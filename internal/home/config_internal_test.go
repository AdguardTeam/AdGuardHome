package home

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/configmigrate"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFilePath(t *testing.T) {
	const (
		realConf       = "real.yaml"
		linkConf       = "conf.link"
		missingConf    = "missing.yaml"
		brokenLinkConf = "broken.link"
	)

	workDir := t.TempDir()
	targetPath := filepath.Join(workDir, realConf)
	linkPath := filepath.Join(workDir, linkConf)
	missingPath := filepath.Join(workDir, missingConf)
	brokenLinkPath := filepath.Join(workDir, brokenLinkConf)

	err := os.Symlink(targetPath, linkPath)
	require.NoError(t, err)

	err = os.Symlink(missingPath, brokenLinkPath)
	require.NoError(t, err)

	f, err := os.Create(targetPath)
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, f.Close)

	otherDir := t.TempDir()

	// Canonicalize the absolute path (e.g., on macOS: /var -> /private/var; on
	// Windows: RUNNER~1 -> runneradmin).
	wantAbs := targetPath
	p, err := filepath.EvalSymlinks(wantAbs)
	if err == nil {
		wantAbs = p
	}

	testCases := []struct {
		name     string
		chDir    string
		confPath string
		want     string
	}{{
		name:     "absolute_path",
		chDir:    "",
		confPath: targetPath,
		want:     wantAbs,
	}, {
		name:     "relative_path",
		chDir:    "",
		confPath: realConf,
		want:     targetPath,
	}, {
		name:     "symlink",
		chDir:    "",
		confPath: linkConf,
		want:     linkPath,
	}, {
		name:     "symlink_broken",
		chDir:    "",
		confPath: brokenLinkConf,
		want:     brokenLinkPath,
	}, {
		name:     "symlink_before_join",
		chDir:    otherDir,
		confPath: linkConf,
		want:     linkPath,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.chDir != "" {
				t.Chdir(tc.chDir)
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			got := configFilePath(ctx, testLogger, workDir, tc.confPath)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseConfig_Migration(t *testing.T) {
	const fixturePath = "../configmigrate/testdata/TestMigrateConfig_Migrate/v34/input.yml"

	want, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		readOnly    bool
		wantChanged bool
	}{{
		name:        "normal",
		readOnly:    false,
		wantChanged: true,
	}, {
		name:        "read_only",
		readOnly:    true,
		wantChanged: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storeGlobals(t)
			config = &configuration{}

			workDir := t.TempDir()
			confPath := filepath.Join(workDir, "AdGuardHome.yaml")
			writeErr := os.WriteFile(confPath, want, 0o600)
			require.NoError(t, writeErr)

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			parseErr := parseConfig(ctx, testLogger, workDir, confPath, tc.readOnly)
			require.NoError(t, parseErr)

			got, readErr := os.ReadFile(confPath)
			require.NoError(t, readErr)

			assert.Equal(t, tc.wantChanged, !bytes.Equal(want, got))
			assert.Equal(t, uint(configmigrate.LastSchemaVersion), config.SchemaVersion)
		})
	}
}
