package aghtest

import (
	"testing"
)

// PrepareTestDir returns the full path to temporary created directory and
// registers the appropriate cleanup for *t.
func PrepareTestDir(t *testing.T) (dir string) {
	t.Helper()

	return prepareTestDir(t)
}
