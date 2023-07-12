// Package rdns processes reverse DNS lookup queries.
package rdns

import (
	"net/netip"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/bluele/gcache"
)

// Interface processes rDNS queries.
type Interface interface {
	// Process makes rDNS request and returns domain name.  changed indicates
	// that domain name was updated since last request.
	Process(ip netip.Addr) (host string, changed bool)
}

// Empty is an empty [Inteface] implementation which does nothing.
type Empty struct{}

// type check
var _ Interface = (*Empty)(nil)

// Process implements the [Interface] interface for Empty.
func (Empty) Process(_ netip.Addr) (host string, changed bool) {
	return "", false
}

// Exchanger is a resolver for clients' addresses.
type Exchanger interface {
	// Exchange tries to resolve the ip in a suitable way, i.e. either as local
	// or as external.
	Exchange(ip netip.Addr) (host string, err error)
}

// Config is the configuration structure for Default.
type Config struct {
	// Exchanger resolves IP addresses to domain names.
	Exchanger Exchanger

	// CacheSize is the maximum size of the cache.  It must be greater than
	// zero.
	CacheSize int

	// CacheTTL is the Time to Live duration for cached IP addresses.
	CacheTTL time.Duration
}

// Default is the default rDNS query processor.
type Default struct {
	// cache is the cache containing IP addresses of clients.  An active IP
	// address is resolved once again after it expires.  If IP address couldn't
	// be resolved, it stays here for some time to prevent further attempts to
	// resolve the same IP.
	cache gcache.Cache

	// exchanger resolves IP addresses to domain names.
	exchanger Exchanger

	// cacheTTL is the Time to Live duration for cached IP addresses.
	cacheTTL time.Duration
}

// New returns a new default rDNS query processor.  conf must not be nil.
func New(conf *Config) (r *Default) {
	return &Default{
		cache:     gcache.New(conf.CacheSize).LRU().Build(),
		exchanger: conf.Exchanger,
		cacheTTL:  conf.CacheTTL,
	}
}

// type check
var _ Interface = (*Default)(nil)

// Process implements the [Interface] interface for Default.
func (r *Default) Process(ip netip.Addr) (host string, changed bool) {
	fromCache, expired := r.findInCache(ip)
	if !expired {
		return fromCache, false
	}

	host, err := r.exchanger.Exchange(ip)
	if err != nil {
		log.Debug("rdns: resolving %q: %s", ip, err)
	}

	item := &cacheItem{
		expiry: time.Now().Add(r.cacheTTL),
		host:   host,
	}

	err = r.cache.Set(ip, item)
	if err != nil {
		log.Debug("rdns: cache: adding item %q: %s", ip, err)
	}

	return host, fromCache == "" || host != fromCache
}

// findInCache finds domain name in the cache.  expired is true if host is not
// valid anymore.
func (r *Default) findInCache(ip netip.Addr) (host string, expired bool) {
	val, err := r.cache.Get(ip)
	if err != nil {
		if !errors.Is(err, gcache.KeyNotFoundError) {
			log.Debug("rdns: cache: retrieving %q: %s", ip, err)
		}

		return "", true
	}

	item, ok := val.(*cacheItem)
	if !ok {
		log.Debug("rdns: cache: %q bad type %T", ip, val)

		return "", true
	}

	return item.host, time.Now().After(item.expiry)
}

// cacheItem represents an item that we will store in the cache.
type cacheItem struct {
	// expiry is the time when cacheItem will expire.
	expiry time.Time

	// host is the domain name of a runtime client.
	host string
}
