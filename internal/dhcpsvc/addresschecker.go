package dhcpsvc

import (
	"context"
	"net/netip"
)

// AddressChecker checks addresses for availability.
type AddressChecker interface {
	// IsAvailable returns true if the address is available in the current
	// subnet.  Any error is a network error.
	IsAvailable(ctx context.Context, ip netip.Addr) (ok bool, err error)
}

// EmptyAddressChecker is an [AddressChecker] implementation that doesn't
// perform any checks.
type EmptyAddressChecker struct{}

// type check
var _ AddressChecker = EmptyAddressChecker{}

// IsAvailable implements the [AddressChecker] interface for
// EmptyAddressChecker.
func (EmptyAddressChecker) IsAvailable(_ context.Context, _ netip.Addr) (ok bool, err error) {
	return true, nil
}
