package home

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFilePath(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)

	workDir := t.TempDir()
	targetPath := filepath.Join(workDir, "real.yaml")
	linkPath := filepath.Join(workDir, "link.yaml")

	err = os.Symlink(targetPath, linkPath)
	require.NoError(t, err)

	_, err = os.Create(targetPath)
	require.NoError(t, err)

	otherDir := t.TempDir()

	testCases := []struct {
		name     string
		chDir    string
		confPath string
		want     string
		workDir  string
	}{{
		name:     "symlink_after_abs",
		chDir:    otherDir,
		confPath: "link.yaml",
		want:     targetPath,
		workDir:  workDir,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.chDir != "" {
				err = os.Chdir(tc.chDir)
				require.NoError(t, err)

				testutil.CleanupAndRequireSuccess(t, func() (err error) {
					return os.Chdir(origDir)
				})
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			got := configFilePath(ctx, testLogger, tc.workDir, tc.confPath)
			assert.Equal(t, tc.want, got)
		})
	}
}
