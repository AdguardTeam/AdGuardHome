package filtering

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

const (
	sbBlocked = "wmconvirus.narod.ru"
	pcBlocked = "pornhub.com"
)

// Helpers.

func purgeCaches(d *DNSFilter) {
	for _, c := range []cache.Cache{
		d.safebrowsingCache,
		d.parentalCache,
		d.safeSearchCache,
	} {
		if c != nil {
			c.Clear()
		}
	}
}

func newForTest(t testing.TB, c *Config, filters []Filter) (f *DNSFilter, setts *Settings) {
	setts = &Settings{
		ProtectionEnabled: true,
		FilteringEnabled:  true,
	}
	if c != nil {
		c.SafeBrowsingCacheSize = 10000
		c.ParentalCacheSize = 10000
		c.SafeSearchCacheSize = 1000
		c.CacheTime = 30
		setts.SafeSearchEnabled = c.SafeSearchEnabled
		setts.SafeBrowsingEnabled = c.SafeBrowsingEnabled
		setts.ParentalEnabled = c.ParentalEnabled
	} else {
		// It must not be nil.
		c = &Config{}
	}
	f, err := New(c, filters)
	require.NoError(t, err)

	purgeCaches(f)

	return f, setts
}

func (d *DNSFilter) checkMatch(t *testing.T, hostname string, setts *Settings) {
	t.Helper()

	res, err := d.CheckHost(hostname, dns.TypeA, setts)
	require.NoErrorf(t, err, "host %q", hostname)

	assert.Truef(t, res.IsFiltered, "host %q", hostname)
}

func (d *DNSFilter) checkMatchIP(t *testing.T, hostname, ip string, qtype uint16, setts *Settings) {
	t.Helper()

	res, err := d.CheckHost(hostname, qtype, setts)
	require.NoErrorf(t, err, "host %q", hostname, err)
	require.NotEmpty(t, res.Rules, "host %q", hostname)

	assert.Truef(t, res.IsFiltered, "host %q", hostname)

	r := res.Rules[0]
	require.NotNilf(t, r.IP, "Expected ip %s to match, actual: %v", ip, r.IP)

	assert.Equalf(t, ip, r.IP.String(), "host %q", hostname)
}

func (d *DNSFilter) checkMatchEmpty(t *testing.T, hostname string, setts *Settings) {
	t.Helper()

	res, err := d.CheckHost(hostname, dns.TypeA, setts)
	require.NoErrorf(t, err, "host %q", hostname)

	assert.Falsef(t, res.IsFiltered, "host %q", hostname)
}

func TestEtcHostsMatching(t *testing.T) {
	addr := "216.239.38.120"
	addr6 := "::1"
	text := fmt.Sprintf(`  %s  google.com www.google.com   # enforce google's safesearch
%s  ipv6.com
0.0.0.0 block.com
0.0.0.1 host2
0.0.0.2 host2
::1 host2
`,
		addr, addr6)
	filters := []Filter{{
		ID: 0, Data: []byte(text),
	}}
	d, setts := newForTest(t, nil, filters)
	t.Cleanup(d.Close)

	d.checkMatchIP(t, "google.com", addr, dns.TypeA, setts)
	d.checkMatchIP(t, "www.google.com", addr, dns.TypeA, setts)
	d.checkMatchEmpty(t, "subdomain.google.com", setts)
	d.checkMatchEmpty(t, "example.org", setts)

	// IPv4 match.
	d.checkMatchIP(t, "block.com", "0.0.0.0", dns.TypeA, setts)

	// Empty IPv6.
	res, err := d.CheckHost("block.com", dns.TypeAAAA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "0.0.0.0 block.com", res.Rules[0].Text)
	assert.Empty(t, res.Rules[0].IP)

	// IPv6 match.
	d.checkMatchIP(t, "ipv6.com", addr6, dns.TypeAAAA, setts)

	// Empty IPv4.
	res, err = d.CheckHost("ipv6.com", dns.TypeA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "::1  ipv6.com", res.Rules[0].Text)
	assert.Empty(t, res.Rules[0].IP)

	// Two IPv4, both must be returned.
	res, err = d.CheckHost("host2", dns.TypeA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 2)

	assert.Equal(t, res.Rules[0].IP, net.IP{0, 0, 0, 1})
	assert.Equal(t, res.Rules[1].IP, net.IP{0, 0, 0, 2})

	// One IPv6 address.
	res, err = d.CheckHost("host2", dns.TypeAAAA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, res.Rules[0].IP, net.IPv6loopback)
}

// Safe Browsing.

