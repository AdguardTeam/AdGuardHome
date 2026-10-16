//go:build darwin || freebsd || netbsd || openbsd

package aghnet

import (
	"context"
	"log/slog"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func canBindPrivilegedPorts(_ context.Context, _ *slog.Logger) (can bool, err error) {
	return aghos.HaveAdminRights()
}
