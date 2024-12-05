// Package dnssvc contains the AdGuard Home DNS service.
//
// TODO(a.garipov): Define, if all methods of a *Service should work with a nil
// receiver.
package dnssvc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"

	// TODO(a.garipov): Add a “dnsproxy proxy” package to shield us from changes
	// and replacement of module dnsproxy.
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
)

// Service is the AdGuard Home DNS service.  A nil *Service is a valid
// [agh.Service] that does nothing.
//
// TODO(a.garipov): Consider saving a [*proxy.Config] instance for those
// fields that are only used in [New] and [Service.Config].
type Service struct {
	logger              *slog.Logger
	proxy               *proxy.Proxy
	bootstraps          []string
	bootstrapResolvers  []*upstream.UpstreamResolver
	upstreams           []string
	dns64Prefixes       []netip.Prefix
	upsTimeout          time.Duration
	running             atomic.Bool
	bootstrapPreferIPv6 bool
	useDNS64            bool
}

// New returns a new properly initialized *Service.  If c is nil, svc is a nil
// *Service that does nothing.  The fields of c must not be modified after
// calling New.
func New(c *Config) (svc *Service, err error) {
	if c == nil {
		return nil, nil
	}

	svc = &Service{
		logger:              c.Logger,
		bootstraps:          c.BootstrapServers,
		upstreams:           c.UpstreamServers,
		dns64Prefixes:       c.DNS64Prefixes,
		upsTimeout:          c.UpstreamTimeout,
		bootstrapPreferIPv6: c.BootstrapPreferIPv6,
		useDNS64:            c.UseDNS64,
	}

	upstreams, resolvers, err := addressesToUpstreams(
		c.UpstreamServers,
		c.BootstrapServers,
		c.UpstreamTimeout,
		c.BootstrapPreferIPv6,
	)
	if err != nil {
		return nil, fmt.Errorf("converting upstreams: %w", err)
	}

	svc.bootstrapResolvers = resolvers
	svc.proxy, err = proxy.New(&proxy.Config{
		Logger:        svc.logger,
		UDPListenAddr: udpAddrs(c.Addresses),
		TCPListenAddr: tcpAddrs(c.Addresses),
		UpstreamConfig: &proxy.UpstreamConfig{
			Upstreams: upstreams,
		},
		UseDNS64:   c.UseDNS64,
		DNS64Prefs: c.DNS64Prefixes,
	})
	if err != nil {
		return nil, fmt.Errorf("proxy: %w", err)
	}

	return svc, nil
}

// addressesToUpstreams is a wrapper around [upstream.AddressToUpstream].  It
// accepts a slice of addresses and other upstream parameters, and returns a
// slice of upstreams.
func addressesToUpstreams(
	upsStrs []string,
	bootstraps []string,
	timeout time.Duration,
	preferIPv6 bool,
) (upstreams []upstream.Upstream, boots []*upstream.UpstreamResolver, err error) {
	opts := &upstream.Options{
		Timeout:    timeout,
		PreferIPv6: preferIPv6,
	}

	boots, err = aghnet.ParseBootstraps(bootstraps, opts)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return nil, nil, err
	}

	// TODO(e.burkov):  Add system hosts resolver here.
	var bootstrap upstream.ParallelResolver
	for _, r := range boots {
		bootstrap = append(bootstrap, upstream.NewCachingResolver(r))
	}

	upstreams = make([]upstream.Upstream, len(upsStrs))
	for i, upsStr := range upsStrs {
		upstreams[i], err = upstream.AddressToUpstream(upsStr, &upstream.Options{
			Bootstrap:  bootstrap,
			Timeout:    timeout,
			PreferIPv6: preferIPv6,
		})
		if err != nil {
			return nil, boots, fmt.Errorf("upstream at index %d: %w", i, err)
		}
	}

	return upstreams, boots, nil
}

// tcpAddrs converts []netip.AddrPort into []*net.TCPAddr.
func tcpAddrs(addrPorts []netip.AddrPort) (tcpAddrs []*net.TCPAddr) {
	if addrPorts == nil {
		return nil
	}

	tcpAddrs = make([]*net.TCPAddr, len(addrPorts))
	for i, a := range addrPorts {
		tcpAddrs[i] = net.TCPAddrFromAddrPort(a)
	}

	return tcpAddrs
}

// udpAddrs converts []netip.AddrPort into []*net.UDPAddr.
func udpAddrs(addrPorts []netip.AddrPort) (udpAddrs []*net.UDPAddr) {
	if addrPorts == nil {
		return nil
	}

	udpAddrs = make([]*net.UDPAddr, len(addrPorts))
	for i, a := range addrPorts {
		udpAddrs[i] = net.UDPAddrFromAddrPort(a)
	}

	return udpAddrs
}

// type check
var _ agh.ServiceWithConfig[*Config] = (*Service)(nil)

// Start implements the [agh.Service] interface for *Service.  svc may be nil.
// After Start exits, all DNS servers have tried to start, but there is no
// guarantee that they did.  Errors from the servers are written to the log.
func (svc *Service) Start(ctx context.Context) (err error) {
	if svc == nil {
		return nil
	}

	defer func() {
		// TODO(a.garipov): [proxy.Proxy.Start] doesn't actually have any way to
		// tell when all servers are actually up, so at best this is merely an
		// assumption.
		svc.running.Store(err == nil)
	}()

	return svc.proxy.Start(ctx)
}

// Shutdown implements the [agh.Service] interface for *Service.  svc may be
// nil.
func (svc *Service) Shutdown(ctx context.Context) (err error) {
	if svc == nil {
		return nil
	}

	errs := []error{
		svc.proxy.Shutdown(ctx),
	}

	for _, b := range svc.bootstrapResolvers {
		errs = append(errs, errors.Annotate(b.Close(), "closing bootstrap %s: %w", b.Address()))
	}

	return errors.Join(errs...)
}

// Config returns the current configuration of the web service.  Config must not
// be called simultaneously with Start.  If svc was initialized with ":0"
// addresses, addrs will not return the actual bound ports until Start is
// finished.
func (svc *Service) Config() (c *Config) {
	// TODO(a.garipov): Do we need to get the TCP addresses separately?

	var addrs []netip.AddrPort
	if svc.running.Load() {
		udpAddrs := svc.proxy.Addrs(proxy.ProtoUDP)
		addrs = make([]netip.AddrPort, len(udpAddrs))
		for i, a := range udpAddrs {
			addrs[i] = a.(*net.UDPAddr).AddrPort()
		}
	} else {
		conf := svc.proxy.Config
		udpAddrs := conf.UDPListenAddr
		addrs = make([]netip.AddrPort, len(udpAddrs))
		for i, a := range udpAddrs {
			addrs[i] = a.AddrPort()
		}
	}

	c = &Config{
		Logger:              svc.logger,
		Addresses:           addrs,
		BootstrapServers:    svc.bootstraps,
		UpstreamServers:     svc.upstreams,
		DNS64Prefixes:       svc.dns64Prefixes,
		UpstreamTimeout:     svc.upsTimeout,
		BootstrapPreferIPv6: svc.bootstrapPreferIPv6,
		UseDNS64:            svc.useDNS64,
	}

	return c
}
