package dnsfilter

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

var setts RequestFilteringSettings

// HELPERS
// SAFE BROWSING
// SAFE SEARCH
// PARENTAL
// FILTERING
// BENCHMARKS

// HELPERS

func purgeCaches() {
	if gctx.safebrowsingCache != nil {
		gctx.safebrowsingCache.Clear()
	}
	if gctx.parentalCache != nil {
		gctx.parentalCache.Clear()
	}
	if gctx.safeSearchCache != nil {
		gctx.safeSearchCache.Clear()
	}
}

func NewForTest(c *Config, filters []Filter) *DNSFilter {
	setts = RequestFilteringSettings{}
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
	if err != nil {
		t.Errorf("Error while matching host %s: %s", hostname, err)
	}
	if !res.IsFiltered {
		t.Errorf("Expected hostname %s to match", hostname)
	}
}

func (d *DNSFilter) checkMatchIP(t *testing.T, hostname, ip string, qtype uint16) {
	t.Helper()

	res, err := d.CheckHost(hostname, qtype, &setts)
	if err != nil {
		t.Errorf("Error while matching host %s: %s", hostname, err)
	}

	if !res.IsFiltered {
		t.Errorf("Expected hostname %s to match", hostname)
	}

	if len(res.Rules) == 0 {
		t.Errorf("Expected result to have rules")

		return
	}

	r := res.Rules[0]
	if r.IP == nil || r.IP.String() != ip {
		t.Errorf("Expected ip %s to match, actual: %v", ip, r.IP)
	}
}

func (d *DNSFilter) checkMatchEmpty(t *testing.T, hostname string) {
	t.Helper()
	res, err := d.CheckHost(hostname, dns.TypeA, &setts)
	if err != nil {
		t.Errorf("Error while matching host %s: %s", hostname, err)
	}
	if res.IsFiltered {
		t.Errorf("Expected hostname %s to not match", hostname)
	}
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
	d := NewForTest(nil, filters)
	defer d.Close()

	d.checkMatchIP(t, "google.com", addr, dns.TypeA)
	d.checkMatchIP(t, "www.google.com", addr, dns.TypeA)
	d.checkMatchEmpty(t, "subdomain.google.com")
	d.checkMatchEmpty(t, "example.org")

	// IPv4
	d.checkMatchIP(t, "block.com", "0.0.0.0", dns.TypeA)

	// ...but empty IPv6
	res, err := d.CheckHost("block.com", dns.TypeAAAA, &setts)
	assert.Nil(t, err)
	assert.True(t, res.IsFiltered)
	if assert.Len(t, res.Rules, 1) {
		assert.Equal(t, "0.0.0.0 block.com", res.Rules[0].Text)
		assert.Len(t, res.Rules[0].IP, 0)
	}

	// IPv6
	d.checkMatchIP(t, "ipv6.com", addr6, dns.TypeAAAA)

	// ...but empty IPv4
	res, err = d.CheckHost("ipv6.com", dns.TypeA, &setts)
	assert.Nil(t, err)
	assert.True(t, res.IsFiltered)
	if assert.Len(t, res.Rules, 1) {
		assert.Equal(t, "::1  ipv6.com", res.Rules[0].Text)
		assert.Len(t, res.Rules[0].IP, 0)
	}

	// 2 IPv4 (return only the first one)
	res, err = d.CheckHost("host2", dns.TypeA, &setts)
	assert.Nil(t, err)
	assert.True(t, res.IsFiltered)
	if assert.Len(t, res.Rules, 1) {
		loopback4 := net.IP{0, 0, 0, 1}
		assert.Equal(t, res.Rules[0].IP, loopback4)
	}

	// ...and 1 IPv6 address
	res, err = d.CheckHost("host2", dns.TypeAAAA, &setts)
	assert.Nil(t, err)
	assert.True(t, res.IsFiltered)
	if assert.Len(t, res.Rules, 1) {
		loopback6 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
		assert.Equal(t, res.Rules[0].IP, loopback6)
	}
}

// SAFE BROWSING

