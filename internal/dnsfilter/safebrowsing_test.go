package dnsfilter

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestSafeBrowsingHash(t *testing.T) {
	// test hostnameToHashes()
	hashes := hostnameToHashes("1.2.3.sub.host.com")
	assert.Len(t, hashes, 3)
	_, ok := hashes[sha256.Sum256([]byte("3.sub.host.com"))]
	assert.True(t, ok)
	_, ok = hashes[sha256.Sum256([]byte("sub.host.com"))]
	assert.True(t, ok)
	_, ok = hashes[sha256.Sum256([]byte("host.com"))]
	assert.True(t, ok)
	_, ok = hashes[sha256.Sum256([]byte("com"))]
	assert.False(t, ok)

	c := &sbCtx{
		svc:        "SafeBrowsing",
		hashToHost: hashes,
	}

	q := c.getQuestion()

	assert.Contains(t, q, "7a1b.")
	assert.Contains(t, q, "af5a.")
	assert.Contains(t, q, "eb11.")
	assert.True(t, strings.HasSuffix(q, "sb.dns.adguard.com."))
}

func TestSafeBrowsingCache(t *testing.T) {
	c := &sbCtx{
		svc:       "SafeBrowsing",
		cacheTime: 100,
	}
	conf := cache.Config{}
	c.cache = cache.New(conf)

	// store in cache hashes for "3.sub.host.com" and "host.com"
	//  and empty data for hash-prefix for "sub.host.com"
	hash := sha256.Sum256([]byte("sub.host.com"))
	c.hashToHost = make(map[[32]byte]string)
	c.hashToHost[hash] = "sub.host.com"
	var hashesArray [][]byte
	hash4 := sha256.Sum256([]byte("3.sub.host.com"))
	hashesArray = append(hashesArray, hash4[:])
	hash2 := sha256.Sum256([]byte("host.com"))
	hashesArray = append(hashesArray, hash2[:])
	c.storeCache(hashesArray)

	// match "3.sub.host.com" or "host.com" from cache
	c.hashToHost = make(map[[32]byte]string)
	hash = sha256.Sum256([]byte("3.sub.host.com"))
	c.hashToHost[hash] = "3.sub.host.com"
	hash = sha256.Sum256([]byte("sub.host.com"))
	c.hashToHost[hash] = "sub.host.com"
	hash = sha256.Sum256([]byte("host.com"))
	c.hashToHost[hash] = "host.com"
	assert.Equal(t, 1, c.getCached())

	// match "sub.host.com" from cache
	c.hashToHost = make(map[[32]byte]string)
	hash = sha256.Sum256([]byte("sub.host.com"))
	c.hashToHost[hash] = "sub.host.com"
	assert.Equal(t, -1, c.getCached())

	// match "sub.host.com" from cache,
	//  but another hash for "nonexisting.com" is not in cache
	//  which means that we must get data from server for it
	c.hashToHost = make(map[[32]byte]string)
	hash = sha256.Sum256([]byte("sub.host.com"))
	c.hashToHost[hash] = "sub.host.com"
	hash = sha256.Sum256([]byte("nonexisting.com"))
	c.hashToHost[hash] = "nonexisting.com"
	assert.Empty(t, c.getCached())

	hash = sha256.Sum256([]byte("sub.host.com"))
	_, ok := c.hashToHost[hash]
	assert.False(t, ok)

	hash = sha256.Sum256([]byte("nonexisting.com"))
	_, ok = c.hashToHost[hash]
	assert.True(t, ok)

	c = &sbCtx{
		svc:       "SafeBrowsing",
		cacheTime: 100,
	}
	conf = cache.Config{}
	c.cache = cache.New(conf)

	hash = sha256.Sum256([]byte("sub.host.com"))
	c.hashToHost = make(map[[32]byte]string)
	c.hashToHost[hash] = "sub.host.com"

	c.cache.Set(hash[0:2], make([]byte, 32))
	assert.Empty(t, c.getCached())
}

// testErrUpstream implements upstream.Upstream interface for replacing real
// upstream in tests.
type testErrUpstream struct{}

// Exchange always returns nil Msg and non-nil error.
func (teu *testErrUpstream) Exchange(*dns.Msg) (*dns.Msg, error) {
	return nil, agherr.Error("bad")
}

