package aghtest

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PrepareTestDir returns the full path to temporary created directory and
// registers the appropriate cleanup for *t.
func PrepareTestDir(t *testing.T) (dir string) {
	t.Helper()

	wd, err := os.Getwd()
	require.Nil(t, err)

	dir, err = ioutil.TempDir(wd, "agh-test")
	require.Nil(t, err)
	require.NotEmpty(t, dir)

	t.Cleanup(func() {
		// TODO(e.burkov): Replace with t.TempDir methods after updating
		// go version to 1.15.
		start := time.Now()
		for {
			err := os.RemoveAll(dir)
			if err == nil {
				break
			}

			if runtime.GOOS != "windows" || time.Since(start) >= 500*time.Millisecond {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		assert.Nil(t, err)
	})

	return dir
}
