// Package hashprefix used for safe browsing and parent control.
package hashprefix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

const (
	// prefixLen is the length of the hash prefix of the filtered hostname.
	prefixLen = 2

	// hashSize is the size of hashed hostname.
	hashSize = sha256.Size

	// hexSize is the size of hexadecimal representation of hashed hostname.
	hexSize = hashSize * 2
)

// prefix is the type of the SHA256 hash prefix used to match against the
// domain-name database.
type prefix [prefixLen]byte

// hostnameHash is the hashed hostname.
//
// TODO(s.chzhen):  Split into prefix and suffix.
type hostnameHash [hashSize]byte

// findMatch returns true if one of the a hostnames matches one of the b.
func findMatch(a, b []hostnameHash) (matched bool) {
	for _, hash := range a {
		if slices.Contains(b, hash) {
			return true
		}
	}

	return false
}

// Config is the configuration structure for safe browsing and parental
// control.
type Config struct {
	// Logger is used for logging the check process.  It must not be nil.
	Logger *slog.Logger

	// Upstream is the upstream DNS server.
	Upstream upstream.Upstream

	// TXTSuffix is the TXT suffix for DNS request.
	TXTSuffix string

	// CacheTime is the time period to store hash.
	CacheTime time.Duration

	// CacheSize is the maximum size of the cache.  If it's zero, cache size is
	// unlimited.
	CacheSize uint
}

type Checker struct {
	// logger is used for logging the check process.
	logger *slog.Logger

	// upstream is the upstream DNS server.
	upstream upstream.Upstream

	// cache stores hostname hashes.
	cache cache.Cache

	// txtSuffix is the TXT suffix for DNS request.
	txtSuffix string

	// cacheTime is the time period to store hash.
	cacheTime time.Duration
}

// New returns Checker.
func New(conf *Config) (c *Checker) {
	return &Checker{
		logger:   conf.Logger,
		upstream: conf.Upstream,
		cache: cache.New(cache.Config{
			EnableLRU: true,
			MaxSize:   conf.CacheSize,
		}),
		txtSuffix: conf.TXTSuffix,
		cacheTime: conf.CacheTime,
	}
}

// Check returns true if request for the host should be blocked.
func (c *Checker) Check(host string) (ok bool, err error) {
	ctx := context.TODO()

	hashes := hostnameToHashes(host)

	l := c.logger.With("host", host)

	found, blocked, hashesToRequest := c.findInCache(hashes)
	if found {
		l.DebugContext(ctx, "found in cache", "blocked", blocked)

		return blocked, nil
	}

	question := c.getQuestion(hashesToRequest)

	l.DebugContext(ctx, "checking", "question", question)
	req := (&dns.Msg{}).SetQuestion(question, dns.TypeTXT)

	resp, err := c.upstream.Exchange(req)
	if err != nil {
		return false, fmt.Errorf("getting hashes: %w", err)
	}

	matched, receivedHashes := c.processAnswer(ctx, l, hashesToRequest, resp)

	c.storeInCache(ctx, hashesToRequest, receivedHashes)

	return matched, nil
}

// hostnameToHashes returns hashes that should be checked by the hash prefix
// filter.
func hostnameToHashes(host string) (hashes []hostnameHash) {
	// subDomainNum defines how many labels should be hashed to match against a
	// hash prefix filter.
	const subDomainNum = 4

	pubSuf, icann := publicsuffix.PublicSuffix(host)
	if !icann {
		// Check the full private domain space.
		pubSuf = ""
	}

	nDots := 0
	i := strings.LastIndexFunc(host, func(r rune) (ok bool) {
		if r == '.' {
			nDots++
		}

		return nDots == subDomainNum
	})
	if i != -1 {
		host = host[i+1:]
	}

	// TODO(e.burkov):  Use pools and [netutil.AppendSubdomains].
	sub := netutil.Subdomains(host)

	for _, s := range sub {
		if s == pubSuf {
			break
		}

		sum := sha256.Sum256([]byte(s))
		hashes = append(hashes, sum)
	}

	return hashes
}

// getQuestion combines hexadecimal encoded prefixes of hashed hostnames into
// string.
func (c *Checker) getQuestion(hashes []hostnameHash) (q string) {
	b := &strings.Builder{}

	for _, hash := range hashes {
		stringutil.WriteToBuilder(b, hex.EncodeToString(hash[:prefixLen]), ".")
	}

	stringutil.WriteToBuilder(b, c.txtSuffix)

	return b.String()
}

// processAnswer returns true if DNS response matches the hash, and received
// hashed hostnames from the upstream.  l must not be nil.
func (c *Checker) processAnswer(
	ctx context.Context,
	l *slog.Logger,
	hashesToRequest []hostnameHash,
	resp *dns.Msg,
) (matched bool, receivedHashes []hostnameHash) {
	txtCount := 0

	for _, a := range resp.Answer {
		txt, ok := a.(*dns.TXT)
		if !ok {
			continue
		}

		txtCount++

		receivedHashes = c.appendHashesFromTXT(ctx, l, receivedHashes, txt)
	}

	l.DebugContext(ctx, "processing answer with TXT", "txt_count", txtCount)

	matched = findMatch(hashesToRequest, receivedHashes)
	if matched {
		l.DebugContext(ctx, "matched")

		return true, receivedHashes
	}

	return false, receivedHashes
}

// appendHashesFromTXT appends received hashed hostnames.  l must not be nil.
func (c *Checker) appendHashesFromTXT(
	ctx context.Context,
	l *slog.Logger,
	hashes []hostnameHash,
	txt *dns.TXT,
) (receivedHashes []hostnameHash) {
	l.DebugContext(ctx, "received hashes", "txt", txt.Txt)

	for _, t := range txt.Txt {
		if len(t) != hexSize {
			l.DebugContext(ctx, "wrong hex size", "len", len(t), "txt", t)

			continue
		}

		buf, err := hex.DecodeString(t)
		if err != nil {
			l.DebugContext(ctx, "decoding hex string", "txt", t, slogutil.KeyError, err)

			continue
		}

		var hash hostnameHash
		copy(hash[:], buf)
		hashes = append(hashes, hash)
	}

	return hashes
}
