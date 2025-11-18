package dhcpsvc

import "net/netip"

// addressChecker checks addresses for availability.
type addressChecker interface {
	// IsAvailable returns true if the address is available in the current
	// subnet.  Any error is a network error.
	IsAvailable(ip netip.Addr) (ok bool, err error)
}

// noopAddressChecker is an implementation of [addressChecker] that doesn't
// perform any checks.
type noopAddressChecker struct{}

// IsAvailable implements the [addressChecker] interface for noopAddressChecker.
func (c noopAddressChecker) IsAvailable(ip netip.Addr) (ok bool, err error) {
	return true, nil
}

// TODO(e.burkov):  Add ICMP implementation of [addressChecker], as required by
// https://datatracker.ietf.org/doc/html/rfc2131#section-2.2.
