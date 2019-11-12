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
	logBufferCap     = 5000            // maximum capacity of buffer before it's flushed to disk
	queryLogFileName = "querylog.json" // .gz added during compression
	getDataLimit     = 500             // GetData(): maximum log entries to return

	// maximum data chunks to parse when filtering entries
	maxFilteringChunks = 10
)

// queryLog is a structure that writes and reads the DNS query log
type queryLog struct {
	conf    Config
	logFile string // path to the log file

	bufferLock    sync.RWMutex
	buffer        []*logEntry
	fileFlushLock sync.Mutex // synchronize a file-flushing goroutine and main thread
	flushPending  bool       // don't start another goroutine while the previous one is still running
	fileWriteLock sync.Mutex
}

// create a new instance of the query log
func newQueryLog(conf Config) *queryLog {
	l := queryLog{}
	l.logFile = filepath.Join(conf.BaseDir, queryLogFileName)
	l.conf = conf
	if !checkInterval(l.conf.Interval) {
		l.conf.Interval = 1
	}
	if l.conf.HTTPRegister != nil {
		l.initWeb()
	}
	go l.periodicRotate()
	return &l
}

func (l *queryLog) Close() {
	_ = l.flushLogBuffer(true)
}

func checkInterval(days uint32) bool {
	return days == 1 || days == 7 || days == 30 || days == 90
}

// Set new configuration at runtime
func (l *queryLog) configure(conf Config) {
	l.conf.Enabled = conf.Enabled
	l.conf.Interval = conf.Interval
}

func (l *queryLog) WriteDiskConfig(dc *DiskConfig) {
	dc.Enabled = l.conf.Enabled
	dc.Interval = l.conf.Interval
}

