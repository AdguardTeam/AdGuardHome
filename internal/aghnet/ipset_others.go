//go:build !linux

package aghnet

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func newIpsetMgr(_ []string) (mgr IpsetManager, err error) {
	return nil, aghos.Unsupported("ipset")
}
