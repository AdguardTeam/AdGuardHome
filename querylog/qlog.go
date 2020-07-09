package querylog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

const (
	queryLogFileName = "querylog.json" // .gz added during compression
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

// logEntry - represents a single log entry
type logEntry struct {
	IP   string    `json:"IP"` // Client IP
	Time time.Time `json:"T"`

	QHost  string `json:"QH"`
	QType  string `json:"QT"`
	QClass string `json:"QC"`

	ClientProto string `json:"CP"` // "" or "doh"

	Answer     []byte `json:",omitempty"` // sometimes empty answers happen like binerdunt.top or rev2.globalrootservers.net
	OrigAnswer []byte `json:",omitempty"`

	Result   dnsfilter.Result
	Elapsed  time.Duration
	Upstream string `json:",omitempty"` // if empty, means it was cached
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

func (l *queryLog) WriteDiskConfig(c *Config) {
	*c = *l.conf
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

		Result:      *params.Result,
		Elapsed:     params.Elapsed,
		Upstream:    params.Upstream,
		ClientProto: params.ClientProto,
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

	if !l.conf.FileEnabled {
		if len(l.buffer) > int(l.conf.MemSize) {
			// writing to file is disabled - just remove the oldest entry from array
			l.buffer = l.buffer[1:]
		}

	} else if !l.flushPending {
		needFlush = len(l.buffer) >= int(l.conf.MemSize)
		if needFlush {
			l.flushPending = true
		}
	}
	l.bufferLock.Unlock()

	// if buffer needs to be flushed to disk, do it now
	if needFlush {
		go func() {
			_ = l.flushLogBuffer(false)
		}()
	}
}
