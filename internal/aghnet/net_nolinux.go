//go:build !linux
// +build !linux

package aghnet

import "github.com/AdguardTeam/AdGuardHome/internal/aghos"

func canBindPrivilegedPorts() (can bool, err error) {
	return aghos.HaveAdminRights()
}