func TestSafeBrowsing(t *testing.T) {
	logOutput := &bytes.Buffer{}
	testutil.ReplaceLogWriter(t, logOutput)
	testutil.ReplaceLogLevel(t, log.DEBUG)

	d := NewForTest(&Config{SafeBrowsingEnabled: true}, nil)
	defer d.Close()
	d.checkMatch(t, "wmconvirus.narod.ru")

	assert.True(t, strings.Contains(logOutput.String(), "SafeBrowsing lookup for wmconvirus.narod.ru"))

	d.checkMatch(t, "test.wmconvirus.narod.ru")
	d.checkMatchEmpty(t, "yandex.ru")
	d.checkMatchEmpty(t, "pornhub.com")

	// test cached result
	d.safeBrowsingServer = "127.0.0.1"
	d.checkMatch(t, "wmconvirus.narod.ru")
	d.checkMatchEmpty(t, "pornhub.com")
	d.safeBrowsingServer = defaultSafebrowsingServer
}

func TestParallelSB(t *testing.T) {
	d := NewForTest(&Config{SafeBrowsingEnabled: true}, nil)
	defer d.Close()
	t.Run("group", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			t.Run(fmt.Sprintf("aaa%d", i), func(t *testing.T) {
				t.Parallel()
				d.checkMatch(t, "wmconvirus.narod.ru")
				d.checkMatch(t, "test.wmconvirus.narod.ru")
				d.checkMatchEmpty(t, "yandex.ru")
				d.checkMatchEmpty(t, "pornhub.com")
			})
		}
	})
}

// SAFE SEARCH

func TestSafeSearch(t *testing.T) {
	d := NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()
	val, ok := d.SafeSearchDomain("www.google.com")
	if !ok {
		t.Errorf("Expected safesearch to find result for www.google.com")
	}
	if val != "forcesafesearch.google.com" {
		t.Errorf("Expected safesearch for google.com to be forcesafesearch.google.com")
	}
}

func TestCheckHostSafeSearchYandex(t *testing.T) {
	d := NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()

	// Slice of yandex domains
	yandex := []string{"yAndeX.ru", "YANdex.COM", "yandex.ua", "yandex.by", "yandex.kz", "www.yandex.com"}

	// Check host for each domain
	for _, host := range yandex {
		res, err := d.CheckHost(host, dns.TypeA, &setts)
		assert.Nil(t, err)
		assert.True(t, res.IsFiltered)
		if assert.Len(t, res.Rules, 1) {
			assert.Equal(t, res.Rules[0].IP.String(), "213.180.193.56")
		}
	}
}

func TestCheckHostSafeSearchGoogle(t *testing.T) {
	d := NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()

	// Slice of google domains
	googleDomains := []string{"www.google.com", "www.google.im", "www.google.co.in", "www.google.iq", "www.google.is", "www.google.it", "www.google.je"}

	// Check host for each domain
	for _, host := range googleDomains {
		res, err := d.CheckHost(host, dns.TypeA, &setts)
		assert.Nil(t, err)
		assert.True(t, res.IsFiltered)
		if assert.Len(t, res.Rules, 1) {
			assert.NotEqual(t, res.Rules[0].IP.String(), "0.0.0.0")
		}
	}
}

func TestSafeSearchCacheYandex(t *testing.T) {
	d := NewForTest(nil, nil)
	defer d.Close()
	domain := "yandex.ru"

	// Check host with disabled safesearch.
	res, err := d.CheckHost(domain, dns.TypeA, &setts)
	assert.Nil(t, err)
	assert.False(t, res.IsFiltered)
	assert.Len(t, res.Rules, 0)

	d = NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()

	res, err = d.CheckHost(domain, dns.TypeA, &setts)
	if err != nil {
		t.Fatalf("CheckHost for safesearh domain %s failed cause %s", domain, err)
	}

	// For yandex we already know valid ip.
	if assert.Len(t, res.Rules, 1) {
		assert.Equal(t, res.Rules[0].IP.String(), "213.180.193.56")
	}

	// Check cache.
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, domain)
	assert.True(t, isFound)
	if assert.Len(t, cachedValue.Rules, 1) {
		assert.Equal(t, cachedValue.Rules[0].IP.String(), "213.180.193.56")
	}
}

