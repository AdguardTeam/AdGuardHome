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

const (
	safeSearchCacheSize = 5000
	cacheTime           = 30 * time.Minute
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

func newForTest(t testing.TB, ssConf filtering.SafeSearchConfig) (ss *DefaultSafeSearch) {
	ss, err := NewDefaultSafeSearch(ssConf, safeSearchCacheSize, cacheTime)
	require.NoError(t, err)

	return ss
}

func TestSafeSearch(t *testing.T) {
	ss := newForTest(t, defaultSafeSearchConf)
	val := ss.SearchHost("www.google.com", dns.TypeA)

	assert.Equal(t, &rules.DNSRewrite{NewCNAME: "forcesafesearch.google.com"}, val)
}

func TestCheckHostSafeSearchYandex(t *testing.T) {
	ss := newForTest(t, defaultSafeSearchConf)

	// Check host for each domain.
	for _, host := range []string{
		"yandex.ru",
		"yAndeX.ru",
		"YANdex.COM",
		"yandex.by",
		"yandex.kz",
		"www.yandex.com",
	} {
		res, err := ss.CheckHost(host, dns.TypeA)
		require.NoError(t, err)

		assert.True(t, res.IsFiltered)

		require.Len(t, res.Rules, 1)

		assert.Equal(t, yandexIP, res.Rules[0].IP)
		assert.EqualValues(t, filtering.SafeSearchListID, res.Rules[0].FilterListID)
	}
}

func TestCheckHostSafeSearchGoogle(t *testing.T) {
	resolver := &aghtest.TestResolver{}
	ip, _ := resolver.HostToIPs("forcesafesearch.google.com")

	ss := newForTest(t, defaultSafeSearchConf)
	ss.resolver = resolver

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
			res, err := ss.CheckHost(host, dns.TypeA)
			require.NoError(t, err)

			assert.True(t, res.IsFiltered)

			require.Len(t, res.Rules, 1)

			assert.Equal(t, ip, res.Rules[0].IP)
			assert.EqualValues(t, filtering.SafeSearchListID, res.Rules[0].FilterListID)
		})
	}
}

func TestSafeSearchCacheYandex(t *testing.T) {
	const domain = "yandex.ru"

	ss := newForTest(t, filtering.SafeSearchConfig{Enabled: false})

	// Check host with disabled safesearch.
	res, err := ss.CheckHost(domain, dns.TypeA)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
	assert.Empty(t, res.Rules)

	ss = newForTest(t, defaultSafeSearchConf)
	res, err = ss.CheckHost(domain, dns.TypeA)
	require.NoError(t, err)

	// For yandex we already know valid IP.
	require.Len(t, res.Rules, 1)

	assert.Equal(t, res.Rules[0].IP, yandexIP)

	// Check cache.
	cachedValue, isFound := ss.getCachedResult(domain)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)

	assert.Equal(t, cachedValue.Rules[0].IP, yandexIP)
}

func TestSafeSearchCacheGoogle(t *testing.T) {
	const domain = "www.google.ru"

	ss := newForTest(t, filtering.SafeSearchConfig{Enabled: false})

	res, err := ss.CheckHost(domain, dns.TypeA)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
	assert.Empty(t, res.Rules)

	resolver := &aghtest.TestResolver{}
	ss = newForTest(t, defaultSafeSearchConf)
	ss.resolver = resolver

	// Lookup for safesearch domain.
	rewrite := ss.SearchHost(domain, dns.TypeA)

	ips, err := resolver.LookupIP(context.Background(), "ip", rewrite.NewCNAME)
	require.NoError(t, err)

	var foundIP net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			foundIP = ip

			break
		}
	}

	res, err = ss.CheckHost(domain, dns.TypeA)
	require.NoError(t, err)
	require.Len(t, res.Rules, 1)

	assert.True(t, res.Rules[0].IP.Equal(foundIP))

	// Check cache.
	cachedValue, isFound := ss.getCachedResult(domain)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)

	assert.True(t, cachedValue.Rules[0].IP.Equal(foundIP))
}

const googleHost = "www.google.com"

var dnsRewriteSink *rules.DNSRewrite

func BenchmarkSafeSearch(b *testing.B) {
	ss := newForTest(b, defaultSafeSearchConf)

	for n := 0; n < b.N; n++ {
		dnsRewriteSink = ss.SearchHost(googleHost, dns.TypeA)
	}

	assert.Equal(b, "forcesafesearch.google.com", dnsRewriteSink.NewCNAME)
}

var dnsRewriteParallelSink *rules.DNSRewrite

func BenchmarkSafeSearch_parallel(b *testing.B) {
	ss := newForTest(b, defaultSafeSearchConf)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			dnsRewriteParallelSink = ss.SearchHost(googleHost, dns.TypeA)
		}
	})

	assert.Equal(b, "forcesafesearch.google.com", dnsRewriteParallelSink.NewCNAME)
}
