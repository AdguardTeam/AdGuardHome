package dhcpsvc

import (
	"bytes"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/timeutil"
)

// Lease is a DHCP lease.
//
// TODO(e.burkov):  Consider moving it to [agh], since it also may be needed in
// [websvc].
//
// TODO(e.burkov):  Add validation method.
type Lease struct {
	// IP is the IP address leased to the client.  It must not be empty.
	IP netip.Addr

	// Expiry is the expiration time of the lease or its blocking expiration
	// time.
	Expiry time.Time

	// Hostname of the client.  It may be empty if the lease is blocked.
	Hostname string

	// HWAddr is the physical hardware (MAC) address.  It must not be nil.
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

// updateExpiry updates the lease expiry time if the current time is past the
// expiry.  For static leases, this operation is a no-op.
func (l *Lease) updateExpiry(clock timeutil.Clock, ttl time.Duration) {
	if l.IsStatic {
		return
	}

	now := clock.Now()
	if now.Before(l.Expiry) {
		return
	}

	l.Expiry = now.Add(ttl)
}