func TestSafeSearchCacheGoogle(t *testing.T) {
	d := NewForTest(nil, nil)
	defer d.Close()
	domain := "www.google.ru"
	res, err := d.CheckHost(domain, dns.TypeA, &setts)
	assert.Nil(t, err)
	assert.False(t, res.IsFiltered)
	assert.Len(t, res.Rules, 0)

	d = NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()

	// Let's lookup for safesearch domain
	safeDomain, ok := d.SafeSearchDomain(domain)
	if !ok {
		t.Fatalf("Failed to get safesearch domain for %s", domain)
	}

	ips, err := net.LookupIP(safeDomain)
	if err != nil {
		t.Fatalf("Failed to lookup for %s", safeDomain)
	}

	ip := ips[0]
	for _, i := range ips {
		if i.To4() != nil {
			ip = i
			break
		}
	}

	res, err = d.CheckHost(domain, dns.TypeA, &setts)
	assert.Nil(t, err)
	if assert.Len(t, res.Rules, 1) {
		assert.True(t, res.Rules[0].IP.Equal(ip))
	}

	// Check cache.
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, domain)
	assert.True(t, isFound)
	if assert.Len(t, cachedValue.Rules, 1) {
		assert.True(t, cachedValue.Rules[0].IP.Equal(ip))
	}
}

// PARENTAL

func TestParentalControl(t *testing.T) {
	logOutput := &bytes.Buffer{}
	testutil.ReplaceLogWriter(t, logOutput)
	testutil.ReplaceLogLevel(t, log.DEBUG)

	d := NewForTest(&Config{ParentalEnabled: true}, nil)
	defer d.Close()
	d.checkMatch(t, "pornhub.com")
	assert.True(t, strings.Contains(logOutput.String(), "Parental lookup for pornhub.com"))
	d.checkMatch(t, "www.pornhub.com")
	d.checkMatchEmpty(t, "www.yandex.ru")
	d.checkMatchEmpty(t, "yandex.ru")
	d.checkMatchEmpty(t, "api.jquery.com")

	// test cached result
	d.parentalServer = "127.0.0.1"
	d.checkMatch(t, "pornhub.com")
	d.checkMatchEmpty(t, "yandex.ru")
	d.parentalServer = defaultParentalServer
}

// FILTERING

const nl = "\n"

const (
	blockingRules  = `||example.org^` + nl
	allowlistRules = `||example.org^` + nl + `@@||test.example.org` + nl
	importantRules = `@@||example.org^` + nl + `||test.example.org^$important` + nl
	regexRules     = `/example\.org/` + nl + `@@||test.example.org^` + nl
	maskRules      = `test*.example.org^` + nl + `exam*.com` + nl
	dnstypeRules   = `||example.org^$dnstype=AAAA` + nl + `@@||test.example.org^` + nl
)