// Clear memory buffer and remove log files
func (l *queryLog) clear() {
	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	l.bufferLock.Lock()
	l.buffer = nil
	l.flushPending = false
	l.bufferLock.Unlock()

	err := os.Remove(l.logFile + ".1")
	if err != nil && !os.IsNotExist(err) {
		log.Error("file remove: %s: %s", l.logFile+".1", err)
	}

	err = os.Remove(l.logFile)
	if err != nil && !os.IsNotExist(err) {
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
	if !l.conf.Enabled {
		return
	}

	var q []byte
	var a []byte
	var err error
	ip := getIPString(addr)

	if question == nil {
		return
	}

	q, err = question.Pack()
	if err != nil {
		log.Printf("failed to pack question for querylog: %s", err)
		return
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

	l.bufferLock.Lock()
	l.buffer = append(l.buffer, &entry)
	needFlush := false
	if !l.flushPending {
		needFlush = len(l.buffer) >= logBufferCap
		if needFlush {
			l.flushPending = true
		}
	}
	l.bufferLock.Unlock()

	// if buffer needs to be flushed to disk, do it now
	if needFlush {
		// write to file
		// do it in separate goroutine -- we are stalling DNS response this whole time
		go l.flushLogBuffer(false) // nolint
	}
}

// Return TRUE if this entry is needed
func isNeeded(entry *logEntry, params getDataParams) bool {
	if params.ResponseStatus == responseStatusFiltered && !entry.Result.IsFiltered {
		return false
	}

	if len(params.Domain) != 0 || params.QuestionType != 0 {
		m := dns.Msg{}
		_ = m.Unpack(entry.Question)

		if params.QuestionType != 0 {
			if m.Question[0].Qtype != params.QuestionType {
				return false
			}
		}

		if len(params.Domain) != 0 && params.StrictMatchDomain {
			if m.Question[0].Name != params.Domain {
				return false
			}
		} else if len(params.Domain) != 0 {
			if strings.Index(m.Question[0].Name, params.Domain) == -1 {
				return false
			}
		}
	}

	if len(params.Client) != 0 && params.StrictMatchClient {
		if entry.IP != params.Client {
			return false
		}
	} else if len(params.Client) != 0 {
		if strings.Index(entry.IP, params.Client) == -1 {
			return false
		}
	}

	return true
}

func (l *queryLog) readFromFile(params getDataParams) ([]*logEntry, int) {
	entries := []*logEntry{}
	olderThan := params.OlderThan
	totalChunks := 0
	total := 0

	r := l.OpenReader()
	if r == nil {
		return entries, 0
	}
	r.BeginRead(olderThan, getDataLimit)
	for totalChunks < maxFilteringChunks {
		first := true
		newEntries := []*logEntry{}
		for {
			entry := r.Next()
			if entry == nil {
				break
			}
			total++

			if first {
				first = false
				olderThan = entry.Time
			}

			if !isNeeded(entry, params) {
				continue
			}
			if len(newEntries) == getDataLimit {
				newEntries = newEntries[1:]
			}
			newEntries = append(newEntries, entry)
		}

		log.Debug("entries: +%d (%d)  older-than:%s", len(newEntries), len(entries), olderThan)

		entries = append(newEntries, entries...)
		if len(entries) > getDataLimit {
			toremove := len(entries) - getDataLimit
			entries = entries[toremove:]
			break
		}
		if first || len(entries) == getDataLimit {
			break
		}
		totalChunks++
		r.BeginReadPrev(olderThan, getDataLimit)
	}

	r.Close()
	return entries, total
}

// Parameters for getData()
type getDataParams struct {
	OlderThan         time.Time          // return entries that are older than this value
	Domain            string             // filter by domain name in question
	Client            string             // filter by client IP
	QuestionType      uint16             // filter by question type
	ResponseStatus    responseStatusType // filter by response status
	StrictMatchDomain bool               // if Domain value must be matched strictly
	StrictMatchClient bool               // if Client value must be matched strictly
}

// Response status
type responseStatusType int32

// Response status constants
const (
	responseStatusAll responseStatusType = iota + 1
	responseStatusFiltered
)

// Get log entries
func (l *queryLog) getData(params getDataParams) []map[string]interface{} {
	var data = []map[string]interface{}{}

	if len(params.Domain) != 0 && params.StrictMatchDomain {
		params.Domain = params.Domain + "."
	}

	now := time.Now()
	entries := []*logEntry{}
	total := 0

	// add from file
	entries, total = l.readFromFile(params)

	if params.OlderThan.IsZero() {
		params.OlderThan = now
	}

	// add from memory buffer
	l.bufferLock.Lock()
	total += len(l.buffer)
	for _, entry := range l.buffer {

		if !isNeeded(entry, params) {
			continue
		}

		if entry.Time.UnixNano() >= params.OlderThan.UnixNano() {
			break
		}

		if len(entries) == getDataLimit {
			entries = entries[1:]
		}
		entries = append(entries, entry)
	}
	l.bufferLock.Unlock()

	// process the elements from latest to oldest
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		var q *dns.Msg
		var a *dns.Msg

		if len(entry.Question) == 0 {
			continue
		}
		q = new(dns.Msg)
		if err := q.Unpack(entry.Question); err != nil {
			log.Tracef("q.Unpack(): %s", err)
			continue
		}
		if len(q.Question) != 1 {
			log.Tracef("len(q.Question) != 1")
			continue
		}

		if len(entry.Answer) > 0 {
			a = new(dns.Msg)
			if err := a.Unpack(entry.Answer); err != nil {
				log.Debug("Failed to unpack dns message answer: %s", err)
				a = nil
			}
		}

		jsonEntry := map[string]interface{}{
			"reason":    entry.Result.Reason.String(),
			"elapsedMs": strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
			"time":      entry.Time.Format(time.RFC3339Nano),
			"client":    entry.IP,
		}
		jsonEntry["question"] = map[string]interface{}{
			"host":  strings.ToLower(strings.TrimSuffix(q.Question[0].Name, ".")),
			"type":  dns.Type(q.Question[0].Qtype).String(),
			"class": dns.Class(q.Question[0].Qclass).String(),
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

	log.Debug("QueryLog: prepared data (%d/%d) older than %s in %s",
		len(entries), total, params.OlderThan, time.Since(now))
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
