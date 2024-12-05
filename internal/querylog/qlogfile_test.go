package querylog

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// prepareTestFile prepares one test query log file with the specified lines
// count.
func prepareTestFile(t *testing.T, dir string, linesNum int) (name string) {
	t.Helper()

	f, err := os.CreateTemp(dir, "*.txt")
	require.NoError(t, err)

	// Use defer and not t.Cleanup to make sure that the file is closed
	// after this function is done.
	defer func() {
		derr := f.Close()
		require.NoError(t, derr)
	}()

	const ans = `"AAAAAAABAAEAAAAAB2V4YW1wbGUDb3JnAAABAAEHZXhhbXBsZQNvcmcAAAEAAQAAAAAABAECAwQ="`
	const format = `{"IP":%q,"T":%q,"QH":"example.org","QT":"A","QC":"IN",` +
		`"Answer":` + ans + `,"Result":{},"Elapsed":0,"Upstream":"upstream"}` + "\n"

	var lineIP uint32
	lineTime := time.Date(2020, 2, 18, 19, 36, 35, 920973000, time.UTC)
	for range linesNum {
		lineIP++
		lineTime = lineTime.Add(time.Second)

		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, lineIP)

		line := fmt.Sprintf(format, ip, lineTime.Format(time.RFC3339Nano))

		_, err = f.WriteString(line)
		require.NoError(t, err)
	}

	return f.Name()
}

// prepareTestFiles prepares several test query log files, each with the
// specified lines count.
func prepareTestFiles(t *testing.T, filesNum, linesNum int) []string {
	t.Helper()

	if filesNum == 0 {
		return []string{}
	}

	dir := t.TempDir()

	files := make([]string, filesNum)
	for i := range files {
		files[filesNum-i-1] = prepareTestFile(t, dir, linesNum)
	}

	return files
}

// newTestQLogFile creates new *qLogFile for tests and registers the required
// cleanup functions.
func newTestQLogFile(t *testing.T, linesNum int) (file *qLogFile) {
	t.Helper()

	testFile := prepareTestFiles(t, 1, linesNum)[0]

	// Create the new qLogFile instance.
	file, err := newQLogFile(testFile)
	require.NoError(t, err)

	assert.NotNil(t, file)
	testutil.CleanupAndRequireSuccess(t, file.Close)

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
			require.NoError(t, err)

			var expPos int64
			if expPos = fileInfo.Size(); expPos > 0 {
				expPos--
			}

			// Seek to the start.
			pos, err := q.SeekStart()
			require.NoError(t, err)
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

	logger := slogutil.NewDiscardLogger()
	ctx := testutil.ContextWithTimeout(t, testTimeout)

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
				require.NoError(t, err)

				ts := readQLogTimestamp(ctx, logger, line)
				assert.NotEqualValues(t, 0, ts)

				// Try seeking to that line now.
				pos, _, err := q.seekTS(ctx, logger, ts)
				require.NoError(t, err)

				assert.NotEqualValues(t, 0, pos)

				testLine, err := q.ReadNext()
				require.NoError(t, err)

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

	logger := slogutil.NewDiscardLogger()
	ctx := testutil.ContextWithTimeout(t, testTimeout)

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
		require.NoError(t, err)

		testCases[2].ts = readQLogTimestamp(ctx, logger, line) - 1
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.NotEqualValues(t, 0, tc.ts)

				var depth int
				_, depth, err = q.seekTS(ctx, logger, tc.ts)
				assert.NotEmpty(t, l.num)
				require.Error(t, err)

				if tc.leq {
					assert.LessOrEqual(t, depth, int(math.Log2(float64(l.num))+3))
				}
			})
		}
	}
}

func getQLogFileLine(q *qLogFile, lineNumber int) (line string, err error) {
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
	// Create the new qLogFile instance.
	q := newTestQLogFile(t, 2)

	// Seek to the start.
	pos, err := q.SeekStart()
	require.NoError(t, err)

	assert.Greater(t, pos, int64(0))

	// Read first line.
	line, err := q.ReadNext()
	require.NoError(t, err)

	assert.Contains(t, line, "0.0.0.2")
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// Read second line.
	line, err = q.ReadNext()
	require.NoError(t, err)

	assert.EqualValues(t, 0, q.position)
	assert.Contains(t, line, "0.0.0.1")
	assert.True(t, strings.HasPrefix(line, "{"), line)
	assert.True(t, strings.HasSuffix(line, "}"), line)

	// Try reading again (there's nothing to read anymore).
	line, err = q.ReadNext()
	require.Equal(t, io.EOF, err)

	assert.Empty(t, line)
}

func newTestQLogFileData(t *testing.T, data string) (file *qLogFile) {
	f, err := os.CreateTemp(t.TempDir(), "*.txt")
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, f.Close)

	_, err = f.WriteString(data)
	require.NoError(t, err)

	file, err = newQLogFile(f.Name())
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, file.Close)

	return file
}

func TestQLog_Seek(t *testing.T) {
	const nl = "\n"
	const strV = "%s"
	const recs = `{"T":"` + strV + `","QH":"wfqvjymurpwegyv","QT":"A","QC":"IN","CP":"","Answer":"","Result":{},"Elapsed":66286385,"Upstream":"tls://unfiltered.adguard-dns.com:853"}` + nl +
		`{"T":"` + strV + `"}` + nl +
		`{"T":"` + strV + `"}` + nl
	timestamp, _ := time.Parse(time.RFC3339Nano, "2020-08-31T18:44:25.376690873+03:00")

	logger := slogutil.NewDiscardLogger()
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	testCases := []struct {
		wantErr   error
		name      string
		delta     int
		wantDepth int
	}{{
		name:      "ok",
		delta:     0,
		wantErr:   nil,
		wantDepth: 2,
	}, {
		name:      "too_late",
		delta:     2,
		wantErr:   errTSTooLate,
		wantDepth: 2,
	}, {
		name:      "too_early",
		delta:     -2,
		wantErr:   errTSTooEarly,
		wantDepth: 1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := fmt.Sprintf(recs,
				timestamp.Add(-time.Second).Format(time.RFC3339Nano),
				timestamp.Format(time.RFC3339Nano),
				timestamp.Add(time.Second).Format(time.RFC3339Nano),
			)

			q := newTestQLogFileData(t, data)

			ts := timestamp.Add(time.Second * time.Duration(tc.delta)).UnixNano()
			_, depth, err := q.seekTS(ctx, logger, ts)
			require.Truef(t, errors.Is(err, tc.wantErr), "%v", err)

			assert.Equal(t, tc.wantDepth, depth)
		})
	}
}
