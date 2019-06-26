package dnsfilter

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/miekg/dns"
)

// HELPERS
// SAFE BROWSING
// SAFE SEARCH
// PARENTAL
// FILTERING
// CLIENTS SETTINGS
// BENCHMARKS

// HELPERS

func purgeCaches() {
	if safebrowsingCache != nil {
		safebrowsingCache.Purge()
	}
	if parentalCache != nil {
		parentalCache.Purge()
	}
}

func _Func() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}

func NewForTest() *Dnsfilter {
	d := New(nil, nil)
	purgeCaches()
	return d
}

func NewForTestFilters(filters map[int]string) *Dnsfilter {
	d := New(nil, filters)
	purgeCaches()
	return d
}

func (d *Dnsfilter) checkMatch(t *testing.T, hostname string) {
	t.Helper()
	ret, err := d.CheckHost(hostname, dns.TypeA, "")
	if err != nil {
		t.Errorf("Error while matching host %s: %s", hostname, err)
	}
	if !ret.IsFiltered {
		t.Errorf("Expected hostname %s to match", hostname)
	}
}

func (d *Dnsfilter) checkMatchIP(t *testing.T, hostname string, ip string, qtype uint16) {
	t.Helper()
	ret, err := d.CheckHost(hostname, qtype, "")
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
	ret, err := d.CheckHost(hostname, dns.TypeA, "")
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
	text := fmt.Sprintf("   %s  google.com www.google.com   # enforce google's safesearch   \n%s  google.com\n0.0.0.0 block.com\n",
		addr, addr6)
	filters := make(map[int]string)
	filters[0] = text
	d := NewForTestFilters(filters)
	defer d.Destroy()

	d.checkMatchIP(t, "google.com", addr, dns.TypeA)
	d.checkMatchIP(t, "www.google.com", addr, dns.TypeA)
	d.checkMatchEmpty(t, "subdomain.google.com")
	d.checkMatchEmpty(t, "example.org")

	// IPv6 address
	d.checkMatchIP(t, "google.com", addr6, dns.TypeAAAA)

	// block both IPv4 and IPv6
	d.checkMatchIP(t, "block.com", "0.0.0.0", dns.TypeA)
	d.checkMatchIP(t, "block.com", "::", dns.TypeAAAA)
}

// SAFE BROWSING

func TestSafeBrowsing(t *testing.T) {
	testCases := []string{
		"",
		"sb.adtidy.org",
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s in %s", tc, _Func()), func(t *testing.T) {
			d := NewForTest()
			defer d.Destroy()
			d.SafeBrowsingEnabled = true
			stats.Safebrowsing.Requests = 0
			d.checkMatch(t, "wmconvirus.narod.ru")
			d.checkMatch(t, "wmconvirus.narod.ru")
			if stats.Safebrowsing.Requests != 1 {
				t.Errorf("Safebrowsing lookup positive cache is not working: %v", stats.Safebrowsing.Requests)
			}
			d.checkMatch(t, "WMconvirus.narod.ru")
			if stats.Safebrowsing.Requests != 1 {
				t.Errorf("Safebrowsing lookup positive cache is not working: %v", stats.Safebrowsing.Requests)
			}
			d.checkMatch(t, "wmconvirus.narod.ru.")
			d.checkMatch(t, "test.wmconvirus.narod.ru")
			d.checkMatch(t, "test.wmconvirus.narod.ru.")
			d.checkMatchEmpty(t, "yandex.ru")
			d.checkMatchEmpty(t, "pornhub.com")
			l := stats.Safebrowsing.Requests
			d.checkMatchEmpty(t, "pornhub.com")
			if stats.Safebrowsing.Requests != l {
				t.Errorf("Safebrowsing lookup negative cache is not working: %v", stats.Safebrowsing.Requests)
			}
		})
	}
}

func TestParallelSB(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()
	d.SafeBrowsingEnabled = true
	t.Run("group", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			t.Run(fmt.Sprintf("aaa%d", i), func(t *testing.T) {
				t.Parallel()
				d.checkMatch(t, "wmconvirus.narod.ru")
				d.checkMatch(t, "wmconvirus.narod.ru.")
				d.checkMatch(t, "test.wmconvirus.narod.ru")
				d.checkMatch(t, "test.wmconvirus.narod.ru.")
				d.checkMatchEmpty(t, "yandex.ru")
				d.checkMatchEmpty(t, "pornhub.com")
			})
		}
	})
}

