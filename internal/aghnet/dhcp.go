package aghnet

import (
	"context"
	"log/slog"
)

// CheckOtherDHCP tries to discover another DHCP server in the network.  l must
// not be nil.
func CheckOtherDHCP(
	ctx context.Context,
	l *slog.Logger,
	ifaceName string,
) (ok4, ok6 bool, err4, err6 error) {
	return checkOtherDHCP(ctx, l, ifaceName)
}