func TestSafeBrowsing(t *testing.T) {
	logOutput := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, logOutput)
	aghtest.ReplaceLogLevel(t, log.DEBUG)

	d, setts := newForTest(t, &Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)

	d.SetSafeBrowsingUpstream(aghtest.NewBlockUpstream(sbBlocked, true))
	d.checkMatch(t, sbBlocked, setts)

	require.Contains(t, logOutput.String(), fmt.Sprintf("safebrowsing lookup for %q", sbBlocked))

	d.checkMatch(t, "test."+sbBlocked, setts)
	d.checkMatchEmpty(t, "yandex.ru", setts)
	d.checkMatchEmpty(t, pcBlocked, setts)

	// Cached result.
	d.safeBrowsingServer = "127.0.0.1"
	d.checkMatch(t, sbBlocked, setts)
	d.checkMatchEmpty(t, pcBlocked, setts)
	d.safeBrowsingServer = defaultSafebrowsingServer
}

func TestParallelSB(t *testing.T) {
	d, setts := newForTest(t, &Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)

	d.SetSafeBrowsingUpstream(aghtest.NewBlockUpstream(sbBlocked, true))

	t.Run("group", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			t.Run(fmt.Sprintf("aaa%d", i), func(t *testing.T) {
				t.Parallel()
				d.checkMatch(t, sbBlocked, setts)
				d.checkMatch(t, "test."+sbBlocked, setts)
				d.checkMatchEmpty(t, "yandex.ru", setts)
				d.checkMatchEmpty(t, pcBlocked, setts)
			})
		}
	})
}

// Safe Search.

func TestSafeSearch(t *testing.T) {
	d, _ := newForTest(t, &Config{SafeSearchEnabled: true}, nil)
	t.Cleanup(d.Close)
	val, ok := d.SafeSearchDomain("www.google.com")
	require.True(t, ok)

	assert.Equal(t, "forcesafesearch.google.com", val)
}

func TestCheckHostSafeSearchYandex(t *testing.T) {
	d, setts := newForTest(t, &Config{
		SafeSearchEnabled: true,
	}, nil)
	t.Cleanup(d.Close)

	yandexIP := net.IPv4(213, 180, 193, 56)

	// Check host for each domain.
	for _, host := range []string{
		"yAndeX.ru",
		"YANdex.COM",
		"yandex.ua",
		"yandex.by",
		"yandex.kz",
		"www.yandex.com",
	} {
		t.Run(strings.ToLower(host), func(t *testing.T) {
			res, err := d.CheckHost(host, dns.TypeA, setts)
			require.NoError(t, err)

			assert.True(t, res.IsFiltered)

			require.Len(t, res.Rules, 1)

			assert.Equal(t, yandexIP, res.Rules[0].IP)
			assert.EqualValues(t, SafeSearchListID, res.Rules[0].FilterListID)
		})
	}
}

func TestCheckHostSafeSearchGoogle(t *testing.T) {
	resolver := &aghtest.TestResolver{}
	d, setts := newForTest(t, &Config{
		SafeSearchEnabled: true,
		CustomResolver:    resolver,
	}, nil)
	t.Cleanup(d.Close)

	ip, _ := resolver.HostToIPs("forcesafesearch.google.com")

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
			res, err := d.CheckHost(host, dns.TypeA, setts)
			require.NoError(t, err)

			assert.True(t, res.IsFiltered)

			require.Len(t, res.Rules, 1)

			assert.Equal(t, ip, res.Rules[0].IP)
			assert.EqualValues(t, SafeSearchListID, res.Rules[0].FilterListID)
		})
	}
}

func TestSafeSearchCacheYandex(t *testing.T) {
	d, setts := newForTest(t, nil, nil)
	t.Cleanup(d.Close)
	const domain = "yandex.ru"

	// Check host with disabled safesearch.
	res, err := d.CheckHost(domain, dns.TypeA, setts)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)

	require.Empty(t, res.Rules)

	yandexIP := net.IPv4(213, 180, 193, 56)

	d, setts = newForTest(t, &Config{SafeSearchEnabled: true}, nil)
	t.Cleanup(d.Close)

	res, err = d.CheckHost(domain, dns.TypeA, setts)
	require.NoError(t, err)

	// For yandex we already know valid IP.
	require.Len(t, res.Rules, 1)
	assert.Equal(t, res.Rules[0].IP, yandexIP)

	// Check cache.
	cachedValue, isFound := getCachedResult(d.safeSearchCache, domain)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)

	assert.Equal(t, cachedValue.Rules[0].IP, yandexIP)
}

