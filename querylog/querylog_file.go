package querylog

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/go-test/deep"
	"github.com/miekg/dns"
)

var (
	fileWriteLock sync.Mutex
)

const enableGzip = false

// flushLogBuffer flushes the current buffer to file and resets the current buffer
func (l *queryLog) flushLogBuffer(fullFlush bool) error {
	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	// flush remainder to file
	l.logBufferLock.Lock()
	needFlush := len(l.logBuffer) >= logBufferCap
	if !needFlush && !fullFlush {
		l.logBufferLock.Unlock()
		return nil
	}
	flushBuffer := l.logBuffer
	l.logBuffer = nil
	l.flushPending = false
	l.logBufferLock.Unlock()
	err := l.flushToFile(flushBuffer)
	if err != nil {
		log.Error("Saving querylog to file failed: %s", err)
		return err
	}
	return nil
}

// flushToFile saves the specified log entries to the query log file
func (l *queryLog) flushToFile(buffer []*logEntry) error {
	if len(buffer) == 0 {
		log.Debug("querylog: there's nothing to write to a file")
		return nil
	}
	start := time.Now()

	var b bytes.Buffer
	e := json.NewEncoder(&b)
	for _, entry := range buffer {
		err := e.Encode(entry)
		if err != nil {
			log.Error("Failed to marshal entry: %s", err)
			return err
		}
	}

	elapsed := time.Since(start)
	log.Debug("%d elements serialized via json in %v: %d kB, %v/entry, %v/entry", len(buffer), elapsed, b.Len()/1024, float64(b.Len())/float64(len(buffer)), elapsed/time.Duration(len(buffer)))

	err := checkBuffer(buffer, b)
	if err != nil {
		log.Error("failed to check buffer: %s", err)
		return err
	}

	var zb bytes.Buffer
	filename := l.logFile

	// gzip enabled?
	if enableGzip {
		filename += ".gz"

		zw := gzip.NewWriter(&zb)
		zw.Name = l.logFile
		zw.ModTime = time.Now()

		_, err = zw.Write(b.Bytes())
		if err != nil {
			log.Error("Couldn't compress to gzip: %s", err)
			zw.Close()
			return err
		}

		if err = zw.Close(); err != nil {
			log.Error("Couldn't close gzip writer: %s", err)
			return err
		}
	} else {
		zb = b
	}

	fileWriteLock.Lock()
	defer fileWriteLock.Unlock()
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Error("failed to create file \"%s\": %s", filename, err)
		return err
	}
	defer f.Close()

	n, err := f.Write(zb.Bytes())
	if err != nil {
		log.Error("Couldn't write to file: %s", err)
		return err
	}

	log.Debug("ok \"%s\": %v bytes written", filename, n)

	return nil
}

func checkBuffer(buffer []*logEntry, b bytes.Buffer) error {
	l := len(buffer)
	d := json.NewDecoder(&b)

	i := 0
	for d.More() {
		entry := &logEntry{}
		err := d.Decode(entry)
		if err != nil {
			log.Error("Failed to decode: %s", err)
			return err
		}
		if diff := deep.Equal(entry, buffer[i]); diff != nil {
			log.Error("decoded buffer differs: %s", diff)
			return fmt.Errorf("decoded buffer differs: %s", diff)
		}
		i++
	}
	if i != l {
		err := fmt.Errorf("check fail: %d vs %d entries", l, i)
		log.Error("%v", err)
		return err
	}
	log.Debug("check ok: %d entries", i)

	return nil
}

func (l *queryLog) rotateQueryLog() error {
	from := l.logFile
	to := l.logFile + ".1"

	if enableGzip {
		from = l.logFile + ".gz"
		to = l.logFile + ".gz.1"
	}

	if _, err := os.Stat(from); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		return nil
	}

	err := os.Rename(from, to)
	if err != nil {
		log.Error("Failed to rename querylog: %s", err)
		return err
	}

	log.Debug("Rotated from %s to %s successfully", from, to)

	return nil
}

func (l *queryLog) periodicQueryLogRotate() {
	for range time.Tick(time.Duration(l.conf.Interval) * time.Hour) {
		err := l.rotateQueryLog()
		if err != nil {
			log.Error("Failed to rotate querylog: %s", err)
			// do nothing, continue rotating
		}
	}
}

// Reader is the DB reader context
type Reader struct {
	f   *os.File
	jd  *json.Decoder
	now time.Time
	ql  *queryLog

	files []string
	ifile int

	count uint64 // returned elements counter
}

// OpenReader locks the file and returns reader object or nil on error
func (l *queryLog) OpenReader() *Reader {
	r := Reader{}
	r.ql = l
	r.now = time.Now()

	return &r
}

// Close closes the reader
func (r *Reader) Close() {
	elapsed := time.Since(r.now)
	var perunit time.Duration
	if r.count > 0 {
		perunit = elapsed / time.Duration(r.count)
	}
	log.Debug("querylog: read %d entries in %v, %v/entry",
		r.count, elapsed, perunit)

	if r.f != nil {
		r.f.Close()
	}
}

// BeginRead starts reading
func (r *Reader) BeginRead() {
	r.files = []string{
		r.ql.logFile,
		r.ql.logFile + ".1",
	}
}

// Next returns the next entry or nil if reading is finished
func (r *Reader) Next() *logEntry { // nolint
	var err error
	for {
		// open file if needed
		if r.f == nil {
			if r.ifile == len(r.files) {
				return nil
			}
			fn := r.files[r.ifile]
			r.f, err = os.Open(fn)
			if err != nil {
				log.Error("Failed to open file \"%s\": %s", fn, err)
				r.ifile++
				continue
			}
		}

		// open decoder if needed
		if r.jd == nil {
			r.jd = json.NewDecoder(r.f)
		}

		// check if there's data
		if !r.jd.More() {
			r.jd = nil
			r.f.Close()
			r.f = nil
			r.ifile++
			continue
		}

		// read data
		var entry logEntry
		err = r.jd.Decode(&entry)
		if err != nil {
			log.Error("Failed to decode: %s", err)
			// next entry can be fine, try more
			continue
		}
		r.count++
		return &entry
	}
}

// Total returns the total number of items
func (r *Reader) Total() int {
	return 0
}

// Fill cache from file
func (l *queryLog) fillFromFile() {
	now := time.Now()
	validFrom := now.Unix() - int64(l.conf.Interval*60*60)
	r := l.OpenReader()
	if r == nil {
		return
	}

	r.BeginRead()

	for {
		entry := r.Next()
		if entry == nil {
			break
		}

		if entry.Time.Unix() < validFrom {
			continue
		}

		if len(entry.Question) == 0 {
			log.Printf("entry question is absent, skipping")
			continue
		}

		if entry.Time.After(now) {
			log.Printf("t %v vs %v is in the future, ignoring", entry.Time, now)
			continue
		}

		q := new(dns.Msg)
		if err := q.Unpack(entry.Question); err != nil {
			log.Printf("failed to unpack dns message question: %s", err)
			continue
		}

		if len(q.Question) != 1 {
			log.Printf("malformed dns message, has no questions, skipping")
			continue
		}

		l.lock.Lock()
		l.cache = append(l.cache, entry)
		if len(l.cache) > queryLogSize {
			toremove := len(l.cache) - queryLogSize
			l.cache = l.cache[toremove:]
		}
		l.lock.Unlock()
	}

	r.Close()
}
