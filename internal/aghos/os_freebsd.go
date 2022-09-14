//go:build freebsd

package aghos

import (
	"os"
	"syscall"
)

func setRlimit(val uint64) (err error) {
	var rlim syscall.Rlimit
	rlim.Max = int64(val)
	rlim.Cur = int64(val)

	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
}

func haveAdminRights() (bool, error) {
	return os.Getuid() == 0, nil
}

func isOpenWrt() (ok bool) {
	return false
}
