//go:build windows
// +build windows

package dhcpd

import "github.com/AdguardTeam/AdGuardHome/internal/aghos"

func CheckIfOtherDHCPServersPresentV4(ifaceName string) (bool, error) {
	return false, aghos.Unsupported("CheckIfOtherDHCPServersPresentV4")
}

func CheckIfOtherDHCPServersPresentV6(ifaceName string) (bool, error) {
	return false, aghos.Unsupported("CheckIfOtherDHCPServersPresentV6")
}