var tests = []struct {
	testname   string
	rules      string
	hostname   string
	isFiltered bool
	reason     Reason
	dnsType    uint16
}{
	{"sanity", "||doubleclick.net^", "www.doubleclick.net", true, FilteredBlockList, dns.TypeA},
	{"sanity", "||doubleclick.net^", "nodoubleclick.net", false, NotFilteredNotFound, dns.TypeA},
	{"sanity", "||doubleclick.net^", "doubleclick.net.ru", false, NotFilteredNotFound, dns.TypeA},
	{"sanity", "||doubleclick.net^", "wmconvirus.narod.ru", false, NotFilteredNotFound, dns.TypeA},

	{"blocking", blockingRules, "example.org", true, FilteredBlockList, dns.TypeA},
	{"blocking", blockingRules, "test.example.org", true, FilteredBlockList, dns.TypeA},
	{"blocking", blockingRules, "test.test.example.org", true, FilteredBlockList, dns.TypeA},
	{"blocking", blockingRules, "testexample.org", false, NotFilteredNotFound, dns.TypeA},
	{"blocking", blockingRules, "onemoreexample.org", false, NotFilteredNotFound, dns.TypeA},

	{"allowlist", allowlistRules, "example.org", true, FilteredBlockList, dns.TypeA},
	{"allowlist", allowlistRules, "test.example.org", false, NotFilteredAllowList, dns.TypeA},
	{"allowlist", allowlistRules, "test.test.example.org", false, NotFilteredAllowList, dns.TypeA},
	{"allowlist", allowlistRules, "testexample.org", false, NotFilteredNotFound, dns.TypeA},
	{"allowlist", allowlistRules, "onemoreexample.org", false, NotFilteredNotFound, dns.TypeA},

	{"important", importantRules, "example.org", false, NotFilteredAllowList, dns.TypeA},
	{"important", importantRules, "test.example.org", true, FilteredBlockList, dns.TypeA},
	{"important", importantRules, "test.test.example.org", true, FilteredBlockList, dns.TypeA},
	{"important", importantRules, "testexample.org", false, NotFilteredNotFound, dns.TypeA},
	{"important", importantRules, "onemoreexample.org", false, NotFilteredNotFound, dns.TypeA},

	{"regex", regexRules, "example.org", true, FilteredBlockList, dns.TypeA},
	{"regex", regexRules, "test.example.org", false, NotFilteredAllowList, dns.TypeA},
	{"regex", regexRules, "test.test.example.org", false, NotFilteredAllowList, dns.TypeA},
	{"regex", regexRules, "testexample.org", true, FilteredBlockList, dns.TypeA},
	{"regex", regexRules, "onemoreexample.org", true, FilteredBlockList, dns.TypeA},

	{"mask", maskRules, "test.example.org", true, FilteredBlockList, dns.TypeA},
	{"mask", maskRules, "test2.example.org", true, FilteredBlockList, dns.TypeA},
	{"mask", maskRules, "example.com", true, FilteredBlockList, dns.TypeA},
	{"mask", maskRules, "exampleeee.com", true, FilteredBlockList, dns.TypeA},
	{"mask", maskRules, "onemoreexamsite.com", true, FilteredBlockList, dns.TypeA},
	{"mask", maskRules, "example.org", false, NotFilteredNotFound, dns.TypeA},
	{"mask", maskRules, "testexample.org", false, NotFilteredNotFound, dns.TypeA},
	{"mask", maskRules, "example.co.uk", false, NotFilteredNotFound, dns.TypeA},

	{"dnstype", dnstypeRules, "onemoreexample.org", false, NotFilteredNotFound, dns.TypeA},
	{"dnstype", dnstypeRules, "example.org", false, NotFilteredNotFound, dns.TypeA},
	{"dnstype", dnstypeRules, "example.org", true, FilteredBlockList, dns.TypeAAAA},
	{"dnstype", dnstypeRules, "test.example.org", false, NotFilteredAllowList, dns.TypeA},
	{"dnstype", dnstypeRules, "test.example.org", false, NotFilteredAllowList, dns.TypeAAAA},
}

func TestMatching(t *testing.T) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s-%s", test.testname, test.hostname), func(t *testing.T) {
			filters := []Filter{{
				ID: 0, Data: []byte(test.rules),
			}}
			d := NewForTest(nil, filters)
			defer d.Close()

			res, err := d.CheckHost(test.hostname, test.dnsType, &setts)
			if err != nil {
				t.Errorf("Error while matching host %s: %s", test.hostname, err)
			}
			if res.IsFiltered != test.isFiltered {
				t.Errorf("Hostname %s has wrong result (%v must be %v)", test.hostname, res.IsFiltered, test.isFiltered)
			}
			if res.Reason != test.reason {
				t.Errorf("Hostname %s has wrong reason (%v must be %v)", test.hostname, res.Reason.String(), test.reason.String())
			}
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
	d := NewForTest(nil, filters)
	d.SetFilters(filters, whiteFilters, false)
	defer d.Close()

	// matched by white filter
	res, err := d.CheckHost("host1", dns.TypeA, &setts)
	assert.True(t, err == nil)
	assert.True(t, !res.IsFiltered && res.Reason == NotFilteredAllowList)
	if assert.Len(t, res.Rules, 1) {
		assert.True(t, res.Rules[0].Text == "||host1^")
	}

	// not matched by white filter, but matched by block filter
	res, err = d.CheckHost("host2", dns.TypeA, &setts)
	assert.True(t, err == nil)
	assert.True(t, res.IsFiltered && res.Reason == FilteredBlockList)
	if assert.Len(t, res.Rules, 1) {
		assert.True(t, res.Rules[0].Text == "||host2^")
	}
}

// CLIENT SETTINGS

func applyClientSettings(setts *RequestFilteringSettings) {
	setts.FilteringEnabled = false
	setts.ParentalEnabled = false
	setts.SafeBrowsingEnabled = true

	rule, _ := rules.NewNetworkRule("||facebook.com^", 0)
	s := ServiceEntry{}
	s.Name = "facebook"
	s.Rules = []*rules.NetworkRule{rule}
	setts.ServicesRules = append(setts.ServicesRules, s)
}

