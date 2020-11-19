package querylog

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"math"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQLogFileEmpty(t *testing.T) {
	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFile := prepareTestFile(testDir, 0)

	// create the new QLogFile instance
	q, err := NewQLogFile(testFile)
	assert.Nil(t, err)
	assert.NotNil(t, q)
	defer q.Close()

	// seek to the start
	pos, err := q.SeekStart()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), pos)

	// try reading anyway
	line, err := q.ReadNext()
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, "", line)
}

func TestQLogFileLarge(t *testing.T) {
	// should be large enough
	count := 50000

	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFile := prepareTestFile(testDir, count)

	// create the new QLogFile instance
	q, err := NewQLogFile(testFile)
	assert.Nil(t, err)
	assert.NotNil(t, q)
	defer q.Close()

	// seek to the start
	pos, err := q.SeekStart()
	assert.Nil(t, err)
	assert.NotEqual(t, int64(0), pos)

	read := 0
	var line string
	for err == nil {
		line, err = q.ReadNext()
		if err == nil {
			assert.True(t, len(line) > 0)
			read += 1
		}
	}

	assert.Equal(t, count, read)
	assert.Equal(t, io.EOF, err)
}

func TestQLogFileSeekLargeFile(t *testing.T) {
	// more or less big file
	count := 10000

	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFile := prepareTestFile(testDir, count)

	// create the new QLogFile instance
	q, err := NewQLogFile(testFile)
	assert.Nil(t, err)
	assert.NotNil(t, q)
	defer q.Close()

	// CASE 1: NOT TOO OLD LINE
	testSeekLineQLogFile(t, q, 300)

	// CASE 2: OLD LINE
	testSeekLineQLogFile(t, q, count-300)

	// CASE 3: FIRST LINE
	testSeekLineQLogFile(t, q, 0)

	// CASE 4: LAST LINE
	testSeekLineQLogFile(t, q, count)

	// CASE 5: Seek non-existent (too low)
	_, _, err = q.Seek(123)
	assert.NotNil(t, err)

	// CASE 6: Seek non-existent (too high)
	ts, _ := time.Parse(time.RFC3339, "2100-01-02T15:04:05Z07:00")
	_, _, err = q.Seek(ts.UnixNano())
	assert.NotNil(t, err)

	// CASE 7: "Almost" found
	line, err := getQLogFileLine(q, count/2)
	assert.Nil(t, err)
	// ALMOST the record we need
	timestamp := readQLogTimestamp(line) - 1
	assert.NotEqual(t, uint64(0), timestamp)
	_, depth, err := q.Seek(timestamp)
	assert.NotNil(t, err)
	assert.True(t, depth <= int(math.Log2(float64(count))+3))
}

func TestQLogFileSeekSmallFile(t *testing.T) {
	// more or less big file
	count := 10

	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFile := prepareTestFile(testDir, count)

	// create the new QLogFile instance
	q, err := NewQLogFile(testFile)
	assert.Nil(t, err)
	assert.NotNil(t, q)
	defer q.Close()

	// CASE 1: NOT TOO OLD LINE
	testSeekLineQLogFile(t, q, 2)

	// CASE 2: OLD LINE
	testSeekLineQLogFile(t, q, count-2)

	// CASE 3: FIRST LINE
	testSeekLineQLogFile(t, q, 0)

	// CASE 4: LAST LINE
	testSeekLineQLogFile(t, q, count)

	// CASE 5: Seek non-existent (too low)
	_, _, err = q.Seek(123)
	assert.NotNil(t, err)

	// CASE 6: Seek non-existent (too high)
	ts, _ := time.Parse(time.RFC3339, "2100-01-02T15:04:05Z07:00")
	_, _, err = q.Seek(ts.UnixNano())
	assert.NotNil(t, err)

	// CASE 7: "Almost" found
	line, err := getQLogFileLine(q, count/2)
	assert.Nil(t, err)
	// ALMOST the record we need
	timestamp := readQLogTimestamp(line) - 1
	assert.NotEqual(t, uint64(0), timestamp)
	_, depth, err := q.Seek(timestamp)
	assert.NotNil(t, err)
	assert.True(t, depth <= int(math.Log2(float64(count))+3))
}

func testSeekLineQLogFile(t *testing.T, q *QLogFile, lineNumber int) {
	line, err := getQLogFileLine(q, lineNumber)
	assert.Nil(t, err)
	ts := readQLogTimestamp(line)
	assert.NotEqual(t, uint64(0), ts)

	// try seeking to that line now
	pos, _, err := q.Seek(ts)
	assert.Nil(t, err)
	assert.NotEqual(t, int64(0), pos)

	testLine, err := q.ReadNext()
	assert.Nil(t, err)
	assert.Equal(t, line, testLine)
}

func getQLogFileLine(q *QLogFile, lineNumber int) (string, error) {
	_, err := q.SeekStart()
	if err != nil {
		return "", err
	}

	for i := 1; i < lineNumber; i++ {
		_, err := q.ReadNext()
		if err != nil {
			return "", err
		}
	}
	return q.ReadNext()
}

