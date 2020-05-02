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
	queryLogFileName = "querylog.json" // .gz added during compression
	getDataLimit     = 500             // GetData(): maximum log entries to return

	// maximum entries to parse when searching
	maxSearchEntries = 50000
)

// queryLog is a structure that writes and reads the DNS query log
type queryLog struct {
	conf    *Config
	lock    sync.Mutex
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
	l.conf = &Config{}
	*l.conf = conf
	if !checkInterval(l.conf.Interval) {
		l.conf.Interval = 1
	}
	return &l
}

func (l *queryLog) Start() {
	if l.conf.HTTPRegister != nil {
		l.initWeb()
	}
	go l.periodicRotate()
}

func (l *queryLog) Close() {
	_ = l.flushLogBuffer(true)
}

func checkInterval(days uint32) bool {
	return days == 1 || days == 7 || days == 30 || days == 90
}

func (l *queryLog) WriteDiskConfig(dc *DiskConfig) {
	dc.Enabled = l.conf.Enabled
	dc.Interval = l.conf.Interval
	dc.MemSize = l.conf.MemSize
	dc.AnonymizeClientIP = l.conf.AnonymizeClientIP
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
	IP   string    `json:"IP"`
	Time time.Time `json:"T"`

	QHost  string `json:"QH"`
	QType  string `json:"QT"`
	QClass string `json:"QC"`

	Answer     []byte `json:",omitempty"` // sometimes empty answers happen like binerdunt.top or rev2.globalrootservers.net
	OrigAnswer []byte `json:",omitempty"`

	Result   dnsfilter.Result
	Elapsed  time.Duration
	Upstream string `json:",omitempty"` // if empty, means it was cached
}

