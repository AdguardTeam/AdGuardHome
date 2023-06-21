// Package whois provides WHOIS functionality.
package whois

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/bluele/gcache"
)

const (
	// DefaultServer is the default WHOIS server.
	DefaultServer = "whois.arin.net"

	// DefaultPort is the default port for WHOIS requests.
	DefaultPort = 43
)

// Interface provides WHOIS functionality.
type Interface interface {
	// Process makes WHOIS request and returns WHOIS information or nil.
	// changed indicates that Info was updated since last request.
	Process(ctx context.Context, ip netip.Addr) (info *Info, changed bool)
}

// Empty is an empty [Interface] implementation which does nothing.
type Empty struct{}

// type check
var _ Interface = (*Empty)(nil)

// Process implements the [Interface] interface for Empty.
func (Empty) Process(_ context.Context, _ netip.Addr) (info *Info, changed bool) {
	return nil, false
}

// Config is the configuration structure for Default.
type Config struct {
	// DialContext specifies the dial function for creating unencrypted TCP
	// connections.
	DialContext func(ctx context.Context, network, addr string) (conn net.Conn, err error)

	// ServerAddr is the address of the WHOIS server.
	ServerAddr string

	// Timeout is the timeout for WHOIS requests.
	Timeout time.Duration

	// CacheTTL is the Time to Live duration for cached IP addresses.
	CacheTTL time.Duration

	// MaxConnReadSize is an upper limit in bytes for reading from net.Conn.
	MaxConnReadSize int64

	// MaxRedirects is the maximum redirects count.
	MaxRedirects int

	// MaxInfoLen is the maximum length of Info fields returned by Process.
	MaxInfoLen int

	// CacheSize is the maximum size of the cache.  It must be greater than
	// zero.
	CacheSize int

	// Port is the port for WHOIS requests.
	Port uint16
}

// Default is the default WHOIS information processor.
type Default struct {
	// cache is the cache containing IP addresses of clients.  An active IP
	// address is resolved once again after it expires.  If IP address couldn't
	// be resolved, it stays here for some time to prevent further attempts to
	// resolve the same IP.
	cache gcache.Cache

	// dialContext connects to a remote server resolving hostname using our own
	// DNS server and unecrypted TCP connection.
	dialContext func(ctx context.Context, network, addr string) (conn net.Conn, err error)

	// serverAddr is the address of the WHOIS server.
	serverAddr string

	// portStr is the port for WHOIS requests.
	portStr string

	// timeout is the timeout for WHOIS requests.
	timeout time.Duration

	// cacheTTL is the Time to Live duration for cached IP addresses.
	cacheTTL time.Duration

	// maxConnReadSize is an upper limit in bytes for reading from net.Conn.
	maxConnReadSize int64

	// maxRedirects is the maximum redirects count.
	maxRedirects int

	// maxInfoLen is the maximum length of Info fields returned by Process.
	maxInfoLen int
}

// New returns a new default WHOIS information processor.  conf must not be
// nil.
func New(conf *Config) (w *Default) {
	return &Default{
		serverAddr:      conf.ServerAddr,
		dialContext:     conf.DialContext,
		timeout:         conf.Timeout,
		cache:           gcache.New(conf.CacheSize).LRU().Build(),
		maxConnReadSize: conf.MaxConnReadSize,
		maxRedirects:    conf.MaxRedirects,
		portStr:         strconv.Itoa(int(conf.Port)),
		maxInfoLen:      conf.MaxInfoLen,
		cacheTTL:        conf.CacheTTL,
	}
}

// trimValue trims s and replaces the last 3 characters of the cut with "..."
// to fit into max.  max must be greater than 3.
func trimValue(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max-3] + "..."
}

// isWHOISComment returns true if the data is empty or is a WHOIS comment.
func isWHOISComment(data []byte) (ok bool) {
	return len(data) == 0 || data[0] == '#' || data[0] == '%'
}

// whoisParse parses a subset of plain-text data from the WHOIS response into a
// string map.  It trims values of the returned map to maxLen.
func whoisParse(data []byte, maxLen int) (info map[string]string) {
	info = map[string]string{}

	var orgname string
	lines := bytes.Split(data, []byte("\n"))
	for _, l := range lines {
		if isWHOISComment(l) {
			continue
		}

		before, after, found := bytes.Cut(l, []byte(":"))
		if !found {
			continue
		}

		key := strings.ToLower(string(before))
		val := strings.TrimSpace(string(after))
		if val == "" {
			continue
		}

		switch key {
		case "orgname", "org-name":
			key = "orgname"
			val = trimValue(val, maxLen)
			orgname = val
		case "city", "country":
			val = trimValue(val, maxLen)
		case "descr", "netname":
			key = "orgname"
			val = stringutil.Coalesce(orgname, val)
			orgname = val
		case "whois":
			key = "whois"
		case "referralserver":
			key = "whois"
			val = strings.TrimPrefix(val, "whois://")
		default:
			continue
		}

		info[key] = val
	}

	return info
}

