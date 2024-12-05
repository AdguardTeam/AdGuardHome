// Package dhcpsvc contains the AdGuard Home DHCP service.
//
// TODO(e.burkov): Add tests.
package dhcpsvc

import (
	"context"
	"net"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
)

const (
	// keyInterface is the key for logging the network interface name.
	keyInterface = "iface"

	// keyFamily is the key for logging the handled address family.
	keyFamily = "family"
)

// Interface is a DHCP service.
//
// TODO(e.burkov):  Separate HostByIP, MACByIP, IPByHost into a separate
// interface.  This is also applicable to Enabled method.
//
// TODO(e.burkov):  Reconsider the requirements for the leases validity.
type Interface interface {
	agh.ServiceWithConfig[*Config]

	// Enabled returns true if DHCP provides information about clients.
	Enabled() (ok bool)

	// HostByIP returns the hostname of the DHCP client with the given IP
	// address.  The address will be netip.Addr{} if there is no such client,
	// due to an assumption that a DHCP client must always have an IP address.
	HostByIP(ip netip.Addr) (host string)

	// MACByIP returns the MAC address for the given IP address leased.  It
	// returns nil if there is no such client, due to an assumption that a DHCP
	// client must always have a MAC address.
	//
	// TODO(e.burkov):  Think of a contract for the returned value.
	MACByIP(ip netip.Addr) (mac net.HardwareAddr)

	// IPByHost returns the IP address of the DHCP client with the given
	// hostname.  The hostname will be an empty string if there is no such
	// client, due to an assumption that a DHCP client must always have a
	// hostname, either set or generated.
	IPByHost(host string) (ip netip.Addr)

	// Leases returns all the active DHCP leases.  The returned slice should be
	// a clone.  The order of leases is undefined.
	//
	// TODO(e.burkov):  Consider implementing iterating methods with appropriate
	// signatures instead of cloning the whole list.
	Leases() (ls []*Lease)

	// AddLease adds a new DHCP lease.  l must be valid.  It returns an error if
	// l already exists.
	AddLease(ctx context.Context, l *Lease) (err error)

	// UpdateStaticLease replaces an existing static DHCP lease.  l must be
	// valid.  It returns an error if the lease with the given hardware address
	// doesn't exist or if other values match another existing lease.
	UpdateStaticLease(ctx context.Context, l *Lease) (err error)

	// RemoveLease removes an existing DHCP lease.  l must be valid.  It returns
	// an error if there is no lease equal to l.
	RemoveLease(ctx context.Context, l *Lease) (err error)

	// Reset removes all the DHCP leases.
	//
	// TODO(e.burkov):  If it's really needed?
	Reset(ctx context.Context) (err error)
}

// Empty is an [Interface] implementation that does nothing.
type Empty struct{}

// type check
var _ agh.ServiceWithConfig[*Config] = Empty{}

// Start implements the [Service] interface for Empty.
func (Empty) Start(_ context.Context) (err error) { return nil }

// Shutdown implements the [Service] interface for Empty.
func (Empty) Shutdown(_ context.Context) (err error) { return nil }

// Config implements the [ServiceWithConfig] interface for Empty.
func (Empty) Config() (conf *Config) { return nil }

// type check
var _ Interface = Empty{}

// Enabled implements the [Interface] interface for Empty.
func (Empty) Enabled() (ok bool) { return false }

// HostByIP implements the [Interface] interface for Empty.
func (Empty) HostByIP(_ netip.Addr) (host string) { return "" }

// MACByIP implements the [Interface] interface for Empty.
func (Empty) MACByIP(_ netip.Addr) (mac net.HardwareAddr) { return nil }

// IPByHost implements the [Interface] interface for Empty.
func (Empty) IPByHost(_ string) (ip netip.Addr) { return netip.Addr{} }

// Leases implements the [Interface] interface for Empty.
func (Empty) Leases() (leases []*Lease) { return nil }

// AddLease implements the [Interface] interface for Empty.
func (Empty) AddLease(_ context.Context, _ *Lease) (err error) { return nil }

// UpdateStaticLease implements the [Interface] interface for Empty.
func (Empty) UpdateStaticLease(_ context.Context, _ *Lease) (err error) { return nil }

// RemoveLease implements the [Interface] interface for Empty.
func (Empty) RemoveLease(_ context.Context, _ *Lease) (err error) { return nil }

// Reset implements the [Interface] interface for Empty.
func (Empty) Reset(_ context.Context) (err error) { return nil }
