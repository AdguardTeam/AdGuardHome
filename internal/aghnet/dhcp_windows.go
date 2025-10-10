//go:build windows

package aghnet

import (
	"context"
	"log/slog"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

func checkOtherDHCP(
	_ context.Context,
	_ *slog.Logger,
	ifaceName string,
) (ok4, ok6 bool, err4, err6 error) {
	return false,
		false,
		aghos.Unsupported("CheckIfOtherDHCPServersPresentV4"),
		aghos.Unsupported("CheckIfOtherDHCPServersPresentV6")
}
