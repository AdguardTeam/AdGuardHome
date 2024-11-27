package updater

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdater_internal(t *testing.T) {
	wd := t.TempDir()

	exePathUnix := filepath.Join(wd, "AdGuardHome.exe")
	exePathWindows := filepath.Join(wd, "AdGuardHome")
	yamlPath := filepath.Join(wd, "AdGuardHome.yaml")
	readmePath := filepath.Join(wd, "README.md")
	licensePath := filepath.Join(wd, "LICENSE.txt")

	require.NoError(t, os.WriteFile(exePathUnix, []byte("AdGuardHome.exe"), 0o755))
	require.NoError(t, os.WriteFile(exePathWindows, []byte("AdGuardHome"), 0o755))
	require.NoError(t, os.WriteFile(yamlPath, []byte("AdGuardHome.yaml"), 0o644))
	require.NoError(t, os.WriteFile(readmePath, []byte("README.md"), 0o644))
	require.NoError(t, os.WriteFile(licensePath, []byte("LICENSE.txt"), 0o644))

	testCases := []struct {
		name        string
		exeName     string
		os          string
		archiveName string
	}{{
		name:        "unix",
		os:          "linux",
		exeName:     "AdGuardHome",
		archiveName: "AdGuardHome.tar.gz",
	}, {
		name:        "windows",
		os:          "windows",
		exeName:     "AdGuardHome.exe",
		archiveName: "AdGuardHome.zip",
	}}

	for _, tc := range testCases {
		exePath := filepath.Join(wd, tc.exeName)

		// Start server for returning package file.
		pkgData, err := os.ReadFile(filepath.Join("testdata", tc.archiveName))
		require.NoError(t, err)

		fakeClient, fakeURL := aghtest.StartHTTPServer(t, pkgData)
		fakeURL = fakeURL.JoinPath(tc.archiveName)

		u := NewUpdater(&Config{
			Client:   fakeClient,
			GOOS:     tc.os,
			Version:  "v0.103.0",
			ExecPath: exePath,
			WorkDir:  wd,
			ConfName: yamlPath,
			// TODO(e.burkov):  Rewrite the test to use a fake version check
			// URL with a fake URLs for the package files.
			VersionCheckURL: &url.URL{},
		})

		u.newVersion = "v0.103.1"
		u.packageURL = fakeURL.String()

		require.NoError(t, u.prepare())
		require.NoError(t, u.downloadPackageFile())
		require.NoError(t, u.unpack())
		require.NoError(t, u.backup(false))
		require.NoError(t, u.replace())

		u.clean()

		require.True(t, t.Run("backup", func(t *testing.T) {
			var d []byte
			d, err = os.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.yaml"))
			require.NoError(t, err)

			assert.Equal(t, "AdGuardHome.yaml", string(d))

			d, err = os.ReadFile(filepath.Join(wd, "agh-backup", tc.exeName))
			require.NoError(t, err)

			assert.Equal(t, tc.exeName, string(d))
		}))

		require.True(t, t.Run("updated", func(t *testing.T) {
			var d []byte
			d, err = os.ReadFile(exePath)
			require.NoError(t, err)

			assert.Equal(t, "1", string(d))

			d, err = os.ReadFile(readmePath)
			require.NoError(t, err)

			assert.Equal(t, "2", string(d))

			d, err = os.ReadFile(licensePath)
			require.NoError(t, err)

			assert.Equal(t, "3", string(d))

			d, err = os.ReadFile(yamlPath)
			require.NoError(t, err)

			assert.Equal(t, "AdGuardHome.yaml", string(d))
		}))
	}
}
