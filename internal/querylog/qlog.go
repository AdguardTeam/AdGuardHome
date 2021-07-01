// Package querylog provides query log functions and interfaces.
package querylog

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

const (
	queryLogFileName = "querylog.json" // .gz added during compression
)

// queryLog is a structure that writes and reads the DNS query log
type queryLog struct {
	findClient func(ids []string) (c *Client, err error)

	conf    *Config
	lock    sync.Mutex
	logFile string // path to the log file

	// bufferLock protects buffer.
	bufferLock sync.RWMutex
	// buffer contains recent log entries.
	buffer []*logEntry

	fileFlushLock sync.Mutex // synchronize a file-flushing goroutine and main thread
	flushPending  bool       // don't start another goroutine while the previous one is still running
	fileWriteLock sync.Mutex
}

// ClientProto values are names of the client protocols.
type ClientProto string

// Client protocol names.
const (
	ClientProtoDoH      ClientProto = "doh"
	ClientProtoDoQ      ClientProto = "doq"
	ClientProtoDoT      ClientProto = "dot"
	ClientProtoDNSCrypt ClientProto = "dnscrypt"
	ClientProtoPlain    ClientProto = ""
)

// NewClientProto validates that the client protocol name is valid and returns
// the name as a ClientProto.
func NewClientProto(s string) (cp ClientProto, err error) {
	switch cp = ClientProto(s); cp {
	case
		ClientProtoDoH,
		ClientProtoDoQ,
		ClientProtoDoT,
		ClientProtoDNSCrypt,
		ClientProtoPlain:

		return cp, nil
	default:
		return "", fmt.Errorf("invalid client proto: %q", s)
	}
}

// logEntry - represents a single log entry
type logEntry struct {
	// client is the found client information, if any.
	client *Client

	IP   net.IP    `json:"IP"` // Client IP
	Time time.Time `json:"T"`

	QHost  string `json:"QH"`
	QType  string `json:"QT"`
	QClass string `json:"QC"`

	ClientID    string      `json:"CID,omitempty"`
	ClientProto ClientProto `json:"CP"`

	Answer     []byte `json:",omitempty"` // sometimes empty answers happen like binerdunt.top or rev2.globalrootservers.net
	OrigAnswer []byte `json:",omitempty"`

	Result   filtering.Result
	Elapsed  time.Duration
	Upstream string `json:",omitempty"` // if empty, means it was cached
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

func checkInterval(ivl time.Duration) (ok bool) {
	// The constants for possible values of query log's rotation interval.
	const (
		quarterDay  = 6 * time.Hour
		day         = 24 * time.Hour
		week        = day * 7
		month       = day * 30
		threeMonths = day * 90
	)

	return ivl == quarterDay || ivl == day || ivl == week || ivl == month || ivl == threeMonths
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

	oldLogFile := l.logFile + ".1"
	err := os.Remove(oldLogFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("removing old log file %q: %s", oldLogFile, err)
	}

	err = os.Remove(l.logFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("removing log file %q: %s", l.logFile, err)
	}

	log.Debug("Query log: cleared")
}

func (l *queryLog) Add(params AddParams) {
	var err error

	if !l.conf.Enabled {
		return
	}

	err = params.validate()
	if err != nil {
		log.Error("querylog: adding record: %s, skipping", err)

		return
	}

	if params.Result == nil {
		params.Result = &filtering.Result{}
	}

	now := time.Now()
	entry := logEntry{
		IP:   l.getClientIP(params.ClientIP),
		Time: now,

		Result:      *params.Result,
		Elapsed:     params.Elapsed,
		Upstream:    params.Upstream,
		ClientID:    params.ClientID,
		ClientProto: params.ClientProto,
	}
	q := params.Question.Question[0]
	entry.QHost = strings.ToLower(q.Name[:len(q.Name)-1]) // remove the last dot
	entry.QType = dns.Type(q.Qtype).String()
	entry.QClass = dns.Class(q.Qclass).String()

	if params.Answer != nil {
		var a []byte
		a, err = params.Answer.Pack()
		if err != nil {
			log.Error("querylog: Answer.Pack(): %s", err)

			return
		}

		entry.Answer = a
	}

	if params.OrigAnswer != nil {
		var a []byte
		a, err = params.OrigAnswer.Pack()
		if err != nil {
			log.Error("querylog: OrigAnswer.Pack(): %s", err)

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
