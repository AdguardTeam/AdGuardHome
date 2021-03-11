// +build !windows

package aghtest

import (
	"testing"
)

func prepareTestDir(t *testing.T) (dir string) {
	t.Helper()

	return t.TempDir()
}
