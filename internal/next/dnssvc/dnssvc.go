// Package dnssvc contains the AdGuard Home DNS service.
//
// TODO(a.garipov): Define, if all methods of a *Service should work with a nil
// receiver.
package dnssvc

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	// TODO(a.garipov): Add a “dnsproxy proxy” package to shield us from changes
	// and replacement of module dnsproxy.
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
)

// Config is the AdGuard Home DNS service configuration structure.
//
// TODO(a.garipov): Add timeout for incoming requests.
type Config struct {
	// Addresses are the addresses on which to serve plain DNS queries.
	Addresses []netip.AddrPort

	// Upstreams are the DNS upstreams to use.  If not set, upstreams are
	// created using data from BootstrapServers, UpstreamServers, and
	// UpstreamTimeout.
	//
	// TODO(a.garipov): Think of a better scheme.  Those other three parameters
	// are here only to make Config work properly.
	Upstreams []upstream.Upstream

	// BootstrapServers are the addresses for bootstrapping the upstream DNS
	// server addresses.
	BootstrapServers []string

	// UpstreamServers are the upstream DNS server addresses to use.
	UpstreamServers []string

	// UpstreamTimeout is the timeout for upstream requests.
	UpstreamTimeout time.Duration
}

// Service is the AdGuard Home DNS service.  A nil *Service is a valid
// [agh.Service] that does nothing.
type Service struct {
	proxy      *proxy.Proxy
	bootstraps []string
	upstreams  []string
	upsTimeout time.Duration
	running    atomic.Bool
}

// New returns a new properly initialized *Service.  If c is nil, svc is a nil
// *Service that does nothing.  The fields of c must not be modified after
// calling New.
func New(c *Config) (svc *Service, err error) {
	if c == nil {
		return nil, nil
	}

	svc = &Service{
		bootstraps: c.BootstrapServers,
		upstreams:  c.UpstreamServers,
		upsTimeout: c.UpstreamTimeout,
	}

	var upstreams []upstream.Upstream
	if len(c.Upstreams) > 0 {
		upstreams = c.Upstreams
	} else {
		upstreams, err = addressesToUpstreams(
			c.UpstreamServers,
			c.BootstrapServers,
			c.UpstreamTimeout,
		)
		if err != nil {
			return nil, fmt.Errorf("converting upstreams: %w", err)
		}
	}

	svc.proxy = &proxy.Proxy{
		Config: proxy.Config{
			UDPListenAddr: udpAddrs(c.Addresses),
			TCPListenAddr: tcpAddrs(c.Addresses),
			UpstreamConfig: &proxy.UpstreamConfig{
				Upstreams: upstreams,
			},
		},
	}

	err = svc.proxy.Init()
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
) (upstreams []upstream.Upstream, err error) {
	upstreams = make([]upstream.Upstream, len(upsStrs))
	for i, upsStr := range upsStrs {
		upstreams[i], err = upstream.AddressToUpstream(upsStr, &upstream.Options{
			Bootstrap: bootstraps,
			Timeout:   timeout,
		})
		if err != nil {
			return nil, fmt.Errorf("upstream at index %d: %w", i, err)
		}
	}

	return upstreams, nil
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
var _ agh.Service = (*Service)(nil)

// Start implements the [agh.Service] interface for *Service.  svc may be nil.
// After Start exits, all DNS servers have tried to start, but there is no
// guarantee that they did.  Errors from the servers are written to the log.
func (svc *Service) Start() (err error) {
	if svc == nil {
		return nil
	}

	defer func() {
		// TODO(a.garipov): [proxy.Proxy.Start] doesn't actually have any way to
		// tell when all servers are actually up, so at best this is merely an
		// assumption.
		svc.running.Store(err == nil)
	}()

	return svc.proxy.Start()
}

// Shutdown implements the [agh.Service] interface for *Service.  svc may be
// nil.
func (svc *Service) Shutdown(ctx context.Context) (err error) {
	if svc == nil {
		return nil
	}

	return svc.proxy.Stop()
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
		Addresses:        addrs,
		BootstrapServers: svc.bootstraps,
		UpstreamServers:  svc.upstreams,
		UpstreamTimeout:  svc.upsTimeout,
	}

	return c
}
