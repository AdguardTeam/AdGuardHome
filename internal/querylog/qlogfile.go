package querylog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

const (
	// Timestamp not found errors.
	errTSNotFound errors.Error = "ts not found"
	errTSTooLate  errors.Error = "ts too late"
	errTSTooEarly errors.Error = "ts too early"

	// maxEntrySize is a maximum size of the entry.
	//
	// TODO: Find a way to grow buffer instead of relying on this value when
	// reading strings.
	maxEntrySize = 16 * 1024

	// bufferSize should be enough for at least this number of entries.
	bufferSize = 100 * maxEntrySize
)

// qLogFile represents a single query log file.  It allows reading from the
// file in the reverse order.
//
// Please note, that this is a stateful object.  Internally, it contains a
// pointer to a specific position in the file, and it reads lines in reverse
// order starting from that position.
type qLogFile struct {
	// file is the query log file.
	file *os.File

	// buffer that we've read from the file.
	buffer []byte

	// lock is a mutex to make it thread-safe.
	lock sync.Mutex

	// position is the position in the file.
	position int64

	// bufferStart is the start of the buffer (in the file).
	bufferStart int64

	// bufferLen is the length of the buffer.
	bufferLen int
}

// newQLogFile initializes a new instance of the qLogFile.
func newQLogFile(path string) (qf *qLogFile, err error) {
	f, err := os.OpenFile(path, os.O_RDONLY, aghos.DefaultPermFile)
	if err != nil {
		return nil, err
	}

	return &qLogFile{file: f}, nil
}

// validateQLogLineIdx returns error if the line index is not valid to continue
// search.
func (q *qLogFile) validateQLogLineIdx(lineIdx, lastProbeLineIdx, ts, fSize int64) (err error) {
	if lineIdx == lastProbeLineIdx {
		if lineIdx == 0 {
			return errTSTooEarly
		}

		// If we're testing the same line twice then most likely the scope is
		// too narrow and we won't find anything anymore in any other file.
		return fmt.Errorf("looking up timestamp %d in %q: %w", ts, q.file.Name(), errTSNotFound)
	} else if lineIdx == fSize {
		return errTSTooLate
	}

	return nil
}

// seekTS performs binary search in the query log file looking for a record
// with the specified timestamp.  Once the record is found, it sets "position"
// so that the next ReadNext call returned that record.
//
// The algorithm is rather simple:
//  1. It starts with the position in the middle of a file.
//  2. Shifts back to the beginning of the line.
//  3. Checks the log record timestamp.
//  4. If it is lower than the timestamp we are looking for, it shifts seek
//     position to 3/4 of the file. Otherwise, to 1/4 of the file.
//  5. It performs the search again, every time the search scope is narrowed
//     twice.
//
// Returns:
//   - It returns the position of the line with the timestamp we were looking
//     for so that when we call "ReadNext" this line was returned.
//   - Depth of the search (how many times we compared timestamps).
//   - If we could not find it, it returns one of the errors described above.
func (q *qLogFile) seekTS(
	ctx context.Context,
	logger *slog.Logger,
	timestamp int64,
) (pos int64, depth int, err error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Empty the buffer.
	q.buffer = nil

	// First of all, check the file size.
	fileInfo, err := q.file.Stat()
	if err != nil {
		return 0, 0, err
	}

	// Define the search scope.

	// Start of the search interval (position in the file).
	start := int64(0)
	// End of the search interval (position in the file).
	end := fileInfo.Size()
	// Probe is the approximate index of the line we'll try to check.
	probe := (end - start) / 2

	var line string
	// Index of the probe line in the file.
	var lineIdx int64
	var lineEndIdx int64
	// Index of the last probe line.
	var lastProbeLineIdx int64
	lastProbeLineIdx = -1

	// Count seek depth in order to detect mistakes.  If depth is too large,
	// we should stop the search.
	for {
		// Get the line at the specified position.
		line, lineIdx, lineEndIdx, err = q.readProbeLine(probe)
		if err != nil {
			return 0, depth, err
		}

		// Check if the line index if invalid.
		err = q.validateQLogLineIdx(lineIdx, lastProbeLineIdx, timestamp, fileInfo.Size())
		if err != nil {
			return 0, depth, err
		}

		// Save the last found idx.
		lastProbeLineIdx = lineIdx

		// Get the timestamp from the query log record.
		ts := readQLogTimestamp(ctx, logger, line)
		if ts == 0 {
			return 0, depth, fmt.Errorf(
				"looking up timestamp %d in %q: record %q has empty timestamp",
				timestamp,
				q.file.Name(),
				line,
			)
		}

		if ts == timestamp {
			// Hurray, returning the result.
			break
		}

		// Narrow the scope and repeat the search.
		if ts > timestamp {
			// If the timestamp we're looking for is OLDER than what we found,
			// then the line is somewhere on the LEFT side from the current
			// probe position.
			end = lineIdx
		} else {
			// If the timestamp we're looking for is NEWER than what we found,
			// then the line is somewhere on the RIGHT side from the current
			// probe position.
			start = lineEndIdx
		}
		probe = start + (end-start)/2

		depth++
		if depth >= 100 {
			return 0, depth, fmt.Errorf(
				"looking up timestamp %d in %q: depth %d too high: %w",
				timestamp,
				q.file.Name(),
				depth,
				errTSNotFound,
			)
		}
	}

	q.position = lineIdx + int64(len(line))
	return q.position, depth, nil
}

