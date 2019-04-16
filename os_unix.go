// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"syscall"

	"github.com/AdguardTeam/golibs/log"
)

// Set user-specified limit of how many fd's we can use
// https://github.com/AdguardTeam/AdGuardHome/issues/659
func setRlimit(val uint) {
	var rlim syscall.Rlimit
	rlim.Max = uint64(val)
	rlim.Cur = uint64(val)
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		log.Error("Setrlimit() failed: %v", err)
	}
}
