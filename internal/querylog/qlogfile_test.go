package querylog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// prepareTestFiles prepares several test query log files, each with the
// specified lines count.
func prepareTestFiles(t *testing.T, filesNum, linesNum int) []string {
	t.Helper()

	if filesNum == 0 {
		return []string{}
	}

	const strV = "\"%s\""
	const nl = "\n"
	const format = `{"IP":` + strV + `,"T":` + strV + `,` +
		`"QH":"example.org","QT":"A","QC":"IN",` +
		`"Answer":"AAAAAAABAAEAAAAAB2V4YW1wbGUDb3JnAAABAAEHZXhhbXBsZQNvcmcAAAEAAQAAAAAABAECAwQ=",` +
		`"Result":{},"Elapsed":0,"Upstream":"upstream"}` + nl

	lineTime, _ := time.Parse(time.RFC3339Nano, "2020-02-18T22:36:35.920973+03:00")
	lineIP := uint32(0)

	dir := aghtest.PrepareTestDir(t)

	files := make([]string, filesNum)
	for j := range files {
		f, err := ioutil.TempFile(dir, "*.txt")
		require.Nil(t, err)
		files[filesNum-j-1] = f.Name()

		for i := 0; i < linesNum; i++ {
			lineIP++
			lineTime = lineTime.Add(time.Second)

			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, lineIP)

			line := fmt.Sprintf(format, ip, lineTime.Format(time.RFC3339Nano))

			_, err = f.WriteString(line)
			require.Nil(t, err)
		}
	}

	return files
}

// prepareTestFile prepares a test query log file with the specified number of
// lines.
func prepareTestFile(t *testing.T, linesCount int) string {
	t.Helper()

	return prepareTestFiles(t, 1, linesCount)[0]
}

// newTestQLogFile creates new *QLogFile for tests and registers the required
// cleanup functions.
func newTestQLogFile(t *testing.T, linesNum int) (file *QLogFile) {
	t.Helper()

	testFile := prepareTestFile(t, linesNum)

	// Create the new QLogFile instance.
	file, err := NewQLogFile(testFile)
	require.Nil(t, err)
	assert.NotNil(t, file)
	t.Cleanup(func() {
		assert.Nil(t, file.Close())
	})

	return file
}

func TestQLogFile_ReadNext(t *testing.T) {
	testCases := []struct {
		name     string
		linesNum int
	}{{
		name:     "empty",
		linesNum: 0,
	}, {
		name:     "large",
		linesNum: 50000,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q := newTestQLogFile(t, tc.linesNum)

			// Calculate the expected position.
			fileInfo, err := q.file.Stat()
			require.Nil(t, err)
			var expPos int64
			if expPos = fileInfo.Size(); expPos > 0 {
				expPos--
			}

			// Seek to the start.
			pos, err := q.SeekStart()
			require.Nil(t, err)
			require.EqualValues(t, expPos, pos)

			var read int
			var line string
			for err == nil {
				line, err = q.ReadNext()
				if err == nil {
					assert.NotEmpty(t, line)
					read++
				}
			}

			require.Equal(t, io.EOF, err)
			assert.Equal(t, tc.linesNum, read)
		})
	}
}

func TestQLogFile_SeekTS_good(t *testing.T) {
	linesCases := []struct {
		name string
		num  int
	}{{
		name: "large",
		num:  10000,
	}, {
		name: "small",
		num:  10,
	}}

	for _, l := range linesCases {
		testCases := []struct {
			name     string
			linesNum int
			line     int
		}{{
			name: "not_too_old",
			line: 2,
		}, {
			name: "old",
			line: l.num - 2,
		}, {
			name: "first",
			line: 0,
		}, {
			name: "last",
			line: l.num,
		}}

		q := newTestQLogFile(t, l.num)

		for _, tc := range testCases {
			t.Run(l.name+"_"+tc.name, func(t *testing.T) {
				line, err := getQLogFileLine(q, tc.line)
				require.Nil(t, err)
				ts := readQLogTimestamp(line)
				assert.NotEqualValues(t, 0, ts)

				// Try seeking to that line now.
				pos, _, err := q.SeekTS(ts)
				require.Nil(t, err)
				assert.NotEqualValues(t, 0, pos)

				testLine, err := q.ReadNext()
				require.Nil(t, err)
				assert.Equal(t, line, testLine)
			})
		}
	}
}

