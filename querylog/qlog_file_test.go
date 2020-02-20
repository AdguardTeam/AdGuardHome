package querylog

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQLogFileEmpty(t *testing.T) {
	// TODO: test empty file
}

func TestQLogFileLarge(t *testing.T) {
	// TODO: test reading large file
}

func TestQLogFileSeek(t *testing.T) {
	// TODO: test seek method on a small file
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
	format := `{"IP":"${IP}","T":"${TIMESTAMP}","QH":"example.org","QT":"A","QC":"IN","Answer":"AAAAAAABAAEAAAAAB2V4YW1wbGUDb3JnAAABAAEHZXhhbXBsZQNvcmcAAAEAAQAAAAAABAECAwQ=","Result":{},"Elapsed":0,"Upstream":"upstream"}`

	lineTime, _ := time.Parse(time.RFC3339Nano, "2020-02-18T22:36:35.920973+03:00")
	lineIP := uint32(0)

	f, _ := ioutil.TempFile(dir, "*.txt")

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

	return f.Name()
}
