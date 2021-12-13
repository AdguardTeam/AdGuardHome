package querylog

import (
	"io"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestQLogReader creates new *QLogReader for tests and registers the
// required cleanup functions.
func newTestQLogReader(t *testing.T, filesNum, linesNum int) (reader *QLogReader) {
	t.Helper()

	testFiles := prepareTestFiles(t, filesNum, linesNum)

	// Create the new QLogReader instance.
	reader, err := NewQLogReader(testFiles)
	require.NoError(t, err)

	assert.NotNil(t, reader)
	testutil.CleanupAndRequireSuccess(t, reader.Close)

	return reader
}

func TestQLogReader(t *testing.T) {
	testCases := []struct {
		name     string
		filesNum int
		linesNum int
	}{{
		name:     "empty",
		filesNum: 0,
		linesNum: 0,
	}, {
		name:     "one_file",
		filesNum: 1,
		linesNum: 10,
	}, {
		name:     "multiple_files",
		filesNum: 5,
		linesNum: 10000,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestQLogReader(t, tc.filesNum, tc.linesNum)

			// Seek to the start.
			err := r.SeekStart()
			require.NoError(t, err)

			// Read everything.
			var read int
			var line string
			for err == nil {
				line, err = r.ReadNext()
				if err == nil {
					assert.NotEmpty(t, line)
					read++
				}
			}

			require.Equal(t, io.EOF, err)
			assert.Equal(t, tc.filesNum*tc.linesNum, read)
		})
	}
}

func TestQLogReader_Seek(t *testing.T) {
	r := newTestQLogReader(t, 2, 10000)

	testCases := []struct {
		name string
		time string
		want error
	}{{
		name: "not_too_old",
		time: "2020-02-18T22:39:35.920973+03:00",
		want: nil,
	}, {
		name: "old",
		time: "2020-02-19T01:28:16.920973+03:00",
		want: nil,
	}, {
		name: "first",
		time: "2020-02-18T22:36:36.920973+03:00",
		want: nil,
	}, {
		name: "last",
		time: "2020-02-19T01:23:16.920973+03:00",
		want: nil,
	}, {
		name: "non-existent_long_ago",
		time: "2000-02-19T01:23:16.920973+03:00",
		want: ErrTSNotFound,
	}, {
		name: "non-existent_far_ahead",
		time: "2100-02-19T01:23:16.920973+03:00",
		want: nil,
	}, {
		name: "non-existent_but_could",
		time: "2020-02-18T22:36:37.000000+03:00",
		want: ErrTSNotFound,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts, err := time.Parse(time.RFC3339Nano, tc.time)
			require.NoError(t, err)

			err = r.seekTS(ts.UnixNano())
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestQLogReader_ReadNext(t *testing.T) {
	const linesNum = 10
	const filesNum = 1
	r := newTestQLogReader(t, filesNum, linesNum)

	testCases := []struct {
		name  string
		start int
		want  error
	}{{
		name:  "ok",
		start: 0,
		want:  nil,
	}, {
		name:  "too_big",
		start: linesNum + 1,
		want:  io.EOF,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := r.SeekStart()
			require.NoError(t, err)

			for i := 1; i < tc.start; i++ {
				_, err = r.ReadNext()
				require.NoError(t, err)
			}

			_, err = r.ReadNext()
			assert.Equal(t, tc.want, err)
		})
	}
}
