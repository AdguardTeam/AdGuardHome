// +build linux

package aghos

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/sys/unix"
)

func canBindPrivilegedPorts() (can bool, err error) {
	cnbs, err := unix.PrctlRetInt(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_IS_SET, unix.CAP_NET_BIND_SERVICE, 0, 0)
	// Don't check the error because it's always nil on Linux.
	adm, _ := haveAdminRights()

	return cnbs == 1 || adm, err
}

func setRlimit(val uint) {
	var rlim syscall.Rlimit
	rlim.Max = uint64(val)
	rlim.Cur = uint64(val)
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		log.Error("Setrlimit() failed: %v", err)
	}
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

	// TODO(e.burkov): Take care of dealing with fs package after updating
	// Go version to 1.16.
	fileInfos, err := ioutil.ReadDir(etcDir)
	if err != nil {
		return false
	}

	// fNameSubstr is a part of a name of the desired file.
	const fNameSubstr = "release"
	osNameData := []byte("OpenWrt")

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}

		fn := fileInfo.Name()
		if !strings.Contains(fn, fNameSubstr) {
			continue
		}

		var body []byte
		body, err = ioutil.ReadFile(filepath.Join(etcDir, fn))
		if err != nil {
			continue
		}

		if bytes.Contains(body, osNameData) {
			return true
		}
	}

	return false
}
