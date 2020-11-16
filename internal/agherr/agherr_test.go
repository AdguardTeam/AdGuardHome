package agherr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func TestError_Error(t *testing.T) {
	testCases := []struct {
		name string
		want string
		err  error
	}{{
		name: "simple",
		want: "a",
		err:  Many("a"),
	}, {
		name: "wrapping",
		want: "a: b",
		err:  Many("a", errors.New("b")),
	}, {
		name: "wrapping several",
		want: "a: b (hidden: c, d)",
		err:  Many("a", errors.New("b"), errors.New("c"), errors.New("d")),
	}, {
		name: "wrapping wrapper",
		want: "a: b: c (hidden: d)",
		err:  Many("a", Many("b", errors.New("c"), errors.New("d"))),
	}}
	for _, tc := range testCases {
		assert.Equal(t, tc.want, tc.err.Error(), tc.name)
	}
}

func TestError_Unwrap(t *testing.T) {
	const (
		errSimple = iota
		errWrapped
		errNil
	)
	errs := []error{
		errSimple:  errors.New("a"),
		errWrapped: fmt.Errorf("%w", errors.New("nested")),
		errNil:     nil,
	}
	testCases := []struct {
		name    string
		want    error
		wrapped error
	}{{
		name:    "simple",
		want:    errs[errSimple],
		wrapped: Many("a", errs[errSimple]),
	}, {
		name:    "nested",
		want:    errs[errWrapped],
		wrapped: Many("b", errs[errWrapped]),
	}, {
		name:    "nil passed",
		want:    errs[errNil],
		wrapped: Many("c", errs[errNil]),
	}, {
		name:    "nil not passed",
		want:    nil,
		wrapped: Many("d"),
	}}
	for _, tc := range testCases {
		assert.Equal(t, tc.want, errors.Unwrap(tc.wrapped), tc.name)
	}
}
