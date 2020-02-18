package querylog

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var ErrSeekNotFound = errors.New("Seek not found the record")

const bufferSize = 64 * 1024 // 64 KB is the buffer size

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
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)

	if err != nil {
		return nil, err
	}

	return &QLogFile{
		file: f,
	}, nil
}

// Seek performs binary search in the query log file looking for a record
// with the specified timestamp.
//
// The algorithm is rather simple:
// 1. It starts with the position in the middle of a file
// 2. Shifts back to the beginning of the line
// 3. Checks the log record timestamp
// 4. If it is lower than the timestamp we are looking for,
// it shifts seek position to 3/4 of the file. Otherwise, to 1/4 of the file.
// 5. It performs the search again, every time the search scope is narrowed twice.
//
// It returns the position of the line with the timestamp we were looking for.
// If we could not find it, it returns 0 and ErrSeekNotFound
func (q *QLogFile) Seek(timestamp uint64) (int64, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// First of all, check the file size
	fileInfo, err := q.file.Stat()
	if err != nil {
		return 0, err
	}

	// Define the search scope
	start := int64(0)
	end := fileInfo.Size()
	probe := (end - start) / 2

	// Get the line
	line, _, err := q.readProbeLine(probe)
	if err != nil {
		return 0, err
	}

	// Get the timestamp from the query log record
	ts := q.readTimestamp(line)

	if ts == timestamp {
		// Hurray, returning the result
		return probe, nil
	}

	// Narrow the scope and repeat the search
	if ts > timestamp {
		end := probe
		probe = (end - start) / 2
	} else {
		start := probe
		probe = (end - start) / 2
	}

	// TODO: temp
	q.position = probe

	// TODO: Check start/stop/probe values and loop this
	return 0, ErrSeekNotFound
}

// SeekStart changes the current position to the end of the file
// Please note that we're reading query log in the reverse order
// and that's why log start is actually the end of file
func (q *QLogFile) SeekStart() (int64, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	// First of all, check the file size
	fileInfo, err := q.file.Stat()
	if err != nil {
		return 0, err
	}

	// Place the position to the very end of file
	q.position = fileInfo.Size() - 1
	if q.position < 0 {
		// TODO: test empty file
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
	if q.buffer == nil || relativePos < maxEntrySize {
		// Time to re-init the buffer
		err := q.initBuffer(position)
		if err != nil {
			return "", 0, err
		}
	}

	// Look for the end of the prev line
	// This is where we'll read from
	var startLine = int64(0)
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
	if (position - bufferSize) > 0 {
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
	// TODO: validate bufferLen
	if err != nil {
		return err
	}

	return nil
}

// readProbeLine reads a line that includes the specified position
// this method is supposed to be used when we use binary search in the Seek method
// in the case of consecutive reads, use readNext (it uses a better buffer)
func (q *QLogFile) readProbeLine(position int64) (string, int64, error) {
	// First of all, we should read a buffer that will include the query log line
	// In order to do this, we'll define the boundaries
	seekPosition := int64(0)
	relativePos := position // position relative to the buffer we're going to read
	if (position - maxEntrySize) > 0 {
		// TODO: cover this case in tests
		seekPosition = position - maxEntrySize
		relativePos = maxEntrySize
	}

	// Seek to this position
	_, err := q.file.Seek(seekPosition, io.SeekStart)
	if err != nil {
		return "", 0, err
	}

	// The buffer size is 2*maxEntrySize
	buffer := make([]byte, maxEntrySize*2)
	bufferLen, err := q.file.Read(buffer)
	if err != nil {
		return "", 0, err
	}

	// Now start looking for the new line character starting
	// from the relativePos and going left
	var startLine = int64(0)
	for i := relativePos - 1; i >= 0; i-- {
		if buffer[i] == '\n' {
			startLine = i + 1
			break
		}
	}
	// Looking for the end of line now
	var endLine = int64(bufferLen)
	for i := relativePos; i < int64(bufferLen); i++ {
		if buffer[i] == '\n' {
			endLine = i
			break
		}
	}

	// Finally we can return the string we were looking for
	lineIdx := startLine + seekPosition
	return string(buffer[startLine:endLine]), lineIdx, nil
}

// readTimestamp reads the timestamp field from the query log line
func (q *QLogFile) readTimestamp(str string) uint64 {
	val := readJSONValue(str, "T")
	if len(val) == 0 {
		val = readJSONValue(str, "Time")
	}

	if len(val) == 0 {
		// TODO: log
		return 0
	}
	tm, err := time.Parse(time.RFC3339, val)
	if err != nil {
		// TODO: log
		return 0
	}
	return uint64(tm.UnixNano())
}
