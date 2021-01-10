// +build freebsd

package sysutil

import (
	"os"
	"syscall"

	"github.com/AdguardTeam/golibs/log"
)

func canBindPrivilegedPorts() (can bool, err error) {
	return HaveAdminRights()
}

func setRlimit(val uint) {
	var rlim syscall.Rlimit
	rlim.Max = int64(val)
	rlim.Cur = int64(val)
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		log.Error("Setrlimit() failed: %v", err)
	}
}

func haveAdminRights() (bool, error) {
	return os.Getuid() == 0, nil
}

func sendProcessSignal(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}