// the only way to verify that custom server option is working is to point it at a server that does serve safebrowsing
func TestSafeBrowsingCustomServerFail(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// w.Write("Hello, client")
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()
	address := ts.Listener.Addr().String()

	d.SafeBrowsingEnabled = true
	d.SetHTTPTimeout(time.Second * 5)
	d.SetSafeBrowsingServer(address) // this will ensure that test fails
	d.checkMatchEmpty(t, "wmconvirus.narod.ru")
}

// SAFE SEARCH

func TestSafeSearch(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()
	_, ok := d.SafeSearchDomain("www.google.com")
	if ok {
		t.Errorf("Expected safesearch to error when disabled")
	}
	d.SafeSearchEnabled = true
	val, ok := d.SafeSearchDomain("www.google.com")
	if !ok {
		t.Errorf("Expected safesearch to find result for www.google.com")
	}
	if val != "forcesafesearch.google.com" {
		t.Errorf("Expected safesearch for google.com to be forcesafesearch.google.com")
	}
}

func TestCheckHostSafeSearchYandex(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()

	// Enable safesearch
	d.SafeSearchEnabled = true

	// Slice of yandex domains
	yandex := []string{"yAndeX.ru", "YANdex.COM", "yandex.ua", "yandex.by", "yandex.kz", "www.yandex.com"}

	// Check host for each domain
	for _, host := range yandex {
		result, err := d.CheckHost(host, dns.TypeA, "")
		if err != nil {
			t.Errorf("SafeSearch doesn't work for yandex domain `%s` cause %s", host, err)
		}

		if result.IP.String() != "213.180.193.56" {
			t.Errorf("SafeSearch doesn't work for yandex domain `%s`", host)
		}
	}
}

func TestCheckHostSafeSearchGoogle(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()

	// Enable safesearch
	d.SafeSearchEnabled = true

	// Slice of google domains
	googleDomains := []string{"www.google.com", "www.google.im", "www.google.co.in", "www.google.iq", "www.google.is", "www.google.it", "www.google.je"}

	// Check host for each domain
	for _, host := range googleDomains {
		result, err := d.CheckHost(host, dns.TypeA, "")
		if err != nil {
			t.Errorf("SafeSearch doesn't work for %s cause %s", host, err)
		}

		if result.IP == nil {
			t.Errorf("SafeSearch doesn't work for %s", host)
		}
	}
}

func TestSafeSearchCacheYandex(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()
	domain := "yandex.ru"

	var result Result
	var err error

	// Check host with disabled safesearch
	result, err = d.CheckHost(domain, dns.TypeA, "")
	if err != nil {
		t.Fatalf("Cannot check host due to %s", err)
	}
	if result.IP != nil {
		t.Fatalf("SafeSearch is not enabled but there is an answer for `%s` !", domain)
	}

	// Enable safesearch
	d.SafeSearchEnabled = true
	result, err = d.CheckHost(domain, dns.TypeA, "")
	if err != nil {
		t.Fatalf("CheckHost for safesearh domain %s failed cause %s", domain, err)
	}

	// Fir yandex we already know valid ip
	if result.IP.String() != "213.180.193.56" {
		t.Fatalf("Wrong IP for %s safesearch: %s", domain, result.IP.String())
	}

	// Check cache
	cachedValue, isFound, err := getCachedReason(safeSearchCache, domain)

	if err != nil {
		t.Fatalf("An error occured during cache search for %s: %s", domain, err)
	}

	if !isFound {
		t.Fatalf("Safesearch cache doesn't work for %s!", domain)
	}

	if cachedValue.IP.String() != "213.180.193.56" {
		t.Fatalf("Wrong IP in cache for %s safesearch: %s", domain, cachedValue.IP.String())
	}
}

func TestSafeSearchCacheGoogle(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()
	domain := "www.google.ru"
	result, err := d.CheckHost(domain, dns.TypeA, "")
	if err != nil {
		t.Fatalf("Cannot check host due to %s", err)
	}
	if result.IP != nil {
		t.Fatalf("SafeSearch is not enabled but there is an answer!")
	}

	// Enable safesearch and check host
	d.SafeSearchEnabled = true

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

	result, err = d.CheckHost(domain, dns.TypeA, "")
	if err != nil {
		t.Fatalf("CheckHost for safesearh domain %s failed cause %s", domain, err)
	}

	if result.IP.String() != ip.String() {
		t.Fatalf("Wrong IP for %s safesearch: %s.  Should be: %s",
			domain, result.IP.String(), ip)
	}

	// Check cache
	cachedValue, isFound, err := getCachedReason(safeSearchCache, domain)

	if err != nil {
		t.Fatalf("An error occured during cache search for %s: %s", domain, err)
	}

	if !isFound {
		t.Fatalf("Safesearch cache doesn't work for %s!", domain)
	}

	if cachedValue.IP.String() != ip.String() {
		t.Fatalf("Wrong IP in cache for %s safesearch: %s", domain, cachedValue.IP.String())
	}
}

