package dnsfilter

import (
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSBPC_checkErrorUpstream(t *testing.T) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)

	ups := &aghtest.TestErrUpstream{}

	d.SetSafeBrowsingUpstream(ups)
	d.SetParentalUpstream(ups)

	setts := &FilteringSettings{
		SafeBrowsingEnabled: true,
		ParentalEnabled:     true,
	}

	_, err := d.checkSafeBrowsing("smthng.com", dns.TypeA, setts)
	assert.Error(t, err)

	_, err = d.checkParental("smthng.com", dns.TypeA, setts)
	assert.Error(t, err)
}

func TestSBPC(t *testing.T) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)

	const hostname = "example.org"

	setts := &FilteringSettings{
		SafeBrowsingEnabled: true,
		ParentalEnabled:     true,
	}

	testCases := []struct {
		name      string
		block     bool
		testFunc  func(host string, _ uint16, _ *FilteringSettings) (res Result, err error)
		testCache cache.Cache
	}{{
		name:      "sb_no_block",
		block:     false,
		testFunc:  d.checkSafeBrowsing,
		testCache: gctx.safebrowsingCache,
	}, {
		name:      "sb_block",
		block:     true,
		testFunc:  d.checkSafeBrowsing,
		testCache: gctx.safebrowsingCache,
	}, {
		name:      "pc_no_block",
		block:     false,
		testFunc:  d.checkParental,
		testCache: gctx.parentalCache,
	}, {
		name:      "pc_block",
		block:     true,
		testFunc:  d.checkParental,
		testCache: gctx.parentalCache,
	}}

	for _, tc := range testCases {
		// Prepare the upstream.
		ups := &aghtest.TestBlockUpstream{
			Hostname: hostname,
			Block:    tc.block,
		}
		d.SetSafeBrowsingUpstream(ups)
		d.SetParentalUpstream(ups)

		t.Run(tc.name, func(t *testing.T) {
			// Firstly, check the request blocking.
			hits := 0
			res, err := tc.testFunc(hostname, dns.TypeA, setts)
			require.NoError(t, err)

			if tc.block {
				assert.True(t, res.IsFiltered)
				require.Len(t, res.Rules, 1)
				hits++
			} else {
				require.False(t, res.IsFiltered)
			}

			// Check the cache state, check the response is now cached.
			assert.Equal(t, 1, tc.testCache.Stats().Count)
			assert.Equal(t, hits, tc.testCache.Stats().Hit)

			// There was one request to an upstream.
			assert.Equal(t, 1, ups.RequestsCount())

			// Now make the same request to check the cache was used.
			res, err = tc.testFunc(hostname, dns.TypeA, setts)
			require.NoError(t, err)

			if tc.block {
				assert.True(t, res.IsFiltered)
				require.Len(t, res.Rules, 1)
			} else {
				require.False(t, res.IsFiltered)
			}

			// Check the cache state, it should've been used.
			assert.Equal(t, 1, tc.testCache.Stats().Count)
			assert.Equal(t, hits+1, tc.testCache.Stats().Hit)

			// Check that there were no additional requests.
			assert.Equal(t, 1, ups.RequestsCount())
		})

		purgeCaches()
	}
}
