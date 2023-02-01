package aghio

import (
	"io"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitReader(t *testing.T) {
	testCases := []struct {
		wantErrMsg string
		name       string
		n          int64
	}{{
		wantErrMsg: "",
		name:       "positive",
		n:          1,
	}, {
		wantErrMsg: "",
		name:       "zero",
		n:          0,
	}, {
		wantErrMsg: "limit must be non-negative",
		name:       "negative",
		n:          -1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LimitReader(nil, tc.n)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestLimitedReader_Read(t *testing.T) {
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
		readCloser := io.NopCloser(strings.NewReader(tc.rStr))
		lreader, err := LimitReader(readCloser, tc.limit)
		require.NoError(t, err)
		require.NotNil(t, lreader)

		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, tc.limit+1)
			n, rerr := lreader.Read(buf)
			require.Equal(t, rerr, tc.err)

			assert.Equal(t, tc.want, n)
		})
	}
}

func TestLimitedReader_LimitReachedError(t *testing.T) {
	testutil.AssertErrorMsg(t, "attempted to read more than 0 bytes", &LimitReachedError{
		Limit: 0,
	})
}
