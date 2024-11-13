package aghtest

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/rdns"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
)

// Interface Mocks
//
// Keep entities in this file in alphabetic order.

// Module adguard-home

// Package aghos

// FSWatcher is a fake [aghos.FSWatcher] implementation for tests.
type FSWatcher struct {
	OnStart  func() (err error)
	OnClose  func() (err error)
	OnEvents func() (e <-chan struct{})
	OnAdd    func(name string) (err error)
}

// type check
var _ aghos.FSWatcher = (*FSWatcher)(nil)

// Start implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Start() (err error) {
	return w.OnStart()
}

// Close implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Close() (err error) {
	return w.OnClose()
}

// Events implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Events() (e <-chan struct{}) {
	return w.OnEvents()
}

// Add implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Add(name string) (err error) {
	return w.OnAdd(name)
}

// Package agh

// ServiceWithConfig is a fake [agh.ServiceWithConfig] implementation for tests.
type ServiceWithConfig[ConfigType any] struct {
	OnStart    func(ctx context.Context) (err error)
	OnShutdown func(ctx context.Context) (err error)
	OnConfig   func() (c ConfigType)
}

// type check
var _ agh.ServiceWithConfig[struct{}] = (*ServiceWithConfig[struct{}])(nil)

// Start implements the [agh.ServiceWithConfig] interface for
// *ServiceWithConfig.
func (s *ServiceWithConfig[_]) Start(ctx context.Context) (err error) {
	return s.OnStart(ctx)
}

// Shutdown implements the [agh.ServiceWithConfig] interface for
// *ServiceWithConfig.
func (s *ServiceWithConfig[_]) Shutdown(ctx context.Context) (err error) {
	return s.OnShutdown(ctx)
}

// Config implements the [agh.ServiceWithConfig] interface for
// *ServiceWithConfig.
func (s *ServiceWithConfig[ConfigType]) Config() (c ConfigType) {
	return s.OnConfig()
}

// Package client

// AddressProcessor is a fake [client.AddressProcessor] implementation for
// tests.
type AddressProcessor struct {
	OnProcess func(ctx context.Context, ip netip.Addr)
	OnClose   func() (err error)
}

// Process implements the [client.AddressProcessor] interface for
// *AddressProcessor.
func (p *AddressProcessor) Process(ctx context.Context, ip netip.Addr) {
	p.OnProcess(ctx, ip)
}

// Close implements the [client.AddressProcessor] interface for
// *AddressProcessor.
func (p *AddressProcessor) Close() (err error) {
	return p.OnClose()
}

// AddressUpdater is a fake [client.AddressUpdater] implementation for tests.
type AddressUpdater struct {
	OnUpdateAddress func(ctx context.Context, ip netip.Addr, host string, info *whois.Info)
}

// UpdateAddress implements the [client.AddressUpdater] interface for
// *AddressUpdater.
func (p *AddressUpdater) UpdateAddress(
	ctx context.Context,
	ip netip.Addr,
	host string,
	info *whois.Info,
) {
	p.OnUpdateAddress(ctx, ip, host, info)
}

// Package dnsforward

// ClientsContainer is a fake [dnsforward.ClientsContainer] implementation for
// tests.
type ClientsContainer struct {
	OnUpstreamConfigByID func(
		id string,
		boot upstream.Resolver,
	) (conf *proxy.CustomUpstreamConfig, err error)
}

// UpstreamConfigByID implements the [dnsforward.ClientsContainer] interface
// for *ClientsContainer.
func (c *ClientsContainer) UpstreamConfigByID(
	id string,
	boot upstream.Resolver,
) (conf *proxy.CustomUpstreamConfig, err error) {
	return c.OnUpstreamConfigByID(id, boot)
}

// Package filtering

// Resolver is a fake [filtering.Resolver] implementation for tests.
type Resolver struct {
	OnLookupIP func(ctx context.Context, network, host string) (ips []net.IP, err error)
}

// LookupIP implements the [filtering.Resolver] interface for *Resolver.
func (r *Resolver) LookupIP(ctx context.Context, network, host string) (ips []net.IP, err error) {
	return r.OnLookupIP(ctx, network, host)
}

// Package rdns

// Exchanger is a fake [rdns.Exchanger] implementation for tests.
type Exchanger struct {
	OnExchange func(ip netip.Addr) (host string, ttl time.Duration, err error)
}

// type check
var _ rdns.Exchanger = (*Exchanger)(nil)

// Exchange implements [rdns.Exchanger] interface for *Exchanger.
func (e *Exchanger) Exchange(ip netip.Addr) (host string, ttl time.Duration, err error) {
	return e.OnExchange(ip)
}

// Module dnsproxy

// Package upstream

// UpstreamMock is a fake [upstream.Upstream] implementation for tests.
//
// TODO(a.garipov): Replace with all uses of Upstream with UpstreamMock and
// rename it to just Upstream.
type UpstreamMock struct {
	OnAddress  func() (addr string)
	OnExchange func(req *dns.Msg) (resp *dns.Msg, err error)
	OnClose    func() (err error)
}

// type check
var _ upstream.Upstream = (*UpstreamMock)(nil)

// Address implements the [upstream.Upstream] interface for *UpstreamMock.
func (u *UpstreamMock) Address() (addr string) {
	return u.OnAddress()
}

// Exchange implements the [upstream.Upstream] interface for *UpstreamMock.
func (u *UpstreamMock) Exchange(req *dns.Msg) (resp *dns.Msg, err error) {
	return u.OnExchange(req)
}

// Close implements the [upstream.Upstream] interface for *UpstreamMock.
func (u *UpstreamMock) Close() (err error) {
	return u.OnClose()
}
