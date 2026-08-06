package dhcpsvc

import (
	"bytes"
	"net"
	"net/netip"
	"slices"
	"time"
)

// Lease is a DHCP lease.
//
// TODO(e.burkov):  Add validation method.
//
// BUG(e.burkov):  The implementation currently relies on the client's hardware
// address for client identification.  This approach is not recommended by RFC
// 9915, so the database should be migrated to use the client's DUID and IAID
// for lease identification.
type Lease struct {
	// IP is the IP address leased to the client.  It must not be empty.
	IP netip.Addr

	// Expiry is the expiration time of the lease or its blocking expiration
	// time.
	Expiry time.Time

	// Hostname of the client.  It may be empty if the lease is blocked.
	Hostname string

	// HWAddr is the physical hardware (MAC) address.  It must be a valid
	// hardware address of length 6, 8, or 20 bytes, see [netutil.ValidateMAC].
	HWAddr net.HardwareAddr

	// IsStatic defines if the lease is static.
	IsStatic bool
}

// Clone returns a deep copy of l.
func (l *Lease) Clone() (clone *Lease) {
	if l == nil {
		return nil
	}

	return &Lease{
		Expiry:   l.Expiry,
		Hostname: l.Hostname,
		HWAddr:   slices.Clone(l.HWAddr),
		IP:       l.IP,
		IsStatic: l.IsStatic,
	}
}

// EUI48AddrLen is the length of a valid EUI-48 hardware address.
const EUI48AddrLen = 6

// blockedHardwareAddr is the hardware address used to mark a lease as blocked.
var blockedHardwareAddr = make(net.HardwareAddr, EUI48AddrLen)

// IsBlocked returns true if the lease is blocked.
func (l *Lease) IsBlocked() (blocked bool) {
	return bytes.Equal(l.HWAddr, blockedHardwareAddr)
}

// isExpiredAt returns true if the lease is expired at now.  For static leases,
// it always returns false.
func (l *Lease) isExpiredAt(now time.Time) (ok bool) {
	if l.IsStatic {
		return false
	}

	return l.Expiry.Before(now)
}

// updateExpiry updates the lease expiry time if the current time is past the
// expiry.  For static leases, this operation is a no-op.
func (l *Lease) updateExpiry(now time.Time, ttl time.Duration) {
	if l.IsStatic {
		return
	}

	if now.Before(l.Expiry) {
		return
	}

	l.Expiry = now.Add(ttl)
}

// compareLeases is a helper function that sorts a slice of leases according to
// their IP.
func compareLeases(a, b *Lease) (res int) {
	return a.IP.Compare(b.IP)
}
