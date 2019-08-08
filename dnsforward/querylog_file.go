package dnsforward

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
	for range time.Tick(queryLogRotationPeriod) {
		err := l.rotateQueryLog()
		if err != nil {
			log.Error("Failed to rotate querylog: %s", err)
			// do nothing, continue rotating
		}
	}
}
