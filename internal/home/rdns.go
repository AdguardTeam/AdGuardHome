package home

import (
	"encoding/binary"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// RDNS resolves clients' addresses to enrich their metadata.
type RDNS struct {
	exchanger dnsforward.RDNSExchanger
	clients   *clientsContainer

	// ipCh used to pass client's IP to rDNS workerLoop.
	ipCh chan netip.Addr

	// ipCache caches the IP addresses to be resolved by rDNS.  The resolved
	// address stays here while it's inside clients.  After leaving clients the
	// address will be resolved once again.  If the address couldn't be
	// resolved, cache prevents further attempts to resolve it for some time.
	ipCache cache.Cache

	// usePrivate stores the state of current private reverse-DNS resolving
	// settings.
	usePrivate atomic.Bool
}

// Default AdGuard Home reverse DNS values.
const (
	revDNSCacheSize = 10000

	// TODO(e.burkov):  Make these values configurable.
	revDNSCacheTTL        = 24 * 60 * 60
	revDNSFailureCacheTTL = 1 * 60 * 60

	revDNSQueueSize = 256
)

// NewRDNS creates and returns initialized RDNS.
func NewRDNS(
	exchanger dnsforward.RDNSExchanger,
	clients *clientsContainer,
	usePrivate bool,
) (rDNS *RDNS) {
	rDNS = &RDNS{
		exchanger: exchanger,
		clients:   clients,
		ipCache: cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  revDNSCacheSize,
		}),
		ipCh: make(chan netip.Addr, revDNSQueueSize),
	}

	rDNS.usePrivate.Store(usePrivate)

	go rDNS.workerLoop()

	return rDNS
}

// ensurePrivateCache ensures that the state of the RDNS cache is consistent
// with the current private client RDNS resolving settings.
//
// TODO(e.burkov): Clearing cache each time this value changed is not a perfect
// approach since only unresolved locally-served addresses should be removed.
// Implement when improving the cache.
func (r *RDNS) ensurePrivateCache() {
	usePrivate := r.exchanger.ResolvesPrivatePTR()
	if r.usePrivate.CompareAndSwap(!usePrivate, usePrivate) {
		r.ipCache.Clear()
	}
}

// isCached returns true if ip is already cached and not expired yet.  It also
// caches it otherwise.
func (r *RDNS) isCached(ip netip.Addr) (ok bool) {
	ipBytes := ip.AsSlice()
	now := uint64(time.Now().Unix())
	if expire := r.ipCache.Get(ipBytes); len(expire) != 0 {
		return binary.BigEndian.Uint64(expire) > now
	}

	return false
}

// cache caches the ip address for ttl seconds.
func (r *RDNS) cache(ip netip.Addr, ttl uint64) {
	ipData := ip.AsSlice()

	ttlData := [8]byte{}
	binary.BigEndian.PutUint64(ttlData[:], uint64(time.Now().Unix())+ttl)

	r.ipCache.Set(ipData, ttlData[:])
}

// Begin adds the ip to the resolving queue if it is not cached or already
// resolved.
func (r *RDNS) Begin(ip netip.Addr) {
	r.ensurePrivateCache()

	if r.isCached(ip) || r.clients.clientSource(ip) > ClientSourceRDNS {
		return
	}

	select {
	case r.ipCh <- ip:
		log.Debug("rdns: %q added to queue", ip)
	default:
		log.Debug("rdns: queue is full")
	}
}

// workerLoop handles incoming IP addresses from ipChan and adds it into
// clients.
func (r *RDNS) workerLoop() {
	defer log.OnPanic("rdns")

	for ip := range r.ipCh {
		ttl := uint64(revDNSCacheTTL)

		host, err := r.exchanger.Exchange(ip.AsSlice())
		if err != nil {
			log.Debug("rdns: resolving %q: %s", ip, err)
			if errors.Is(err, dnsforward.ErrRDNSFailed) {
				// Cache failure for a less time.
				ttl = revDNSFailureCacheTTL
			}
		}

		r.cache(ip, ttl)

		if host != "" {
			_ = r.clients.AddHost(ip, host, ClientSourceRDNS)
		}
	}
}
