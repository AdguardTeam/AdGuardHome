package dnsfilter

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

var setts FilteringSettings

// Helpers.

func purgeCaches() {
	for _, c := range []cache.Cache{
		gctx.safebrowsingCache,
		gctx.parentalCache,
		gctx.safeSearchCache,
	} {
		if c != nil {
			c.Clear()
		}
	}
}

func newForTest(c *Config, filters []Filter) *DNSFilter {
	setts = FilteringSettings{
		FilteringEnabled: true,
	}
	setts.FilteringEnabled = true
	if c != nil {
		c.SafeBrowsingCacheSize = 10000
		c.ParentalCacheSize = 10000
		c.SafeSearchCacheSize = 1000
		c.CacheTime = 30
		setts.SafeSearchEnabled = c.SafeSearchEnabled
		setts.SafeBrowsingEnabled = c.SafeBrowsingEnabled
		setts.ParentalEnabled = c.ParentalEnabled
	}
	d := New(c, filters)
	purgeCaches()
	return d
}

func (d *DNSFilter) checkMatch(t *testing.T, hostname string) {
	t.Helper()

	res, err := d.CheckHost(hostname, dns.TypeA, &setts)
	require.Nilf(t, err, "Error while matching host %s: %s", hostname, err)
	assert.Truef(t, res.IsFiltered, "Expected hostname %s to match", hostname)
}

func (d *DNSFilter) checkMatchIP(t *testing.T, hostname, ip string, qtype uint16) {
	t.Helper()

	res, err := d.CheckHost(hostname, qtype, &setts)
	require.Nilf(t, err, "Error while matching host %s: %s", hostname, err)
	assert.Truef(t, res.IsFiltered, "Expected hostname %s to match", hostname)

	require.NotEmpty(t, res.Rules, "Expected result to have rules")
	r := res.Rules[0]
	require.NotNilf(t, r.IP, "Expected ip %s to match, actual: %v", ip, r.IP)
	assert.Equalf(t, ip, r.IP.String(), "Expected ip %s to match, actual: %v", ip, r.IP)
}

func (d *DNSFilter) checkMatchEmpty(t *testing.T, hostname string) {
	t.Helper()

	res, err := d.CheckHost(hostname, dns.TypeA, &setts)
	require.Nilf(t, err, "Error while matching host %s: %s", hostname, err)
	assert.Falsef(t, res.IsFiltered, "Expected hostname %s to not match", hostname)
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
	d := newForTest(nil, filters)
	t.Cleanup(d.Close)

	d.checkMatchIP(t, "google.com", addr, dns.TypeA)
	d.checkMatchIP(t, "www.google.com", addr, dns.TypeA)
	d.checkMatchEmpty(t, "subdomain.google.com")
	d.checkMatchEmpty(t, "example.org")

	// IPv4 match.
	d.checkMatchIP(t, "block.com", "0.0.0.0", dns.TypeA)

	// Empty IPv6.
	res, err := d.CheckHost("block.com", dns.TypeAAAA, &setts)
	require.Nil(t, err)
	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)
	assert.Equal(t, "0.0.0.0 block.com", res.Rules[0].Text)
	assert.Empty(t, res.Rules[0].IP)

	// IPv6 match.
	d.checkMatchIP(t, "ipv6.com", addr6, dns.TypeAAAA)

	// Empty IPv4.
	res, err = d.CheckHost("ipv6.com", dns.TypeA, &setts)
	require.Nil(t, err)
	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)
	assert.Equal(t, "::1  ipv6.com", res.Rules[0].Text)
	assert.Empty(t, res.Rules[0].IP)

	// Two IPv4, the first one returned.
	res, err = d.CheckHost("host2", dns.TypeA, &setts)
	require.Nil(t, err)
	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)
	assert.Equal(t, res.Rules[0].IP, net.IP{0, 0, 0, 1})

	// One IPv6 address.
	res, err = d.CheckHost("host2", dns.TypeAAAA, &setts)
	require.Nil(t, err)
	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)
	assert.Equal(t, res.Rules[0].IP, net.IPv6loopback)
}

// Safe Browsing.

func TestSafeBrowsing(t *testing.T) {
	logOutput := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, logOutput)
	aghtest.ReplaceLogLevel(t, log.DEBUG)

	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)
	const matching = "wmconvirus.narod.ru"
	d.SetSafeBrowsingUpstream(&aghtest.TestBlockUpstream{
		Hostname: matching,
		Block:    true,
	})
	d.checkMatch(t, matching)

	require.Contains(t, logOutput.String(), "SafeBrowsing lookup for "+matching)

	d.checkMatch(t, "test."+matching)
	d.checkMatchEmpty(t, "yandex.ru")
	d.checkMatchEmpty(t, "pornhub.com")

	// Cached result.
	d.safeBrowsingServer = "127.0.0.1"
	d.checkMatch(t, matching)
	d.checkMatchEmpty(t, "pornhub.com")
	d.safeBrowsingServer = defaultSafebrowsingServer
}

