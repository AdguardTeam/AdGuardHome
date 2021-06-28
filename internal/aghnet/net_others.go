//go:build !(linux || darwin)
// +build !linux,!darwin

package aghnet

import (
	"fmt"
	"runtime"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func canBindPrivilegedPorts() (can bool, err error) {
	return aghos.HaveAdminRights()
}

func ifaceHasStaticIP(string) (bool, error) {
	return false, fmt.Errorf("cannot check if IP is static: not supported on %s", runtime.GOOS)
}

func ifaceSetStaticIP(string) error {
	return fmt.Errorf("cannot set static IP on %s", runtime.GOOS)
}
