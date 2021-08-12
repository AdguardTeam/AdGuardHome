//go:build !(linux || darwin || freebsd || openbsd)
// +build !linux,!darwin,!freebsd,!openbsd

package aghnet

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func canBindPrivilegedPorts() (can bool, err error) {
	return aghos.HaveAdminRights()
}

func ifaceHasStaticIP(string) (ok bool, err error) {
	return false, aghos.Unsupported("checking static ip")
}

func ifaceSetStaticIP(string) (err error) {
	return aghos.Unsupported("setting static ip")
}
