package client

import (
	"context"
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/rdns"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
)

// ErrClosed is returned from [AddressProcessor.Close] if it's closed more than
// once.
const ErrClosed errors.Error = "use of closed address processor"

// AddressProcessor is the interface for types that can process clients.
type AddressProcessor interface {
	Process(ctx context.Context, ip netip.Addr)
	Close() (err error)
}

// EmptyAddrProc is an [AddressProcessor] that does nothing.
type EmptyAddrProc struct{}

// type check
var _ AddressProcessor = EmptyAddrProc{}

// Process implements the [AddressProcessor] interface for EmptyAddrProc.
func (EmptyAddrProc) Process(_ context.Context, _ netip.Addr) {}

// Close implements the [AddressProcessor] interface for EmptyAddrProc.
func (EmptyAddrProc) Close() (_ error) { return nil }

// DefaultAddrProcConfig is the configuration structure for address processors.
type DefaultAddrProcConfig struct {
	// BaseLogger is used to create loggers with custom prefixes for sources of
	// information about runtime clients.  It must not be nil.
	BaseLogger *slog.Logger

	// DialContext is used to create TCP connections to WHOIS servers.
	// DialContext must not be nil if [DefaultAddrProcConfig.UseWHOIS] is true.
	DialContext aghnet.DialContextFunc

	// Exchanger is used to perform rDNS queries.  Exchanger must not be nil if
	// [DefaultAddrProcConfig.UseRDNS] is true.
	Exchanger rdns.Exchanger

	// PrivateSubnets are used to determine if an incoming IP address is
	// private.  It must not be nil.
	PrivateSubnets netutil.SubnetSet

	// AddressUpdater is used to update the information about a client's IP
	// address.  It must not be nil.
	AddressUpdater AddressUpdater

	// InitialAddresses are the addresses that are queued for processing
	// immediately by [NewDefaultAddrProc].
	InitialAddresses []netip.Addr

	// CatchPanics, if true, makes the address processor catch and log panics.
	//
	// TODO(a.garipov): Consider better ways to do this or apply this method to
	// other parts of the codebase.
	CatchPanics bool

	// UseRDNS, if true, enables resolving of client IP addresses using reverse
	// DNS.
	UseRDNS bool

	// UsePrivateRDNS, if true, enables resolving of private client IP addresses
	// using reverse DNS.  See [DefaultAddrProcConfig.PrivateSubnets].
	UsePrivateRDNS bool

	// UseWHOIS, if true, enables resolving of client IP addresses using WHOIS.
	UseWHOIS bool
}

// AddressUpdater is the interface for storages of DNS clients that can update
// information about them.
//
// TODO(a.garipov): Consider using the actual client storage once it is moved
// into this package.
type AddressUpdater interface {
	// UpdateAddress updates information about an IP address, setting host (if
	// not empty) and WHOIS information (if not nil).
	UpdateAddress(ctx context.Context, ip netip.Addr, host string, info *whois.Info)
}

// DefaultAddrProc processes incoming client addresses with rDNS and WHOIS, if
// configured, and updates that information in a client storage.
type DefaultAddrProc struct {
	// logger is used to log the operation of address processor.
	logger *slog.Logger

	// clientIPsMu serializes closure of clientIPs and access to isClosed.
	clientIPsMu *sync.Mutex

	// clientIPs is the channel queueing client processing tasks.
	clientIPs chan netip.Addr

	// rdns is used to perform rDNS lookups of clients' IP addresses.
	rdns rdns.Interface

	// whois is used to perform WHOIS lookups of clients' IP addresses.
	whois whois.Interface

	// addrUpdater is used to update the information about a client's IP
	// address.
	addrUpdater AddressUpdater

	// privateSubnets are used to determine if an incoming IP address is
	// private.
	privateSubnets netutil.SubnetSet

	// isClosed is set to true once the address processor is closed.
	isClosed bool

	// usePrivateRDNS, if true, enables resolving of private client IP addresses
	// using reverse DNS.
	usePrivateRDNS bool
}

const (
	// defaultQueueSize is the size of queue of IPs for rDNS and WHOIS
	// processing.
	defaultQueueSize = 255

	// defaultCacheSize is the maximum size of the cache for rDNS and WHOIS
	// processing.  It must be greater than zero.
	defaultCacheSize = 10_000

	// defaultIPTTL is the Time to Live duration for IP addresses cached by
	// rDNS and WHOIS.
	defaultIPTTL = 1 * time.Hour
)

// NewDefaultAddrProc returns a new running client address processor.  c must
// not be nil.
func NewDefaultAddrProc(c *DefaultAddrProcConfig) (p *DefaultAddrProc) {
	p = &DefaultAddrProc{
		logger:         c.BaseLogger.With(slogutil.KeyPrefix, "addrproc"),
		clientIPsMu:    &sync.Mutex{},
		clientIPs:      make(chan netip.Addr, defaultQueueSize),
		rdns:           &rdns.Empty{},
		addrUpdater:    c.AddressUpdater,
		whois:          &whois.Empty{},
		privateSubnets: c.PrivateSubnets,
		usePrivateRDNS: c.UsePrivateRDNS,
	}

	if c.UseRDNS {
		p.rdns = rdns.New(&rdns.Config{
			Logger:    c.BaseLogger.With(slogutil.KeyPrefix, "rdns"),
			Exchanger: c.Exchanger,
			CacheSize: defaultCacheSize,
			CacheTTL:  defaultIPTTL,
		})
	}

	if c.UseWHOIS {
		p.whois = newWHOIS(c.BaseLogger.With(slogutil.KeyPrefix, "whois"), c.DialContext)
	}

	// TODO(s.chzhen):  Pass context.
	ctx := context.TODO()

	go p.process(ctx, c.CatchPanics)

	for _, ip := range c.InitialAddresses {
		p.Process(ctx, ip)
	}

	return p
}

