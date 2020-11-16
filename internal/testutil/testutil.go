// Package testutil contains utilities for testing.
package testutil

import (
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
