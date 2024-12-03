package querylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/c2h5oh/datasize"
)

// flushLogBuffer flushes the current buffer to file and resets the current
// buffer.
func (l *queryLog) flushLogBuffer(ctx context.Context) (err error) {
	defer func() { err = errors.Annotate(err, "flushing log buffer: %w") }()

	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	b, err := l.encodeEntries(ctx)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	return l.flushToFile(ctx, b)
}

// encodeEntries returns JSON encoded log entries, logs estimated time, clears
// the log buffer.
func (l *queryLog) encodeEntries(ctx context.Context) (b *bytes.Buffer, err error) {
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

	size := b.Len()
	elapsed := time.Since(start)
	l.logger.DebugContext(
		ctx,
		"serialized elements via json",
		"count", bufLen,
		"elapsed", elapsed,
		"size", datasize.ByteSize(size),
		"size_per_entry", datasize.ByteSize(float64(size)/float64(bufLen)),
		"time_per_entry", elapsed/time.Duration(bufLen),
	)

	l.buffer.Clear()
	l.flushPending = false

	return b, nil
}

// flushToFile saves the encoded log entries to the query log file.
func (l *queryLog) flushToFile(ctx context.Context, b *bytes.Buffer) (err error) {
	l.fileWriteLock.Lock()
	defer l.fileWriteLock.Unlock()

	filename := l.logFile

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, aghos.DefaultPermFile)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", filename, err)
	}

	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	n, err := f.Write(b.Bytes())
	if err != nil {
		return fmt.Errorf("writing to file %q: %w", filename, err)
	}

	l.logger.DebugContext(ctx, "flushed to file", "file", filename, "size", datasize.ByteSize(n))

	return nil
}

func (l *queryLog) rotate(ctx context.Context) error {
	from := l.logFile
	to := l.logFile + ".1"

	err := os.Rename(from, to)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			l.logger.DebugContext(ctx, "no log to rotate")

			return nil
		}

		return fmt.Errorf("failed to rename old file: %w", err)
	}

	l.logger.DebugContext(ctx, "renamed log file", "from", from, "to", to)

	return nil
}

func (l *queryLog) readFileFirstTimeValue(ctx context.Context) (first time.Time, err error) {
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

	l.logger.DebugContext(ctx, "oldest log entry", "entry_time", val)

	return t, nil
}

func (l *queryLog) periodicRotate(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, l.logger)

	l.checkAndRotate(ctx)

	// rotationCheckIvl is the period of time between checking the need for
	// rotating log files.  It's smaller of any available rotation interval to
	// increase time accuracy.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/3823.
	const rotationCheckIvl = 1 * time.Hour

	rotations := time.NewTicker(rotationCheckIvl)
	defer rotations.Stop()

	for range rotations.C {
		l.checkAndRotate(ctx)
	}
}

// checkAndRotate rotates log files if those are older than the specified
// rotation interval.
func (l *queryLog) checkAndRotate(ctx context.Context) {
	var rotationIvl time.Duration
	func() {
		l.confMu.RLock()
		defer l.confMu.RUnlock()

		rotationIvl = l.conf.RotationIvl
	}()

	oldest, err := l.readFileFirstTimeValue(ctx)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		l.logger.ErrorContext(ctx, "reading oldest record for rotation", slogutil.KeyError, err)

		return
	}

	if rotTime, now := oldest.Add(rotationIvl), time.Now(); rotTime.After(now) {
		l.logger.DebugContext(
			ctx,
			"not rotating",
			"now", now.Format(time.RFC3339),
			"rotate_time", rotTime.Format(time.RFC3339),
		)

		return
	}

	err = l.rotate(ctx)
	if err != nil {
		l.logger.ErrorContext(ctx, "rotating", slogutil.KeyError, err)

		return
	}

	l.logger.DebugContext(ctx, "rotated successfully")
}
