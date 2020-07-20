// +build ignore

package home

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoUpdate(t *testing.T) {
	config.DNS.Port = 0
	Context.workDir = "..." // set absolute path
	newver := "v0.96"

	data := `{
		"version": "v0.96",
		"announcement": "AdGuard Home v0.96 is now available!",
		"announcement_url": "",
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
	uu, err := getUpdateInfo([]byte(data))
	if err != nil {
		t.Fatalf("getUpdateInfo: %s", err)
	}

	u := updateInfo{
		pkgURL:           "https://static.adguard.com/adguardhome/beta/AdGuardHome_linux_armv6.tar.gz",
		pkgName:          Context.workDir + "/agh-update-" + newver + "/AdGuardHome_linux_amd64.tar.gz",
		newVer:           newver,
		updateDir:        Context.workDir + "/agh-update-" + newver,
		backupDir:        Context.workDir + "/agh-backup",
		configName:       Context.workDir + "/AdGuardHome.yaml",
		updateConfigName: Context.workDir + "/agh-update-" + newver + "/AdGuardHome/AdGuardHome.yaml",
		curBinName:       Context.workDir + "/AdGuardHome",
		bkpBinName:       Context.workDir + "/agh-backup/AdGuardHome",
		newBinName:       Context.workDir + "/agh-update-" + newver + "/AdGuardHome/AdGuardHome",
	}

	assert.Equal(t, uu.pkgURL, u.pkgURL)
	assert.Equal(t, uu.pkgName, u.pkgName)
	assert.Equal(t, uu.newVer, u.newVer)
	assert.Equal(t, uu.updateDir, u.updateDir)
	assert.Equal(t, uu.backupDir, u.backupDir)
	assert.Equal(t, uu.configName, u.configName)
	assert.Equal(t, uu.updateConfigName, u.updateConfigName)
	assert.Equal(t, uu.curBinName, u.curBinName)
	assert.Equal(t, uu.bkpBinName, u.bkpBinName)
	assert.Equal(t, uu.newBinName, u.newBinName)

	e := doUpdate(&u)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	os.RemoveAll(u.backupDir)
}

func TestTargzFileUnpack(t *testing.T) {
	fn := "../dist/AdGuardHome_linux_amd64.tar.gz"
	outdir := "../test-unpack"
	defer os.RemoveAll(outdir)
	_ = os.Mkdir(outdir, 0755)
	files, e := targzFileUnpack(fn, outdir)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	t.Logf("%v", files)
}

func TestZipFileUnpack(t *testing.T) {
	fn := "../dist/AdGuardHome_windows_amd64.zip"
	outdir := "../test-unpack"
	_ = os.Mkdir(outdir, 0755)
	files, e := zipFileUnpack(fn, outdir)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	t.Logf("%v", files)
	os.RemoveAll(outdir)
}