func TestParallelSB(t *testing.T) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	t.Cleanup(d.Close)
	const matching = "wmconvirus.narod.ru"
	d.SetSafeBrowsingUpstream(&aghtest.TestBlockUpstream{
		Hostname: matching,
		Block:    true,
	})

	t.Run("group", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			t.Run(fmt.Sprintf("aaa%d", i), func(t *testing.T) {
				t.Parallel()
				d.checkMatch(t, matching)
				d.checkMatch(t, "test."+matching)
				d.checkMatchEmpty(t, "yandex.ru")
				d.checkMatchEmpty(t, "pornhub.com")
			})
		}
	})
}

// Safe Search.

func TestSafeSearch(t *testing.T) {
	d := newForTest(&Config{SafeSearchEnabled: true}, nil)
	t.Cleanup(d.Close)
	val, ok := d.SafeSearchDomain("www.google.com")
	require.True(t, ok, "Expected safesearch to find result for www.google.com")
	assert.Equal(t, "forcesafesearch.google.com", val, "Expected safesearch for google.com to be forcesafesearch.google.com")
}

func TestCheckHostSafeSearchYandex(t *testing.T) {
	d := newForTest(&Config{SafeSearchEnabled: true}, nil)
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
			res, err := d.CheckHost(host, dns.TypeA, &setts)
			require.Nil(t, err)
			assert.True(t, res.IsFiltered)

			require.Len(t, res.Rules, 1)
			assert.Equal(t, yandexIP, res.Rules[0].IP)
		})
	}
}

func TestCheckHostSafeSearchGoogle(t *testing.T) {
	resolver := &aghtest.TestResolver{}
	d := newForTest(&Config{
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
			res, err := d.CheckHost(host, dns.TypeA, &setts)
			require.Nil(t, err)
			assert.True(t, res.IsFiltered)
			require.Len(t, res.Rules, 1)
			assert.Equal(t, ip, res.Rules[0].IP)
		})
	}
}

func TestSafeSearchCacheYandex(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	const domain = "yandex.ru"

	// Check host with disabled safesearch.
	res, err := d.CheckHost(domain, dns.TypeA, &setts)
	require.Nil(t, err)
	assert.False(t, res.IsFiltered)
	require.Empty(t, res.Rules)

	yandexIP := net.IPv4(213, 180, 193, 56)

	d = newForTest(&Config{SafeSearchEnabled: true}, nil)
	t.Cleanup(d.Close)

	res, err = d.CheckHost(domain, dns.TypeA, &setts)
	require.Nilf(t, err, "CheckHost for safesearh domain %s failed cause %s", domain, err)

	// For yandex we already know valid IP.
	require.Len(t, res.Rules, 1)
	assert.Equal(t, res.Rules[0].IP, yandexIP)

	// Check cache.
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, domain)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)
	assert.Equal(t, cachedValue.Rules[0].IP, yandexIP)
}

func TestSafeSearchCacheGoogle(t *testing.T) {
	resolver := &aghtest.TestResolver{}
	d := newForTest(&Config{
		CustomResolver: resolver,
	}, nil)
	t.Cleanup(d.Close)

	const domain = "www.google.ru"
	res, err := d.CheckHost(domain, dns.TypeA, &setts)
	require.Nil(t, err)
	assert.False(t, res.IsFiltered)
	require.Empty(t, res.Rules)

	d = newForTest(&Config{SafeSearchEnabled: true}, nil)
	t.Cleanup(d.Close)
	d.resolver = resolver

	// Lookup for safesearch domain.
	safeDomain, ok := d.SafeSearchDomain(domain)
	require.Truef(t, ok, "Failed to get safesearch domain for %s", domain)

	ips, err := resolver.LookupIP(context.Background(), "ip", safeDomain)
	require.Nilf(t, err, "Failed to lookup for %s", safeDomain)

	var ip net.IP
	for _, foundIP := range ips {
		if foundIP.To4() != nil {
			ip = foundIP

			break
		}
	}

	res, err = d.CheckHost(domain, dns.TypeA, &setts)
	require.Nil(t, err)
	require.Len(t, res.Rules, 1)
	assert.True(t, res.Rules[0].IP.Equal(ip))

	// Check cache.
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, domain)
	require.True(t, isFound)
	require.Len(t, cachedValue.Rules, 1)
	assert.True(t, cachedValue.Rules[0].IP.Equal(ip))
}

// Parental.