// PARENTAL

func TestParentalControl(t *testing.T) {
	d := NewForTest()
	defer d.Destroy()
	d.ParentalEnabled = true
	d.ParentalSensitivity = 3
	d.checkMatch(t, "pornhub.com")
	d.checkMatch(t, "pornhub.com")
	if stats.Parental.Requests != 1 {
		t.Errorf("Parental lookup positive cache is not working")
	}
	d.checkMatch(t, "PORNhub.com")
	if stats.Parental.Requests != 1 {
		t.Errorf("Parental lookup positive cache is not working")
	}
	d.checkMatch(t, "www.pornhub.com")
	d.checkMatch(t, "pornhub.com.")
	d.checkMatch(t, "www.pornhub.com.")
	d.checkMatchEmpty(t, "www.yandex.ru")
	d.checkMatchEmpty(t, "yandex.ru")
	l := stats.Parental.Requests
	d.checkMatchEmpty(t, "yandex.ru")
	if stats.Parental.Requests != l {
		t.Errorf("Parental lookup negative cache is not working")
	}

	d.checkMatchEmpty(t, "api.jquery.com")
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
			filters := make(map[int]string)
			filters[0] = test.rules
			d := NewForTestFilters(filters)
			defer d.Destroy()

			ret, err := d.CheckHost(test.hostname, dns.TypeA, "")
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

// CLIENT SETTINGS

func applyClientSettings(clientAddr string, setts *RequestFilteringSettings) {
	setts.FilteringEnabled = false
	setts.ParentalEnabled = false
}

func TestClientSettings(t *testing.T) {
	var r Result
	filters := make(map[int]string)
	filters[0] = "||example.org^\n"
	d := NewForTestFilters(filters)
	defer d.Destroy()
	d.ParentalEnabled = true
	d.ParentalSensitivity = 3

	// no client settings:

	// blocked by filters
	r, _ = d.CheckHost("example.org", dns.TypeA, "1.1.1.1")
	if !r.IsFiltered || r.Reason != FilteredBlackList {
		t.Fatalf("CheckHost FilteredBlackList")
	}

	// blocked by parental
	r, _ = d.CheckHost("pornhub.com", dns.TypeA, "1.1.1.1")
	if !r.IsFiltered || r.Reason != FilteredParental {
		t.Fatalf("CheckHost FilteredParental")
	}

	// override client settings:
	d.FilterHandler = applyClientSettings

	// override filtering settings
	r, _ = d.CheckHost("example.org", dns.TypeA, "1.1.1.1")
	if r.IsFiltered {
		t.Fatalf("CheckHost")
	}

	// override parental settings
	r, _ = d.CheckHost("pornhub.com", dns.TypeA, "1.1.1.1")
	if r.IsFiltered {
		t.Fatalf("CheckHost")
	}
}

// BENCHMARKS

func BenchmarkSafeBrowsing(b *testing.B) {
	d := NewForTest()
	defer d.Destroy()
	d.SafeBrowsingEnabled = true
	for n := 0; n < b.N; n++ {
		hostname := "wmconvirus.narod.ru"
		ret, err := d.CheckHost(hostname, dns.TypeA, "")
		if err != nil {
			b.Errorf("Error while matching host %s: %s", hostname, err)
		}
		if !ret.IsFiltered {
			b.Errorf("Expected hostname %s to match", hostname)
		}
	}
}

func BenchmarkSafeBrowsingParallel(b *testing.B) {
	d := NewForTest()
	defer d.Destroy()
	d.SafeBrowsingEnabled = true
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			hostname := "wmconvirus.narod.ru"
			ret, err := d.CheckHost(hostname, dns.TypeA, "")
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
	d := NewForTest()
	defer d.Destroy()
	d.SafeSearchEnabled = true
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
	d := NewForTest()
	defer d.Destroy()
	d.SafeSearchEnabled = true
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

func TestDnsfilterDialCache(t *testing.T) {
	d := Dnsfilter{}
	dialCache = gcache.New(1).LRU().Expiration(30 * time.Minute).Build()

	d.shouldBeInDialCache("hostname")
	if searchInDialCache("hostname") != "" {
		t.Errorf("searchInDialCache")
	}
	addToDialCache("hostname", "1.1.1.1")
	if searchInDialCache("hostname") != "1.1.1.1" {
		t.Errorf("searchInDialCache")
	}
}
