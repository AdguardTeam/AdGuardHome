package querylog

import (
	"bytes"
	"encoding/json"
	"os"
	"time"

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

	var err error
	var zb bytes.Buffer
	filename := l.logFile
	zb = b

	l.fileWriteLock.Lock()
	defer l.fileWriteLock.Unlock()
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

	log.Debug("querylog: ok \"%s\": %v bytes written", filename, n)

	return nil
}

func (l *queryLog) rotate() error {
	from := l.logFile
	to := l.logFile + ".1"

	if _, err := os.Stat(from); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		return nil
	}

	err := os.Rename(from, to)
	if err != nil {
		log.Error("querylog: failed to rename file: %s", err)
		return err
	}

	log.Debug("querylog: renamed %s -> %s", from, to)
	return nil
}

func (l *queryLog) readFileFirstTimeValue() int64 {
	f, err := os.Open(l.logFile)
	if err != nil {
		return -1
	}
	defer f.Close()

	buf := make([]byte, 500)
	r, err := f.Read(buf)
	if err != nil {
		return -1
	}
	buf = buf[:r]

	val := readJSONValue(string(buf), "T")
	t, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		return -1
	}

	log.Debug("querylog: the oldest log entry: %s", val)
	return t.Unix()
}

func (l *queryLog) periodicRotate() {
	intervalSeconds := uint64(l.conf.Interval) * 24 * 60 * 60
	for {
		oldest := l.readFileFirstTimeValue()
		if uint64(oldest)+intervalSeconds <= uint64(time.Now().Unix()) {
			_ = l.rotate()
		}

		time.Sleep(24 * time.Hour)
	}
}
