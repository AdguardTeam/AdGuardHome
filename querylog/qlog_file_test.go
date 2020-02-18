package querylog

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Check adding and loading (with filtering) entries from disk and memory
func TestQLogFile(t *testing.T) {
	conf := Config{
		Enabled:  true,
		Interval: 1,
		MemSize:  100,
	}
	conf.BaseDir = prepareTestDir()
	defer func() { _ = os.RemoveAll(conf.BaseDir) }()
	l := newQueryLog(conf)

	// add disk entries
	addEntry(l, "example.org", "1.2.3.4", "0.1.2.4")
	addEntry(l, "example.org", "1.2.3.4", "0.1.2.5")

	// write to disk
	_ = l.flushLogBuffer(true)

	// create the new QLogFile instance
	q, err := NewQLogFile(l.logFile)
	assert.Nil(t, err)
	assert.NotNil(t, q)

	// seek to the start
	pos, err := q.SeekStart()
	assert.Nil(t, err)
	assert.True(t, pos > 0)

	// read first line
	line, err := q.ReadNext()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(line, "0.1.2.5"), line)
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// read second line
	line, err = q.ReadNext()
	assert.Nil(t, err)
	assert.Equal(t, int64(0), q.position)
	assert.True(t, strings.Contains(line, "0.1.2.4"), line)
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// try reading again (there's nothing to read anymore)
	line, err = q.ReadNext()
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, "", line)
}
