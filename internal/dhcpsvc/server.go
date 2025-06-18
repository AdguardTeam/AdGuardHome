package dhcpsvc

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/google/gopacket"
)

// DHCPServer is a DHCP server for both IPv4 and IPv6 address families.
//
// TODO(e.burkov):  Rename to Default.
type DHCPServer struct {
	// enabled indicates whether the DHCP server is enabled and can provide
	// information about its clients.
	enabled *atomic.Bool

	// logger logs common DHCP events.
	logger *slog.Logger

	// packetSource is the source of DHCP packets to process.
	//
	// TODO(e.burkov):  Implement and set.
	packetSource gopacket.PacketSource

	// localTLD is the top-level domain name to use for resolving DHCP clients'
	// hostnames.
	localTLD string

	// dbFilePath is the path to the database file containing the DHCP leases.
	//
	// TODO(e.burkov):  Consider extracting the database logic into a separate
	// interface to prevent packages that only need lease data from depending on
	// the entire server and to simplify testing.
	dbFilePath string

	// leasesMu protects the leases index as well as leases in the interfaces.
	leasesMu *sync.RWMutex

	// leases stores the DHCP leases for quick lookups.
	leases *leaseIndex

	// interfaces4 is the set of IPv4 interfaces sorted by interface name.
	interfaces4 dhcpInterfacesV4

	// interfaces6 is the set of IPv6 interfaces sorted by interface name.
	interfaces6 dhcpInterfacesV6

	// icmpTimeout is the timeout for checking another DHCP server's presence.
	icmpTimeout time.Duration
}

// New creates a new DHCP server with the given configuration.  conf must be
// valid.
//
// TODO(e.burkov):  Use.
func New(ctx context.Context, conf *Config) (srv *DHCPServer, err error) {
	l := conf.Logger
	if !conf.Enabled {
		l.DebugContext(ctx, "disabled")

		// TODO(e.burkov):  Perhaps return [Empty]?
		return nil, nil
	}

	ifaces4, ifaces6 := newInterfaces(ctx, l, conf.Interfaces)

	enabled := &atomic.Bool{}
	enabled.Store(conf.Enabled)

	srv = &DHCPServer{
		enabled:     enabled,
		logger:      l,
		localTLD:    conf.LocalDomainName,
		leasesMu:    &sync.RWMutex{},
		leases:      newLeaseIndex(),
		interfaces4: ifaces4,
		interfaces6: ifaces6,
		icmpTimeout: conf.ICMPTimeout,
		dbFilePath:  conf.DBFilePath,
	}

	err = srv.dbLoad(ctx)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	return srv, nil
}

// newInterfaces creates interfaces for the given map of interface names to
// their configurations.  ifaces must be valid, baseLogger must not be nil.
func newInterfaces(
	ctx context.Context,
	baseLogger *slog.Logger,
	ifaces map[string]*InterfaceConfig,
) (v4 dhcpInterfacesV4, v6 dhcpInterfacesV6) {
	// TODO(e.burkov):  Add validations scoped to the network interfaces set.
	v4 = make(dhcpInterfacesV4, 0, len(ifaces))
	v6 = make(dhcpInterfacesV6, 0, len(ifaces))

	for _, name := range slices.Sorted(maps.Keys(ifaces)) {
		iface := ifaces[name]
		ifaceLogger := baseLogger.With(keyInterface, name)

		iface4 := newDHCPInterfaceV4(
			ctx,
			ifaceLogger.With(keyFamily, netutil.AddrFamilyIPv4),
			name,
			iface.IPv4,
		)
		if iface4 != nil {
			v4 = append(v4, iface4)
		}

		iface6 := newDHCPInterfaceV6(
			ctx,
			ifaceLogger.With(keyFamily, netutil.AddrFamilyIPv6),
			name,
			iface.IPv6,
		)
		if iface6 != nil {
			v6 = append(v6, iface6)
		}
	}

	return v4, v6
}

// type check
//
// TODO(e.burkov):  Uncomment when the [Interface] interface is implemented.
// var _ Interface = (*DHCPServer)(nil)

