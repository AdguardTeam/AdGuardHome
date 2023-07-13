package home

import (
	"context"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/rdns"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/log"
)

// TODO(a.garipov): It is currently hard to add tests for this structure due to
// strong coupling between it and Context.dnsServer with Context.clients.
// Resolve this coupling and add proper testing.

// clientAddrProcessor processes incoming client addresses with rDNS and WHOIS,
// if configured.
type clientAddrProcessor struct {
	rdns  rdns.Interface
	whois whois.Interface
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

// newClientAddrProcessor returns a new client address processor.  c must not be
// nil.
func newClientAddrProcessor(c *clientSourcesConfig) (p *clientAddrProcessor) {
	p = &clientAddrProcessor{}

	if c.RDNS {
		p.rdns = rdns.New(&rdns.Config{
			Exchanger: Context.dnsServer,
			CacheSize: defaultCacheSize,
			CacheTTL:  defaultIPTTL,
		})
	} else {
		p.rdns = rdns.Empty{}
	}

	if c.WHOIS {
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

		p.whois = whois.New(&whois.Config{
			DialContext:     customDialContext,
			ServerAddr:      whois.DefaultServer,
			Port:            whois.DefaultPort,
			Timeout:         defaultTimeout,
			CacheSize:       defaultCacheSize,
			MaxConnReadSize: defaultMaxConnReadSize,
			MaxRedirects:    defaultMaxRedirects,
			MaxInfoLen:      defaultMaxInfoLen,
			CacheTTL:        defaultIPTTL,
		})
	} else {
		p.whois = whois.Empty{}
	}

	return p
}

// process processes the incoming client IP-address information.  It is intended
// to be used as a goroutine.
func (p *clientAddrProcessor) process(clientIPs <-chan netip.Addr) {
	defer log.OnPanic("clientAddrProcessor.process")

	log.Info("home: processing client addresses")

	for ip := range clientIPs {
		p.processRDNS(ip)
		p.processWHOIS(ip)
	}

	log.Info("home: finished processing client addresses")
}

// processRDNS resolves the clients' IP addresses using reverse DNS.
func (p *clientAddrProcessor) processRDNS(ip netip.Addr) {
	start := time.Now()
	log.Debug("home: processing client %s with rdns", ip)
	defer func() {
		log.Debug("home: finished processing client %s with rdns in %s", ip, time.Since(start))
	}()

	ok := Context.dnsServer.ShouldResolveClient(ip)
	if !ok {
		return
	}

	host, changed := p.rdns.Process(ip)
	if host == "" || !changed {
		return
	}

	ok = Context.clients.AddHost(ip, host, ClientSourceRDNS)
	if ok {
		return
	}

	log.Debug("dns: setting rdns info for client %q: already set with higher priority source", ip)
}

// processWHOIS looks up the information aobut clients' IP addresses in the
// WHOIS databases.
func (p *clientAddrProcessor) processWHOIS(ip netip.Addr) {
	start := time.Now()
	log.Debug("home: processing client %s with whois", ip)
	defer func() {
		log.Debug("home: finished processing client %s with whois in %s", ip, time.Since(start))
	}()

	// TODO(s.chzhen):  Move the timeout logic from WHOIS configuration to the
	// context.
	info, changed := p.whois.Process(context.Background(), ip)
	if info == nil || !changed {
		return
	}

	Context.clients.setWHOISInfo(ip, info)
}
