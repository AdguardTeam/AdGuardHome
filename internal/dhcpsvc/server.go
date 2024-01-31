package dhcpsvc

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// DHCPServer is a DHCP server for both IPv4 and IPv6 address families.
type DHCPServer struct {
	// enabled indicates whether the DHCP server is enabled and can provide
	// information about its clients.
	enabled *atomic.Bool

	// localTLD is the top-level domain name to use for resolving DHCP clients'
	// hostnames.
	localTLD string

	// leasesMu protects the ipIndex and nameIndex fields against concurrent
	// access, as well as leaseHandlers within the interfaces.
	leasesMu *sync.RWMutex

	// leaseByIP is a lookup shortcut for leases by their IP addresses.
	leaseByIP map[netip.Addr]*Lease

	// leaseByName is a lookup shortcut for leases by their hostnames.
	//
	// TODO(e.burkov):  Use a slice of leases with the same hostname?
	leaseByName map[string]*Lease

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
		leaseByIP:   map[netip.Addr]*Lease{},
		leaseByName: map[string]*Lease{},
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

	return leases
}

// HostByIP implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) HostByIP(ip netip.Addr) (host string) {
	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	if l, ok := srv.leaseByIP[ip]; ok {
		return l.Hostname
	}

	return ""
}

// MACByIP implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) MACByIP(ip netip.Addr) (mac net.HardwareAddr) {
	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	if l, ok := srv.leaseByIP[ip]; ok {
		return l.HWAddr
	}

	return nil
}

// IPByHost implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) IPByHost(host string) (ip netip.Addr) {
	lowered := strings.ToLower(host)

	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	if l, ok := srv.leaseByName[lowered]; ok {
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

	maps.Clear(srv.leaseByIP)
	maps.Clear(srv.leaseByName)

	return nil
}

// AddLease implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) AddLease(l *Lease) (err error) {
	var ok bool
	var iface *netInterface

	addr := l.IP

	if addr.Is4() {
		iface, ok = srv.interfaces4.find(addr)
	} else {
		iface, ok = srv.interfaces6.find(addr)
	}
	if !ok {
		return fmt.Errorf("no interface for IP address %s", addr)
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	err = iface.insertLease(l)
	if err != nil {
		return err
	}

	srv.leaseByIP[l.IP] = l
	srv.leaseByName[strings.ToLower(l.Hostname)] = l

	return nil
}
