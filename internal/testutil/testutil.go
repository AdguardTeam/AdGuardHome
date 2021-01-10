// Package testutil contains utilities for testing.
package testutil

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/AdguardTeam/golibs/log"
)

// DiscardLogOutput runs tests with discarded logger output.
func DiscardLogOutput(m *testing.M) {
	// TODO(e.burkov): Refactor code and tests to not use the global mutable
	// logger.
	log.SetOutput(ioutil.Discard)

	os.Exit(m.Run())
}

// ReplaceLogWriter moves logger output to w and uses Cleanup method of t to
// revert changes.
func ReplaceLogWriter(t *testing.T, w io.Writer) {
	stdWriter := log.Writer()
	t.Cleanup(func() {
		log.SetOutput(stdWriter)
	})
	log.SetOutput(w)
}

// ReplaceLogLevel sets logging level to l and uses Cleanup method of t to
// revert changes.
func ReplaceLogLevel(t *testing.T, l int) {
	switch l {
	case log.INFO, log.DEBUG, log.ERROR:
		// Go on.
	default:
		t.Fatalf("wrong l value (must be one of %v, %v, %v)", log.INFO, log.DEBUG, log.ERROR)
	}

	stdLevel := log.GetLevel()
	t.Cleanup(func() {
		log.SetLevel(stdLevel)
	})
	log.SetLevel(l)
}