// query sends request to a server and returns the response or error.
func (w *Default) query(ctx context.Context, target, serverAddr string) (data []byte, err error) {
	addr, _, _ := net.SplitHostPort(serverAddr)
	if addr == DefaultServer {
		// Display type flags for query.
		//
		// See https://www.arin.net/resources/registry/whois/rws/api/#nicname-whois-queries.
		target = "n + " + target
	}

	conn, err := w.dialContext(ctx, "tcp", serverAddr)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}
	defer func() { err = errors.WithDeferred(err, conn.Close()) }()

	r, err := aghio.LimitReader(conn, w.maxConnReadSize)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(w.timeout))
	_, err = io.WriteString(conn, target+"\r\n")
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	// This use of ReadAll is now safe, because we limited the conn Reader.
	data, err = io.ReadAll(r)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	return data, nil
}

// queryAll queries WHOIS server and handles redirects.
func (w *Default) queryAll(ctx context.Context, target string) (info map[string]string, err error) {
	server := net.JoinHostPort(w.serverAddr, w.portStr)
	var data []byte

	for i := 0; i < w.maxRedirects; i++ {
		data, err = w.query(ctx, target, server)
		if err != nil {
			// Don't wrap the error since it's informative enough as is.
			return nil, err
		}

		log.Debug("whois: received response (%d bytes) from %q about %q", len(data), server, target)

		info = whoisParse(data, w.maxInfoLen)
		redir, ok := info["whois"]
		if !ok {
			return info, nil
		}

		redir = strings.ToLower(redir)

		_, _, err = net.SplitHostPort(redir)
		if err != nil {
			server = net.JoinHostPort(redir, w.portStr)
		} else {
			server = redir
		}

		log.Debug("whois: redirected to %q about %q", redir, target)
	}

	return nil, fmt.Errorf("whois: redirect loop")
}

// type check
var _ Interface = (*Default)(nil)

// Process makes WHOIS request and returns WHOIS information or nil.  changed
// indicates that Info was updated since last request.
func (w *Default) Process(ctx context.Context, ip netip.Addr) (wi *Info, changed bool) {
	if netutil.IsSpecialPurposeAddr(ip) {
		return nil, false
	}

	wi, expired := w.findInCache(ip)
	if wi != nil && !expired {
		// Don't return an empty struct so that the frontend doesn't get
		// confused.
		if (*wi == Info{}) {
			return nil, false
		}

		return wi, false
	}

	var info Info

	defer func() {
		item := toCacheItem(info, w.cacheTTL)
		err := w.cache.Set(ip, item)
		if err != nil {
			log.Debug("whois: cache: adding item %q: %s", ip, err)
		}
	}()

	kv, err := w.queryAll(ctx, ip.String())
	if err != nil {
		log.Debug("whois: quering about %q: %s", ip, err)

		return nil, true
	}

	info = Info{
		City:    kv["city"],
		Country: kv["country"],
		Orgname: kv["orgname"],
	}

	// Don't return an empty struct so that the frontend doesn't get confused.
	if (info == Info{}) {
		return nil, true
	}

	return &info, wi == nil || info != *wi
}

// findInCache finds Info in the cache.  expired indicates that Info is valid.
func (w *Default) findInCache(ip netip.Addr) (wi *Info, expired bool) {
	val, err := w.cache.Get(ip)
	if err != nil {
		if !errors.Is(err, gcache.KeyNotFoundError) {
			log.Debug("whois: cache: retrieving info about %q: %s", ip, err)
		}

		return nil, false
	}

	item, ok := val.(*cacheItem)
	if !ok {
		log.Debug("whois: cache: %q bad type %T", ip, val)

		return nil, false
	}

	return fromCacheItem(item)
}

// Info is the filtered WHOIS data for a runtime client.
type Info struct {
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
	Orgname string `json:"orgname,omitempty"`
}

// cacheItem represents an item that we will store in the cache.
type cacheItem struct {
	// expiry is the time when cacheItem will expire.
	expiry time.Time

	// info is the WHOIS data for a runtime client.
	info *Info
}

// toCacheItem creates a cached item from a WHOIS info and Time to Live
// duration.
func toCacheItem(info Info, ttl time.Duration) (item *cacheItem) {
	return &cacheItem{
		expiry: time.Now().Add(ttl),
		info:   &info,
	}
}

// fromCacheItem creates a WHOIS info from the cached item.  expired indicates
// that WHOIS info is valid.  item must not be nil.
func fromCacheItem(item *cacheItem) (info *Info, expired bool) {
	if time.Now().After(item.expiry) {
		return item.info, true
	}

	return item.info, false
}
