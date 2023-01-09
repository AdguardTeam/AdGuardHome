package updater

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(a.garipov): Rewrite these tests.

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func startHTTPServer(data string) (l net.Listener, portStr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(data))
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	go func() { _ = http.Serve(listener, mux) }()
	return listener, strconv.FormatUint(uint64(listener.Addr().(*net.TCPAddr).Port), 10)
}

func TestUpdateGetVersion(t *testing.T) {
	const jsonData = `{
  "version": "v0.103.0-beta.2",
  "announcement": "AdGuard Home v0.103.0-beta.2 is now available!",
  "announcement_url": "https://github.com/AdguardTeam/AdGuardHome/internal/releases",
  "selfupdate_min_version": "v0.0",
  "download_windows_amd64": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_windows_amd64.zip",
  "download_windows_386": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_windows_386.zip",
  "download_darwin_amd64": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_darwin_amd64.zip",
  "download_darwin_386": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_darwin_386.zip",
  "download_linux_amd64": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_amd64.tar.gz",
  "download_linux_386": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_386.tar.gz",
  "download_linux_arm": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv6.tar.gz",
  "download_linux_armv5": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv5.tar.gz",
  "download_linux_armv6": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv6.tar.gz",
  "download_linux_armv7": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv7.tar.gz",
  "download_linux_arm64": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_arm64.tar.gz",
  "download_linux_mips": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mips_softfloat.tar.gz",
  "download_linux_mipsle": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mipsle_softfloat.tar.gz",
  "download_linux_mips64": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mips64_softfloat.tar.gz",
  "download_linux_mips64le": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mips64le_softfloat.tar.gz",
  "download_freebsd_386": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_386.tar.gz",
  "download_freebsd_amd64": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_amd64.tar.gz",
  "download_freebsd_arm": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_armv6.tar.gz",
  "download_freebsd_armv5": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_armv5.tar.gz",
  "download_freebsd_armv6": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_armv6.tar.gz",
  "download_freebsd_armv7": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_armv7.tar.gz",
  "download_freebsd_arm64": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_freebsd_arm64.tar.gz"
}`

	l, lport := startHTTPServer(jsonData)
	testutil.CleanupAndRequireSuccess(t, l.Close)

	u := NewUpdater(&Config{
		Client:  &http.Client{},
		Version: "v0.103.0-beta.1",
		Channel: version.ChannelBeta,
		GOARCH:  "arm",
		GOOS:    "linux",
	})

	fakeURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("127.0.0.1", lport),
		Path:   path.Join("adguardhome", version.ChannelBeta, "version.json"),
	}
	u.versionCheckURL = fakeURL.String()

	info, err := u.VersionInfo(false)
	require.NoError(t, err)

	assert.Equal(t, "v0.103.0-beta.2", info.NewVersion)
	assert.Equal(t, "AdGuard Home v0.103.0-beta.2 is now available!", info.Announcement)
	assert.Equal(t, "https://github.com/AdguardTeam/AdGuardHome/internal/releases", info.AnnouncementURL)
	assert.Equal(t, aghalg.NBTrue, info.CanAutoUpdate)

	// check cached
	_, err = u.VersionInfo(false)
	require.NoError(t, err)
}

func TestUpdate(t *testing.T) {
	wd := t.TempDir()

	exePath := filepath.Join(wd, "AdGuardHome")
	yamlPath := filepath.Join(wd, "AdGuardHome.yaml")
	readmePath := filepath.Join(wd, "README.md")
	licensePath := filepath.Join(wd, "LICENSE.txt")

	require.NoError(t, os.WriteFile(exePath, []byte("AdGuardHome"), 0o755))
	require.NoError(t, os.WriteFile(yamlPath, []byte("AdGuardHome.yaml"), 0o644))
	require.NoError(t, os.WriteFile(readmePath, []byte("README.md"), 0o644))
	require.NoError(t, os.WriteFile(licensePath, []byte("LICENSE.txt"), 0o644))

	// start server for returning package file
	pkgData, err := os.ReadFile("testdata/AdGuardHome.tar.gz")
	require.NoError(t, err)

	l, lport := startHTTPServer(string(pkgData))
	testutil.CleanupAndRequireSuccess(t, l.Close)

	u := NewUpdater(&Config{
		Client:  &http.Client{},
		Version: "v0.103.0",
	})

	fakeURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("127.0.0.1", lport),
		Path:   "AdGuardHome.tar.gz",
	}

	u.workDir = wd
	u.confName = yamlPath
	u.newVersion = "v0.103.1"
	u.packageURL = fakeURL.String()

	require.NoError(t, u.prepare(exePath))
	require.NoError(t, u.downloadPackageFile())
	require.NoError(t, u.unpack())
	// require.NoError(t, u.check())
	require.NoError(t, u.backup(false))
	require.NoError(t, u.replace())

	u.clean()

	// check backup files
	d, err := os.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "AdGuardHome.yaml", string(d))

	d, err = os.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome"))
	require.NoError(t, err)

	assert.Equal(t, "AdGuardHome", string(d))

	// check updated files
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
}

