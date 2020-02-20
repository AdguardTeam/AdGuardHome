package querylog

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQLogReaderEmpty(t *testing.T) {
	r, err := NewQLogReader([]string{})
	assert.Nil(t, err)
	assert.NotNil(t, r)
	defer r.Close()

	// seek to the start
	err = r.SeekStart()
	assert.Nil(t, err)

	line, err := r.ReadNext()
	assert.Equal(t, "", line)
	assert.Equal(t, io.EOF, err)
}

func TestQLogReaderOneFile(t *testing.T) {
	// let's do one small file
	count := 10
	filesCount := 1

	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFiles := prepareTestFiles(testDir, filesCount, count)

	r, err := NewQLogReader(testFiles)
	assert.Nil(t, err)
	assert.NotNil(t, r)
	defer r.Close()

	// seek to the start
	err = r.SeekStart()
	assert.Nil(t, err)

	// read everything
	read := 0
	var line string
	for err == nil {
		line, err = r.ReadNext()
		if err == nil {
			assert.True(t, len(line) > 0)
			read += 1
		}
	}

	assert.Equal(t, count*filesCount, read)
	assert.Equal(t, io.EOF, err)
}

func TestQLogReaderMultipleFiles(t *testing.T) {
	// should be large enough
	count := 10000
	filesCount := 5

	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFiles := prepareTestFiles(testDir, filesCount, count)

	r, err := NewQLogReader(testFiles)
	assert.Nil(t, err)
	assert.NotNil(t, r)
	defer r.Close()

	// seek to the start
	err = r.SeekStart()
	assert.Nil(t, err)

	// read everything
	read := 0
	var line string
	for err == nil {
		line, err = r.ReadNext()
		if err == nil {
			assert.True(t, len(line) > 0)
			read += 1
		}
	}

	assert.Equal(t, count*filesCount, read)
	assert.Equal(t, io.EOF, err)
}

func TestQLogReaderSeek(t *testing.T) {
	// more or less big file
	count := 10000
	filesCount := 2

	testDir := prepareTestDir()
	defer func() { _ = os.RemoveAll(testDir) }()
	testFiles := prepareTestFiles(testDir, filesCount, count)

	r, err := NewQLogReader(testFiles)
	assert.Nil(t, err)
	assert.NotNil(t, r)
	defer r.Close()

	// CASE 1: NOT TOO OLD LINE
	testSeekLineQLogReader(t, r, 300)

	// CASE 2: OLD LINE
	testSeekLineQLogReader(t, r, count-300)

	// CASE 3: FIRST LINE
	testSeekLineQLogReader(t, r, 0)

	// CASE 4: LAST LINE
	testSeekLineQLogReader(t, r, count)

	// CASE 5: Seek non-existent (too low)
	err = r.Seek(123)
	assert.NotNil(t, err)

	// CASE 6: Seek non-existent (too high)
	ts, _ := time.Parse(time.RFC3339, "2100-01-02T15:04:05Z07:00")
	err = r.Seek(ts.UnixNano())
	assert.NotNil(t, err)
}

func testSeekLineQLogReader(t *testing.T, r *QLogReader, lineNumber int) {
	line, err := getQLogReaderLine(r, lineNumber)
	assert.Nil(t, err)
	ts := readQLogTimestamp(line)
	assert.NotEqual(t, uint64(0), ts)

	// try seeking to that line now
	err = r.Seek(ts)
	assert.Nil(t, err)

	testLine, err := r.ReadNext()
	assert.Nil(t, err)
	assert.Equal(t, line, testLine)
}

func getQLogReaderLine(r *QLogReader, lineNumber int) (string, error) {
	err := r.SeekStart()
	if err != nil {
		return "", err
	}

	for i := 1; i < lineNumber; i++ {
		_, err := r.ReadNext()
		if err != nil {
			return "", err
		}
	}
	return r.ReadNext()
}