// Check behaviour without any per-client settings,
//  then apply per-client settings and check behaviour once again
func TestClientSettings(t *testing.T) {
	var r Result
	filters := []Filter{{
		ID: 0, Data: []byte("||example.org^\n"),
	}}
	d := NewForTest(&Config{ParentalEnabled: true, SafeBrowsingEnabled: false}, filters)
	defer d.Close()

	// no client settings:

	// blocked by filters
	r, _ = d.CheckHost("example.org", dns.TypeA, &setts)
	if !r.IsFiltered || r.Reason != FilteredBlockList {
		t.Fatalf("CheckHost FilteredBlockList")
	}

	// blocked by parental
	r, _ = d.CheckHost("pornhub.com", dns.TypeA, &setts)
	if !r.IsFiltered || r.Reason != FilteredParental {
		t.Fatalf("CheckHost FilteredParental")
	}

	// safesearch is disabled
	r, _ = d.CheckHost("wmconvirus.narod.ru", dns.TypeA, &setts)
	if r.IsFiltered {
		t.Fatalf("CheckHost safesearch")
	}

	// not blocked
	r, _ = d.CheckHost("facebook.com", dns.TypeA, &setts)
	assert.True(t, !r.IsFiltered)

	// override client settings:
	applyClientSettings(&setts)

	// override filtering settings
	r, _ = d.CheckHost("example.org", dns.TypeA, &setts)
	if r.IsFiltered {
		t.Fatalf("CheckHost")
	}

	// override parental settings (force disable parental)
	r, _ = d.CheckHost("pornhub.com", dns.TypeA, &setts)
	if r.IsFiltered {
		t.Fatalf("CheckHost")
	}

	// override safesearch settings (force enable safesearch)
	r, _ = d.CheckHost("wmconvirus.narod.ru", dns.TypeA, &setts)
	if !r.IsFiltered || r.Reason != FilteredSafeBrowsing {
		t.Fatalf("CheckHost FilteredSafeBrowsing")
	}

	// blocked by additional rules
	r, _ = d.CheckHost("facebook.com", dns.TypeA, &setts)
	assert.True(t, r.IsFiltered && r.Reason == FilteredBlockedService)
}

// BENCHMARKS

func BenchmarkSafeBrowsing(b *testing.B) {
	d := NewForTest(&Config{SafeBrowsingEnabled: true}, nil)
	defer d.Close()
	for n := 0; n < b.N; n++ {
		hostname := "wmconvirus.narod.ru"
		res, err := d.CheckHost(hostname, dns.TypeA, &setts)
		if err != nil {
			b.Errorf("Error while matching host %s: %s", hostname, err)
		}
		if !res.IsFiltered {
			b.Errorf("Expected hostname %s to match", hostname)
		}
	}
}

func BenchmarkSafeBrowsingParallel(b *testing.B) {
	d := NewForTest(&Config{SafeBrowsingEnabled: true}, nil)
	defer d.Close()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			hostname := "wmconvirus.narod.ru"
			res, err := d.CheckHost(hostname, dns.TypeA, &setts)
			if err != nil {
				b.Errorf("Error while matching host %s: %s", hostname, err)
			}
			if !res.IsFiltered {
				b.Errorf("Expected hostname %s to match", hostname)
			}
		}
	})
}

func BenchmarkSafeSearch(b *testing.B) {
	d := NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()
	for n := 0; n < b.N; n++ {
		val, ok := d.SafeSearchDomain("www.google.com")
		if !ok {
			b.Errorf("Expected safesearch to find result for www.google.com")
		}
		if val != "forcesafesearch.google.com" {
			b.Errorf("Expected safesearch for google.com to be forcesafesearch.google.com")
		}
	}
}

func BenchmarkSafeSearchParallel(b *testing.B) {
	d := NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			val, ok := d.SafeSearchDomain("www.google.com")
			if !ok {
				b.Errorf("Expected safesearch to find result for www.google.com")
			}
			if val != "forcesafesearch.google.com" {
				b.Errorf("Expected safesearch for google.com to be forcesafesearch.google.com")
			}
		}
	})
}
