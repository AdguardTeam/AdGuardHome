package aghtest

import (
	"context"
	"io/fs"
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
)

// Interface Mocks
//
// Keep entities in this file in alphabetic order.

// Standard Library

// Package fs

// type check
var _ fs.FS = &FS{}

// FS is a mock [fs.FS] implementation for tests.
type FS struct {
	OnOpen func(name string) (fs.File, error)
}

// Open implements the [fs.FS] interface for *FS.
func (fsys *FS) Open(name string) (fs.File, error) {
	return fsys.OnOpen(name)
}

// type check
var _ fs.GlobFS = &GlobFS{}

// GlobFS is a mock [fs.GlobFS] implementation for tests.
type GlobFS struct {
	// FS is embedded here to avoid implementing all it's methods.
	FS
	OnGlob func(pattern string) ([]string, error)
}

// Glob implements the [fs.GlobFS] interface for *GlobFS.
func (fsys *GlobFS) Glob(pattern string) ([]string, error) {
	return fsys.OnGlob(pattern)
}

// type check
var _ fs.StatFS = &StatFS{}

// StatFS is a mock [fs.StatFS] implementation for tests.
type StatFS struct {
	// FS is embedded here to avoid implementing all it's methods.
	FS
	OnStat func(name string) (fs.FileInfo, error)
}

// Stat implements the [fs.StatFS] interface for *StatFS.
func (fsys *StatFS) Stat(name string) (fs.FileInfo, error) {
	return fsys.OnStat(name)
}

// Package net

// type check
var _ net.Listener = (*Listener)(nil)

// Listener is a mock [net.Listener] implementation for tests.
type Listener struct {
	OnAccept func() (conn net.Conn, err error)
	OnAddr   func() (addr net.Addr)
	OnClose  func() (err error)
}

// Accept implements the [net.Listener] interface for *Listener.
func (l *Listener) Accept() (conn net.Conn, err error) {
	return l.OnAccept()
}

// Addr implements the [net.Listener] interface for *Listener.
func (l *Listener) Addr() (addr net.Addr) {
	return l.OnAddr()
}

// Close implements the [net.Listener] interface for *Listener.
func (l *Listener) Close() (err error) {
	return l.OnClose()
}

// Module adguard-home

// Package aghos

// type check
var _ aghos.FSWatcher = (*FSWatcher)(nil)

// FSWatcher is a mock [aghos.FSWatcher] implementation for tests.
type FSWatcher struct {
	OnEvents func() (e <-chan struct{})
	OnAdd    func(name string) (err error)
	OnClose  func() (err error)
}

// Events implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Events() (e <-chan struct{}) {
	return w.OnEvents()
}

// Add implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Add(name string) (err error) {
	return w.OnAdd(name)
}

// Close implements the [aghos.FSWatcher] interface for *FSWatcher.
func (w *FSWatcher) Close() (err error) {
	return w.OnClose()
}

// Package agh

// type check
var _ agh.ServiceWithConfig[struct{}] = (*ServiceWithConfig[struct{}])(nil)

// ServiceWithConfig is a mock [agh.ServiceWithConfig] implementation for tests.
type ServiceWithConfig[ConfigType any] struct {
	OnStart    func() (err error)
	OnShutdown func(ctx context.Context) (err error)
	OnConfig   func() (c ConfigType)
}

// Start implements the [agh.ServiceWithConfig] interface for
// *ServiceWithConfig.
func (s *ServiceWithConfig[_]) Start() (err error) {
	return s.OnStart()
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

// Module dnsproxy

// Package upstream

// type check
var _ upstream.Upstream = (*UpstreamMock)(nil)

// UpstreamMock is a mock [upstream.Upstream] implementation for tests.
//
// TODO(a.garipov): Replace with all uses of Upstream with UpstreamMock and
// rename it to just Upstream.
type UpstreamMock struct {
	OnAddress  func() (addr string)
	OnExchange func(req *dns.Msg) (resp *dns.Msg, err error)
	OnClose    func() (err error)
}

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
