// Package aghtest contains utilities for testing.
package aghtest

import (
	"io"
	"testing"

	"github.com/AdguardTeam/golibs/log"
)

// ReplaceLogWriter moves logger output to w and uses Cleanup method of t to
// revert changes.
func ReplaceLogWriter(t testing.TB, w io.Writer) {
	t.Helper()

	prev := log.Writer()
	t.Cleanup(func() { log.SetOutput(prev) })
	log.SetOutput(w)
}

// ReplaceLogLevel sets logging level to l and uses Cleanup method of t to
// revert changes.
func ReplaceLogLevel(t testing.TB, l log.Level) {
	t.Helper()

	switch l {
	case log.INFO, log.DEBUG, log.ERROR:
		// Go on.
	default:
		t.Fatalf("wrong l value (must be one of %v, %v, %v)", log.INFO, log.DEBUG, log.ERROR)
	}

	prev := log.GetLevel()
	t.Cleanup(func() { log.SetLevel(prev) })
	log.SetLevel(l)
}
