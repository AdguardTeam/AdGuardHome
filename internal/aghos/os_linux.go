//go:build linux
// +build linux

package aghos

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func setRlimit(val uint64) (err error) {
	var rlim syscall.Rlimit
	rlim.Max = val
	rlim.Cur = val

	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
}

func haveAdminRights() (bool, error) {
	// The error is nil because the platform-independent function signature
	// requires returning an error.
	return os.Getuid() == 0, nil
}

func sendProcessSignal(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}

func isOpenWrt() (ok bool) {
	const etcDir = "/etc"

	dirEnts, err := os.ReadDir(etcDir)
	if err != nil {
		return false
	}

	// fNameSubstr is a part of a name of the desired file.
	const fNameSubstr = "release"
	osNameData := []byte("OpenWrt")

	for _, dirEnt := range dirEnts {
		if dirEnt.IsDir() {
			continue
		}

		fn := dirEnt.Name()
		if !strings.Contains(fn, fNameSubstr) {
			continue
		}

		var body []byte
		body, err = os.ReadFile(filepath.Join(etcDir, fn))
		if err != nil {
			continue
		}

		if bytes.Contains(body, osNameData) {
			return true
		}
	}

	return false
}
