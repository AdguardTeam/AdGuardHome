package dhcpsvc

import (
	"net"
	"net/netip"
	"slices"
	"time"
)

// Lease is a DHCP lease.
//
// TODO(e.burkov):  Consider moving it to [agh], since it also may be needed in
// [websvc].
//
// TODO(e.burkov):  Add validation method.
type Lease struct {
	// IP is the IP address leased to the client.
	IP netip.Addr

	// Expiry is the expiration time of the lease.
	Expiry time.Time

	// Hostname of the client.
	Hostname string

	// HWAddr is the physical hardware address (MAC address).
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
