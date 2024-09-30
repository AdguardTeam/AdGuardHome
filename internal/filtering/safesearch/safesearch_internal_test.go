package safesearch

import (
	"net/netip"
	"testing"
	"time"

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
	Ecosia:     true,
	Google:     true,
	Pixabay:    true,
	Yandex:     true,
	YouTube:    true,
}

var yandexIP = netip.AddrFrom4([4]byte{213, 180, 193, 56})

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

const googleHost = "www.google.com"

var dnsRewriteSink *rules.DNSRewrite

func BenchmarkSafeSearch(b *testing.B) {
	ss := newForTest(b, defaultSafeSearchConf)

	for range b.N {
		dnsRewriteSink = ss.searchHost(googleHost, testQType)
	}

	assert.Equal(b, "forcesafesearch.google.com", dnsRewriteSink.NewCNAME)
}