func TestQLogFile_SeekTS_bad(t *testing.T) {
	linesCases := []struct {
		name string
		num  int
	}{{
		name: "large",
		num:  10000,
	}, {
		name: "small",
		num:  10,
	}}

	for _, l := range linesCases {
		testCases := []struct {
			name string
			ts   int64
			leq  bool
		}{{
			name: "non-existent_long_ago",
		}, {
			name: "non-existent_far_ahead",
		}, {
			name: "almost",
			leq:  true,
		}}

		q := newTestQLogFile(t, l.num)
		testCases[0].ts = 123

		lateTS, _ := time.Parse(time.RFC3339, "2100-01-02T15:04:05Z07:00")
		testCases[1].ts = lateTS.UnixNano()

		line, err := getQLogFileLine(q, l.num/2)
		require.Nil(t, err)
		testCases[2].ts = readQLogTimestamp(line) - 1

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.NotEqualValues(t, 0, tc.ts)

				var depth int
				_, depth, err = q.SeekTS(tc.ts)
				assert.NotEmpty(t, l.num)
				require.NotNil(t, err)
				if tc.leq {
					assert.LessOrEqual(t, depth, int(math.Log2(float64(l.num))+3))
				}
			})
		}
	}
}

func getQLogFileLine(q *QLogFile, lineNumber int) (line string, err error) {
	if _, err = q.SeekStart(); err != nil {
		return line, err
	}

	for i := 1; i < lineNumber; i++ {
		if _, err = q.ReadNext(); err != nil {
			return line, err
		}
	}

	return q.ReadNext()
}

// Check adding and loading (with filtering) entries from disk and memory.
func TestQLogFile(t *testing.T) {
	// Create the new QLogFile instance.
	q := newTestQLogFile(t, 2)

	// Seek to the start.
	pos, err := q.SeekStart()
	require.Nil(t, err)
	assert.Greater(t, pos, int64(0))

	// Read first line.
	line, err := q.ReadNext()
	require.Nil(t, err)
	assert.Contains(t, line, "0.0.0.2")
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// Read second line.
	line, err = q.ReadNext()
	require.Nil(t, err)
	assert.EqualValues(t, 0, q.position)
	assert.Contains(t, line, "0.0.0.1")
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// Try reading again (there's nothing to read anymore).
	line, err = q.ReadNext()
	require.Equal(t, io.EOF, err)
	assert.Empty(t, line)
}

func NewTestQLogFileData(t *testing.T, data string) (file *QLogFile) {
	f, err := ioutil.TempFile(aghtest.PrepareTestDir(t), "*.txt")
	require.Nil(t, err)
	t.Cleanup(func() {
		assert.Nil(t, f.Close())
	})

	_, err = f.WriteString(data)
	require.Nil(t, err)

	file, err = NewQLogFile(f.Name())
	require.Nil(t, err)
	t.Cleanup(func() {
		assert.Nil(t, file.Close())
	})

	return file
}

func TestQLog_Seek(t *testing.T) {
	const nl = "\n"
	const strV = "%s"
	const recs = `{"T":"` + strV + `","QH":"wfqvjymurpwegyv","QT":"A","QC":"IN","CP":"","Answer":"","Result":{},"Elapsed":66286385,"Upstream":"tls://dns-unfiltered.adguard.com:853"}` + nl +
		`{"T":"` + strV + `"}` + nl +
		`{"T":"` + strV + `"}` + nl
	timestamp, _ := time.Parse(time.RFC3339Nano, "2020-08-31T18:44:25.376690873+03:00")

	testCases := []struct {
		name      string
		delta     int
		wantErr   error
		wantDepth int
	}{{
		name:      "ok",
		delta:     0,
		wantErr:   nil,
		wantDepth: 2,
	}, {
		name:      "too_late",
		delta:     2,
		wantErr:   ErrTSTooLate,
		wantDepth: 2,
	}, {
		name:      "too_early",
		delta:     -2,
		wantErr:   ErrTSTooEarly,
		wantDepth: 1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := fmt.Sprintf(recs,
				timestamp.Add(-time.Second).Format(time.RFC3339Nano),
				timestamp.Format(time.RFC3339Nano),
				timestamp.Add(time.Second).Format(time.RFC3339Nano),
			)

			q := NewTestQLogFileData(t, data)

			_, depth, err := q.SeekTS(timestamp.Add(time.Second * time.Duration(tc.delta)).UnixNano())
			require.Truef(t, errors.Is(err, tc.wantErr), "%v", err)
			assert.Equal(t, tc.wantDepth, depth)
		})
	}
}