func TestSafeSearchCacheGoogle(t *testing.T) {
	resolver := &aghtest.TestResolver{}
	d, setts := newForTest(t, &Config{
		CustomResolver: resolver,
	}, nil)
	t.Cleanup(d.Close)

	const domain = "www.google.ru"
	res, err := d.CheckHost(domain, dns.TypeA, setts)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)

	require.Empty(t, res.Rules)

	d, setts = newForTest(t, &Config{SafeSearchEnabled: true}, nil)
	t.Cleanup(d.Close)
	d.resolver = resolver

	// Lookup for safesearch domain.
	safeDomain, ok := d.SafeSearchDomain(domain)
	require.True(t, ok)

	ips, err := resolver.LookupIP(context.Background(), "ip", safeDomain)
	require.NoError(t, err)

	var ip net.IP
	for _, foundIP := range ips {
		if foundIP.To4() != nil {
			ip = foundIP

			break
		}
	}

	res, err = d.CheckHost(domain, dns.TypeA, setts)
	require.NoError(t, err)
	require.Len(t, res.Rules, 1)

	assert.True(t, res.Rules[0].IP.Equal(ip))

	// Check cache.
	cachedValue, isFound := getCachedResult(d.safeSearchCache, domain)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)

	assert.True(t, cachedValue.Rules[0].IP.Equal(ip))
}

// Parental.

func TestParentalControl(t *testing.T) {
	logOutput := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, logOutput)
	aghtest.ReplaceLogLevel(t, log.DEBUG)

	d, setts := newForTest(t, &Config{ParentalEnabled: true}, nil)
	t.Cleanup(d.Close)

	d.SetParentalUpstream(aghtest.NewBlockUpstream(pcBlocked, true))
	d.checkMatch(t, pcBlocked, setts)
	require.Contains(t, logOutput.String(), fmt.Sprintf("parental lookup for %q", pcBlocked))

	d.checkMatch(t, "www."+pcBlocked, setts)
	d.checkMatchEmpty(t, "www.yandex.ru", setts)
	d.checkMatchEmpty(t, "yandex.ru", setts)
	d.checkMatchEmpty(t, "api.jquery.com", setts)

	// Test cached result.
	d.parentalServer = "127.0.0.1"
	d.checkMatch(t, pcBlocked, setts)
	d.checkMatchEmpty(t, "yandex.ru", setts)
}

// Filtering.

func TestMatching(t *testing.T) {
	const nl = "\n"
	const (
		blockingRules  = `||example.org^` + nl
		allowlistRules = `||example.org^` + nl + `@@||test.example.org` + nl
		importantRules = `@@||example.org^` + nl + `||test.example.org^$important` + nl
		regexRules     = `/example\.org/` + nl + `@@||test.example.org^` + nl
		maskRules      = `test*.example.org^` + nl + `exam*.com` + nl
		dnstypeRules   = `||example.org^$dnstype=AAAA` + nl + `@@||test.example.org^` + nl
	)
	testCases := []struct {
		name           string
		rules          string
		host           string
		wantReason     Reason
		wantIsFiltered bool
		wantDNSType    uint16
	}{{
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           "www.doubleclick.net",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           "nodoubleclick.net",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           "doubleclick.net.ru",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           sbBlocked,
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "test.test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "test.test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "test.test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "test.test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "testexample.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "onemoreexample.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "test2.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "example.com",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "exampleeee.com",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "onemoreexamsite.com",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "example.co.uk",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeAAAA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeAAAA,
	}}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.name, tc.host), func(t *testing.T) {
			filters := []Filter{{ID: 0, Data: []byte(tc.rules)}}
			d, setts := newForTest(t, nil, filters)
			t.Cleanup(d.Close)

			res, err := d.CheckHost(tc.host, tc.wantDNSType, setts)
			require.NoError(t, err)

			assert.Equalf(t, tc.wantIsFiltered, res.IsFiltered, "Hostname %s has wrong result (%v must be %v)", tc.host, res.IsFiltered, tc.wantIsFiltered)
			assert.Equalf(t, tc.wantReason, res.Reason, "Hostname %s has wrong reason (%v must be %v)", tc.host, res.Reason, tc.wantReason)
		})
	}
}

func TestWhitelist(t *testing.T) {
	rules := `||host1^
||host2^
`
	filters := []Filter{{
		ID: 0, Data: []byte(rules),
	}}

	whiteRules := `||host1^
||host3^
`
	whiteFilters := []Filter{{
		ID: 0, Data: []byte(whiteRules),
	}}
	d, setts := newForTest(t, nil, filters)

	err := d.SetFilters(filters, whiteFilters, false)
	require.NoError(t, err)

	t.Cleanup(d.Close)

	// Matched by white filter.
	res, err := d.CheckHost("host1", dns.TypeA, setts)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
	assert.Equal(t, res.Reason, NotFilteredAllowList)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "||host1^", res.Rules[0].Text)

	// Not matched by white filter, but matched by block filter.
	res, err = d.CheckHost("host2", dns.TypeA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)
	assert.Equal(t, res.Reason, FilteredBlockList)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "||host2^", res.Rules[0].Text)
}

