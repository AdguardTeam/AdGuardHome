package safesearch_test

import (
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

// Common test constants.
const (
	// TODO(a.garipov): Add IPv6 tests.
	testQType     = dns.TypeA
	testCacheSize = 5000
	testCacheTTL  = 30 * time.Minute
)

// testConf is the default safe search configuration for tests.
var testConf = filtering.SafeSearchConfig{
	CustomResolver: nil,

	Enabled: true,

	Bing:       true,
	DuckDuckGo: true,
	Google:     true,
	Pixabay:    true,
	Yandex:     true,
	YouTube:    true,
}

// yandexIP is the expected IP address of Yandex safe search results.  Keep in
// sync with the rules data.
var yandexIP = net.IPv4(213, 180, 193, 56)

func TestDefault_CheckHost_yandex(t *testing.T) {
	conf := testConf
	ss, err := safesearch.NewDefault(conf, "", testCacheSize, testCacheTTL)
	require.NoError(t, err)

	// Check host for each domain.
	for _, host := range []string{
		"yandex.ru",
		"yAndeX.ru",
		"YANdex.COM",
		"yandex.by",
		"yandex.kz",
		"www.yandex.com",
	} {
		var res filtering.Result
		res, err = ss.CheckHost(host, testQType)
		require.NoError(t, err)

		assert.True(t, res.IsFiltered)

		require.Len(t, res.Rules, 1)

		assert.Equal(t, yandexIP, res.Rules[0].IP)
		assert.EqualValues(t, filtering.SafeSearchListID, res.Rules[0].FilterListID)
	}
}

func TestDefault_CheckHost_google(t *testing.T) {
	resolver := &aghtest.TestResolver{}
	ip, _ := resolver.HostToIPs("forcesafesearch.google.com")

	conf := testConf
	conf.CustomResolver = resolver
	ss, err := safesearch.NewDefault(conf, "", testCacheSize, testCacheTTL)
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
			res, err = ss.CheckHost(host, testQType)
			require.NoError(t, err)

			assert.True(t, res.IsFiltered)

			require.Len(t, res.Rules, 1)

			assert.Equal(t, ip, res.Rules[0].IP)
			assert.EqualValues(t, filtering.SafeSearchListID, res.Rules[0].FilterListID)
		})
	}
}

func TestDefault_Update(t *testing.T) {
	conf := testConf
	ss, err := safesearch.NewDefault(conf, "", testCacheSize, testCacheTTL)
	require.NoError(t, err)

	res, err := ss.CheckHost("www.yandex.com", testQType)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	err = ss.Update(filtering.SafeSearchConfig{
		Enabled: true,
		Google:  false,
	})
	require.NoError(t, err)

	res, err = ss.CheckHost("www.yandex.com", testQType)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)

	err = ss.Update(filtering.SafeSearchConfig{
		Enabled: false,
		Google:  true,
	})
	require.NoError(t, err)

	res, err = ss.CheckHost("www.yandex.com", testQType)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
}
