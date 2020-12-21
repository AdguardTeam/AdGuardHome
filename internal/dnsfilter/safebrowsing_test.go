package dnsfilter

import (
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestSafeBrowsingHash(t *testing.T) {
	// test hostnameToHashes()
	hashes := hostnameToHashes("1.2.3.sub.host.com")
	assert.Equal(t, 3, len(hashes))
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

	assert.True(t, strings.Contains(q, "7a1b."))
	assert.True(t, strings.Contains(q, "af5a."))
	assert.True(t, strings.Contains(q, "eb11."))
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
	assert.Equal(t, 0, c.getCached())

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
	assert.Equal(t, 0, c.getCached())
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
	d := NewForTest(&Config{SafeBrowsingEnabled: true}, nil)
	defer d.Close()

	ups := &testErrUpstream{}

	d.safeBrowsingUpstream = ups
	d.parentalUpstream = ups

	_, err := d.checkSafeBrowsing("smthng.com")
	assert.NotNil(t, err)

	_, err = d.checkParental("smthng.com")
	assert.NotNil(t, err)
}
