package aghio

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitReadCloser(t *testing.T) {
	testCases := []struct {
		name string
		n    int64
		want error
	}{{
		name: "positive",
		n:    1,
		want: nil,
	}, {
		name: "zero",
		n:    0,
		want: nil,
	}, {
		name: "negative",
		n:    -1,
		want: fmt.Errorf("aghio: invalid n in LimitReadCloser: -1"),
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
		name  string
		limit int64
		rStr  string
		want  int
		err   error
	}{{
		name:  "perfectly_match",
		limit: 3,
		rStr:  "abc",
		want:  3,
		err:   nil,
	}, {
		name:  "eof",
		limit: 3,
		rStr:  "",
		want:  0,
		err:   io.EOF,
	}, {
		name:  "limit_reached",
		limit: 0,
		rStr:  "abc",
		want:  0,
		err: &LimitReachedError{
			Limit: 0,
		},
	}, {
		name:  "truncated",
		limit: 2,
		rStr:  "abc",
		want:  2,
		err:   nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			readCloser := ioutil.NopCloser(strings.NewReader(tc.rStr))
			buf := make([]byte, tc.limit+1)

			lreader, err := LimitReadCloser(readCloser, tc.limit)
			assert.Nil(t, err)

			n, err := lreader.Read(buf)
			assert.Equal(t, n, tc.want)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestLimitedReadCloser_LimitReachedError(t *testing.T) {
	testCases := []struct {
		name string
		want string
		err  error
	}{{
		name: "simplest",
		want: "attempted to read more than 0 bytes",
		err: &LimitReachedError{
			Limit: 0,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.err.Error())
		})
	}
}
