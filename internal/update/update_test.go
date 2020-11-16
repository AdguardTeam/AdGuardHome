package update

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func startHTTPServer(data string) (net.Listener, uint16) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(data))
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	go func() { _ = http.Serve(listener, mux) }()
	return listener, uint16(listener.Addr().(*net.TCPAddr).Port)
}

func TestUpdateGetVersion(t *testing.T) {
	const jsonData = `{
  "version": "v0.103.0-beta2",
  "announcement": "AdGuard Home v0.103.0-beta2 is now available!",
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
	defer func() { _ = l.Close() }()

	u := NewUpdater(Config{
		Client:        &http.Client{},
		VersionURL:    fmt.Sprintf("http://127.0.0.1:%d/", lport),
		OS:            "linux",
		Arch:          "arm",
		VersionString: "v0.103.0-beta1",
	})

	info, err := u.GetVersionResponse(false)
	assert.Nil(t, err)
	assert.Equal(t, "v0.103.0-beta2", info.NewVersion)
	assert.Equal(t, "AdGuard Home v0.103.0-beta2 is now available!", info.Announcement)
	assert.Equal(t, "https://github.com/AdguardTeam/AdGuardHome/internal/releases", info.AnnouncementURL)
	assert.Equal(t, "v0.0", info.SelfUpdateMinVersion)
	assert.True(t, info.CanAutoUpdate)

	_ = l.Close()

	// check cached
	_, err = u.GetVersionResponse(false)
	assert.Nil(t, err)
}

func TestUpdate(t *testing.T) {
	_ = os.Mkdir("aghtest", 0755)
	defer func() {
		_ = os.RemoveAll("aghtest")
	}()

	// create "current" files
	assert.Nil(t, ioutil.WriteFile("aghtest/AdGuardHome", []byte("AdGuardHome"), 0755))
	assert.Nil(t, ioutil.WriteFile("aghtest/README.md", []byte("README.md"), 0644))
	assert.Nil(t, ioutil.WriteFile("aghtest/LICENSE.txt", []byte("LICENSE.txt"), 0644))
	assert.Nil(t, ioutil.WriteFile("aghtest/AdGuardHome.yaml", []byte("AdGuardHome.yaml"), 0644))

	// start server for returning package file
	pkgData, err := ioutil.ReadFile("test/AdGuardHome.tar.gz")
	assert.Nil(t, err)
	l, lport := startHTTPServer(string(pkgData))
	defer func() { _ = l.Close() }()

	u := NewUpdater(Config{
		Client:        &http.Client{},
		PackageURL:    fmt.Sprintf("http://127.0.0.1:%d/AdGuardHome.tar.gz", lport),
		VersionString: "v0.103.0",
		NewVersion:    "v0.103.1",
		ConfigName:    "aghtest/AdGuardHome.yaml",
		WorkDir:       "aghtest",
	})

	assert.Nil(t, u.prepare())
	u.currentExeName = "aghtest/AdGuardHome"
	assert.Nil(t, u.downloadPackageFile(u.PackageURL, u.packageName))
	assert.Nil(t, u.unpack())
	// assert.Nil(t, u.check())
	assert.Nil(t, u.backup())
	assert.Nil(t, u.replace())
	u.clean()

	// check backup files
	d, err := ioutil.ReadFile("aghtest/agh-backup/AdGuardHome.yaml")
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))

	d, err = ioutil.ReadFile("aghtest/agh-backup/AdGuardHome")
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome", string(d))

	// check updated files
	d, err = ioutil.ReadFile("aghtest/AdGuardHome")
	assert.Nil(t, err)
	assert.Equal(t, "1", string(d))

	d, err = ioutil.ReadFile("aghtest/README.md")
	assert.Nil(t, err)
	assert.Equal(t, "2", string(d))

	d, err = ioutil.ReadFile("aghtest/LICENSE.txt")
	assert.Nil(t, err)
	assert.Equal(t, "3", string(d))

	d, err = ioutil.ReadFile("aghtest/AdGuardHome.yaml")
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))
}

func TestUpdateWindows(t *testing.T) {
	_ = os.Mkdir("aghtest", 0755)
	defer func() {
		_ = os.RemoveAll("aghtest")
	}()

	// create "current" files
	assert.Nil(t, ioutil.WriteFile("aghtest/AdGuardHome.exe", []byte("AdGuardHome.exe"), 0755))
	assert.Nil(t, ioutil.WriteFile("aghtest/README.md", []byte("README.md"), 0644))
	assert.Nil(t, ioutil.WriteFile("aghtest/LICENSE.txt", []byte("LICENSE.txt"), 0644))
	assert.Nil(t, ioutil.WriteFile("aghtest/AdGuardHome.yaml", []byte("AdGuardHome.yaml"), 0644))

	// start server for returning package file
	pkgData, err := ioutil.ReadFile("test/AdGuardHome.zip")
	assert.Nil(t, err)
	l, lport := startHTTPServer(string(pkgData))
	defer func() { _ = l.Close() }()

	u := NewUpdater(Config{
		WorkDir:       "aghtest",
		Client:        &http.Client{},
		PackageURL:    fmt.Sprintf("http://127.0.0.1:%d/AdGuardHome.zip", lport),
		OS:            "windows",
		VersionString: "v0.103.0",
		NewVersion:    "v0.103.1",
		ConfigName:    "aghtest/AdGuardHome.yaml",
	})

	assert.Nil(t, u.prepare())
	u.currentExeName = "aghtest/AdGuardHome.exe"
	assert.Nil(t, u.downloadPackageFile(u.PackageURL, u.packageName))
	assert.Nil(t, u.unpack())
	// assert.Nil(t, u.check())
	assert.Nil(t, u.backup())
	assert.Nil(t, u.replace())
	u.clean()

	// check backup files
	d, err := ioutil.ReadFile("aghtest/agh-backup/AdGuardHome.yaml")
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))

	d, err = ioutil.ReadFile("aghtest/agh-backup/AdGuardHome.exe")
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.exe", string(d))

	// check updated files
	d, err = ioutil.ReadFile("aghtest/AdGuardHome.exe")
	assert.Nil(t, err)
	assert.Equal(t, "1", string(d))

	d, err = ioutil.ReadFile("aghtest/README.md")
	assert.Nil(t, err)
	assert.Equal(t, "2", string(d))

	d, err = ioutil.ReadFile("aghtest/LICENSE.txt")
	assert.Nil(t, err)
	assert.Equal(t, "3", string(d))

	d, err = ioutil.ReadFile("aghtest/AdGuardHome.yaml")
	assert.Nil(t, err)
	assert.Equal(t, "AdGuardHome.yaml", string(d))
}