// Start implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) Start(ctx context.Context) (err error) {
	srv.logger.DebugContext(ctx, "starting dhcp server")

	// TODO(e.burkov):  Listen to configured interfaces.

	go srv.serve(context.WithoutCancel(ctx))

	return nil
}

func (srv *DHCPServer) Shutdown(ctx context.Context) (err error) {
	srv.logger.DebugContext(ctx, "shutting down dhcp server")

	// TODO(e.burkov):  Close the packet source.

	return nil
}

// Enabled implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) Enabled() (ok bool) {
	return srv.enabled.Load()
}

// Leases implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) Leases() (leases []*Lease) {
	srv.leasesMu.RLock()
	defer srv.leasesMu.RUnlock()

	for l := range srv.leases.rangeLeases {
		leases = append(leases, l)
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
func (srv *DHCPServer) Reset(ctx context.Context) (err error) {
	defer func() { err = errors.Annotate(err, "resetting leases: %w") }()

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	srv.resetLeases()
	err = srv.dbStore(ctx)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	srv.logger.DebugContext(ctx, "reset leases")

	return nil
}

// resetLeases resets the leases for all network interfaces of the server.  It
// expects the DHCPServer.leasesMu to be locked.
func (srv *DHCPServer) resetLeases() {
	for _, iface := range srv.interfaces4 {
		iface.common.reset()
	}
	for _, iface := range srv.interfaces6 {
		iface.common.reset()
	}
	srv.leases.clear()
}

// AddLease implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) AddLease(ctx context.Context, l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "adding lease: %w") }()

	addr := l.IP
	iface, err := srv.ifaceForAddr(addr)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	err = srv.leases.add(l, iface)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	err = srv.dbStore(ctx)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	iface.logger.DebugContext(
		ctx, "added lease",
		"hostname", l.Hostname,
		"ip", l.IP,
		"mac", l.HWAddr,
		"is_static", l.IsStatic,
	)

	return nil
}

// UpdateStaticLease implements the [Interface] interface for *DHCPServer.
//
// TODO(e.burkov):  Support moving leases between interfaces.
func (srv *DHCPServer) UpdateStaticLease(ctx context.Context, l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "updating static lease: %w") }()

	addr := l.IP
	iface, err := srv.ifaceForAddr(addr)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	err = srv.leases.update(l, iface)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	err = srv.dbStore(ctx)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	iface.logger.DebugContext(
		ctx, "updated lease",
		"hostname", l.Hostname,
		"ip", l.IP,
		"mac", l.HWAddr,
		"is_static", l.IsStatic,
	)

	return nil
}

// RemoveLease implements the [Interface] interface for *DHCPServer.
func (srv *DHCPServer) RemoveLease(ctx context.Context, l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "removing lease: %w") }()

	addr := l.IP
	iface, err := srv.ifaceForAddr(addr)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	err = srv.leases.remove(l, iface)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	err = srv.dbStore(ctx)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	iface.logger.DebugContext(
		ctx, "removed lease",
		"hostname", l.Hostname,
		"ip", l.IP,
		"mac", l.HWAddr,
		"is_static", l.IsStatic,
	)

	return nil
}

// removeLeaseByAddr removes the lease with the given IP address from the
// server.  It returns an error if the lease can't be removed.
//
//lint:ignore U1000 TODO(e.burkov):  Use
func (srv *DHCPServer) removeLeaseByAddr(ctx context.Context, addr netip.Addr) (err error) {
	defer func() { err = errors.Annotate(err, "removing lease by address: %w") }()

	iface, err := srv.ifaceForAddr(addr)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	srv.leasesMu.Lock()
	defer srv.leasesMu.Unlock()

	l, ok := srv.leases.leaseByAddr(addr)
	if !ok {
		return fmt.Errorf("no lease for ip %s", addr)
	}

	err = srv.leases.remove(l, iface)
	if err != nil {
		// Don't wrap the error since there is already an annotation deferred.
		return err
	}

	err = srv.dbStore(ctx)
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return err
	}

	iface.logger.DebugContext(
		ctx, "removed lease",
		"hostname", l.Hostname,
		"ip", l.IP,
		"mac", l.HWAddr,
		"is_static", l.IsStatic,
	)

	return nil
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
