package main

import (
	"os"
	"testing"
)

func testDoUpdate(t *testing.T) {
	config.DNS.Port = 0
	u := updateInfo{
		pkgURL:           "https://github.com/AdguardTeam/AdGuardHome/releases/download/v0.95/AdGuardHome_v0.95_linux_amd64.tar.gz",
		pkgName:          "./AdGuardHome_v0.95_linux_amd64.tar.gz",
		newVer:           "v0.95",
		updateDir:        "./update-v0.95",
		backupDir:        "./backup-v0.94",
		configName:       "./AdGuardHome.yaml",
		updateConfigName: "./update-v0.95/AdGuardHome/AdGuardHome.yaml",
		curBinName:       "./AdGuardHome",
		bkpBinName:       "./backup-v0.94/AdGuardHome",
		newBinName:       "./update-v0.95/AdGuardHome/AdGuardHome",
	}
	e := doUpdate(&u)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	os.RemoveAll(u.backupDir)
	os.RemoveAll(u.updateDir)
}

func testTargzFileUnpack(t *testing.T) {
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

func testZipFileUnpack(t *testing.T) {
	fn := "./dist/AdGuardHome_v0.95_Windows_amd64.zip"
	outdir := "./test-unpack"
	_ = os.Mkdir(outdir, 0755)
	e := zipFileUnpack(fn, outdir)
	if e != nil {
		t.Fatalf("FAILED: %s", e)
	}
	os.RemoveAll(outdir)
}
