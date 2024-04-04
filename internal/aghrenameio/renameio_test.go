package aghrenameio_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghrenameio"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPerm is the common permission mode for tests.
const testPerm fs.FileMode = 0o644

// Common file data for tests.
var (
	initialData = []byte("initial data\n")
	newData     = []byte("new data\n")
)

func TestPendingFile(t *testing.T) {
	t.Parallel()

	targetPath := newInitialFile(t)
	f, err := aghrenameio.NewPendingFile(targetPath, testPerm)
	require.NoError(t, err)

	_, err = f.Write(newData)
	require.NoError(t, err)

	err = f.CloseReplace()
	require.NoError(t, err)

	gotData, err := os.ReadFile(targetPath)
	require.NoError(t, err)

	assert.Equal(t, newData, gotData)
}

// newInitialFile is a test helper that returns the path to the file containing
// [initialData].
func newInitialFile(t *testing.T) (targetPath string) {
	t.Helper()

	dir := t.TempDir()
	targetPath = filepath.Join(dir, "target")

	err := os.WriteFile(targetPath, initialData, 0o644)
	require.NoError(t, err)

	return targetPath
}

func TestWithDeferredCleanup(t *testing.T) {
	t.Parallel()

	const testError errors.Error = "test error"

	testCases := []struct {
		error      error
		name       string
		wantErrMsg string
		wantData   []byte
	}{{
		name:       "success",
		error:      nil,
		wantErrMsg: "",
		wantData:   newData,
	}, {
		name:       "error",
		error:      testError,
		wantErrMsg: testError.Error(),
		wantData:   initialData,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			targetPath := newInitialFile(t)
			f, err := aghrenameio.NewPendingFile(targetPath, testPerm)
			require.NoError(t, err)

			_, err = f.Write(newData)
			require.NoError(t, err)

			err = aghrenameio.WithDeferredCleanup(tc.error, f)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			gotData, err := os.ReadFile(targetPath)
			require.NoError(t, err)

			assert.Equal(t, tc.wantData, gotData)
		})
	}
}
