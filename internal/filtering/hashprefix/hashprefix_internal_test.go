package hashprefix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/dnsproxy/dnsproxytest"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/faketime"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cacheTime = 10 * time.Minute
	cacheSize = 10000
)

// testLogger is a logger used in tests.
var testLogger = slogutil.NewDiscardLogger()

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

	c := New(&Config{
		Logger:    testLogger,
		TXTSuffix: suf,
	})

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
		host:    "foo.dyndns.org",
		wantLen: 3,
	}, {
		name:    "sub_private_domain_v2",
		host:    "bar.foo.dyndns.org",
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
	const testTimeout = 1 * time.Second

	c := New(&Config{
		Logger:    testLogger,
		CacheTime: cacheTime,
	})

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
	c.storeInCache(testutil.ContextWithTimeout(t, testTimeout), hashes, hashesArray)

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

	c = New(&Config{
		Logger:    testLogger,
		CacheTime: cacheTime,
	})

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
			Logger:    testLogger,
			CacheTime: cacheTime,
			CacheSize: cacheSize,
		})

		// Prepare the upstream.
		ups := aghtest.NewBlockUpstream(hostname, tc.wantBlock)

		var numReq int
		onExchange := ups.OnExchange
		ups.OnExchange = func(ctx context.Context, req *dns.Msg) (resp *dns.Msg, err error) {
			numReq++

			return onExchange(ctx, req)
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

// newHashPrefixResp returns a TXT response that does not match hostname hashes.
func newHashPrefixResp(req *dns.Msg) (resp *dns.Msg) {
	resp = (&dns.Msg{}).SetReply(req)
	resp.Answer = []dns.RR{&dns.TXT{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Txt: []string{strings.Repeat("ab", hashSize)},
	}}

	return resp
}

func TestChecker_Check_circuitBreaker(t *testing.T) {
	const (
		hostFail  = "fail.example"
		hostSkip  = "skip.example"
		hostRetry = "retry.example"
		hostOK    = "ok.example"
		hostNext  = "next.example"
	)

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	clock := &faketime.Clock{
		OnNow: func() (cur time.Time) { return now },
	}

	var calls atomic.Int32
	ups := &dnsproxytest.Upstream{
		OnAddress: func() (addr string) { return "sb.example" },
		OnClose:   func() (err error) { return nil },
		OnExchange: func(_ context.Context, _ *dns.Msg) (resp *dns.Msg, err error) {
			calls.Add(1)

			return nil, aghtest.ErrUpstream
		},
	}

	c := New(&Config{
		Logger:    testLogger,
		Upstream:  ups,
		Clock:     clock,
		CacheTime: cacheTime,
		CacheSize: cacheSize,
		TXTSuffix: "sb.dns.adguard.com.",
	})

	blocked, err := c.Check(hostFail)
	require.ErrorIs(t, err, aghtest.ErrUpstream)
	assert.False(t, blocked)
	assert.Equal(t, int32(1), calls.Load())

	// The next lookup must not reach the upstream.  A slow upstream makes a
	// missed breaker visible as a slow call.
	ups.OnExchange = func(_ context.Context, _ *dns.Msg) (resp *dns.Msg, err error) {
		calls.Add(1)
		time.Sleep(200 * time.Millisecond)

		return nil, aghtest.ErrUpstream
	}

	start := time.Now()
	blocked, err = c.Check(hostSkip)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.False(t, blocked)
	assert.Equal(t, int32(1), calls.Load())
	assert.Less(t, elapsed, 50*time.Millisecond)

	now = now.Add(upstreamBackoff)
	ups.OnExchange = func(_ context.Context, _ *dns.Msg) (resp *dns.Msg, err error) {
		calls.Add(1)

		return nil, aghtest.ErrUpstream
	}

	blocked, err = c.Check(hostRetry)
	require.ErrorIs(t, err, aghtest.ErrUpstream)
	assert.False(t, blocked)
	assert.Equal(t, int32(2), calls.Load())

	now = now.Add(upstreamBackoff)
	ups.OnExchange = func(_ context.Context, req *dns.Msg) (resp *dns.Msg, err error) {
		calls.Add(1)

		return newHashPrefixResp(req), nil
	}

	blocked, err = c.Check(hostOK)
	require.NoError(t, err)
	assert.False(t, blocked)
	assert.Equal(t, int32(3), calls.Load())
	assert.Zero(t, c.openedUntilNanos.Load())

	// Drop cached prefixes so the next host must use the upstream if the
	// circuit was reset.
	c.cache = cache.New(cache.Config{
		EnableLRU: true,
		MaxSize:   cacheSize,
	})

	blocked, err = c.Check(hostNext)
	require.NoError(t, err)
	assert.False(t, blocked)
	assert.Equal(t, int32(4), calls.Load())
}

func TestChecker_Check_circuitBreaker_cachedBlock(t *testing.T) {
	const (
		blockedHost = "evil.example"
		otherHost   = "other.example"
	)

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	clock := &faketime.Clock{
		OnNow: func() (cur time.Time) { return now },
	}

	var calls atomic.Int32
	ups := aghtest.NewBlockUpstream(blockedHost, true)
	onExchange := ups.OnExchange
	ups.OnExchange = func(ctx context.Context, req *dns.Msg) (resp *dns.Msg, err error) {
		calls.Add(1)

		return onExchange(ctx, req)
	}

	c := New(&Config{
		Logger:    testLogger,
		Upstream:  ups,
		Clock:     clock,
		CacheTime: cacheTime,
		CacheSize: cacheSize,
		TXTSuffix: "sb.dns.adguard.com.",
	})

	blocked, err := c.Check(blockedHost)
	require.NoError(t, err)
	require.True(t, blocked)
	require.Equal(t, int32(1), calls.Load())

	ups.OnExchange = func(_ context.Context, _ *dns.Msg) (resp *dns.Msg, err error) {
		calls.Add(1)

		return nil, aghtest.ErrUpstream
	}

	_, err = c.Check(otherHost)
	require.ErrorIs(t, err, aghtest.ErrUpstream)
	require.Equal(t, int32(2), calls.Load())

	// A cached blocked host stays blocked and does not hit the upstream while
	// the circuit is open.
	blocked, err = c.Check(blockedHost)
	require.NoError(t, err)
	assert.True(t, blocked)
	assert.Equal(t, int32(2), calls.Load())
}

func TestChecker_Check_circuitBreaker_concurrent(t *testing.T) {
	const (
		goroutines = 32
		host       = "example.org"
		otherHost  = "other.example"
	)

	var calls atomic.Int32
	ups := &dnsproxytest.Upstream{
		OnAddress: func() (addr string) { return "sb.example" },
		OnClose:   func() (err error) { return nil },
		OnExchange: func(_ context.Context, _ *dns.Msg) (resp *dns.Msg, err error) {
			calls.Add(1)

			return nil, aghtest.ErrUpstream
		},
	}

	c := New(&Config{
		Logger:    testLogger,
		Upstream:  ups,
		CacheTime: cacheTime,
		CacheSize: cacheSize,
		TXTSuffix: "sb.dns.adguard.com.",
	})

	_, err := c.Check(host)
	require.ErrorIs(t, err, aghtest.ErrUpstream)
	require.Equal(t, int32(1), calls.Load())

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()

			blocked, checkErr := c.Check(otherHost)
			assert.NoError(t, checkErr)
			assert.False(t, blocked)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), calls.Load())
}
