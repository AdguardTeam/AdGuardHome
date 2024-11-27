package updater_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func TestUpdater_Update(t *testing.T) {
	const jsonData = `{
  "version": "v0.103.0-beta.2",
  "announcement": "AdGuard Home v0.103.0-beta.2 is now available!",
  "announcement_url": "https://github.com/AdguardTeam/AdGuardHome/internal/releases",
  "selfupdate_min_version": "v0.0",
  "download_linux_amd64": "%s"
}`

	const packagePath = "/AdGuardHome.tar.gz"

	wd := t.TempDir()

	exePath := filepath.Join(wd, "AdGuardHome")
	yamlPath := filepath.Join(wd, "AdGuardHome.yaml")
	readmePath := filepath.Join(wd, "README.md")
	licensePath := filepath.Join(wd, "LICENSE.txt")

	require.NoError(t, os.WriteFile(exePath, []byte("AdGuardHome"), 0o755))
	require.NoError(t, os.WriteFile(yamlPath, []byte("AdGuardHome.yaml"), 0o644))
	require.NoError(t, os.WriteFile(readmePath, []byte("README.md"), 0o644))
	require.NoError(t, os.WriteFile(licensePath, []byte("LICENSE.txt"), 0o644))

	pkgData, err := os.ReadFile("testdata/AdGuardHome_unix.tar.gz")
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc(packagePath, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(pkgData)
	})

	versionPath := path.Join("/adguardhome", version.ChannelBeta, "version.json")
	mux.HandleFunc(versionPath, func(w http.ResponseWriter, r *http.Request) {
		var u string
		u, err = url.JoinPath("http://", r.Host, packagePath)
		require.NoError(t, err)

		_, _ = fmt.Fprintf(w, jsonData, u)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	srvURL, err := url.Parse(srv.URL)
	require.NoError(t, err)

	versionCheckURL := srvURL.JoinPath(versionPath)
	require.NoError(t, err)

	u := updater.NewUpdater(&updater.Config{
		Client:          srv.Client(),
		GOARCH:          "amd64",
		GOOS:            "linux",
		Version:         "v0.103.0",
		ConfName:        yamlPath,
		WorkDir:         wd,
		ExecPath:        exePath,
		VersionCheckURL: versionCheckURL,
	})

	_, err = u.VersionInfo(false)
	require.NoError(t, err)

	err = u.Update(true)
	require.NoError(t, err)

	// check backup files
	d, err := os.ReadFile(filepath.Join(wd, "agh-backup", "LICENSE.txt"))
	require.NoError(t, err)

	assert.Equal(t, "LICENSE.txt", string(d))

	d, err = os.ReadFile(filepath.Join(wd, "agh-backup", "README.md"))
	require.NoError(t, err)

	assert.Equal(t, "README.md", string(d))

	// check updated files
	_, err = os.Stat(exePath)
	require.NoError(t, err)

	d, err = os.ReadFile(readmePath)
	require.NoError(t, err)

	assert.Equal(t, "2", string(d))

	d, err = os.ReadFile(licensePath)
	require.NoError(t, err)

	assert.Equal(t, "3", string(d))

	d, err = os.ReadFile(yamlPath)
	require.NoError(t, err)

	assert.Equal(t, "AdGuardHome.yaml", string(d))

	t.Run("config_check", func(t *testing.T) {
		// TODO(s.chzhen):  Test on Windows also.
		if runtime.GOOS == "windows" {
			t.Skip("skipping config check test on windows")
		}

		err = u.Update(false)
		assert.NoError(t, err)
	})

	t.Run("api_fail", func(t *testing.T) {
		srv.Close()

		err = u.Update(true)
		var urlErr *url.Error
		assert.ErrorAs(t, err, &urlErr)
	})
}