func TestUpdateWindows(t *testing.T) {
	wd := t.TempDir()

	exePath := filepath.Join(wd, "AdGuardHome.exe")
	yamlPath := filepath.Join(wd, "AdGuardHome.yaml")
	readmePath := filepath.Join(wd, "README.md")
	licensePath := filepath.Join(wd, "LICENSE.txt")

	require.NoError(t, os.WriteFile(exePath, []byte("AdGuardHome.exe"), 0o755))
	require.NoError(t, os.WriteFile(yamlPath, []byte("AdGuardHome.yaml"), 0o644))
	require.NoError(t, os.WriteFile(readmePath, []byte("README.md"), 0o644))
	require.NoError(t, os.WriteFile(licensePath, []byte("LICENSE.txt"), 0o644))

	// start server for returning package file
	pkgData, err := os.ReadFile("testdata/AdGuardHome.zip")
	require.NoError(t, err)

	l, lport := startHTTPServer(string(pkgData))
	testutil.CleanupAndRequireSuccess(t, l.Close)

	u := NewUpdater(&Config{
		Client:  &http.Client{},
		GOOS:    "windows",
		Version: "v0.103.0",
	})

	fakeURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("127.0.0.1", lport),
		Path:   "AdGuardHome.zip",
	}

	u.workDir = wd
	u.confName = yamlPath
	u.newVersion = "v0.103.1"
	u.packageURL = fakeURL.String()

	require.NoError(t, u.prepare(exePath))
	require.NoError(t, u.downloadPackageFile())
	require.NoError(t, u.unpack())
	// assert.Nil(t, u.check())
	require.NoError(t, u.backup(false))
	require.NoError(t, u.replace())

	u.clean()

	// check backup files
	d, err := os.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "AdGuardHome.yaml", string(d))

	d, err = os.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.exe"))
	require.NoError(t, err)

	assert.Equal(t, "AdGuardHome.exe", string(d))

	// check updated files
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
}

func TestUpdater_VersionInto_ARM(t *testing.T) {
	const jsonData = `{
  "version": "v0.103.0-beta.2",
  "announcement": "AdGuard Home v0.103.0-beta.2 is now available!",
  "announcement_url": "https://github.com/AdguardTeam/AdGuardHome/internal/releases",
  "selfupdate_min_version": "v0.0",
  "download_linux_armv7": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_armv7.tar.gz"
}`

	l, lport := startHTTPServer(jsonData)
	testutil.CleanupAndRequireSuccess(t, l.Close)

	u := NewUpdater(&Config{
		Client:  &http.Client{},
		Version: "v0.103.0-beta.1",
		Channel: version.ChannelBeta,
		GOARCH:  "arm",
		GOOS:    "linux",
		GOARM:   "7",
	})

	fakeURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("127.0.0.1", lport),
		Path:   path.Join("adguardhome", version.ChannelBeta, "version.json"),
	}
	u.versionCheckURL = fakeURL.String()

	info, err := u.VersionInfo(false)
	require.NoError(t, err)

	assert.Equal(t, "v0.103.0-beta.2", info.NewVersion)
	assert.Equal(t, "AdGuard Home v0.103.0-beta.2 is now available!", info.Announcement)
	assert.Equal(t, "https://github.com/AdguardTeam/AdGuardHome/internal/releases", info.AnnouncementURL)
	assert.Equal(t, aghalg.NBTrue, info.CanAutoUpdate)
}

func TestUpdater_VersionInto_MIPS(t *testing.T) {
	const jsonData = `{
  "version": "v0.103.0-beta.2",
  "announcement": "AdGuard Home v0.103.0-beta.2 is now available!",
  "announcement_url": "https://github.com/AdguardTeam/AdGuardHome/internal/releases",
  "selfupdate_min_version": "v0.0",
  "download_linux_mips_softfloat": "https://static.adtidy.org/adguardhome/beta/AdGuardHome_linux_mips_softfloat.tar.gz"
}`

	l, lport := startHTTPServer(jsonData)
	testutil.CleanupAndRequireSuccess(t, l.Close)

	u := NewUpdater(&Config{
		Client:  &http.Client{},
		Version: "v0.103.0-beta.1",
		Channel: version.ChannelBeta,
		GOARCH:  "mips",
		GOOS:    "linux",
		GOMIPS:  "softfloat",
	})

	fakeURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("127.0.0.1", lport),
		Path:   path.Join("adguardhome", version.ChannelBeta, "version.json"),
	}
	u.versionCheckURL = fakeURL.String()

	info, err := u.VersionInfo(false)
	require.NoError(t, err)

	assert.Equal(t, "v0.103.0-beta.2", info.NewVersion)
	assert.Equal(t, "AdGuard Home v0.103.0-beta.2 is now available!", info.Announcement)
	assert.Equal(t, "https://github.com/AdguardTeam/AdGuardHome/internal/releases", info.AnnouncementURL)
	assert.Equal(t, aghalg.NBTrue, info.CanAutoUpdate)
}
