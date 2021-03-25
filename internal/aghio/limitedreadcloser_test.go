package aghio

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitReadCloser(t *testing.T) {
	testCases := []struct {
		want error
		name string
		n    int64
	}{{
		want: nil,
		name: "positive",
		n:    1,
	}, {
		want: nil,
		name: "zero",
		n:    0,
	}, {
		want: fmt.Errorf("aghio: invalid n in LimitReadCloser: -1"),
		name: "negative",
		n:    -1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LimitReadCloser(nil, tc.n)
			assert.Equal(t, tc.want, err)
		})
	}
}

func TestLimitedReadCloser_Read(t *testing.T) {
	testCases := []struct {
		err   error
		name  string
		rStr  string
		limit int64
		want  int
	}{{
		err:   nil,
		name:  "perfectly_match",
		rStr:  "abc",
		limit: 3,
		want:  3,
	}, {
		err:   io.EOF,
		name:  "eof",
		rStr:  "",
		limit: 3,
		want:  0,
	}, {
		err: &LimitReachedError{
			Limit: 0,
		},
		name:  "limit_reached",
		rStr:  "abc",
		limit: 0,
		want:  0,
	}, {
		err:   nil,
		name:  "truncated",
		rStr:  "abc",
		limit: 2,
		want:  2,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			readCloser := ioutil.NopCloser(strings.NewReader(tc.rStr))
			buf := make([]byte, tc.limit+1)

			lreader, err := LimitReadCloser(readCloser, tc.limit)
			require.NoError(t, err)

			n, err := lreader.Read(buf)
			require.Equal(t, tc.err, err)
			assert.Equal(t, tc.want, n)
		})
	}
}

func TestLimitedReadCloser_LimitReachedError(t *testing.T) {
	testCases := []struct {
		err  error
		name string
		want string
	}{{
		err: &LimitReachedError{
			Limit: 0,
		},
		name: "simplest",
		want: "attempted to read more than 0 bytes",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.err.Error())
		})
	}
}
