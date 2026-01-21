package aghtest

import (
	"context"
	"net/http"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	nextagh "github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/rdns"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
)

// FSWatcher is a fake [aghos.FSWatcher] implementation for tests.
type FSWatcher struct {
	OnStart    func(ctx context.Context) (err error)
	OnShutdown func(ctx context.Context) (err error)
	OnEvents   func() (e <-chan aghos.Event)
	OnAdd      func(name string) (err error)
	OnRemove   func(name string) (err error)
}

// NewFSWatcher returns a new *FSWatcher all methods of which panic.
func NewFSWatcher() (w *FSWatcher) {
	return &FSWatcher{
		OnStart:    func(ctx context.Context) (_ error) { panic(testutil.UnexpectedCall(ctx)) },
		OnShutdown: func(ctx context.Context) (_ error) { panic(testutil.UnexpectedCall(ctx)) },
		OnEvents:   func() (_ <-chan aghos.Event) { panic(testutil.UnexpectedCall()) },
		OnAdd:      func(name string) (_ error) { panic(testutil.UnexpectedCall(name)) },
		OnRemove:   func(name string) (_ error) { panic(testutil.UnexpectedCall(name)) },
	}
}

// type check
var _ aghos.FSWatcher = (*FSWatcher)(nil)

// Start implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Start(ctx context.Context) (err error) {
	return w.OnStart(ctx)
}

// Shutdown implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Shutdown(ctx context.Context) (err error) {
	return w.OnShutdown(ctx)
}

// Events implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Events() (e <-chan aghos.Event) {
	return w.OnEvents()
}

// Add implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Add(name string) (err error) {
	return w.OnAdd(name)
}

// Remove implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Remove(name string) (err error) {
	return w.OnRemove(name)
}

// ServiceWithConfig is a fake [nextagh.ServiceWithConfig] implementation for
// tests.
type ServiceWithConfig[ConfigType any] struct {
	OnStart    func(ctx context.Context) (err error)
	OnShutdown func(ctx context.Context) (err error)
	OnConfig   func() (c ConfigType)
}

// type check
var _ nextagh.ServiceWithConfig[struct{}] = (*ServiceWithConfig[struct{}])(nil)

// Start implements the [nextagh.ServiceWithConfig] interface for
// *ServiceWithConfig.
func (s *ServiceWithConfig[_]) Start(ctx context.Context) (err error) {
	return s.OnStart(ctx)
}

// Shutdown implements the [nextagh.ServiceWithConfig] interface for
// *ServiceWithConfig.
func (s *ServiceWithConfig[_]) Shutdown(ctx context.Context) (err error) {
	return s.OnShutdown(ctx)
}

// Config implements the [nextagh.ServiceWithConfig] interface for
// *ServiceWithConfig.
func (s *ServiceWithConfig[ConfigType]) Config() (c ConfigType) {
	return s.OnConfig()
}

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

// Exchanger is a fake [rdns.Exchanger] implementation for tests.
type Exchanger struct {
	OnExchange func(ctx context.Context, ip netip.Addr) (host string, ttl time.Duration, err error)
}

// type check
var _ rdns.Exchanger = (*Exchanger)(nil)

// Exchange implements [rdns.Exchanger] interface for *Exchanger.
func (e *Exchanger) Exchange(
	ctx context.Context,
	ip netip.Addr,
) (host string, ttl time.Duration, err error) {
	return e.OnExchange(ctx, ip)
}

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

// ConfigModifier is a fake [agh.ConfigModifier] implementation for tests.
type ConfigModifier struct {
	OnApply func(ctx context.Context)
}

// type check
var _ agh.ConfigModifier = (*ConfigModifier)(nil)

// Apply implements the [agh.ConfigModifier] interface for *ConfigModifier.
func (m *ConfigModifier) Apply(ctx context.Context) {
	m.OnApply(ctx)
}

// Registrar is a fake [aghhttp.Registrar] implementation for tests.
type Registrar struct {
	OnRegister func(method, path string, h http.HandlerFunc)
}

// type check
var _ aghhttp.Registrar = (*Registrar)(nil)

// Register implements the [aghhttp.Registrar] interface for *Registrar.
func (m *Registrar) Register(method, path string, h http.HandlerFunc) {
	m.OnRegister(method, path, h)
}
