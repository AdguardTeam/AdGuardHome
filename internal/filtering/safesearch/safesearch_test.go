package safesearch_test

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests and contexts.
const testTimeout = 1 * time.Second

// Common test constants.
const (
	// TODO(a.garipov): Add IPv6 tests.
	testQType     = dns.TypeA
	testCacheSize = 5000
	testCacheTTL  = 30 * time.Minute
)

// testConf is the default safe search configuration for tests.
var testConf = filtering.SafeSearchConfig{
	Enabled: true,

	Bing:       true,
	DuckDuckGo: true,
	Ecosia:     true,
	Google:     true,
	Pixabay:    true,
	Yandex:     true,
	YouTube:    true,
}

// yandexIP is the expected IP address of Yandex safe search results.  Keep in
// sync with the rules data.
var yandexIP = netip.AddrFrom4([4]byte{213, 180, 193, 56})

func TestDefault_CheckHost_yandex(t *testing.T) {
	conf := testConf
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	ss, err := safesearch.NewDefault(ctx, &safesearch.DefaultConfig{
		Logger:         slogutil.NewDiscardLogger(),
		ServicesConfig: conf,
		CacheSize:      testCacheSize,
		CacheTTL:       testCacheTTL,
	})
	require.NoError(t, err)

	hosts := []string{
		"yandex.ru",
		"yAndeX.ru",
		"YANdex.COM",
		"yandex.by",
		"yandex.kz",
		"www.yandex.com",
	}

	testCases := []struct {
		want netip.Addr
		name string
		qt   uint16
	}{{
		want: yandexIP,
		name: "a",
		qt:   dns.TypeA,
	}, {
		want: netip.Addr{},
		name: "aaaa",
		qt:   dns.TypeAAAA,
	}, {
		want: netip.Addr{},
		name: "https",
		qt:   dns.TypeHTTPS,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, host := range hosts {
				// Check host for each domain.
				var res filtering.Result
				res, err = ss.CheckHost(ctx, host, tc.qt)
				require.NoError(t, err)

				assert.True(t, res.IsFiltered)
				assert.Equal(t, filtering.FilteredSafeSearch, res.Reason)

				if tc.want == (netip.Addr{}) {
					assert.Empty(t, res.Rules)
				} else {
					require.Len(t, res.Rules, 1)

					rule := res.Rules[0]
					assert.Equal(t, tc.want, rule.IP)
					assert.Equal(t, rulelist.URLFilterIDSafeSearch, rule.FilterListID)
				}
			}
		})
	}
}

func TestDefault_CheckHost_google(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	ss, err := safesearch.NewDefault(ctx, &safesearch.DefaultConfig{
		Logger:         slogutil.NewDiscardLogger(),
		ServicesConfig: testConf,
		CacheSize:      testCacheSize,
		CacheTTL:       testCacheTTL,
	})
	require.NoError(t, err)

	// Check host for each domain.
	for _, host := range []string{
		"www.google.com",
		"www.google.im",
		"www.google.co.in",
		"www.google.iq",
		"www.google.is",
		"www.google.it",
		"www.google.je",
	} {
		t.Run(host, func(t *testing.T) {
			var res filtering.Result
			res, err = ss.CheckHost(ctx, host, testQType)
			require.NoError(t, err)

			assert.True(t, res.IsFiltered)
			assert.Equal(t, filtering.FilteredSafeSearch, res.Reason)
			assert.Equal(t, "forcesafesearch.google.com", res.CanonName)
			assert.Empty(t, res.Rules)
		})
	}
}

// testResolver is a [filtering.Resolver] for tests.
//
// TODO(a.garipov): Move to aghtest and use everywhere.
type testResolver struct {
	OnLookupIP func(ctx context.Context, network, host string) (ips []net.IP, err error)
}

// type check
var _ filtering.Resolver = (*testResolver)(nil)

// LookupIP implements the [filtering.Resolver] interface for *testResolver.
func (r *testResolver) LookupIP(
	ctx context.Context,
	network string,
	host string,
) (ips []net.IP, err error) {
	return r.OnLookupIP(ctx, network, host)
}

func TestDefault_CheckHost_duckduckgoAAAA(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	ss, err := safesearch.NewDefault(ctx, &safesearch.DefaultConfig{
		Logger:         slogutil.NewDiscardLogger(),
		ServicesConfig: testConf,
		CacheSize:      testCacheSize,
		CacheTTL:       testCacheTTL,
	})
	require.NoError(t, err)

	// The DuckDuckGo safe-search addresses are resolved through CNAMEs, but
	// DuckDuckGo doesn't have a safe-search IPv6 address.  The result should be
	// the same as the one for Yandex IPv6.  That is, a NODATA response.
	res, err := ss.CheckHost(ctx, "www.duckduckgo.com", dns.TypeAAAA)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)
	assert.Equal(t, filtering.FilteredSafeSearch, res.Reason)
	assert.Equal(t, "safe.duckduckgo.com", res.CanonName)
	assert.Empty(t, res.Rules)
}

func TestDefault_Update(t *testing.T) {
	conf := testConf
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	ss, err := safesearch.NewDefault(ctx, &safesearch.DefaultConfig{
		Logger:         slogutil.NewDiscardLogger(),
		ServicesConfig: conf,
		CacheSize:      testCacheSize,
		CacheTTL:       testCacheTTL,
	})
	require.NoError(t, err)

	res, err := ss.CheckHost(ctx, "www.yandex.com", testQType)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	err = ss.Update(ctx, filtering.SafeSearchConfig{
		Enabled: true,
		Google:  false,
	})
	require.NoError(t, err)

	res, err = ss.CheckHost(ctx, "www.yandex.com", testQType)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)

	err = ss.Update(ctx, filtering.SafeSearchConfig{
		Enabled: false,
		Google:  true,
	})
	require.NoError(t, err)

	res, err = ss.CheckHost(ctx, "www.yandex.com", testQType)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
}
