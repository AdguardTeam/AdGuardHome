package updater

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/stretchr/testify/assert"
)

// TODO(a.garipov): Rewrite these tests.

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
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
  "download_windows_amd64": "https://static.adguard.com/adguardhome/beta/AdGuardHome_windows_amd64.zip",
  "download_windows_386": "https://static.adguard.com/adguardhome/beta/AdGuardHome_windows_386.zip",
  "download_darwin_amd64": "https://static.adguard.com/adguardhome/beta/AdGuardHome_darwin_amd64.zip",
  "download_darwin_386": "https://static.adguard.com/adguardhome/beta/AdGuardHome_darwin_386.zip",
  "download_linux_amd64": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_amd64.tar.gz",
  "download_linux_386": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_386.tar.gz",
  "download_linux_arm": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_armv6.tar.gz",
  "download_linux_armv5": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_armv5.tar.gz",
  "download_linux_armv6": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_armv6.tar.gz",
  "download_linux_armv7": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_armv7.tar.gz",
  "download_linux_arm64": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_arm64.tar.gz",
  "download_linux_mips": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_mips_softfloat.tar.gz",
  "download_linux_mipsle": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_mipsle_softfloat.tar.gz",
  "download_linux_mips64": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_mips64_softfloat.tar.gz",
  "download_linux_mips64le": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_mips64le_softfloat.tar.gz",
  "download_freebsd_386": "https://static.adguard.com/adguardhome/beta/AdGuardHome_freebsd_386.tar.gz",
  "download_freebsd_amd64": "https://static.adguard.com/adguardhome/beta/AdGuardHome_freebsd_amd64.tar.gz",
  "download_freebsd_arm": "https://static.adguard.com/adguardhome/beta/AdGuardHome_freebsd_armv6.tar.gz",
  "download_freebsd_armv5": "https://static.adguard.com/adguardhome/beta/AdGuardHome_freebsd_armv5.tar.gz",
  "download_freebsd_armv6": "https://static.adguard.com/adguardhome/beta/AdGuardHome_freebsd_armv6.tar.gz",
  "download_freebsd_armv7": "https://static.adguard.com/adguardhome/beta/AdGuardHome_freebsd_armv7.tar.gz",
  "download_freebsd_arm64": "https://static.adguard.com/adguardhome/beta/AdGuardHome_freebsd_arm64.tar.gz"
}`

	l, lport := startHTTPServer(jsonData)
	t.Cleanup(func() { assert.Nil(t, l.Close()) })

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
	assert.Nil(t, err)
	assert.Equal(t, "v0.103.0-beta.2", info.NewVersion)
	assert.Equal(t, "AdGuard Home v0.103.0-beta.2 is now available!", info.Announcement)
	assert.Equal(t, "https://github.com/AdguardTeam/AdGuardHome/internal/releases", info.AnnouncementURL)
	assert.Equal(t, "v0.0", info.SelfUpdateMinVersion)
	if assert.NotNil(t, info.CanAutoUpdate) {
		assert.True(t, *info.CanAutoUpdate)
	}

	// check cached
	_, err = u.VersionInfo(false)
	assert.Nil(t, err)
}

func TestUpdate(t *testing.T) {
	wd := t.TempDir()

	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "AdGuardHome"), []byte("AdGuardHome"), 0o755))
	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "README.md"), []byte("README.md"), 0o644))
	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "LICENSE.txt"), []byte("LICENSE.txt"), 0o644))
	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "AdGuardHome.yaml"), []byte("AdGuardHome.yaml"), 0o644))

	// start server for returning package file
	pkgData, err := ioutil.ReadFile("testdata/AdGuardHome.tar.gz")
	assert.Nil(t, err)
	l, lport := startHTTPServer(string(pkgData))
	t.Cleanup(func() { assert.Nil(t, l.Close()) })

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
	u.confName = filepath.Join(u.workDir, "AdGuardHome.yaml")
	u.newVersion = "v0.103.1"
	u.packageURL = fakeURL.String()

	assert.Nil(t, u.prepare())
	u.currentExeName = filepath.Join(wd, "AdGuardHome")
	assert.Nil(t, u.downloadPackageFile(u.packageURL, u.packageName))
	assert.Nil(t, u.unpack())
	// assert.Nil(t, u.check())
	assert.Nil(t, u.backup())
	assert.Nil(t, u.replace())
	u.clean()

	// check backup files
	d, err := ioutil.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.yaml"))
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome"))
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome", string(d))

	// check updated files
	d, err = ioutil.ReadFile(filepath.Join(wd, "AdGuardHome"))
	assert.Nil(t, err)
	assert.Equal(t, "1", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "README.md"))
	assert.Nil(t, err)
	assert.Equal(t, "2", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "LICENSE.txt"))
	assert.Nil(t, err)
	assert.Equal(t, "3", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "AdGuardHome.yaml"))
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))
}

func TestUpdateWindows(t *testing.T) {
	wd := t.TempDir()

	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "AdGuardHome.exe"), []byte("AdGuardHome.exe"), 0o755))
	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "README.md"), []byte("README.md"), 0o644))
	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "LICENSE.txt"), []byte("LICENSE.txt"), 0o644))
	assert.Nil(t, ioutil.WriteFile(filepath.Join(wd, "AdGuardHome.yaml"), []byte("AdGuardHome.yaml"), 0o644))

	// start server for returning package file
	pkgData, err := ioutil.ReadFile("testdata/AdGuardHome.zip")
	assert.Nil(t, err)

	l, lport := startHTTPServer(string(pkgData))
	t.Cleanup(func() { assert.Nil(t, l.Close()) })

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
	u.confName = filepath.Join(u.workDir, "AdGuardHome.yaml")
	u.newVersion = "v0.103.1"
	u.packageURL = fakeURL.String()

	assert.Nil(t, u.prepare())
	u.currentExeName = filepath.Join(wd, "AdGuardHome.exe")
	assert.Nil(t, u.downloadPackageFile(u.packageURL, u.packageName))
	assert.Nil(t, u.unpack())
	// assert.Nil(t, u.check())
	assert.Nil(t, u.backup())
	assert.Nil(t, u.replace())
	u.clean()

	// check backup files
	d, err := ioutil.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.yaml"))
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "agh-backup", "AdGuardHome.exe"))
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.exe", string(d))

	// check updated files
	d, err = ioutil.ReadFile(filepath.Join(wd, "AdGuardHome.exe"))
	assert.Nil(t, err)
	assert.Equal(t, "1", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "README.md"))
	assert.Nil(t, err)
	assert.Equal(t, "2", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "LICENSE.txt"))
	assert.Nil(t, err)
	assert.Equal(t, "3", string(d))

	d, err = ioutil.ReadFile(filepath.Join(wd, "AdGuardHome.yaml"))
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))
}

func TestUpdater_VersionInto_ARM(t *testing.T) {
	const jsonData = `{
  "version": "v0.103.0-beta.2",
  "announcement": "AdGuard Home v0.103.0-beta.2 is now available!",
  "announcement_url": "https://github.com/AdguardTeam/AdGuardHome/internal/releases",
  "selfupdate_min_version": "v0.0",
  "download_linux_armv7": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_armv7.tar.gz"
}`

	l, lport := startHTTPServer(jsonData)
	t.Cleanup(func() { assert.Nil(t, l.Close()) })

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
	assert.Nil(t, err)
	assert.Equal(t, "v0.103.0-beta.2", info.NewVersion)
	assert.Equal(t, "AdGuard Home v0.103.0-beta.2 is now available!", info.Announcement)
	assert.Equal(t, "https://github.com/AdguardTeam/AdGuardHome/internal/releases", info.AnnouncementURL)
	assert.Equal(t, "v0.0", info.SelfUpdateMinVersion)
	if assert.NotNil(t, info.CanAutoUpdate) {
		assert.True(t, *info.CanAutoUpdate)
	}
}

func TestUpdater_VersionInto_MIPS(t *testing.T) {
	const jsonData = `{
  "version": "v0.103.0-beta.2",
  "announcement": "AdGuard Home v0.103.0-beta.2 is now available!",
  "announcement_url": "https://github.com/AdguardTeam/AdGuardHome/internal/releases",
  "selfupdate_min_version": "v0.0",
  "download_linux_mips_softfloat": "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_mips_softfloat.tar.gz"
}`

	l, lport := startHTTPServer(jsonData)
	t.Cleanup(func() { assert.Nil(t, l.Close()) })

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
	assert.Nil(t, err)
	assert.Equal(t, "v0.103.0-beta.2", info.NewVersion)
	assert.Equal(t, "AdGuard Home v0.103.0-beta.2 is now available!", info.Announcement)
	assert.Equal(t, "https://github.com/AdguardTeam/AdGuardHome/internal/releases", info.AnnouncementURL)
	assert.Equal(t, "v0.0", info.SelfUpdateMinVersion)
	if assert.NotNil(t, info.CanAutoUpdate) {
		assert.True(t, *info.CanAutoUpdate)
	}
}