// Check adding and loading (with filtering) entries from disk and memory
func TestQLogFile(t *testing.T) {
	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFile := prepareTestFile(testDir, 2)

	// create the new QLogFile instance
	q, err := NewQLogFile(testFile)
	assert.Nil(t, err)
	assert.NotNil(t, q)
	defer q.Close()

	// seek to the start
	pos, err := q.SeekStart()
	assert.Nil(t, err)
	assert.True(t, pos > 0)

	// read first line
	line, err := q.ReadNext()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(line, "0.0.0.2"), line)
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// read second line
	line, err = q.ReadNext()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), q.position)
	assert.True(t, strings.Contains(line, "0.0.0.1"), line)
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// try reading again (there's nothing to read anymore)
	line, err = q.ReadNext()
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, "", line)
}

// prepareTestFile - prepares a test query log file with the specified number of lines
func prepareTestFile(dir string, linesCount int) string {
	return prepareTestFiles(dir, 1, linesCount)[0]
}

// prepareTestFiles - prepares several test query log files
// each of them -- with the specified linesCount
func prepareTestFiles(dir string, filesCount, linesCount int) []string {
	format := `{"IP":"${IP}","T":"${TIMESTAMP}","QH":"example.org","QT":"A","QC":"IN","Answer":"AAAAAAABAAEAAAAAB2V4YW1wbGUDb3JnAAABAAEHZXhhbXBsZQNvcmcAAAEAAQAAAAAABAECAwQ=","Result":{},"Elapsed":0,"Upstream":"upstream"}`

	lineTime, _ := time.Parse(time.RFC3339Nano, "2020-02-18T22:36:35.920973+03:00")
	lineIP := uint32(0)

	files := make([]string, filesCount)
	for j := 0; j < filesCount; j++ {
		f, _ := ioutil.TempFile(dir, "*.txt")
		files[filesCount-j-1] = f.Name()

		for i := 0; i < linesCount; i++ {
			lineIP += 1
			lineTime = lineTime.Add(time.Second)

			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, lineIP)

			line := format
			line = strings.ReplaceAll(line, "${IP}", ip.String())
			line = strings.ReplaceAll(line, "${TIMESTAMP}", lineTime.Format(time.RFC3339Nano))

			_, _ = f.WriteString(line)
			_, _ = f.WriteString("\n")
		}
	}

	return files
}

func TestQLogSeek(t *testing.T) {
	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()

	d := `{"T":"2020-08-31T18:44:23.911246629+03:00","QH":"wfqvjymurpwegyv","QT":"A","QC":"IN","CP":"","Answer":"","Result":{},"Elapsed":66286385,"Upstream":"tls://dns-unfiltered.adguard.com:853"}
{"T":"2020-08-31T18:44:25.376690873+03:00"}
{"T":"2020-08-31T18:44:25.382540454+03:00"}`
	f, _ := ioutil.TempFile(testDir, "*.txt")
	_, _ = f.WriteString(d)
	defer f.Close()

	q, err := NewQLogFile(f.Name())
	assert.Nil(t, err)
	defer q.Close()

	target, _ := time.Parse(time.RFC3339, "2020-08-31T18:44:25.376690873+03:00")

	_, depth, err := q.Seek(target.UnixNano())
	assert.Nil(t, err)
	assert.Equal(t, 1, depth)
}

func TestQLogSeek_ErrTSTooLate(t *testing.T) {
	testDir := prepareTestDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	d := `{"T":"2020-08-31T18:44:23.911246629+03:00","QH":"wfqvjymurpwegyv","QT":"A","QC":"IN","CP":"","Answer":"","Result":{},"Elapsed":66286385,"Upstream":"tls://dns-unfiltered.adguard.com:853"}
{"T":"2020-08-31T18:44:25.376690873+03:00"}
{"T":"2020-08-31T18:44:25.382540454+03:00"}
`
	f, err := ioutil.TempFile(testDir, "*.txt")
	assert.Nil(t, err)
	defer f.Close()

	_, err = f.WriteString(d)
	assert.Nil(t, err)

	q, err := NewQLogFile(f.Name())
	assert.Nil(t, err)
	defer q.Close()

	target, err := time.Parse(time.RFC3339, "2020-08-31T18:44:25.382540454+03:00")
	assert.Nil(t, err)

	_, depth, err := q.Seek(target.UnixNano() + int64(time.Second))
	assert.Equal(t, ErrTSTooLate, err)
	assert.Equal(t, 2, depth)
}

func TestQLogSeek_ErrTSTooEarly(t *testing.T) {
	testDir := prepareTestDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(testDir)
	})

	d := `{"T":"2020-08-31T18:44:23.911246629+03:00","QH":"wfqvjymurpwegyv","QT":"A","QC":"IN","CP":"","Answer":"","Result":{},"Elapsed":66286385,"Upstream":"tls://dns-unfiltered.adguard.com:853"}
{"T":"2020-08-31T18:44:25.376690873+03:00"}
{"T":"2020-08-31T18:44:25.382540454+03:00"}
`
	f, err := ioutil.TempFile(testDir, "*.txt")
	assert.Nil(t, err)
	defer f.Close()

	_, err = f.WriteString(d)
	assert.Nil(t, err)

	q, err := NewQLogFile(f.Name())
	assert.Nil(t, err)
	defer q.Close()

	target, err := time.Parse(time.RFC3339, "2020-08-31T18:44:23.911246629+03:00")
	assert.Nil(t, err)

	_, depth, err := q.Seek(target.UnixNano() - int64(time.Second))
	assert.Equal(t, ErrTSTooEarly, err)
	assert.Equal(t, 1, depth)
}