func (l *queryLog) Add(params AddParams) {
	if !l.conf.Enabled {
		return
	}

	if params.Question == nil || len(params.Question.Question) != 1 || len(params.Question.Question[0].Name) == 0 ||
		params.ClientIP == nil {
		return
	}

	if params.Result == nil {
		params.Result = &dnsfilter.Result{}
	}

	now := time.Now()
	entry := logEntry{
		IP:   l.getClientIP(params.ClientIP.String()),
		Time: now,

		Result:   *params.Result,
		Elapsed:  params.Elapsed,
		Upstream: params.Upstream,
	}
	q := params.Question.Question[0]
	entry.QHost = strings.ToLower(q.Name[:len(q.Name)-1]) // remove the last dot
	entry.QType = dns.Type(q.Qtype).String()
	entry.QClass = dns.Class(q.Qclass).String()

	if params.Answer != nil {
		a, err := params.Answer.Pack()
		if err != nil {
			log.Info("Querylog: Answer.Pack(): %s", err)
			return
		}
		entry.Answer = a
	}

	if params.OrigAnswer != nil {
		a, err := params.OrigAnswer.Pack()
		if err != nil {
			log.Info("Querylog: OrigAnswer.Pack(): %s", err)
			return
		}
		entry.OrigAnswer = a
	}

	l.bufferLock.Lock()
	l.buffer = append(l.buffer, &entry)
	needFlush := false
	if !l.flushPending {
		needFlush = len(l.buffer) >= int(l.conf.MemSize)
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

// Parameters for getData()
type getDataParams struct {
	OlderThan         time.Time          // return entries that are older than this value
	Domain            string             // filter by domain name in question
	Client            string             // filter by client IP
	QuestionType      string             // filter by question type
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

// Gets log entries
func (l *queryLog) getData(params getDataParams) map[string]interface{} {
	now := time.Now()

	if len(params.Client) != 0 && l.conf.AnonymizeClientIP {
		params.Client = l.getClientIP(params.Client)
	}

	// add from file
	fileEntries, oldest, total := l.searchFiles(params)

	if params.OlderThan.IsZero() {
		// In case if the timer is not precise (for instance, on Windows)
		// We really want to get all records including those added just before the call
		params.OlderThan = now.Add(time.Millisecond)
	}

	// add from memory buffer
	l.bufferLock.Lock()
	total += len(l.buffer)
	memoryEntries := make([]*logEntry, 0)

	// go through the buffer in the reverse order
	// from NEWER to OLDER
	for i := len(l.buffer) - 1; i >= 0; i-- {
		entry := l.buffer[i]

		if entry.Time.UnixNano() >= params.OlderThan.UnixNano() {
			// Ignore entries newer than what was requested
			continue
		}

		if !matchesGetDataParams(entry, params) {
			continue
		}

		memoryEntries = append(memoryEntries, entry)
	}
	l.bufferLock.Unlock()

	// now let's get a unified collection
	entries := append(memoryEntries, fileEntries...)
	if len(entries) > getDataLimit {
		// remove extra records
		entries = entries[:getDataLimit]
	}
	if len(entries) == getDataLimit {
		// change the "oldest" value here.
		// we cannot use the "oldest" we got from "searchFiles" anymore
		// because after adding in-memory records and removing extra records
		// the situation has changed
		oldest = entries[len(entries)-1].Time
	}

	// init the response object
	var data = []map[string]interface{}{}

	// the elements order is already reversed (from newer to older)
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		jsonEntry := l.logEntryToJSONEntry(entry)
		data = append(data, jsonEntry)
	}

	log.Debug("QueryLog: prepared data (%d/%d) older than %s in %s",
		len(entries), total, params.OlderThan, time.Since(now))

	var result = map[string]interface{}{}
	result["oldest"] = ""
	if !oldest.IsZero() {
		result["oldest"] = oldest.Format(time.RFC3339Nano)
	}
	result["data"] = data
	return result
}

// Get Client IP address
func (l *queryLog) getClientIP(clientIP string) string {
	if l.conf.AnonymizeClientIP {
		ip := net.ParseIP(clientIP)
		if ip != nil {
			ip4 := ip.To4()
			const AnonymizeClientIP4Mask = 24
			const AnonymizeClientIP6Mask = 112
			if ip4 != nil {
				clientIP = ip4.Mask(net.CIDRMask(AnonymizeClientIP4Mask, 32)).String()
			} else {
				clientIP = ip.Mask(net.CIDRMask(AnonymizeClientIP6Mask, 128)).String()
			}
		}
	}

	return clientIP
}

func (l *queryLog) logEntryToJSONEntry(entry *logEntry) map[string]interface{} {
	var msg *dns.Msg

	if len(entry.Answer) > 0 {
		msg = new(dns.Msg)
		if err := msg.Unpack(entry.Answer); err != nil {
			log.Debug("Failed to unpack dns message answer: %s: %s", err, string(entry.Answer))
			msg = nil
		}
	}

	jsonEntry := map[string]interface{}{
		"reason":    entry.Result.Reason.String(),
		"elapsedMs": strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
		"time":      entry.Time.Format(time.RFC3339Nano),
		"client":    l.getClientIP(entry.IP),
	}
	jsonEntry["question"] = map[string]interface{}{
		"host":  entry.QHost,
		"type":  entry.QType,
		"class": entry.QClass,
	}

	if msg != nil {
		jsonEntry["status"] = dns.RcodeToString[msg.Rcode]

		opt := msg.IsEdns0()
		dnssecOk := false
		if opt != nil {
			dnssecOk = opt.Do()
		}
		jsonEntry["answer_dnssec"] = dnssecOk
	}

	if len(entry.Result.Rule) > 0 {
		jsonEntry["rule"] = entry.Result.Rule
		jsonEntry["filterId"] = entry.Result.FilterID
	}

	if len(entry.Result.ServiceName) != 0 {
		jsonEntry["service_name"] = entry.Result.ServiceName
	}

	answers := answerToMap(msg)
	if answers != nil {
		jsonEntry["answer"] = answers
	}

	if len(entry.OrigAnswer) != 0 {
		a := new(dns.Msg)
		err := a.Unpack(entry.OrigAnswer)
		if err == nil {
			answers = answerToMap(a)
			if answers != nil {
				jsonEntry["original_answer"] = answers
			}
		} else {
			log.Debug("Querylog: msg.Unpack(entry.OrigAnswer): %s: %s", err, string(entry.OrigAnswer))
		}
	}

	return jsonEntry
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
			answer["value"] = v.A.String()
		case *dns.AAAA:
			answer["value"] = v.AAAA.String()
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
