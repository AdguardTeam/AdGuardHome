package aghnet

import (
	"fmt"
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
)

// ValidateHardwareAddress returns an error if hwa is not a valid EUI-48,
// EUI-64, or 20-octet InfiniBand link-layer address.
func ValidateHardwareAddress(hwa net.HardwareAddr) (err error) {
	defer agherr.Annotate("validating hardware address %q: %w", &err, hwa)

	switch l := len(hwa); l {
	case 0:
		return agherr.Error("address is empty")
	case 6, 8, 20:
		return nil
	default:
		return fmt.Errorf("bad len: %d", l)
	}
}
