package querylog

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

const (
	logBufferCap     = 5000            // maximum capacity of logBuffer before it's flushed to disk
	queryLogFileName = "querylog.json" // .gz added during compression
	queryLogSize     = 5000            // maximum API response for /querylog
)

// queryLog is a structure that writes and reads the DNS query log
type queryLog struct {
	conf    Config
	logFile string // path to the log file

	logBufferLock sync.RWMutex
	logBuffer     []*logEntry
	fileFlushLock sync.Mutex // synchronize a file-flushing goroutine and main thread
	flushPending  bool       // don't start another goroutine while the previous one is still running

	cache []*logEntry
	lock  sync.RWMutex
}

// create a new instance of the query log
func newQueryLog(conf Config) *queryLog {
	l := queryLog{}
	l.logFile = filepath.Join(conf.BaseDir, queryLogFileName)
	l.conf = conf
	go l.periodicQueryLogRotate()
	go l.fillFromFile()
	return &l
}

func (l *queryLog) Close() {
	_ = l.flushLogBuffer(true)
}

func (l *queryLog) Configure(conf Config) {
	l.conf = conf
}

func (l *queryLog) Clear() {
	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	l.logBufferLock.Lock()
	l.logBuffer = nil
	l.flushPending = false
	l.logBufferLock.Unlock()

	l.lock.Lock()
	l.cache = nil
	l.lock.Unlock()

	err := os.Remove(l.logFile + ".1")
	if err != nil {
		log.Error("file remove: %s: %s", l.logFile+".1", err)
	}

	err = os.Remove(l.logFile)
	if err != nil {
		log.Error("file remove: %s: %s", l.logFile, err)
	}

	log.Debug("Query log: cleared")
}

type logEntry struct {
	Question []byte
	Answer   []byte `json:",omitempty"` // sometimes empty answers happen like binerdunt.top or rev2.globalrootservers.net
	Result   dnsfilter.Result
	Time     time.Time
	Elapsed  time.Duration
	IP       string
	Upstream string `json:",omitempty"` // if empty, means it was cached
}

// getIPString is a helper function that extracts IP address from net.Addr
func getIPString(addr net.Addr) string {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.TCPAddr:
		return addr.IP.String()
	}
	return ""
}

func (l *queryLog) Add(question *dns.Msg, answer *dns.Msg, result *dnsfilter.Result, elapsed time.Duration, addr net.Addr, upstream string) {
	var q []byte
	var a []byte
	var err error
	ip := getIPString(addr)

	if question != nil {
		q, err = question.Pack()
		if err != nil {
			log.Printf("failed to pack question for querylog: %s", err)
			return
		}
	}

	if answer != nil {
		a, err = answer.Pack()
		if err != nil {
			log.Printf("failed to pack answer for querylog: %s", err)
			return
		}
	}

	if result == nil {
		result = &dnsfilter.Result{}
	}

	now := time.Now()
	entry := logEntry{
		Question: q,
		Answer:   a,
		Result:   *result,
		Time:     now,
		Elapsed:  elapsed,
		IP:       ip,
		Upstream: upstream,
	}

	l.logBufferLock.Lock()
	l.logBuffer = append(l.logBuffer, &entry)
	needFlush := false
	if !l.flushPending {
		needFlush = len(l.logBuffer) >= logBufferCap
		if needFlush {
			l.flushPending = true
		}
	}
	l.logBufferLock.Unlock()
	l.lock.Lock()
	l.cache = append(l.cache, &entry)
	if len(l.cache) > queryLogSize {
		toremove := len(l.cache) - queryLogSize
		l.cache = l.cache[toremove:]
	}
	l.lock.Unlock()

	// if buffer needs to be flushed to disk, do it now
	if needFlush {
		// write to file
		// do it in separate goroutine -- we are stalling DNS response this whole time
		go l.flushLogBuffer(false) // nolint
	}
}

func (l *queryLog) GetData() []map[string]interface{} {
	l.lock.RLock()
	values := make([]*logEntry, len(l.cache))
	copy(values, l.cache)
	l.lock.RUnlock()

	// reverse it so that newest is first
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}

	// iterate
	var data = []map[string]interface{}{}
	for _, entry := range values {
		var q *dns.Msg
		var a *dns.Msg

		if len(entry.Question) > 0 {
			q = new(dns.Msg)
			if err := q.Unpack(entry.Question); err != nil {
				// ignore, log and move on
				log.Printf("Failed to unpack dns message question: %s", err)
				q = nil
			}
		}
		if len(entry.Answer) > 0 {
			a = new(dns.Msg)
			if err := a.Unpack(entry.Answer); err != nil {
				// ignore, log and move on
				log.Printf("Failed to unpack dns message question: %s", err)
				a = nil
			}
		}

		jsonEntry := map[string]interface{}{
			"reason":    entry.Result.Reason.String(),
			"elapsedMs": strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
			"time":      entry.Time.Format(time.RFC3339),
			"client":    entry.IP,
		}
		if q != nil {
			jsonEntry["question"] = map[string]interface{}{
				"host":  strings.ToLower(strings.TrimSuffix(q.Question[0].Name, ".")),
				"type":  dns.Type(q.Question[0].Qtype).String(),
				"class": dns.Class(q.Question[0].Qclass).String(),
			}
		}

		if a != nil {
			jsonEntry["status"] = dns.RcodeToString[a.Rcode]
		}
		if len(entry.Result.Rule) > 0 {
			jsonEntry["rule"] = entry.Result.Rule
			jsonEntry["filterId"] = entry.Result.FilterID
		}

		if len(entry.Result.ServiceName) != 0 {
			jsonEntry["service_name"] = entry.Result.ServiceName
		}

		answers := answerToMap(a)
		if answers != nil {
			jsonEntry["answer"] = answers
		}

		data = append(data, jsonEntry)
	}

	return data
}

func answerToMap(a *dns.Msg) []map[string]interface{} {
	if a == nil || len(a.Answer) == 0 {
		return nil
	}

	var answers = []map[string]interface{}{}
	for _, k := range a.Answer {
		header := k.Header()
		answer := map[string]interface{}{
			"type": dns.TypeToString[header.Rrtype],
			"ttl":  header.Ttl,
		}
		// try most common record types
		switch v := k.(type) {
		case *dns.A:
			answer["value"] = v.A
		case *dns.AAAA:
			answer["value"] = v.AAAA
		case *dns.MX:
			answer["value"] = fmt.Sprintf("%v %v", v.Preference, v.Mx)
		case *dns.CNAME:
			answer["value"] = v.Target
		case *dns.NS:
			answer["value"] = v.Ns
		case *dns.SPF:
			answer["value"] = v.Txt
		case *dns.TXT:
			answer["value"] = v.Txt
		case *dns.PTR:
			answer["value"] = v.Ptr
		case *dns.SOA:
			answer["value"] = fmt.Sprintf("%v %v %v %v %v %v %v", v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
		case *dns.CAA:
			answer["value"] = fmt.Sprintf("%v %v \"%v\"", v.Flag, v.Tag, v.Value)
		case *dns.HINFO:
			answer["value"] = fmt.Sprintf("\"%v\" \"%v\"", v.Cpu, v.Os)
		case *dns.RRSIG:
			answer["value"] = fmt.Sprintf("%v %v %v %v %v %v %v %v %v", dns.TypeToString[v.TypeCovered], v.Algorithm, v.Labels, v.OrigTtl, v.Expiration, v.Inception, v.KeyTag, v.SignerName, v.Signature)
		default:
			// type unknown, marshall it as-is
			answer["value"] = v
		}
		answers = append(answers, answer)
	}

	return answers
}
