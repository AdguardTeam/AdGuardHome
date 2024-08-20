// Package hashprefix used for safe browsing and parent control.
package hashprefix

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
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
	// Upstream is the upstream DNS server.
	Upstream upstream.Upstream

	// ServiceName is the name of the service.
	ServiceName string

	// TXTSuffix is the TXT suffix for DNS request.
	TXTSuffix string

	// CacheTime is the time period to store hash.
	CacheTime time.Duration

	// CacheSize is the maximum size of the cache.  If it's zero, cache size is
	// unlimited.
	CacheSize uint
}

type Checker struct {
	// upstream is the upstream DNS server.
	upstream upstream.Upstream

	// cache stores hostname hashes.
	cache cache.Cache

	// svc is the name of the service.
	svc string

	// txtSuffix is the TXT suffix for DNS request.
	txtSuffix string

	// cacheTime is the time period to store hash.
	cacheTime time.Duration
}

// New returns Checker.
func New(conf *Config) (c *Checker) {
	return &Checker{
		upstream: conf.Upstream,
		cache: cache.New(cache.Config{
			EnableLRU: true,
			MaxSize:   conf.CacheSize,
		}),
		svc:       conf.ServiceName,
		txtSuffix: conf.TXTSuffix,
		cacheTime: conf.CacheTime,
	}
}

// Check returns true if request for the host should be blocked.
func (c *Checker) Check(host string) (ok bool, err error) {
	hashes := hostnameToHashes(host)

	found, blocked, hashesToRequest := c.findInCache(hashes)
	if found {
		log.Debug("%s: found %q in cache, blocked: %t", c.svc, host, blocked)

		return blocked, nil
	}

	question := c.getQuestion(hashesToRequest)

	log.Debug("%s: checking %s: %s", c.svc, host, question)
	req := (&dns.Msg{}).SetQuestion(question, dns.TypeTXT)

	resp, err := c.upstream.Exchange(req)
	if err != nil {
		return false, fmt.Errorf("getting hashes: %w", err)
	}

	matched, receivedHashes := c.processAnswer(hashesToRequest, resp, host)

	c.storeInCache(hashesToRequest, receivedHashes)

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
// hashed hostnames from the upstream.
func (c *Checker) processAnswer(
	hashesToRequest []hostnameHash,
	resp *dns.Msg,
	host string,
) (matched bool, receivedHashes []hostnameHash) {
	txtCount := 0

	for _, a := range resp.Answer {
		txt, ok := a.(*dns.TXT)
		if !ok {
			continue
		}

		txtCount++

		receivedHashes = c.appendHashesFromTXT(receivedHashes, txt, host)
	}

	log.Debug("%s: received answer for %s with %d TXT count", c.svc, host, txtCount)

	matched = findMatch(hashesToRequest, receivedHashes)
	if matched {
		log.Debug("%s: matched %s", c.svc, host)

		return true, receivedHashes
	}

	return false, receivedHashes
}

// appendHashesFromTXT appends received hashed hostnames.
func (c *Checker) appendHashesFromTXT(
	hashes []hostnameHash,
	txt *dns.TXT,
	host string,
) (receivedHashes []hostnameHash) {
	log.Debug("%s: received hashes for %s: %v", c.svc, host, txt.Txt)

	for _, t := range txt.Txt {
		if len(t) != hexSize {
			log.Debug("%s: wrong hex size %d for %s %s", c.svc, len(t), host, t)

			continue
		}

		buf, err := hex.DecodeString(t)
		if err != nil {
			log.Debug("%s: decoding hex string %s: %s", c.svc, t, err)

			continue
		}

		var hash hostnameHash
		copy(hash[:], buf)
		hashes = append(hashes, hash)
	}

	return hashes
}
