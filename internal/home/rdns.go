package home

import (
	"encoding/binary"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
)

// RDNS resolves clients' addresses to enrich their metadata.
type RDNS struct {
	exchanger dnsforward.RDNSExchanger
	clients   *clientsContainer

	// usePrivate is used to store the state of current private RDNS resolving
	// settings and to react to it's changes.
	usePrivate uint32

	// ipCh used to pass client's IP to rDNS workerLoop.
	ipCh chan netip.Addr

	// ipCache caches the IP addresses to be resolved by rDNS.  The resolved
	// address stays here while it's inside clients.  After leaving clients the
	// address will be resolved once again.  If the address couldn't be
	// resolved, cache prevents further attempts to resolve it for some time.
	ipCache cache.Cache
}

// Default rDNS values.
const (
	defaultRDNSCacheSize = 10000
	defaultRDNSCacheTTL  = 1 * 60 * 60
	defaultRDNSIPChSize  = 256
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
			MaxCount:  defaultRDNSCacheSize,
		}),
		ipCh: make(chan netip.Addr, defaultRDNSIPChSize),
	}
	if usePrivate {
		rDNS.usePrivate = 1
	}

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
	var usePrivate uint32
	if r.exchanger.ResolvesPrivatePTR() {
		usePrivate = 1
	}

	if atomic.CompareAndSwapUint32(&r.usePrivate, 1-usePrivate, usePrivate) {
		r.ipCache.Clear()
	}
}

// isCached returns true if ip is already cached and not expired yet.  It also
// caches it otherwise.
func (r *RDNS) isCached(ip netip.Addr) (ok bool) {
	ipBytes := ip.AsSlice()
	now := uint64(time.Now().Unix())
	if expire := r.ipCache.Get(ipBytes); len(expire) != 0 {
		if binary.BigEndian.Uint64(expire) > now {
			return true
		}
	}

	// The cache entry either expired or doesn't exist.
	ttl := make([]byte, 8)
	binary.BigEndian.PutUint64(ttl, now+defaultRDNSCacheTTL)
	r.ipCache.Set(ipBytes, ttl)

	return false
}

// Begin adds the ip to the resolving queue if it is not cached or already
// resolved.
func (r *RDNS) Begin(ip netip.Addr) {
	r.ensurePrivateCache()

	if r.isCached(ip) || r.clients.exists(ip, ClientSourceRDNS) {
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
		host, err := r.exchanger.Exchange(ip.AsSlice())
		if err != nil {
			log.Debug("rdns: resolving %q: %s", ip, err)

			continue
		} else if host == "" {
			continue
		}

		_ = r.clients.AddHost(ip, host, ClientSourceRDNS)
	}
}
