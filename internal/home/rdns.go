package home

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
)

// RDNS resolves clients' addresses to enrich their metadata.
type RDNS struct {
	exchanger dnsforward.RDNSExchanger
	clients   *clientsContainer

	// ipCh used to pass client's IP to rDNS workerLoop.
	ipCh chan net.IP

	// ipCache caches the IP addresses to be resolved by rDNS.  The resolved
	// address stays here while it's inside clients.  After leaving clients
	// the address will be resolved once again.  If the address couldn't be
	// resolved, cache prevents further attempts to resolve it for some
	// time.
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
) (rDNS *RDNS) {
	rDNS = &RDNS{
		exchanger: exchanger,
		clients:   clients,
		ipCache: cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  defaultRDNSCacheSize,
		}),
		ipCh: make(chan net.IP, defaultRDNSIPChSize),
	}

	go rDNS.workerLoop()

	return rDNS
}

// Begin adds the ip to the resolving queue if it is not cached or already
// resolved.
func (r *RDNS) Begin(ip net.IP) {
	now := uint64(time.Now().Unix())
	if expire := r.ipCache.Get(ip); len(expire) != 0 {
		if binary.BigEndian.Uint64(expire) > now {
			return
		}
	}

	// The cache entry either expired or doesn't exist.
	ttl := make([]byte, 8)
	binary.BigEndian.PutUint64(ttl, now+defaultRDNSCacheTTL)
	r.ipCache.Set(ip, ttl)

	id := ip.String()
	if r.clients.Exists(id, ClientSourceRDNS) {
		return
	}

	select {
	case r.ipCh <- ip:
		log.Tracef("rdns: %q added to queue", ip)
	default:
		log.Tracef("rdns: queue is full")
	}
}

// workerLoop handles incoming IP addresses from ipChan and adds it into
// clients.
func (r *RDNS) workerLoop() {
	defer agherr.LogPanic("rdns")

	for ip := range r.ipCh {
		host, err := r.exchanger.Exchange(ip)
		if err != nil {
			log.Error("rdns: resolving %q: %s", ip, err)

			continue
		}

		if host == "" {
			continue
		}

		// Don't handle any errors since AddHost doesn't return non-nil
		// errors for now.
		_, _ = r.clients.AddHost(ip.String(), host, ClientSourceRDNS)
	}
}
