package agherr

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_Error(t *testing.T) {
	testCases := []struct {
		err  error
		name string
		want string
	}{{
		err:  Many("a"),
		name: "simple",
		want: "a",
	}, {
		err:  Many("a", errors.New("b")),
		name: "wrapping",
		want: "a: b",
	}, {
		err:  Many("a", errors.New("b"), errors.New("c"), errors.New("d")),
		name: "wrapping several",
		want: "a: b (hidden: c, d)",
	}, {
		err:  Many("a", Many("b", errors.New("c"), errors.New("d"))),
		name: "wrapping wrapper",
		want: "a: b: c (hidden: d)",
	}}

	for _, tc := range testCases {
		assert.Equal(t, tc.want, tc.err.Error(), tc.name)
	}
}

func TestError_Unwrap(t *testing.T) {
	var _ wrapper = &manyError{}

	const (
		errSimple = iota
		errWrapped
		errNil
	)

	errs := []error{
		errSimple:  errors.New("a"),
		errWrapped: fmt.Errorf("err: %w", errors.New("nested")),
		errNil:     nil,
	}

	testCases := []struct {
		want    error
		wrapped error
		name    string
	}{{
		want:    errs[errSimple],
		wrapped: Many("a", errs[errSimple]),
		name:    "simple",
	}, {
		want:    errs[errWrapped],
		wrapped: Many("b", errs[errWrapped]),
		name:    "nested",
	}, {
		want:    errs[errNil],
		wrapped: Many("c", errs[errNil]),
		name:    "nil passed",
	}, {
		want:    nil,
		wrapped: Many("d"),
		name:    "nil not passed",
	}}

	for _, tc := range testCases {
		assert.Equal(t, tc.want, errors.Unwrap(tc.wrapped), tc.name)
	}
}

func TestAnnotate(t *testing.T) {
	const s = "1234"
	const wantMsg = `bad string "1234": test`

	// Don't use const, because we can't take a pointer of a constant.
	var errTest error = Error("test")

	t.Run("nil", func(t *testing.T) {
		var errPtr *error
		assert.NotPanics(t, func() {
			Annotate("bad string %q: %w", errPtr, s)
		})
	})

	t.Run("non_nil", func(t *testing.T) {
		errPtr := &errTest
		assert.NotPanics(t, func() {
			Annotate("bad string %q: %w", errPtr, s)
		})

		require.NotNil(t, errPtr)

		err := *errPtr
		require.Error(t, err)

		assert.Equal(t, wantMsg, err.Error())
	})

	t.Run("defer", func(t *testing.T) {
		f := func() (err error) {
			defer Annotate("bad string %q: %w", &errTest, s)

			return errTest
		}

		err := f()
		require.Error(t, err)

		assert.Equal(t, wantMsg, err.Error())
	})
}

func TestLogPanic(t *testing.T) {
	buf := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, buf)

	t.Run("prefix", func(t *testing.T) {
		const (
			panicMsg        = "spooky!"
			prefix          = "packagename"
			errWithNoPrefix = "[error] recovered from panic: spooky!"
			errWithPrefix   = "[error] packagename: recovered from panic: spooky!"
		)

		panicFunc := func(prefix string) {
			defer LogPanic(prefix)

			panic(panicMsg)
		}

		panicFunc("")
		assert.Contains(t, buf.String(), errWithNoPrefix)
		buf.Reset()

		panicFunc(prefix)
		assert.Contains(t, buf.String(), errWithPrefix)
		buf.Reset()
	})

	t.Run("don't_panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			defer LogPanic("")
		})

		assert.Empty(t, buf.String())
	})
}
