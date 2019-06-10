// +build ignore

package home

import (
	"os"
	"testing"
)

func TestDoUpdate(t *testing.T) {
	config.DNS.Port = 0
	config.ourWorkingDir = "."
	u := updateInfo{
		pkgURL:           "https://github.com/AdguardTeam/AdGuardHome/releases/download/v0.95/AdGuardHome_v0.95_linux_amd64.tar.gz",
		pkgName:          "./AdGuardHome_v0.95_linux_amd64.tar.gz",
		newVer:           "v0.95",
		updateDir:        "./agh-update-v0.95",
		backupDir:        "./agh-backup-v0.94",
		configName:       "./AdGuardHome.yaml",
		updateConfigName: "./agh-update-v0.95/AdGuardHome/AdGuardHome.yaml",
		curBinName:       "./AdGuardHome",
		bkpBinName:       "./agh-backup-v0.94/AdGuardHome",
		newBinName:       "./agh-update-v0.95/AdGuardHome/AdGuardHome",
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
