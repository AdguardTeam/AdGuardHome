package dnsforward

import (
	"fmt"
	"net"
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
	logBufferCap           = 5000            // maximum capacity of logBuffer before it's flushed to disk
	queryLogTimeLimit      = time.Hour * 24  // how far in the past we care about querylogs
	queryLogRotationPeriod = time.Hour * 24  // rotate the log every 24 hours
	queryLogFileName       = "querylog.json" // .gz added during compression
	queryLogSize           = 5000            // maximum API response for /querylog
	queryLogTopSize        = 500             // Keep in memory only top N values
)

// queryLog is a structure that writes and reads the DNS query log
type queryLog struct {
	logFile string // path to the log file

	logBufferLock sync.RWMutex
	logBuffer     []*logEntry
	fileFlushLock sync.Mutex // synchronize a file-flushing goroutine and main thread
	flushPending  bool       // don't start another goroutine while the previous one is still running

	queryLogCache []*logEntry
	queryLogLock  sync.RWMutex
}

// newQueryLog creates a new instance of the query log
func newQueryLog(baseDir string) *queryLog {
	l := &queryLog{
		logFile: filepath.Join(baseDir, queryLogFileName),
	}
	return l
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

func (l *queryLog) logRequest(question *dns.Msg, answer *dns.Msg, result *dnsfilter.Result, elapsed time.Duration, addr net.Addr, upstream string) *logEntry {
	var q []byte
	var a []byte
	var err error
	ip := GetIPString(addr)

	if question != nil {
		q, err = question.Pack()
		if err != nil {
			log.Printf("failed to pack question for querylog: %s", err)
			return nil
		}
	}

	if answer != nil {
		a, err = answer.Pack()
		if err != nil {
			log.Printf("failed to pack answer for querylog: %s", err)
			return nil
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
	l.queryLogLock.Lock()
	l.queryLogCache = append(l.queryLogCache, &entry)
	if len(l.queryLogCache) > queryLogSize {
		toremove := len(l.queryLogCache) - queryLogSize
		l.queryLogCache = l.queryLogCache[toremove:]
	}
	l.queryLogLock.Unlock()

	// if buffer needs to be flushed to disk, do it now
	if needFlush {
		// write to file
		// do it in separate goroutine -- we are stalling DNS response this whole time
		go l.flushLogBuffer(false) // nolint
	}

	return &entry
}

// getQueryLogJson returns a map with the current query log ready to be converted to a JSON
func (l *queryLog) getQueryLog() []map[string]interface{} {
	l.queryLogLock.RLock()
	values := make([]*logEntry, len(l.queryLogCache))
	copy(values, l.queryLogCache)
	l.queryLogLock.RUnlock()

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
