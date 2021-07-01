package querylog

import (
	"bytes"
	"encoding/json"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// flushLogBuffer flushes the current buffer to file and resets the current buffer
func (l *queryLog) flushLogBuffer(fullFlush bool) error {
	if !l.conf.FileEnabled {
		return nil
	}

	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	// flush remainder to file
	l.bufferLock.Lock()
	needFlush := len(l.buffer) >= int(l.conf.MemSize)
	if !needFlush && !fullFlush {
		l.bufferLock.Unlock()
		return nil
	}
	flushBuffer := l.buffer
	l.buffer = nil
	l.flushPending = false
	l.bufferLock.Unlock()
	err := l.flushToFile(flushBuffer)
	if err != nil {
		log.Error("Saving querylog to file failed: %s", err)
		return err
	}
	return nil
}

// flushToFile saves the specified log entries to the query log file
func (l *queryLog) flushToFile(buffer []*logEntry) (err error) {
	if len(buffer) == 0 {
		log.Debug("querylog: there's nothing to write to a file")
		return nil
	}
	start := time.Now()

	var b bytes.Buffer
	e := json.NewEncoder(&b)
	for _, entry := range buffer {
		err = e.Encode(entry)
		if err != nil {
			log.Error("Failed to marshal entry: %s", err)

			return err
		}
	}

	elapsed := time.Since(start)
	log.Debug("%d elements serialized via json in %v: %d kB, %v/entry, %v/entry", len(buffer), elapsed, b.Len()/1024, float64(b.Len())/float64(len(buffer)), elapsed/time.Duration(len(buffer)))

	var zb bytes.Buffer
	filename := l.logFile
	zb = b

	l.fileWriteLock.Lock()
	defer l.fileWriteLock.Unlock()
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		log.Error("failed to create file \"%s\": %s", filename, err)
		return err
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	n, err := f.Write(zb.Bytes())
	if err != nil {
		log.Error("Couldn't write to file: %s", err)
		return err
	}

	log.Debug("querylog: ok \"%s\": %v bytes written", filename, n)

	return nil
}

func (l *queryLog) rotate() error {
	from := l.logFile
	to := l.logFile + ".1"

	err := os.Rename(from, to)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		log.Error("querylog: failed to rename file: %s", err)

		return err
	}

	log.Debug("querylog: renamed %s -> %s", from, to)

	return nil
}

func (l *queryLog) readFileFirstTimeValue() (first time.Time, err error) {
	var f *os.File
	f, err = os.Open(l.logFile)
	if err != nil {
		return time.Time{}, err
	}

	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	buf := make([]byte, 512)
	var r int
	r, err = f.Read(buf)
	if err != nil {
		return time.Time{}, err
	}

	val := readJSONValue(string(buf[:r]), `"T":"`)
	t, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		return time.Time{}, err
	}

	log.Debug("querylog: the oldest log entry: %s", val)

	return t, nil
}

func (l *queryLog) periodicRotate() {
	defer log.OnPanic("querylog: rotating")

	var err error
	for {
		var oldest time.Time
		oldest, err = l.readFileFirstTimeValue()
		if err != nil {
			log.Debug("%s", err)
		}

		if oldest.Add(l.conf.RotationIvl).After(time.Now()) {
			err = l.rotate()
			if err != nil {
				log.Debug("%s", err)
			}
		}

		// What?
		time.Sleep(24 * time.Hour)
	}
}
