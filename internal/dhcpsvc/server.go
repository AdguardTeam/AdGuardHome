package dhcpsvc

import (
	"fmt"
	"net"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/exp/maps"
)

// DHCPServer is a DHCP server for both IPv4 and IPv6 address families.
type DHCPServer struct {
	// enabled indicates whether the DHCP server is enabled and can provide
	// information about its clients.
	enabled *atomic.Bool

	// localTLD is the top-level domain name to use for resolving DHCP clients'
	// hostnames.
	localTLD string

	// leasesMu protects the leases index as well as leases in the interfaces.
	leasesMu *sync.RWMutex

	// leases stores the DHCP leases for quick lookups.
	leases *leaseIndex

	// interfaces4 is the set of IPv4 interfaces sorted by interface name.
	interfaces4 netInterfacesV4

	// interfaces6 is the set of IPv6 interfaces sorted by interface name.
	interfaces6 netInterfacesV6

	// icmpTimeout is the timeout for checking another DHCP server's presence.
	icmpTimeout time.Duration
}

// New creates a new DHCP server with the given configuration.  It returns an
// error if the given configuration can't be used.
//
// TODO(e.burkov):  Use.
func New(conf *Config) (srv *DHCPServer, err error) {
	if !conf.Enabled {
		// TODO(e.burkov):  Perhaps return [Empty]?
		return nil, nil
	}

	// TODO(e.burkov):  Add validations scoped to the network interfaces set.
	ifaces4 := make(netInterfacesV4, 0, len(conf.Interfaces))
	ifaces6 := make(netInterfacesV6, 0, len(conf.Interfaces))

	ifaceNames := maps.Keys(conf.Interfaces)
	slices.Sort(ifaceNames)

	var i4 *netInterfaceV4
	var i6 *netInterfaceV6

	for _, ifaceName := range ifaceNames {
		iface := conf.Interfaces[ifaceName]

		i4, err = newNetInterfaceV4(ifaceName, iface.IPv4)
		if err != nil {
			return nil, fmt.Errorf("interface %q: ipv4: %w", ifaceName, err)
		} else if i4 != nil {
			ifaces4 = append(ifaces4, i4)
		}

		i6 = newNetInterfaceV6(ifaceName, iface.IPv6)
		if i6 != nil {
			ifaces6 = append(ifaces6, i6)
		}
	}

	enabled := &atomic.Bool{}
	enabled.Store(conf.Enabled)

	srv = &DHCPServer{
		enabled:     enabled,
		localTLD:    conf.LocalDomainName,
		leasesMu:    &sync.RWMutex{},
		leases:      newLeaseIndex(),
		interfaces4: ifaces4,
		interfaces6: ifaces6,
		icmpTimeout: conf.ICMPTimeout,
	}

	// TODO(e.burkov):  Load leases.

	return srv, nil
}

// type check
//
// TODO(e.burkov):  Uncomment when the [Interface] interface is implemented.
// var _ Interface = (*DHCPServer)(nil)

// Enabled implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) Enabled() (ok bool) {
	return srv.enabled.Load()
}

// Leases implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) Leases() (leases []*Lease) {
	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	for _, iface := range srv.interfaces4 {
		for _, lease := range iface.leases {
			leases = append(leases, lease.Clone())
		}
	}
	for _, iface := range srv.interfaces6 {
		for _, lease := range iface.leases {
			leases = append(leases, lease.Clone())
		}
	}

	return leases
}

// HostByIP implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) HostByIP(ip netip.Addr) (host string) {
	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	if l, ok := srv.leases.leaseByAddr(ip); ok {
		return l.Hostname
	}

	return ""
}

// MACByIP implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) MACByIP(ip netip.Addr) (mac net.HardwareAddr) {
	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	if l, ok := srv.leases.leaseByAddr(ip); ok {
		return l.HWAddr
	}

	return nil
}

// IPByHost implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) IPByHost(host string) (ip netip.Addr) {
	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	if l, ok := srv.leases.leaseByName(host); ok {
		return l.IP
	}

	return netip.Addr{}
}

// Reset implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) Reset() (err error) {
	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	for _, iface := range srv.interfaces4 {
		iface.reset()
	}
	for _, iface := range srv.interfaces6 {
		iface.reset()
	}
	srv.leases.clear()

	return nil
}

// AddLease implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) AddLease(l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "adding lease: %w") }()

	addr := l.IP
	iface, err := srv.ifaceForAddr(addr)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	return srv.leases.add(l, iface)
}

// UpdateStaticLease implements the [Interface] interface for *DHCPServer.
//
// TODO(e.burkov):  Support moving leases between interfaces.
func (srv *DHCPServer) UpdateStaticLease(l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "updating static lease: %w") }()

	addr := l.IP
	iface, err := srv.ifaceForAddr(addr)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	return srv.leases.update(l, iface)
}

// RemoveLease implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) RemoveLease(l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "removing lease: %w") }()

	addr := l.IP
	iface, err := srv.ifaceForAddr(addr)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	return srv.leases.remove(l, iface)
}

// ifaceForAddr returns the handled network interface for the given IP address,
// or an error if no such interface exists.
func (srv *DHCPServer) ifaceForAddr(addr netip.Addr) (iface *netInterface, err error) {
	var ok bool
	if addr.Is4() {
		iface, ok = srv.interfaces4.find(addr)
	} else {
		iface, ok = srv.interfaces6.find(addr)
	}
	if !ok {
		return nil, fmt.Errorf("no interface for ip %s", addr)
	}

	return iface, nil
}
