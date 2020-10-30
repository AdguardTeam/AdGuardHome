package dnsfilter

import (
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/cache"
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
		svc: "SafeBrowsing",
	}

	// test getQuestion()
	c.hashToHost = hashes
	q := c.getQuestion()
	assert.True(t, strings.Index(q, "7a1b.") >= 0)
	assert.True(t, strings.Index(q, "af5a.") >= 0)
	assert.True(t, strings.Index(q, "eb11.") >= 0)
	assert.True(t, strings.Index(q, "sb.dns.adguard.com.") > 0)
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
}
