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
	"github.com/AdguardTeam/AdGuardHome/internal/aghslog"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/ratelimit"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
)

// Service is the AdGuard Home DNS service.  A nil *Service is a valid
// [agh.Service] that does nothing.
//
// TODO(a.garipov): Consider saving a [*proxy.Config] instance for those
// fields that are only used in [New] and [Service.Config].
type Service struct {
	// logger is used for logging the operation of the DNS service.
	logger *slog.Logger

	// proxy is the current DNS proxy.
	proxy *proxy.Proxy

	// proxyConf contains the fields that have been used to create proxy to
	// return them in [Service.Config].
	proxyConf *proxy.Config

	// The fields below have been used to create proxy and are saved to return
	// them in [Service.Config].

	bootstraps          []string
	bootstrapResolvers  []*upstream.UpstreamResolver
	upstreams           []string
	upstreamTimeout     time.Duration
	bootstrapPreferIPv6 bool

	// The fields above have been used to create proxy and are saved to return
	// them in [Service.Config].

	// running is true when the service has started.
	running atomic.Bool
}

// New returns a new properly initialized *Service.  If c is nil, svc is a nil
// *Service that does nothing.  The fields of c must not be modified after
// calling New.
func New(c *Config) (svc *Service, err error) {
	if c == nil {
		return nil, nil
	}

	rlMw, err := newRatelimitMw(c.Logger, c.Ratelimit)
	if err != nil {
		return nil, fmt.Errorf("ratelimit middleware: %w", err)
	}

	svc = &Service{
		logger: c.Logger,
		proxyConf: &proxy.Config{
			UpstreamMode:   c.UpstreamMode,
			DNS64Prefs:     c.DNS64Prefixes,
			CacheSizeBytes: c.CacheSize,
			CacheEnabled:   c.CacheEnabled,
			RefuseAny:      c.RefuseAny,
			UseDNS64:       c.UseDNS64,
		},
		bootstraps:          c.BootstrapServers,
		upstreams:           c.UpstreamServers,
		upstreamTimeout:     c.UpstreamTimeout,
		bootstrapPreferIPv6: c.BootstrapPreferIPv6,
	}

	upstreams, resolvers, err := addressesToUpstreams(
		svc.logger.With(slogutil.KeyPrefix, aghslog.PrefixDNSProxy),
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
		Logger: svc.logger,
		UpstreamConfig: &proxy.UpstreamConfig{
			Upstreams: upstreams,
		},
		UDPListenAddr:  udpAddrs(c.Addresses),
		TCPListenAddr:  tcpAddrs(c.Addresses),
		UpstreamMode:   svc.proxyConf.UpstreamMode,
		RequestHandler: rlMw.Wrap(proxy.DefaultHandler{}),
		DNS64Prefs:     svc.proxyConf.DNS64Prefs,
		CacheEnabled:   svc.proxyConf.CacheEnabled,
		RefuseAny:      svc.proxyConf.RefuseAny,
		UseDNS64:       svc.proxyConf.UseDNS64,
	})
	if err != nil {
		return nil, fmt.Errorf("proxy: %w", err)
	}

	return svc, nil
}

// newRatelimitMw returns the ratelimit middleware.  In case of invalid
// ratelimit configuration returns an error. l must not be nil.
func newRatelimitMw(l *slog.Logger, limit int) (mw proxy.Middleware, err error) {
	if limit == 0 {
		return proxy.MiddlewareFunc(proxy.PassThrough), nil
	}

	rlConf := &ratelimit.Config{
		Logger:        l.With(slogutil.KeyPrefix, "ratelimit"),
		Ratelimit:     uint(limit),
		SubnetLenIPv4: netutil.IPv4BitLen,
		SubnetLenIPv6: netutil.IPv6BitLen,
	}
	if err = rlConf.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return ratelimit.NewMiddleware(rlConf), nil
}

// addressesToUpstreams is a wrapper around [upstream.AddressToUpstream].  It
// accepts a slice of addresses and other upstream parameters, and returns a
// slice of upstreams.  logger must not be nil.
func addressesToUpstreams(
	logger *slog.Logger,
	upsStrs []string,
	bootstraps []string,
	timeout time.Duration,
	preferIPv6 bool,
) (upstreams []upstream.Upstream, boots []*upstream.UpstreamResolver, err error) {
	boots, err = aghnet.ParseBootstraps(bootstraps, &upstream.Options{
		Logger:     logger.With(aghslog.KeyUpstreamType, aghslog.UpstreamTypeBootstrap),
		Timeout:    timeout,
		PreferIPv6: preferIPv6,
	})
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
			Logger:     logger.With(aghslog.KeyUpstreamType, aghslog.UpstreamTypeMain),
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

	// TODO(d.kolyshev): Fill ratelimit.
	c = &Config{
		Logger:              svc.logger,
		UpstreamMode:        svc.proxyConf.UpstreamMode,
		Addresses:           addrs,
		BootstrapServers:    svc.bootstraps,
		UpstreamServers:     svc.upstreams,
		DNS64Prefixes:       svc.proxyConf.DNS64Prefs,
		UpstreamTimeout:     svc.upstreamTimeout,
		CacheSize:           svc.proxyConf.CacheSizeBytes,
		BootstrapPreferIPv6: svc.bootstrapPreferIPv6,
		CacheEnabled:        svc.proxyConf.CacheEnabled,
		RefuseAny:           svc.proxyConf.RefuseAny,
		UseDNS64:            svc.proxyConf.UseDNS64,
	}

	return c
}
