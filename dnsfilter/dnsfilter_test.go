package dnsfilter

import (
	"fmt"
	"net"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

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

func _Func() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}

func NewForTest(c *Config, filters []Filter) *Dnsfilter {
	setts = RequestFilteringSettings{}
	setts.FilteringEnabled = true
	if c != nil {
		c.SafeBrowsingCacheSize = 1000
		c.SafeSearchCacheSize = 1000
		c.ParentalCacheSize = 1000
		c.CacheTime = 30
		setts.SafeSearchEnabled = c.SafeSearchEnabled
		setts.SafeBrowsingEnabled = c.SafeBrowsingEnabled
		setts.ParentalEnabled = c.ParentalEnabled
	}
	d := New(c, filters)
	purgeCaches()
	return d
}

func (d *Dnsfilter) checkMatch(t *testing.T, hostname string) {
	t.Helper()
	ret, err := d.CheckHost(hostname, dns.TypeA, &setts)
	if err != nil {
		t.Errorf("Error while matching host %s: %s", hostname, err)
	}
	if !ret.IsFiltered {
		t.Errorf("Expected hostname %s to match", hostname)
	}
}

func (d *Dnsfilter) checkMatchIP(t *testing.T, hostname string, ip string, qtype uint16) {
	t.Helper()
	ret, err := d.CheckHost(hostname, qtype, &setts)
	if err != nil {
		t.Errorf("Error while matching host %s: %s", hostname, err)
	}
	if !ret.IsFiltered {
		t.Errorf("Expected hostname %s to match", hostname)
	}
	if ret.IP == nil || ret.IP.String() != ip {
		t.Errorf("Expected ip %s to match, actual: %v", ip, ret.IP)
	}
}