// SeekStart changes the current position to the end of the file.  Please note,
// that we're reading query log in the reverse order and that's why log start
// is actually the end of file.
//
// Returns nil if we were able to change the current position.  Returns error
// in any other case.
func (q *qLogFile) SeekStart() (int64, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// Empty the buffer.
	q.buffer = nil

	// First of all, check the file size.
	fileInfo, err := q.file.Stat()
	if err != nil {
		return 0, err
	}

	// Place the position to the very end of file.
	q.position = fileInfo.Size() - 1
	if q.position < 0 {
		q.position = 0
	}

	return q.position, nil
}

// ReadNext reads the next line (in the reverse order) from the file and shifts
// the current position left to the next (actually prev) line.
//
// Returns io.EOF if there's nothing more to read.
func (q *qLogFile) ReadNext() (string, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.position == 0 {
		return "", io.EOF
	}

	line, lineIdx, err := q.readNextLine(q.position)
	if err != nil {
		return "", err
	}

	// Shift position.
	if lineIdx == 0 {
		q.position = 0
	} else {
		// There's usually a line break before the line, so we should shift one
		// more char left from the line "\nline".
		q.position = lineIdx - 1
	}
	return line, err
}

// Close frees the underlying resources.
func (q *qLogFile) Close() error {
	return q.file.Close()
}

// readNextLine reads the next line from the specified position.  This line
// actually have to END on that position.
//
// The algorithm is:
//  1. Check if we have the buffer initialized.
//  2. If it is so, scan it and look for the line there.
//  3. If we cannot find the line there, read the prev chunk into the buffer.
//  4. Read the line from the buffer.
func (q *qLogFile) readNextLine(position int64) (string, int64, error) {
	relativePos := position - q.bufferStart
	if q.buffer == nil || (relativePos < maxEntrySize && q.bufferStart != 0) {
		// Time to re-init the buffer.
		err := q.initBuffer(position)
		if err != nil {
			return "", 0, err
		}
		relativePos = position - q.bufferStart
	}

	// Look for the end of the prev line, this is where we'll read from.
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

// initBuffer initializes the qLogFile buffer.  The goal is to read a chunk of
// file that includes the line with the specified position.
func (q *qLogFile) initBuffer(position int64) error {
	q.bufferStart = int64(0)
	if position > bufferSize {
		q.bufferStart = position - bufferSize
	}

	// Seek to this position.
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

// readProbeLine reads a line that includes the specified position.  This
// method is supposed to be used when we use binary search in the Seek method.
// In the case of consecutive reads, use readNext, cause it uses better buffer.
func (q *qLogFile) readProbeLine(position int64) (string, int64, int64, error) {
	// First of all, we should read a buffer that will include the query log
	// line.  In order to do this, we'll define the boundaries.
	seekPosition := int64(0)
	// Position relative to the buffer we're going to read.
	relativePos := position
	if position > maxEntrySize {
		seekPosition = position - maxEntrySize
		relativePos = maxEntrySize
	}

	// Seek to this position.
	_, err := q.file.Seek(seekPosition, io.SeekStart)
	if err != nil {
		return "", 0, 0, err
	}

	// The buffer size is 2*maxEntrySize.
	buffer := make([]byte, maxEntrySize*2)
	bufferLen, err := q.file.Read(buffer)
	if err != nil {
		return "", 0, 0, err
	}

	// Now start looking for the new line character starting from the
	// relativePos and going left.
	startLine := int64(0)
	for i := relativePos - 1; i >= 0; i-- {
		if buffer[i] == '\n' {
			startLine = i + 1
			break
		}
	}
	// Looking for the end of line now.
	endLine := int64(bufferLen)
	lineEndIdx := endLine + seekPosition
	for i := relativePos; i < int64(bufferLen); i++ {
		if buffer[i] == '\n' {
			endLine = i
			lineEndIdx = endLine + seekPosition + 1
			break
		}
	}

	// Finally we can return the string we were looking for.
	lineIdx := startLine + seekPosition
	return string(buffer[startLine:endLine]), lineIdx, lineEndIdx, nil
}

// readJSONValue reads a JSON string in form of '"key":"value"'.  prefix must
// be of the form '"key":"' to generate less garbage.
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

// readQLogTimestamp reads the timestamp field from the query log line.
func readQLogTimestamp(ctx context.Context, logger *slog.Logger, str string) int64 {
	val := readJSONValue(str, `"T":"`)
	if len(val) == 0 {
		val = readJSONValue(str, `"Time":"`)
	}

	if len(val) == 0 {
		logger.ErrorContext(ctx, "couldn't find timestamp", "line", str)

		return 0
	}

	tm, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		logger.ErrorContext(ctx, "couldn't parse timestamp", "value", val, slogutil.KeyError, err)

		return 0
	}

	return tm.UnixNano()
}
