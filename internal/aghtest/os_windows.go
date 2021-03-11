// +build windows

package aghtest

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxRetryDur = 1000 * time.Millisecond
	retryDur    = 5 * time.Millisecond
)

func prepareTestDir(t *testing.T) (dir string) {
	// Windows, including the version of Windows Server that Github Actions
	// uses, apparently likes to overly eagerly inspect new directories with
	// its Windows Defender.  Disabling it might require additional
	// workarounds, and until we've figured it out, just retry the deletion
	// until the error goes away.
	//
	// The code is largely inspired by the one that has been introduced into
	// the go command itself.  We should probably make a proposal to use the
	// same mechanism in t.TempDir.
	//
	// See https://go-review.googlesource.com/c/go/+/172337.
	//
	// See https://github.com/golang/go/issues/44919.

	t.Helper()

	wd, err := os.Getwd()
	require.Nil(t, err)

	dir, err = ioutil.TempDir(wd, "agh-test")
	require.Nil(t, err)
	require.NotEmpty(t, dir)

	t.Cleanup(func() {
		start := time.Now()
		for {
			err = os.RemoveAll(dir)
			if err == nil {
				break
			}

			if time.Since(start) >= maxRetryDur {
				break
			}

			time.Sleep(retryDur)
		}

		assert.Nil(t, err, "after %s", time.Since(start))
	})

	return dir
}
