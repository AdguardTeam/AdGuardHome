package dhcpd

import (
	"fmt"
	"math"
	"math/big"
	"net"

	"github.com/AdguardTeam/golibs/errors"
)

// ipRange is an inclusive range of IP addresses.  A nil range is a range that
// doesn't contain any IP addresses.
//
// It is safe for concurrent use.
//
// TODO(a.garipov): Perhaps create an optimized version with uint32 for IPv4
// ranges?  Or use one of uint128 packages?
//
// TODO(e.burkov):  Use netip.Addr.
type ipRange struct {
	start *big.Int
	end   *big.Int
}

// maxRangeLen is the maximum IP range length.  The bitsets used in servers only
// accept uints, which can have the size of 32 bit.
const maxRangeLen = math.MaxUint32

// newIPRange creates a new IP address range.  start must be less than end.  The
// resulting range must not be greater than maxRangeLen.
func newIPRange(start, end net.IP) (r *ipRange, err error) {
	defer func() { err = errors.Annotate(err, "invalid ip range: %w") }()

	// Make sure that both are 16 bytes long to simplify handling in
	// methods.
	start, end = start.To16(), end.To16()

	startInt := (&big.Int{}).SetBytes(start)
	endInt := (&big.Int{}).SetBytes(end)
	diff := (&big.Int{}).Sub(endInt, startInt)

	if diff.Sign() <= 0 {
		return nil, fmt.Errorf("start is greater than or equal to end")
	} else if !diff.IsUint64() || diff.Uint64() > maxRangeLen {
		return nil, fmt.Errorf("range is too large")
	}

	r = &ipRange{
		start: startInt,
		end:   endInt,
	}

	return r, nil
}

// contains returns true if r contains ip.
func (r *ipRange) contains(ip net.IP) (ok bool) {
	if r == nil {
		return false
	}

	ipInt := (&big.Int{}).SetBytes(ip.To16())

	return r.containsInt(ipInt)
}

// containsInt returns true if r contains ipInt.  For internal use only.
func (r *ipRange) containsInt(ipInt *big.Int) (ok bool) {
	return ipInt.Cmp(r.start) >= 0 && ipInt.Cmp(r.end) <= 0
}

// ipPredicate is a function that is called on every IP address in
// (*ipRange).find.  ip is given in the 16-byte form.
type ipPredicate func(ip net.IP) (ok bool)

// find finds the first IP address in r for which p returns true.  ip is in the
// 16-byte form.
func (r *ipRange) find(p ipPredicate) (ip net.IP) {
	if r == nil {
		return nil
	}

	ip = make(net.IP, net.IPv6len)
	_1 := big.NewInt(1)
	for i := (&big.Int{}).Set(r.start); i.Cmp(r.end) <= 0; i.Add(i, _1) {
		i.FillBytes(ip)
		if p(ip) {
			return ip
		}
	}

	return nil
}

// offset returns the offset of ip from the beginning of r.  It returns 0 and
// false if ip is not in r.
func (r *ipRange) offset(ip net.IP) (offset uint64, ok bool) {
	if r == nil {
		return 0, false
	}

	ip = ip.To16()
	ipInt := (&big.Int{}).SetBytes(ip)
	if !r.containsInt(ipInt) {
		return 0, false
	}

	offsetInt := (&big.Int{}).Sub(ipInt, r.start)

	// Assume that the range was checked against maxRangeLen during
	// construction.
	return offsetInt.Uint64(), true
}

// String implements the fmt.Stringer interface for *ipRange.
func (r *ipRange) String() (s string) {
	return fmt.Sprintf("%s-%s", r.start, r.end)
}
