//go:build !linux

package ipset

import (
	"context"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func newManager(_ context.Context, _ *Config) (mgr Manager, err error) {
	return nil, aghos.Unsupported("ipset")
}
