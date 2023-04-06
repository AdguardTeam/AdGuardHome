package safesearch

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(a.garipov): Move as much of this as possible into proper external tests.

const (
	// TODO(a.garipov): Add IPv6 tests.
	testQType     = dns.TypeA
	testCacheSize = 5000
	testCacheTTL  = 30 * time.Minute
)

var defaultSafeSearchConf = filtering.SafeSearchConfig{
	Enabled:    true,
	Bing:       true,
	DuckDuckGo: true,
	Google:     true,
	Pixabay:    true,
	Yandex:     true,
	YouTube:    true,
}

var yandexIP = net.IPv4(213, 180, 193, 56)

func newForTest(t testing.TB, ssConf filtering.SafeSearchConfig) (ss *Default) {
	ss, err := NewDefault(ssConf, "", testCacheSize, testCacheTTL)
	require.NoError(t, err)

	return ss
}

func TestSafeSearch(t *testing.T) {
	ss := newForTest(t, defaultSafeSearchConf)
	val := ss.searchHost("www.google.com", testQType)

	assert.Equal(t, &rules.DNSRewrite{NewCNAME: "forcesafesearch.google.com"}, val)
}

func TestSafeSearchCacheYandex(t *testing.T) {
	const domain = "yandex.ru"

	ss := newForTest(t, filtering.SafeSearchConfig{Enabled: false})

	// Check host with disabled safesearch.
	res, err := ss.CheckHost(domain, testQType)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
	assert.Empty(t, res.Rules)

	ss = newForTest(t, defaultSafeSearchConf)
	res, err = ss.CheckHost(domain, testQType)
	require.NoError(t, err)

	// For yandex we already know valid IP.
	require.Len(t, res.Rules, 1)

	assert.Equal(t, res.Rules[0].IP, yandexIP)

	// Check cache.
	cachedValue, isFound := ss.getCachedResult(domain, testQType)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)

	assert.Equal(t, cachedValue.Rules[0].IP, yandexIP)
}

func TestSafeSearchCacheGoogle(t *testing.T) {
	const domain = "www.google.ru"

	ss := newForTest(t, filtering.SafeSearchConfig{Enabled: false})

	res, err := ss.CheckHost(domain, testQType)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
	assert.Empty(t, res.Rules)

	resolver := &aghtest.TestResolver{}
	ss = newForTest(t, defaultSafeSearchConf)
	ss.resolver = resolver

	// Lookup for safesearch domain.
	rewrite := ss.searchHost(domain, testQType)

	ips, err := resolver.LookupIP(context.Background(), "ip", rewrite.NewCNAME)
	require.NoError(t, err)

	var foundIP net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			foundIP = ip

			break
		}
	}

	res, err = ss.CheckHost(domain, testQType)
	require.NoError(t, err)
	require.Len(t, res.Rules, 1)

	assert.True(t, res.Rules[0].IP.Equal(foundIP))

	// Check cache.
	cachedValue, isFound := ss.getCachedResult(domain, testQType)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)

	assert.True(t, cachedValue.Rules[0].IP.Equal(foundIP))
}

const googleHost = "www.google.com"

var dnsRewriteSink *rules.DNSRewrite

func BenchmarkSafeSearch(b *testing.B) {
	ss := newForTest(b, defaultSafeSearchConf)

	for n := 0; n < b.N; n++ {
		dnsRewriteSink = ss.searchHost(googleHost, testQType)
	}

	assert.Equal(b, "forcesafesearch.google.com", dnsRewriteSink.NewCNAME)
}