func (teu *testErrUpstream) Address() string {
	return ""
}

func TestSBPC_checkErrorUpstream(t *testing.T) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)

	ups := &testErrUpstream{}

	d.safeBrowsingUpstream = ups
	d.parentalUpstream = ups

	_, err := d.checkSafeBrowsing("smthng.com")
	assert.NotNil(t, err)

	_, err = d.checkParental("smthng.com")
	assert.NotNil(t, err)
}

// testSbUpstream implements upstream.Upstream interface for replacing real
// upstream in tests.
type testSbUpstream struct {
	hostname      string
	block         bool
	requestsCount int
	counterLock   sync.RWMutex
}

// Exchange returns a message depending on the upstream settings (hostname, block)
func (u *testSbUpstream) Exchange(r *dns.Msg) (*dns.Msg, error) {
	u.counterLock.Lock()
	u.requestsCount++
	u.counterLock.Unlock()

	hash := sha256.Sum256([]byte(u.hostname))
	prefix := hash[0:2]
	hashToReturn := hex.EncodeToString(prefix) + strings.Repeat("ab", 28)
	if u.block {
		hashToReturn = hex.EncodeToString(hash[:])
	}

	m := &dns.Msg{}
	m.Answer = []dns.RR{
		&dns.TXT{
			Hdr: dns.RR_Header{
				Name: r.Question[0].Name,
			},
			Txt: []string{
				hashToReturn,
			},
		},
	}

	return m, nil
}

func (u *testSbUpstream) Address() string {
	return ""
}

func TestSBPC_sbValidResponse(t *testing.T) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)

	ups := &testSbUpstream{}
	d.safeBrowsingUpstream = ups
	d.parentalUpstream = ups

	// Prepare the upstream
	ups.hostname = "example.org"
	ups.block = false
	ups.requestsCount = 0

	// First - check that the request is not blocked
	res, err := d.checkSafeBrowsing("example.org")
	assert.Nil(t, err)
	assert.False(t, res.IsFiltered)

	// Check the cache state, check that the response is now cached
	assert.Equal(t, 1, gctx.safebrowsingCache.Stats().Count)
	assert.Equal(t, 0, gctx.safebrowsingCache.Stats().Hit)

	// There was one request to an upstream
	assert.Equal(t, 1, ups.requestsCount)

	// Now make the same request to check that the cache was used
	res, err = d.checkSafeBrowsing("example.org")
	assert.Nil(t, err)
	assert.False(t, res.IsFiltered)

	// Check the cache state, it should've been used
	assert.Equal(t, 1, gctx.safebrowsingCache.Stats().Count)
	assert.Equal(t, 1, gctx.safebrowsingCache.Stats().Hit)

	// Check that there were no additional requests
	assert.Equal(t, 1, ups.requestsCount)
}

func TestSBPC_pcBlockedResponse(t *testing.T) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)

	ups := &testSbUpstream{}
	d.safeBrowsingUpstream = ups
	d.parentalUpstream = ups

	// Prepare the upstream
	// Make sure that the upstream will return a response that matches the queried domain
	ups.hostname = "example.com"
	ups.block = true
	ups.requestsCount = 0

	// Make a lookup
	res, err := d.checkParental("example.com")
	assert.Nil(t, err)
	assert.True(t, res.IsFiltered)
	assert.Len(t, res.Rules, 1)

	// Check the cache state, check that the response is now cached
	assert.Equal(t, 1, gctx.parentalCache.Stats().Count)
	assert.Equal(t, 1, gctx.parentalCache.Stats().Hit)

	// There was one request to an upstream
	assert.Equal(t, 1, ups.requestsCount)

	// Make a second lookup for the same domain
	res, err = d.checkParental("example.com")
	assert.Nil(t, err)
	assert.True(t, res.IsFiltered)
	assert.Len(t, res.Rules, 1)

	// Check the cache state, it should've been used
	assert.Equal(t, 1, gctx.parentalCache.Stats().Count)
	assert.Equal(t, 2, gctx.parentalCache.Stats().Hit)

	// Check that there were no additional requests
	assert.Equal(t, 1, ups.requestsCount)
}