// Client Settings.

func applyClientSettings(setts *Settings) {
	setts.FilteringEnabled = false
	setts.ParentalEnabled = false
	setts.SafeBrowsingEnabled = true

	rule, _ := rules.NewNetworkRule("||facebook.com^", 0)
	s := ServiceEntry{}
	s.Name = "facebook"
	s.Rules = []*rules.NetworkRule{rule}
	setts.ServicesRules = append(setts.ServicesRules, s)
}

func TestClientSettings(t *testing.T) {
	d, setts := newForTest(t,
		&Config{
			ParentalEnabled:     true,
			SafeBrowsingEnabled: false,
		},
		[]Filter{{
			ID: 0, Data: []byte("||example.org^\n"),
		}},
	)
	t.Cleanup(d.Close)

	d.SetParentalUpstream(aghtest.NewBlockUpstream(pcBlocked, true))
	d.SetSafeBrowsingUpstream(aghtest.NewBlockUpstream(sbBlocked, true))

	type testCase struct {
		name       string
		host       string
		before     bool
		wantReason Reason
	}
	testCases := []testCase{{
		name:       "filters",
		host:       "example.org",
		before:     true,
		wantReason: FilteredBlockList,
	}, {
		name:       "parental",
		host:       pcBlocked,
		before:     true,
		wantReason: FilteredParental,
	}, {
		name:       "safebrowsing",
		host:       sbBlocked,
		before:     false,
		wantReason: FilteredSafeBrowsing,
	}, {
		name:       "additional_rules",
		host:       "facebook.com",
		before:     false,
		wantReason: FilteredBlockedService,
	}}

	makeTester := func(tc testCase, before bool) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()

			r, err := d.CheckHost(tc.host, dns.TypeA, setts)
			require.NoError(t, err)

			if before {
				assert.True(t, r.IsFiltered)
				assert.Equal(t, tc.wantReason, r.Reason)
			} else {
				assert.False(t, r.IsFiltered)
			}
		}
	}

	// Check behaviour without any per-client settings, then apply per-client
	// settings and check behavior once again.
	for _, tc := range testCases {
		t.Run(tc.name, makeTester(tc, tc.before))
	}

	applyClientSettings(setts)

	for _, tc := range testCases {
		t.Run(tc.name, makeTester(tc, !tc.before))
	}
}

// Benchmarks.

func BenchmarkSafeBrowsing(b *testing.B) {
	d, setts := newForTest(b, &Config{SafeBrowsingEnabled: true}, nil)
	b.Cleanup(d.Close)

	d.SetSafeBrowsingUpstream(aghtest.NewBlockUpstream(sbBlocked, true))

	for n := 0; n < b.N; n++ {
		res, err := d.CheckHost(sbBlocked, dns.TypeA, setts)
		require.NoError(b, err)

		assert.Truef(b, res.IsFiltered, "expected hostname %q to match", sbBlocked)
	}
}

func BenchmarkSafeBrowsingParallel(b *testing.B) {
	d, setts := newForTest(b, &Config{SafeBrowsingEnabled: true}, nil)
	b.Cleanup(d.Close)

	d.SetSafeBrowsingUpstream(aghtest.NewBlockUpstream(sbBlocked, true))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			res, err := d.CheckHost(sbBlocked, dns.TypeA, setts)
			require.NoError(b, err)

			assert.Truef(b, res.IsFiltered, "expected hostname %q to match", sbBlocked)
		}
	})
}

func BenchmarkSafeSearch(b *testing.B) {
	d, _ := newForTest(b, &Config{SafeSearchEnabled: true}, nil)
	b.Cleanup(d.Close)
	for n := 0; n < b.N; n++ {
		val, ok := d.SafeSearchDomain("www.google.com")
		require.True(b, ok)

		assert.Equal(b, "forcesafesearch.google.com", val, "Expected safesearch for google.com to be forcesafesearch.google.com")
	}
}

func BenchmarkSafeSearchParallel(b *testing.B) {
	d, _ := newForTest(b, &Config{SafeSearchEnabled: true}, nil)
	b.Cleanup(d.Close)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			val, ok := d.SafeSearchDomain("www.google.com")
			require.True(b, ok)

			assert.Equal(b, "forcesafesearch.google.com", val, "Expected safesearch for google.com to be forcesafesearch.google.com")
		}
	})
}