func TestParentalControl(t *testing.T) {
	logOutput := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, logOutput)
	aghtest.ReplaceLogLevel(t, log.DEBUG)

	d := newForTest(&Config{ParentalEnabled: true}, nil)
	t.Cleanup(d.Close)
	const matching = "pornhub.com"
	d.SetParentalUpstream(&aghtest.TestBlockUpstream{
		Hostname: matching,
		Block:    true,
	})

	d.checkMatch(t, matching)
	require.Contains(t, logOutput.String(), "Parental lookup for "+matching)
	d.checkMatch(t, "www."+matching)
	d.checkMatchEmpty(t, "www.yandex.ru")
	d.checkMatchEmpty(t, "yandex.ru")
	d.checkMatchEmpty(t, "api.jquery.com")

	// Test cached result.
	d.parentalServer = "127.0.0.1"
	d.checkMatch(t, matching)
	d.checkMatchEmpty(t, "yandex.ru")
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
		host:           "wmconvirus.narod.ru",
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
			d := newForTest(nil, filters)
			t.Cleanup(d.Close)

			res, err := d.CheckHost(tc.host, tc.wantDNSType, &setts)
			require.Nilf(t, err, "Error while matching host %s: %s", tc.host, err)
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
	d := newForTest(nil, filters)

	require.Nil(t, d.SetFilters(filters, whiteFilters, false))
	t.Cleanup(d.Close)

	// Matched by white filter.
	res, err := d.CheckHost("host1", dns.TypeA, &setts)
	require.Nil(t, err)
	assert.False(t, res.IsFiltered)
	assert.Equal(t, res.Reason, NotFilteredAllowList)
	require.Len(t, res.Rules, 1)
	assert.Equal(t, "||host1^", res.Rules[0].Text)

	// Not matched by white filter, but matched by block filter.
	res, err = d.CheckHost("host2", dns.TypeA, &setts)
	require.Nil(t, err)
	assert.True(t, res.IsFiltered)
	assert.Equal(t, res.Reason, FilteredBlockList)
	require.Len(t, res.Rules, 1)
	assert.Equal(t, "||host2^", res.Rules[0].Text)
}

// Client Settings.

func applyClientSettings(setts *FilteringSettings) {
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
	d := newForTest(
		&Config{
			ParentalEnabled:     true,
			SafeBrowsingEnabled: false,
		},
		[]Filter{{
			ID: 0, Data: []byte("||example.org^\n"),
		}},
	)
	t.Cleanup(d.Close)
	d.SetParentalUpstream(&aghtest.TestBlockUpstream{
		Hostname: "pornhub.com",
		Block:    true,
	})
	d.SetSafeBrowsingUpstream(&aghtest.TestBlockUpstream{
		Hostname: "wmconvirus.narod.ru",
		Block:    true,
	})

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
		host:       "pornhub.com",
		before:     true,
		wantReason: FilteredParental,
	}, {
		name:       "safebrowsing",
		host:       "wmconvirus.narod.ru",
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
			r, _ := d.CheckHost(tc.host, dns.TypeA, &setts)
			if before {
				assert.True(t, r.IsFiltered)
				assert.Equal(t, tc.wantReason, r.Reason)
			} else {
				assert.False(t, r.IsFiltered)
			}
		}
	}

	// Check behaviour without any per-client settings, then apply per-client
	// settings and check behaviour once again.
	for _, tc := range testCases {
		t.Run(tc.name, makeTester(tc, tc.before))
	}

	applyClientSettings(&setts)

	for _, tc := range testCases {
		t.Run(tc.name, makeTester(tc, !tc.before))
	}
}

// Benchmarks.

func BenchmarkSafeBrowsing(b *testing.B) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	b.Cleanup(d.Close)
	blocked := "wmconvirus.narod.ru"
	d.SetSafeBrowsingUpstream(&aghtest.TestBlockUpstream{
		Hostname: blocked,
		Block:    true,
	})
	for n := 0; n < b.N; n++ {
		res, err := d.CheckHost(blocked, dns.TypeA, &setts)
		require.Nilf(b, err, "Error while matching host %s: %s", blocked, err)
		assert.True(b, res.IsFiltered, "Expected hostname %s to match", blocked)
	}
}

func BenchmarkSafeBrowsingParallel(b *testing.B) {
	d := newForTest(&Config{SafeBrowsingEnabled: true}, nil)
	b.Cleanup(d.Close)
	blocked := "wmconvirus.narod.ru"
	d.SetSafeBrowsingUpstream(&aghtest.TestBlockUpstream{
		Hostname: blocked,
		Block:    true,
	})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			res, err := d.CheckHost(blocked, dns.TypeA, &setts)
			require.Nilf(b, err, "Error while matching host %s: %s", blocked, err)
			assert.True(b, res.IsFiltered, "Expected hostname %s to match", blocked)
		}
	})
}

func BenchmarkSafeSearch(b *testing.B) {
	d := newForTest(&Config{SafeSearchEnabled: true}, nil)
	b.Cleanup(d.Close)
	for n := 0; n < b.N; n++ {
		val, ok := d.SafeSearchDomain("www.google.com")
		require.True(b, ok, "Expected safesearch to find result for www.google.com")
		assert.Equal(b, "forcesafesearch.google.com", val, "Expected safesearch for google.com to be forcesafesearch.google.com")
	}
}

func BenchmarkSafeSearchParallel(b *testing.B) {
	d := newForTest(&Config{SafeSearchEnabled: true}, nil)
	b.Cleanup(d.Close)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			val, ok := d.SafeSearchDomain("www.google.com")
			require.True(b, ok, "Expected safesearch to find result for www.google.com")
			assert.Equal(b, "forcesafesearch.google.com", val, "Expected safesearch for google.com to be forcesafesearch.google.com")
		}
	})
}
