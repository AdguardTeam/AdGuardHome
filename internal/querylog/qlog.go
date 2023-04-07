// Package querylog provides query log functions and interfaces.
package querylog

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
)

const (
	queryLogFileName = "querylog.json" // .gz added during compression
)

// queryLog is a structure that writes and reads the DNS query log
type queryLog struct {
	findClient func(ids []string) (c *Client, err error)

	// confMu protects conf.
	confMu *sync.RWMutex
	conf   *Config

	// logFile is the path to the log file.
	logFile string

	// bufferLock protects buffer.
	bufferLock sync.RWMutex
	// buffer contains recent log entries.  The entries in this buffer must not
	// be modified.
	buffer []*logEntry

	fileFlushLock sync.Mutex // synchronize a file-flushing goroutine and main thread
	flushPending  bool       // don't start another goroutine while the previous one is still running
	fileWriteLock sync.Mutex

	anonymizer *aghnet.IPMut
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

func (l *queryLog) Start() {
	if l.conf.HTTPRegister != nil {
		l.initWeb()
	}

	go l.periodicRotate()
}

func (l *queryLog) Close() {
	l.confMu.RLock()
	defer l.confMu.RUnlock()

	if l.conf.FileEnabled {
		err := l.flushLogBuffer()
		if err != nil {
			log.Error("querylog: closing: %s", err)
		}
	}
}

func checkInterval(ivl time.Duration) (ok bool) {
	// The constants for possible values of query log's rotation interval.
	const (
		quarterDay  = timeutil.Day / 4
		day         = timeutil.Day
		week        = timeutil.Day * 7
		month       = timeutil.Day * 30
		threeMonths = timeutil.Day * 90
	)

	return ivl == quarterDay || ivl == day || ivl == week || ivl == month || ivl == threeMonths
}

// validateIvl returns an error if ivl is less than an hour or more than a
// year.
func validateIvl(ivl time.Duration) (err error) {
	if ivl < time.Hour {
		return errors.Error("less than an hour")
	}

	if ivl > timeutil.Day*365 {
		return errors.Error("more than a year")
	}

	return nil
}

func (l *queryLog) WriteDiskConfig(c *Config) {
	l.confMu.RLock()
	defer l.confMu.RUnlock()

	*c = *l.conf
	c.Ignored = l.conf.Ignored.Clone()
}

// Clear memory buffer and remove log files
func (l *queryLog) clear() {
	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	func() {
		l.bufferLock.Lock()
		defer l.bufferLock.Unlock()

		l.buffer = nil
		l.flushPending = false
	}()

	oldLogFile := l.logFile + ".1"
	err := os.Remove(oldLogFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("removing old log file %q: %s", oldLogFile, err)
	}

	err = os.Remove(l.logFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("removing log file %q: %s", l.logFile, err)
	}

	log.Debug("querylog: cleared")
}

func (l *queryLog) Add(params *AddParams) {
	var isEnabled, fileIsEnabled bool
	var memSize uint32
	func() {
		l.confMu.RLock()
		defer l.confMu.RUnlock()

		isEnabled, fileIsEnabled = l.conf.Enabled, l.conf.FileEnabled
		memSize = l.conf.MemSize
	}()

	if !isEnabled {
		return
	}

	err := params.validate()
	if err != nil {
		log.Error("querylog: adding record: %s, skipping", err)

		return
	}

	if params.Result == nil {
		params.Result = &filtering.Result{}
	}

	now := time.Now()
	q := params.Question.Question[0]
	entry := &logEntry{
		Time: now,

		QHost:  strings.ToLower(q.Name[:len(q.Name)-1]),
		QType:  dns.Type(q.Qtype).String(),
		QClass: dns.Class(q.Qclass).String(),

		ClientID:    params.ClientID,
		ClientProto: params.ClientProto,

		Result:   *params.Result,
		Upstream: params.Upstream,

		IP: params.ClientIP,

		Elapsed: params.Elapsed,

		Cached:            params.Cached,
		AuthenticatedData: params.AuthenticatedData,
	}

	if params.ReqECS != nil {
		entry.ReqECS = params.ReqECS.String()
	}

	entry.addResponse(params.Answer, false)
	entry.addResponse(params.OrigAnswer, true)

	needFlush := false
	func() {
		l.bufferLock.Lock()
		defer l.bufferLock.Unlock()

		l.buffer = append(l.buffer, entry)

		if !fileIsEnabled {
			if len(l.buffer) > int(memSize) {
				// Writing to file is disabled, so just remove the oldest entry
				// from the slices.
				//
				// TODO(a.garipov): This should be replaced by a proper ring
				// buffer, but it's currently difficult to do that.
				l.buffer[0] = nil
				l.buffer = l.buffer[1:]
			}
		} else if !l.flushPending {
			needFlush = len(l.buffer) >= int(memSize)
			if needFlush {
				l.flushPending = true
			}
		}
	}()

	if needFlush {
		go func() {
			flushErr := l.flushLogBuffer()
			if flushErr != nil {
				log.Error("querylog: flushing after adding: %s", err)
			}
		}()
	}
}

// ShouldLog returns true if request for the host should be logged.
func (l *queryLog) ShouldLog(host string, _, _ uint16, ids []string) bool {
	l.confMu.RLock()
	defer l.confMu.RUnlock()

	c, err := l.findClient(ids)
	if err != nil {
		log.Error("querylog: finding client: %s", err)
	}

	if c != nil && c.IgnoreQueryLog {
		return false
	}

	return !l.isIgnored(host)
}

// isIgnored returns true if the host is in the ignored domains list.  It
// assumes that l.confMu is locked for reading.
func (l *queryLog) isIgnored(host string) bool {
	return l.conf.Ignored.Has(host)
}
