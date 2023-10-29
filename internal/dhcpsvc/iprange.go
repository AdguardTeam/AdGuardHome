package dhcpsvc

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
)

// ipRange is an inclusive range of IP addresses.  A zero range doesn't contain
// any IP addresses.
//
// It is safe for concurrent use.
type ipRange struct {
	start netip.Addr
	end   netip.Addr
}

// maxRangeLen is the maximum IP range length.  The bitsets used in servers only
// accept uints, which can have the size of 32 bit.
//
// TODO(a.garipov, e.burkov):  Reconsider the value for IPv6.
const maxRangeLen = math.MaxUint32

// newIPRange creates a new IP address range.  start must be less than end.  The
// resulting range must not be greater than maxRangeLen.
func newIPRange(start, end netip.Addr) (r ipRange, err error) {
	defer func() { err = errors.Annotate(err, "invalid ip range: %w") }()

	switch false {
	case start.Is4() == end.Is4():
		return ipRange{}, fmt.Errorf("%s and %s must be within the same address family", start, end)
	case start.Less(end):
		return ipRange{}, fmt.Errorf("start %s is greater than or equal to end %s", start, end)
	default:
		diff := (&big.Int{}).Sub(
			(&big.Int{}).SetBytes(end.AsSlice()),
			(&big.Int{}).SetBytes(start.AsSlice()),
		)

		if !diff.IsUint64() || diff.Uint64() > maxRangeLen {
			return ipRange{}, fmt.Errorf("range length must be within %d", uint32(maxRangeLen))
		}
	}

	return ipRange{
		start: start,
		end:   end,
	}, nil
}

// contains returns true if r contains ip.
func (r ipRange) contains(ip netip.Addr) (ok bool) {
	// Assume that the end was checked to be within the same address family as
	// the start during construction.
	return r.start.Is4() == ip.Is4() && !ip.Less(r.start) && !r.end.Less(ip)
}

// ipPredicate is a function that is called on every IP address in
// [ipRange.find].
type ipPredicate func(ip netip.Addr) (ok bool)

// find finds the first IP address in r for which p returns true.  It returns an
// empty [netip.Addr] if there are no addresses that satisfy p.
//
// TODO(e.burkov):  Use.
func (r ipRange) find(p ipPredicate) (ip netip.Addr) {
	for ip = r.start; !r.end.Less(ip); ip = ip.Next() {
		if p(ip) {
			return ip
		}
	}

	return netip.Addr{}
}

// offset returns the offset of ip from the beginning of r.  It returns 0 and
// false if ip is not in r.
func (r ipRange) offset(ip netip.Addr) (offset uint64, ok bool) {
	if !r.contains(ip) {
		return 0, false
	}

	startData, ipData := r.start.As16(), ip.As16()
	be := binary.BigEndian

	// Assume that the range length was checked against maxRangeLen during
	// construction.
	return be.Uint64(ipData[8:]) - be.Uint64(startData[8:]), true
}

// String implements the fmt.Stringer interface for *ipRange.
func (r ipRange) String() (s string) {
	return fmt.Sprintf("%s-%s", r.start, r.end)
}
