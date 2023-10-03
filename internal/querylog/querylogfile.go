package querylog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// flushLogBuffer flushes the current buffer to file and resets the current
// buffer.
func (l *queryLog) flushLogBuffer() (err error) {
	defer func() { err = errors.Annotate(err, "flushing log buffer: %w") }()
	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	b, err := l.encodeEntries()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	return l.flushToFile(b)
}

// encodeEntries returns JSON encoded log entries, logs estimated time, clears
// the log buffer.
func (l *queryLog) encodeEntries() (b *bytes.Buffer, err error) {
	l.bufferLock.Lock()
	defer l.bufferLock.Unlock()

	bufLen := l.buffer.Len()
	if bufLen == 0 {
		return nil, errors.Error("nothing to write to a file")
	}

	start := time.Now()

	b = &bytes.Buffer{}
	e := json.NewEncoder(b)

	l.buffer.Range(func(entry *logEntry) (cont bool) {
		err = e.Encode(entry)

		return err == nil
	})

	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	elapsed := time.Since(start)
	log.Debug("%d elements serialized via json in %v: %d kB, %v/entry, %v/entry", bufLen, elapsed, b.Len()/1024, float64(b.Len())/float64(bufLen), elapsed/time.Duration(bufLen))

	l.buffer.Clear()
	l.flushPending = false

	return b, nil
}

// flushToFile saves the encoded log entries to the query log file.
func (l *queryLog) flushToFile(b *bytes.Buffer) (err error) {
	l.fileWriteLock.Lock()
	defer l.fileWriteLock.Unlock()

	filename := l.logFile

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", filename, err)
	}

	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	n, err := f.Write(b.Bytes())
	if err != nil {
		return fmt.Errorf("writing to file %q: %w", filename, err)
	}

	log.Debug("querylog: ok %q: %v bytes written", filename, n)

	return nil
}

func (l *queryLog) rotate() error {
	from := l.logFile
	to := l.logFile + ".1"

	err := os.Rename(from, to)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug("querylog: no log to rotate")

			return nil
		}

		return fmt.Errorf("failed to rename old file: %w", err)
	}

	log.Debug("querylog: renamed %s into %s", from, to)

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

	l.checkAndRotate()

	// rotationCheckIvl is the period of time between checking the need for
	// rotating log files.  It's smaller of any available rotation interval to
	// increase time accuracy.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/3823.
	const rotationCheckIvl = 1 * time.Hour

	rotations := time.NewTicker(rotationCheckIvl)
	defer rotations.Stop()

	for range rotations.C {
		l.checkAndRotate()
	}
}

// checkAndRotate rotates log files if those are older than the specified
// rotation interval.
func (l *queryLog) checkAndRotate() {
	var rotationIvl time.Duration
	func() {
		l.confMu.RLock()
		defer l.confMu.RUnlock()

		rotationIvl = l.conf.RotationIvl
	}()

	oldest, err := l.readFileFirstTimeValue()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("querylog: reading oldest record for rotation: %s", err)

		return
	}

	if rotTime, now := oldest.Add(rotationIvl), time.Now(); rotTime.After(now) {
		log.Debug(
			"querylog: %s <= %s, not rotating",
			now.Format(time.RFC3339),
			rotTime.Format(time.RFC3339),
		)

		return
	}

	err = l.rotate()
	if err != nil {
		log.Error("querylog: rotating: %s", err)

		return
	}

	log.Debug("querylog: rotated successfully")
}
