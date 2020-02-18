// +build freebsd

package util

import (
	"os"
	"syscall"

	"github.com/AdguardTeam/golibs/log"
)

// Set user-specified limit of how many fd's we can use
// https://github.com/AdguardTeam/AdGuardHome/issues/659
func SetRlimit(val uint) {
	var rlim syscall.Rlimit
	rlim.Max = int64(val)
	rlim.Cur = int64(val)
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		log.Error("Setrlimit() failed: %v", err)
	}
}

// Check if the current user has root (administrator) rights
func HaveAdminRights() (bool, error) {
	return os.Getuid() == 0, nil
}

// SendProcessSignal - send signal to a process
func SendProcessSignal(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}
