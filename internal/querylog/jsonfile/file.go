package jsonfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
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
	oldest, err := l.readFileFirstTimeValue()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("querylog: reading oldest record for rotation: %s", err)

		return
	}

	if rot, now := oldest.Add(l.conf.RotationIvl), time.Now(); rot.After(now) {
		log.Debug(
			"querylog: %s <= %s, not rotating",
			now.Format(time.RFC3339),
			rot.Format(time.RFC3339),
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

// Timestamp not found errors.
const (
	ErrTSNotFound errors.Error = "ts not found"
	ErrTSTooLate  errors.Error = "ts too late"
	ErrTSTooEarly errors.Error = "ts too early"
)

// TODO: Find a way to grow buffer instead of relying on this value when reading strings
const maxEntrySize = 16 * 1024

// buffer should be enough for at least this number of entries
const bufferSize = 100 * maxEntrySize

// QLogFile represents a single query log file
// It allows reading from the file in the reverse order
//
// Please note that this is a stateful object.
// Internally, it contains a pointer to a specific position in the file,
// and it reads lines in reverse order starting from that position.
type QLogFile struct {
	file     *os.File // the query log file
	position int64    // current position in the file

	buffer      []byte // buffer that we've read from the file
	bufferStart int64  // start of the buffer (in the file)
	bufferLen   int    // buffer len

	lock sync.Mutex // We use mutex to make it thread-safe
}

// NewQLogFile initializes a new instance of the QLogFile
func NewQLogFile(path string) (*QLogFile, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return &QLogFile{
		file: f,
	}, nil
}

// seekTS performs binary search in the query log file looking for a record
// with the specified timestamp. Once the record is found, it sets
// "position" so that the next ReadNext call returned that record.
//
// The algorithm is rather simple:
// 1. It starts with the position in the middle of a file
// 2. Shifts back to the beginning of the line
// 3. Checks the log record timestamp
// 4. If it is lower than the timestamp we are looking for,
// it shifts seek position to 3/4 of the file. Otherwise, to 1/4 of the file.
// 5. It performs the search again, every time the search scope is narrowed twice.
//
// Returns:
// * It returns the position of the the line with the timestamp we were looking for
// so that when we call "ReadNext" this line was returned.
// * Depth of the search (how many times we compared timestamps).
// * If we could not find it, it returns one of the errors described above.
func (q *QLogFile) seekTS(timestamp int64) (int64, int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Empty the buffer
	q.buffer = nil

	// First of all, check the file size
	fileInfo, err := q.file.Stat()
	if err != nil {
		return 0, 0, err
	}

	// Define the search scope
	start := int64(0)          // start of the search interval (position in the file)
	end := fileInfo.Size()     // end of the search interval (position in the file)
	probe := (end - start) / 2 // probe -- approximate index of the line we'll try to check
	var line string
	var lineIdx int64 // index of the probe line in the file
	var lineEndIdx int64
	var lastProbeLineIdx int64 // index of the last probe line
	lastProbeLineIdx = -1

	// Count seek depth in order to detect mistakes
	// If depth is too large, we should stop the search
	depth := 0

	for {
		// Get the line at the specified position
		line, lineIdx, lineEndIdx, err = q.readProbeLine(probe)
		if err != nil {
			return 0, depth, err
		}

		if lineIdx == lastProbeLineIdx {
			if lineIdx == 0 {
				return 0, depth, ErrTSTooEarly
			}

			// If we're testing the same line twice then most likely
			// the scope is too narrow and we won't find anything
			// anymore in any other file.
			return 0, depth, fmt.Errorf("looking up timestamp %d in %q: %w", timestamp, q.file.Name(), ErrTSNotFound)
		} else if lineIdx == fileInfo.Size() {
			return 0, depth, ErrTSTooLate
		}

		// Save the last found idx
		lastProbeLineIdx = lineIdx

		// Get the timestamp from the query log record
		ts := readQLogTimestamp(line)
		if ts == 0 {
			return 0, depth, fmt.Errorf("looking up timestamp %d in %q: record %q has empty timestamp", timestamp, q.file.Name(), line)
		}

		if ts == timestamp {
			// Hurray, returning the result
			break
		}

		// Narrow the scope and repeat the search
		if ts > timestamp {
			// If the timestamp we're looking for is OLDER than what we found
			// Then the line is somewhere on the LEFT side from the current probe position
			end = lineIdx
		} else {
			// If the timestamp we're looking for is NEWER than what we found
			// Then the line is somewhere on the RIGHT side from the current probe position
			start = lineEndIdx
		}
		probe = start + (end-start)/2

		depth++
		if depth >= 100 {
			return 0, depth, fmt.Errorf("looking up timestamp %d in %q: depth %d too high: %w", timestamp, q.file.Name(), depth, ErrTSNotFound)
		}
	}

	q.position = lineIdx + int64(len(line))
	return q.position, depth, nil
}

// SeekStart changes the current position to the end of the file
// Please note that we're reading query log in the reverse order
// and that's why log start is actually the end of file
//
// Returns nil if we were able to change the current position.
// Returns error in any other case.
func (q *QLogFile) SeekStart() (int64, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Empty the buffer
	q.buffer = nil

	// First of all, check the file size
	fileInfo, err := q.file.Stat()
	if err != nil {
		return 0, err
	}

	// Place the position to the very end of file
	q.position = fileInfo.Size() - 1
	if q.position < 0 {
		q.position = 0
	}
	return q.position, nil
}

// ReadNext reads the next line (in the reverse order) from the file
// and shifts the current position left to the next (actually prev) line.
// returns io.EOF if there's nothing to read more
func (q *QLogFile) ReadNext() (string, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.position == 0 {
		return "", io.EOF
	}

	line, lineIdx, err := q.readNextLine(q.position)
	if err != nil {
		return "", err
	}

	// Shift position
	if lineIdx == 0 {
		q.position = 0
	} else {
		// there's usually a line break before the line
		// so we should shift one more char left from the line
		// line\nline
		q.position = lineIdx - 1
	}
	return line, err
}

// Close frees the underlying resources
func (q *QLogFile) Close() error {
	return q.file.Close()
}

// readNextLine reads the next line from the specified position
// this line actually have to END on that position.
//
// the algorithm is:
// 1. check if we have the buffer initialized
// 2. if it is, scan it and look for the line there
// 3. if we cannot find the line there, read the prev chunk into the buffer
// 4. read the line from the buffer
func (q *QLogFile) readNextLine(position int64) (string, int64, error) {
	relativePos := position - q.bufferStart
	if q.buffer == nil || (relativePos < maxEntrySize && q.bufferStart != 0) {
		// Time to re-init the buffer
		err := q.initBuffer(position)
		if err != nil {
			return "", 0, err
		}
		relativePos = position - q.bufferStart
	}

	// Look for the end of the prev line
	// This is where we'll read from
	startLine := int64(0)
	for i := relativePos - 1; i >= 0; i-- {
		if q.buffer[i] == '\n' {
			startLine = i + 1
			break
		}
	}

	line := string(q.buffer[startLine:relativePos])
	lineIdx := q.bufferStart + startLine
	return line, lineIdx, nil
}

// initBuffer initializes the QLogFile buffer.
// the goal is to read a chunk of file that includes the line with the specified position.
func (q *QLogFile) initBuffer(position int64) error {
	q.bufferStart = int64(0)
	if position > bufferSize {
		q.bufferStart = position - bufferSize
	}

	// Seek to this position
	_, err := q.file.Seek(q.bufferStart, io.SeekStart)
	if err != nil {
		return err
	}

	if q.buffer == nil {
		q.buffer = make([]byte, bufferSize)
	}

	q.bufferLen, err = q.file.Read(q.buffer)

	return err
}

// readProbeLine reads a line that includes the specified position
// this method is supposed to be used when we use binary search in the Seek method
// in the case of consecutive reads, use readNext (it uses a better buffer)
func (q *QLogFile) readProbeLine(position int64) (string, int64, int64, error) {
	// First of all, we should read a buffer that will include the query log line
	// In order to do this, we'll define the boundaries
	seekPosition := int64(0)
	relativePos := position // position relative to the buffer we're going to read
	if position > maxEntrySize {
		seekPosition = position - maxEntrySize
		relativePos = maxEntrySize
	}

	// Seek to this position
	_, err := q.file.Seek(seekPosition, io.SeekStart)
	if err != nil {
		return "", 0, 0, err
	}

	// The buffer size is 2*maxEntrySize
	buffer := make([]byte, maxEntrySize*2)
	bufferLen, err := q.file.Read(buffer)
	if err != nil {
		return "", 0, 0, err
	}

	// Now start looking for the new line character starting
	// from the relativePos and going left
	startLine := int64(0)
	for i := relativePos - 1; i >= 0; i-- {
		if buffer[i] == '\n' {
			startLine = i + 1
			break
		}
	}
	// Looking for the end of line now
	endLine := int64(bufferLen)
	lineEndIdx := endLine + seekPosition
	for i := relativePos; i < int64(bufferLen); i++ {
		if buffer[i] == '\n' {
			endLine = i
			lineEndIdx = endLine + seekPosition + 1
			break
		}
	}

	// Finally we can return the string we were looking for
	lineIdx := startLine + seekPosition
	return string(buffer[startLine:endLine]), lineIdx, lineEndIdx, nil
}

// readJSONvalue reads a JSON string in form of '"key":"value"'.  prefix must be
// of the form '"key":"' to generate less garbage.
func readJSONValue(s, prefix string) string {
	i := strings.Index(s, prefix)
	if i == -1 {
		return ""
	}

	start := i + len(prefix)
	i = strings.IndexByte(s[start:], '"')
	if i == -1 {
		return ""
	}

	end := start + i
	return s[start:end]
}

// readQLogTimestamp reads the timestamp field from the query log line
func readQLogTimestamp(str string) int64 {
	val := readJSONValue(str, `"T":"`)
	if len(val) == 0 {
		val = readJSONValue(str, `"Time":"`)
	}

	if len(val) == 0 {
		log.Error("Couldn't find timestamp: %s", str)
		return 0
	}
	tm, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		log.Error("Couldn't parse timestamp: %s", val)
		return 0
	}
	return tm.UnixNano()
}
