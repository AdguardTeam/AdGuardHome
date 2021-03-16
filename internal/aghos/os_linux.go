// +build linux

package aghos

import (
	"os"
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
