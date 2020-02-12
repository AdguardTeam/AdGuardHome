// +build ignore

package home

import (
	"os"
	"testing"
)

func TestDoUpdate(t *testing.T) {

	config.DNS.Port = 0
	Context.workDir = "..." // set absolute path
	newver := "v0.96"

	data := `{
		"version": "v0.96",
		"announcement": "AdGuard Home v0.96 is now available!",
		"announcement_url": "",
		"download_windows_amd64": "",
		"download_windows_386": "",
		"download_darwin_amd64": "",
		"download_linux_amd64": "https://github.com/AdguardTeam/AdGuardHome/releases/download/v0.96/AdGuardHome_linux_amd64.tar.gz",
		"download_linux_386": "",
		"download_linux_arm": "",
		"download_linux_arm64": "",
		"download_linux_mips": "",
		"download_linux_mipsle": "",
		"selfupdate_min_version": "v0.0"
	}`
	uu, err := getUpdateInfo([]byte(data))
	if err != nil {
		t.Fatalf("getUpdateInfo: %s", err)
	}

	u := updateInfo{
		pkgURL:           "https://github.com/AdguardTeam/AdGuardHome/releases/download/" + newver + "/AdGuardHome_linux_amd64.tar.gz",
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

	if uu.pkgURL != u.pkgURL ||
		uu.pkgName != u.pkgName ||
		uu.newVer != u.newVer ||
		uu.updateDir != u.updateDir ||
		uu.backupDir != u.backupDir ||
		uu.configName != u.configName ||
		uu.updateConfigName != u.updateConfigName ||
		uu.curBinName != u.curBinName ||
		uu.bkpBinName != u.bkpBinName ||
		uu.newBinName != u.newBinName {
		t.Fatalf("getUpdateInfo: %v != %v", uu, u)
	}

	e := doUpdate(&u)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	os.RemoveAll(u.backupDir)
}

func TestTargzFileUnpack(t *testing.T) {
	fn := "./dist/AdGuardHome_v0.95_linux_amd64.tar.gz"
	outdir := "./test-unpack"
	_ = os.Mkdir(outdir, 0755)
	files, e := targzFileUnpack(fn, outdir)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	t.Logf("%v", files)
	os.RemoveAll(outdir)
}

func TestZipFileUnpack(t *testing.T) {
	fn := "./dist/AdGuardHome_v0.95_Windows_amd64.zip"
	outdir := "./test-unpack"
	_ = os.Mkdir(outdir, 0755)
	files, e := zipFileUnpack(fn, outdir)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	t.Logf("%v", files)
	os.RemoveAll(outdir)
}