// newWHOIS returns a whois.Interface instance using the given function for
// dialing.
func newWHOIS(logger *slog.Logger, dialFunc aghnet.DialContextFunc) (w whois.Interface) {
	// TODO(s.chzhen):  Consider making configurable.
	const (
		// defaultTimeout is the timeout for WHOIS requests.
		defaultTimeout = 5 * time.Second

		// defaultMaxConnReadSize is an upper limit in bytes for reading from a
		// net.Conn.
		defaultMaxConnReadSize = 64 * 1024

		// defaultMaxRedirects is the maximum redirects count.
		defaultMaxRedirects = 5

		// defaultMaxInfoLen is the maximum length of whois.Info fields.
		defaultMaxInfoLen = 250
	)

	return whois.New(&whois.Config{
		Logger:          logger,
		DialContext:     dialFunc,
		ServerAddr:      whois.DefaultServer,
		Port:            whois.DefaultPort,
		Timeout:         defaultTimeout,
		CacheSize:       defaultCacheSize,
		MaxConnReadSize: defaultMaxConnReadSize,
		MaxRedirects:    defaultMaxRedirects,
		MaxInfoLen:      defaultMaxInfoLen,
		CacheTTL:        defaultIPTTL,
	})
}

// type check
var _ AddressProcessor = (*DefaultAddrProc)(nil)

// Process implements the [AddressProcessor] interface for *DefaultAddrProc.
func (p *DefaultAddrProc) Process(ctx context.Context, ip netip.Addr) {
	p.clientIPsMu.Lock()
	defer p.clientIPsMu.Unlock()

	if p.isClosed {
		return
	}

	select {
	case p.clientIPs <- ip:
		// Go on.
	default:
		p.logger.DebugContext(ctx, "ip channel is full", "len", len(p.clientIPs))
	}
}

// process processes the incoming client IP-address information.  It is intended
// to be used as a goroutine.  Once clientIPs is closed, process exits.
func (p *DefaultAddrProc) process(ctx context.Context, catchPanics bool) {
	if catchPanics {
		defer slogutil.RecoverAndLog(ctx, p.logger)
	}

	p.logger.InfoContext(ctx, "processing addresses")

	for ip := range p.clientIPs {
		host := p.processRDNS(ctx, ip)
		info := p.processWHOIS(ctx, ip)

		p.addrUpdater.UpdateAddress(ctx, ip, host, info)
	}

	p.logger.InfoContext(ctx, "finished processing addresses")
}

// processRDNS resolves the clients' IP addresses using reverse DNS.  host is
// empty if there were errors or if the information hasn't changed.
func (p *DefaultAddrProc) processRDNS(ctx context.Context, ip netip.Addr) (host string) {
	start := time.Now()
	p.logger.DebugContext(ctx, "processing rdns", "ip", ip)
	defer func() {
		p.logger.DebugContext(
			ctx,
			"finished processing rdns",
			"ip", ip,
			"host", host,
			"elapsed", time.Since(start),
		)
	}()

	ok := p.shouldResolve(ip)
	if !ok {
		return
	}

	host, changed := p.rdns.Process(ctx, ip)
	if !changed {
		host = ""
	}

	return host
}

// shouldResolve returns false if ip is a loopback address, or ip is private and
// resolving of private addresses is disabled.
func (p *DefaultAddrProc) shouldResolve(ip netip.Addr) (ok bool) {
	return !ip.IsLoopback() && (p.usePrivateRDNS || !p.privateSubnets.Contains(ip))
}

// processWHOIS looks up the information about clients' IP addresses in the
// WHOIS databases.  info is nil if there were errors or if the information
// hasn't changed.
func (p *DefaultAddrProc) processWHOIS(ctx context.Context, ip netip.Addr) (info *whois.Info) {
	start := time.Now()
	p.logger.DebugContext(ctx, "processing whois", "ip", ip)
	defer func() {
		p.logger.DebugContext(
			ctx,
			"finished processing whois",
			"ip", ip,
			"whois", info,
			"elapsed", time.Since(start),
		)
	}()

	// TODO(s.chzhen):  Move the timeout logic from WHOIS configuration to the
	// context.
	info, changed := p.whois.Process(ctx, ip)
	if !changed {
		info = nil
	}

	return info
}

// Close implements the [AddressProcessor] interface for *DefaultAddrProc.
func (p *DefaultAddrProc) Close() (err error) {
	p.clientIPsMu.Lock()
	defer p.clientIPsMu.Unlock()

	if p.isClosed {
		return ErrClosed
	}

	close(p.clientIPs)
	p.isClosed = true

	return nil
}
