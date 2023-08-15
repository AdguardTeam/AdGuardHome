//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"fmt"

	"github.com/AdguardTeam/golibs/errors"
)

// wrapErrs is a helper to wrap the errors from two independent underlying
// connections.
func wrapErrs(action string, udpConnErr, rawConnErr error) (err error) {
	switch {
	case udpConnErr != nil && rawConnErr != nil:
		return fmt.Errorf("%s both connections: %s", action, errors.Join(udpConnErr, rawConnErr))
	case udpConnErr != nil:
		return fmt.Errorf("%s udp connection: %w", action, udpConnErr)
	case rawConnErr != nil:
		return fmt.Errorf("%s raw connection: %w", action, rawConnErr)
	default:
		return nil
	}
}
