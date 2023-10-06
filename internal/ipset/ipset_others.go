//go:build !linux

package ipset

import (
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func newManager(_ []string) (mgr Manager, err error) {
	return nil, aghos.Unsupported("ipset")
}