func (d *Dnsfilter) checkMatchEmpty(t *testing.T, hostname string) {
	t.Helper()
	ret, err := d.CheckHost(hostname, dns.TypeA, &setts)
	if err != nil {
		t.Errorf("Error while matching host %s: %s", hostname, err)
	}
	if ret.IsFiltered {
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
	filters := []Filter{Filter{
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
	ret, err := d.CheckHost("block.com", dns.TypeAAAA, &setts)
	assert.True(t, err == nil && ret.IsFiltered && ret.IP != nil && len(ret.IP) == 0)
	assert.True(t, ret.Rule == "0.0.0.0 block.com")

	// IPv6
	d.checkMatchIP(t, "ipv6.com", addr6, dns.TypeAAAA)

	// ...but empty IPv4
	ret, err = d.CheckHost("ipv6.com", dns.TypeA, &setts)
	assert.True(t, err == nil && ret.IsFiltered && ret.IP != nil && len(ret.IP) == 0)

	// 2 IPv4 (return only the first one)
	ret, err = d.CheckHost("host2", dns.TypeA, &setts)
	assert.True(t, err == nil && ret.IsFiltered)
	assert.True(t, ret.IP != nil && ret.IP.Equal(net.ParseIP("0.0.0.1")))

	// ...and 1 IPv6 address
	ret, err = d.CheckHost("host2", dns.TypeAAAA, &setts)
	assert.True(t, err == nil && ret.IsFiltered)
	assert.True(t, ret.IP != nil && ret.IP.Equal(net.ParseIP("::1")))
}

// SAFE BROWSING

func TestSafeBrowsing(t *testing.T) {
	d := NewForTest(&Config{SafeBrowsingEnabled: true}, nil)
	defer d.Close()
	gctx.stats.Safebrowsing.Requests = 0
	d.checkMatch(t, "wmconvirus.narod.ru")
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
		result, err := d.CheckHost(host, dns.TypeA, &setts)
		if err != nil {
			t.Errorf("SafeSearch doesn't work for yandex domain `%s` cause %s", host, err)
		}

		if result.IP.String() != "213.180.193.56" {
			t.Errorf("SafeSearch doesn't work for yandex domain `%s`", host)
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
		result, err := d.CheckHost(host, dns.TypeA, &setts)
		if err != nil {
			t.Errorf("SafeSearch doesn't work for %s cause %s", host, err)
		}

		if result.IP == nil {
			t.Errorf("SafeSearch doesn't work for %s", host)
		}
	}
}

func TestSafeSearchCacheYandex(t *testing.T) {
	d := NewForTest(nil, nil)
	defer d.Close()
	domain := "yandex.ru"

	var result Result
	var err error

	// Check host with disabled safesearch
	result, err = d.CheckHost(domain, dns.TypeA, &setts)
	if err != nil {
		t.Fatalf("Cannot check host due to %s", err)
	}
	if result.IP != nil {
		t.Fatalf("SafeSearch is not enabled but there is an answer for `%s` !", domain)
	}

	d = NewForTest(&Config{SafeSearchEnabled: true}, nil)
	defer d.Close()

	result, err = d.CheckHost(domain, dns.TypeA, &setts)
	if err != nil {
		t.Fatalf("CheckHost for safesearh domain %s failed cause %s", domain, err)
	}

	// Fir yandex we already know valid ip
	if result.IP.String() != "213.180.193.56" {
		t.Fatalf("Wrong IP for %s safesearch: %s", domain, result.IP.String())
	}

	// Check cache
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, domain)

	if !isFound {
		t.Fatalf("Safesearch cache doesn't work for %s!", domain)
	}

	if cachedValue.IP.String() != "213.180.193.56" {
		t.Fatalf("Wrong IP in cache for %s safesearch: %s", domain, cachedValue.IP.String())
	}
}

func TestSafeSearchCacheGoogle(t *testing.T) {
	d := NewForTest(nil, nil)
	defer d.Close()
	domain := "www.google.ru"
	result, err := d.CheckHost(domain, dns.TypeA, &setts)
	if err != nil {
		t.Fatalf("Cannot check host due to %s", err)
	}
	if result.IP != nil {
		t.Fatalf("SafeSearch is not enabled but there is an answer!")
	}

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

	t.Logf("IP addresses: %v", ips)
	ip := ips[0]
	for _, i := range ips {
		if i.To4() != nil {
			ip = i
			break
		}
	}

	result, err = d.CheckHost(domain, dns.TypeA, &setts)
	if err != nil {
		t.Fatalf("CheckHost for safesearh domain %s failed cause %s", domain, err)
	}

	if result.IP.String() != ip.String() {
		t.Fatalf("Wrong IP for %s safesearch: %s.  Should be: %s",
			domain, result.IP.String(), ip)
	}

	// Check cache
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, domain)

	if !isFound {
		t.Fatalf("Safesearch cache doesn't work for %s!", domain)
	}

	if cachedValue.IP.String() != ip.String() {
		t.Fatalf("Wrong IP in cache for %s safesearch: %s", domain, cachedValue.IP.String())
	}
}

// PARENTAL

func TestParentalControl(t *testing.T) {
	d := NewForTest(&Config{ParentalEnabled: true}, nil)
	defer d.Close()
	d.checkMatch(t, "pornhub.com")
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

var blockingRules = "||example.org^\n"
var whitelistRules = "||example.org^\n@@||test.example.org\n"
var importantRules = "@@||example.org^\n||test.example.org^$important\n"
var regexRules = "/example\\.org/\n@@||test.example.org^\n"
var maskRules = "test*.example.org^\nexam*.com\n"

var tests = []struct {
	testname   string
	rules      string
	hostname   string
	isFiltered bool
	reason     Reason
}{
	{"sanity", "||doubleclick.net^", "www.doubleclick.net", true, FilteredBlackList},
	{"sanity", "||doubleclick.net^", "nodoubleclick.net", false, NotFilteredNotFound},
	{"sanity", "||doubleclick.net^", "doubleclick.net.ru", false, NotFilteredNotFound},
	{"sanity", "||doubleclick.net^", "wmconvirus.narod.ru", false, NotFilteredNotFound},

	{"blocking", blockingRules, "example.org", true, FilteredBlackList},
	{"blocking", blockingRules, "test.example.org", true, FilteredBlackList},
	{"blocking", blockingRules, "test.test.example.org", true, FilteredBlackList},
	{"blocking", blockingRules, "testexample.org", false, NotFilteredNotFound},
	{"blocking", blockingRules, "onemoreexample.org", false, NotFilteredNotFound},

	{"whitelist", whitelistRules, "example.org", true, FilteredBlackList},
	{"whitelist", whitelistRules, "test.example.org", false, NotFilteredWhiteList},
	{"whitelist", whitelistRules, "test.test.example.org", false, NotFilteredWhiteList},
	{"whitelist", whitelistRules, "testexample.org", false, NotFilteredNotFound},
	{"whitelist", whitelistRules, "onemoreexample.org", false, NotFilteredNotFound},

	{"important", importantRules, "example.org", false, NotFilteredWhiteList},
	{"important", importantRules, "test.example.org", true, FilteredBlackList},
	{"important", importantRules, "test.test.example.org", true, FilteredBlackList},
	{"important", importantRules, "testexample.org", false, NotFilteredNotFound},
	{"important", importantRules, "onemoreexample.org", false, NotFilteredNotFound},

	{"regex", regexRules, "example.org", true, FilteredBlackList},
	{"regex", regexRules, "test.example.org", false, NotFilteredWhiteList},
	{"regex", regexRules, "test.test.example.org", false, NotFilteredWhiteList},
	{"regex", regexRules, "testexample.org", true, FilteredBlackList},
	{"regex", regexRules, "onemoreexample.org", true, FilteredBlackList},

	{"mask", maskRules, "test.example.org", true, FilteredBlackList},
	{"mask", maskRules, "test2.example.org", true, FilteredBlackList},
	{"mask", maskRules, "example.com", true, FilteredBlackList},
	{"mask", maskRules, "exampleeee.com", true, FilteredBlackList},
	{"mask", maskRules, "onemoreexamsite.com", true, FilteredBlackList},
	{"mask", maskRules, "example.org", false, NotFilteredNotFound},
	{"mask", maskRules, "testexample.org", false, NotFilteredNotFound},
	{"mask", maskRules, "example.co.uk", false, NotFilteredNotFound},
}

func TestMatching(t *testing.T) {
	for _, test := range tests {
		t.Run(fmt.Sprintf("%s-%s", test.testname, test.hostname), func(t *testing.T) {
			filters := []Filter{Filter{
				ID: 0, Data: []byte(test.rules),
			}}
			d := NewForTest(nil, filters)
			defer d.Close()

			ret, err := d.CheckHost(test.hostname, dns.TypeA, &setts)
			if err != nil {
				t.Errorf("Error while matching host %s: %s", test.hostname, err)
			}
			if ret.IsFiltered != test.isFiltered {
				t.Errorf("Hostname %s has wrong result (%v must be %v)", test.hostname, ret.IsFiltered, test.isFiltered)
			}
			if ret.Reason != test.reason {
				t.Errorf("Hostname %s has wrong reason (%v must be %v)", test.hostname, ret.Reason.String(), test.reason.String())
			}
		})
	}
}

func TestWhitelist(t *testing.T) {
	rules := `||host1^
||host2^
`
	filters := []Filter{Filter{
		ID: 0, Data: []byte(rules),
	}}

	whiteRules := `||host1^
||host3^
`
	whiteFilters := []Filter{Filter{
		ID: 0, Data: []byte(whiteRules),
	}}
	d := NewForTest(nil, filters)
	d.SetFilters(filters, whiteFilters, false)
	defer d.Close()

	// matched by white filter
	ret, err := d.CheckHost("host1", dns.TypeA, &setts)
	assert.True(t, err == nil)
	assert.True(t, !ret.IsFiltered && ret.Reason == NotFilteredWhiteList)
	assert.True(t, ret.Rule == "||host1^")

	// not matched by white filter, but matched by block filter
	ret, err = d.CheckHost("host2", dns.TypeA, &setts)
	assert.True(t, err == nil)
	assert.True(t, ret.IsFiltered && ret.Reason == FilteredBlackList)
	assert.True(t, ret.Rule == "||host2^")

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
	filters := []Filter{Filter{
		ID: 0, Data: []byte("||example.org^\n"),
	}}
	d := NewForTest(&Config{ParentalEnabled: true, SafeBrowsingEnabled: false}, filters)
	defer d.Close()

	// no client settings:

	// blocked by filters
	r, _ = d.CheckHost("example.org", dns.TypeA, &setts)
	if !r.IsFiltered || r.Reason != FilteredBlackList {
		t.Fatalf("CheckHost FilteredBlackList")
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

func prepareTestDir() string {
	const dir = "./agh-test"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// BENCHMARKS

func BenchmarkSafeBrowsing(b *testing.B) {
	d := NewForTest(&Config{SafeBrowsingEnabled: true}, nil)
	defer d.Close()
	for n := 0; n < b.N; n++ {
		hostname := "wmconvirus.narod.ru"
		ret, err := d.CheckHost(hostname, dns.TypeA, &setts)
		if err != nil {
			b.Errorf("Error while matching host %s: %s", hostname, err)
		}
		if !ret.IsFiltered {
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
			ret, err := d.CheckHost(hostname, dns.TypeA, &setts)
			if err != nil {
				b.Errorf("Error while matching host %s: %s", hostname, err)
			}
			if !ret.IsFiltered {
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
