package hashprefix

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cacheTime = 10 * time.Minute
	cacheSize = 10000
)

func TestChcker_getQuestion(t *testing.T) {
	const suf = "sb.dns.adguard.com."

	// test hostnameToHashes()
	hashes := hostnameToHashes("1.2.3.sub.host.com")
	assert.Len(t, hashes, 3)

	hash := hostnameHash(sha256.Sum256([]byte("3.sub.host.com")))
	hexPref1 := hex.EncodeToString(hash[:prefixLen])
	assert.True(t, slices.Contains(hashes, hash))

	hash = sha256.Sum256([]byte("sub.host.com"))
	hexPref2 := hex.EncodeToString(hash[:prefixLen])
	assert.True(t, slices.Contains(hashes, hash))

	hash = sha256.Sum256([]byte("host.com"))
	hexPref3 := hex.EncodeToString(hash[:prefixLen])
	assert.True(t, slices.Contains(hashes, hash))

	hash = sha256.Sum256([]byte("com"))
	assert.False(t, slices.Contains(hashes, hash))

	c := &Checker{
		svc:       "SafeBrowsing",
		txtSuffix: suf,
	}

	q := c.getQuestion(hashes)

	assert.Contains(t, q, hexPref1)
	assert.Contains(t, q, hexPref2)
	assert.Contains(t, q, hexPref3)
	assert.True(t, strings.HasSuffix(q, suf))
}

func TestHostnameToHashes(t *testing.T) {
	testCases := []struct {
		name    string
		host    string
		wantLen int
	}{{
		name:    "basic",
		host:    "example.com",
		wantLen: 1,
	}, {
		name:    "sub_basic",
		host:    "www.example.com",
		wantLen: 2,
	}, {
		name:    "private_domain",
		host:    "foo.co.uk",
		wantLen: 1,
	}, {
		name:    "sub_private_domain",
		host:    "bar.foo.co.uk",
		wantLen: 2,
	}, {
		name:    "private_domain_v2",
		host:    "foo.blogspot.co.uk",
		wantLen: 4,
	}, {
		name:    "sub_private_domain_v2",
		host:    "bar.foo.blogspot.co.uk",
		wantLen: 4,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hashes := hostnameToHashes(tc.host)
			assert.Len(t, hashes, tc.wantLen)
		})
	}
}

func TestChecker_storeInCache(t *testing.T) {
	c := &Checker{
		svc:       "SafeBrowsing",
		cacheTime: cacheTime,
	}
	conf := cache.Config{}
	c.cache = cache.New(conf)

	// store in cache hashes for "3.sub.host.com" and "host.com"
	//  and empty data for hash-prefix for "sub.host.com"
	hashes := []hostnameHash{}
	hash := hostnameHash(sha256.Sum256([]byte("sub.host.com")))
	hashes = append(hashes, hash)
	var hashesArray []hostnameHash
	hash4 := sha256.Sum256([]byte("3.sub.host.com"))
	hashesArray = append(hashesArray, hash4)
	hash2 := sha256.Sum256([]byte("host.com"))
	hashesArray = append(hashesArray, hash2)
	c.storeInCache(hashes, hashesArray)

	// match "3.sub.host.com" or "host.com" from cache
	hashes = []hostnameHash{}
	hash = sha256.Sum256([]byte("3.sub.host.com"))
	hashes = append(hashes, hash)
	hash = sha256.Sum256([]byte("sub.host.com"))
	hashes = append(hashes, hash)
	hash = sha256.Sum256([]byte("host.com"))
	hashes = append(hashes, hash)
	found, blocked, _ := c.findInCache(hashes)
	assert.True(t, found)
	assert.True(t, blocked)

	// match "sub.host.com" from cache
	hashes = []hostnameHash{}
	hash = sha256.Sum256([]byte("sub.host.com"))
	hashes = append(hashes, hash)
	found, blocked, _ = c.findInCache(hashes)
	assert.True(t, found)
	assert.False(t, blocked)

	// Match "sub.host.com" from cache.  Another hash for "host.example" is not
	// in the cache, so get data for it from the server.
	hashes = []hostnameHash{}
	hash = sha256.Sum256([]byte("sub.host.com"))
	hashes = append(hashes, hash)
	hash = sha256.Sum256([]byte("host.example"))
	hashes = append(hashes, hash)
	found, _, hashesToRequest := c.findInCache(hashes)
	assert.False(t, found)

	hash = sha256.Sum256([]byte("sub.host.com"))
	ok := slices.Contains(hashesToRequest, hash)
	assert.False(t, ok)

	hash = sha256.Sum256([]byte("host.example"))
	ok = slices.Contains(hashesToRequest, hash)
	assert.True(t, ok)

	c = &Checker{
		svc:       "SafeBrowsing",
		cacheTime: cacheTime,
	}
	c.cache = cache.New(cache.Config{})

	hashes = []hostnameHash{}
	hash = sha256.Sum256([]byte("sub.host.com"))
	hashes = append(hashes, hash)

	c.cache.Set(hash[:prefixLen], make([]byte, expirySize+hashSize))
	found, _, _ = c.findInCache(hashes)
	assert.False(t, found)
}

func TestChecker_Check(t *testing.T) {
	const hostname = "example.org"

	testCases := []struct {
		name      string
		wantBlock bool
	}{{
		name:      "sb_no_block",
		wantBlock: false,
	}, {
		name:      "sb_block",
		wantBlock: true,
	}, {
		name:      "pc_no_block",
		wantBlock: false,
	}, {
		name:      "pc_block",
		wantBlock: true,
	}}

	for _, tc := range testCases {
		c := New(&Config{
			CacheTime: cacheTime,
			CacheSize: cacheSize,
		})

		// Prepare the upstream.
		ups := aghtest.NewBlockUpstream(hostname, tc.wantBlock)

		var numReq int
		onExchange := ups.OnExchange
		ups.OnExchange = func(req *dns.Msg) (resp *dns.Msg, err error) {
			numReq++

			return onExchange(req)
		}

		c.upstream = ups

		t.Run(tc.name, func(t *testing.T) {
			// Firstly, check the request blocking.
			hits := 0
			res := false
			res, err := c.Check(hostname)
			require.NoError(t, err)

			if tc.wantBlock {
				assert.True(t, res)
				hits++
			} else {
				require.False(t, res)
			}

			// Check the cache state, check the response is now cached.
			assert.Equal(t, 1, c.cache.Stats().Count)
			assert.Equal(t, hits, c.cache.Stats().Hit)

			// There was one request to an upstream.
			assert.Equal(t, 1, numReq)

			// Now make the same request to check the cache was used.
			res, err = c.Check(hostname)
			require.NoError(t, err)

			if tc.wantBlock {
				assert.True(t, res)
			} else {
				require.False(t, res)
			}

			// Check the cache state, it should've been used.
			assert.Equal(t, 1, c.cache.Stats().Count)
			assert.Equal(t, hits+1, c.cache.Stats().Hit)

			// Check that there were no additional requests.
			assert.Equal(t, 1, numReq)
		})
	}
}
