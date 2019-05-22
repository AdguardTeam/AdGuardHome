package dnsfilter

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/bluele/gcache"
	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

const defaultCacheSize = 64 * 1024 // in number of elements
const defaultCacheTime = 30 * time.Minute

const defaultHTTPTimeout = 5 * time.Minute
const defaultHTTPMaxIdleConnections = 100

const defaultSafebrowsingServer = "sb.adtidy.org"
const defaultSafebrowsingURL = "https://%s/safebrowsing-lookup-hash.html?prefixes=%s"
const defaultParentalServer = "pctrl.adguard.com"
const defaultParentalURL = "https://%s/check-parental-control-hash?prefixes=%s&sensitivity=%d"
const maxDialCacheSize = 2 // the number of host names for safebrowsing and parental control

// ErrInvalidSyntax is returned by AddRule when the rule is invalid
var ErrInvalidSyntax = errors.New("dnsfilter: invalid rule syntax")

// ErrAlreadyExists is returned by AddRule when the rule was already added to the filter
var ErrAlreadyExists = errors.New("dnsfilter: rule was already added")

const shortcutLength = 6 // used for rule search optimization, 6 hits the sweet spot

const enableFastLookup = true         // flag for debugging, must be true in production for faster performance
const enableDelayedCompilation = true // flag for debugging, must be true in production for faster performance

// Config allows you to configure DNS filtering with New() or just change variables directly.
type Config struct {
	FilteringTempFilename string `yaml:"filtering_temp_filename"` // temporary file for storing unused filtering rules
	ParentalSensitivity   int    `yaml:"parental_sensitivity"`    // must be either 3, 10, 13 or 17
	ParentalEnabled       bool   `yaml:"parental_enabled"`
	SafeSearchEnabled     bool   `yaml:"safesearch_enabled"`
	SafeBrowsingEnabled   bool   `yaml:"safebrowsing_enabled"`
	ResolverAddress       string // DNS server address
}

type privateConfig struct {
	parentalServer     string // access via methods
	safeBrowsingServer string // access via methods
}

// LookupStats store stats collected during safebrowsing or parental checks
type LookupStats struct {
	Requests   uint64 // number of HTTP requests that were sent
	CacheHits  uint64 // number of lookups that didn't need HTTP requests
	Pending    int64  // number of currently pending HTTP requests
	PendingMax int64  // maximum number of pending HTTP requests
}

// Stats store LookupStats for safebrowsing, parental and safesearch
type Stats struct {
	Safebrowsing LookupStats
	Parental     LookupStats
	Safesearch   LookupStats
}

// Dnsfilter holds added rules and performs hostname matches against the rules
type Dnsfilter struct {
	rulesStorage    *urlfilter.RulesStorage
	filteringEngine *urlfilter.DNSEngine

	// HTTP lookups for safebrowsing and parental
	client    http.Client     // handle for http client -- single instance as recommended by docs
	transport *http.Transport // handle for http transport used by http client

	Config // for direct access by library users, even a = assignment
	privateConfig
}

// Filter represents a filter list
type Filter struct {
	ID   int64  `json:"id"`         // auto-assigned when filter is added (see nextFilterID), json by default keeps ID uppercase but we need lowercase
	Data []byte `json:"-" yaml:"-"` // List of rules divided by '\n'
}

//go:generate stringer -type=Reason

// Reason holds an enum detailing why it was filtered or not filtered
type Reason int

const (
	// reasons for not filtering

	// NotFilteredNotFound - host was not find in any checks, default value for result
	NotFilteredNotFound Reason = iota
	// NotFilteredWhiteList - the host is explicitly whitelisted
	NotFilteredWhiteList
	// NotFilteredError - there was a transitive error during check
	NotFilteredError

	// reasons for filtering

	// FilteredBlackList - the host was matched to be advertising host
	FilteredBlackList
	// FilteredSafeBrowsing - the host was matched to be malicious/phishing
	FilteredSafeBrowsing
	// FilteredParental - the host was matched to be outside of parental control settings
	FilteredParental
	// FilteredInvalid - the request was invalid and was not processed
	FilteredInvalid
	// FilteredSafeSearch - the host was replaced with safesearch variant
	FilteredSafeSearch
)

// these variables need to survive coredns reload
var (
	stats             Stats
	dialCache         gcache.Cache // "host" -> "IP" cache for safebrowsing and parental control servers
	safebrowsingCache gcache.Cache
	parentalCache     gcache.Cache
	safeSearchCache   gcache.Cache
)

// Result holds state of hostname check
type Result struct {
	IsFiltered bool   `json:",omitempty"` // True if the host name is filtered
	Reason     Reason `json:",omitempty"` // Reason for blocking / unblocking
	Rule       string `json:",omitempty"` // Original rule text
	IP         net.IP `json:",omitempty"` // Not nil only in the case of a hosts file syntax
	FilterID   int64  `json:",omitempty"` // Filter ID the rule belongs to
}

// Matched can be used to see if any match at all was found, no matter filtered or not
func (r Reason) Matched() bool {
	return r != NotFilteredNotFound
}

// CheckHost tries to match host against rules, then safebrowsing and parental if they are enabled
func (d *Dnsfilter) CheckHost(host string, qtype uint16) (Result, error) {
	// sometimes DNS clients will try to resolve ".", which is a request to get root servers
	if host == "" {
		return Result{Reason: NotFilteredNotFound}, nil
	}
	host = strings.ToLower(host)
	// prevent recursion
	if host == d.parentalServer || host == d.safeBrowsingServer {
		return Result{}, nil
	}

	// try filter lists first
	result, err := d.matchHost(host, qtype)
	if err != nil {
		return result, err
	}
	if result.Reason.Matched() {
		return result, nil
	}

	// check safeSearch if no match
	if d.SafeSearchEnabled {
		result, err = d.checkSafeSearch(host)
		if err != nil {
			log.Printf("Failed to safesearch HTTP lookup, ignoring check: %v", err)
			return Result{}, nil
		}

		if result.Reason.Matched() {
			return result, nil
		}
	}

	// check safebrowsing if no match
	if d.SafeBrowsingEnabled {
		result, err = d.checkSafeBrowsing(host)
		if err != nil {
			// failed to do HTTP lookup -- treat it as if we got empty response, but don't save cache
			log.Printf("Failed to do safebrowsing HTTP lookup, ignoring check: %v", err)
			return Result{}, nil
		}
		if result.Reason.Matched() {
			return result, nil
		}
	}

	// check parental if no match
	if d.ParentalEnabled {
		result, err = d.checkParental(host)
		if err != nil {
			// failed to do HTTP lookup -- treat it as if we got empty response, but don't save cache
			log.Printf("Failed to do parental HTTP lookup, ignoring check: %v", err)
			return Result{}, nil
		}
		if result.Reason.Matched() {
			return result, nil
		}
	}

	// nothing matched, return nothing
	return Result{}, nil
}

func getCachedReason(cache gcache.Cache, host string) (result Result, isFound bool, err error) {
	isFound = false // not found yet

	// get raw value
	rawValue, err := cache.Get(host)
	if err == gcache.KeyNotFoundError {
		// not a real error, just not found
		err = nil
		return
	}
	if err != nil {
		// real error
		return
	}

	// since it can be something else, validate that it belongs to proper type
	cachedValue, ok := rawValue.(Result)
	if !ok {
		// this is not our type -- error
		text := "SHOULD NOT HAPPEN: entry with invalid type was found in lookup cache"
		log.Println(text)
		err = errors.New(text)
		return
	}
	isFound = ok
	return cachedValue, isFound, err
}

// for each dot, hash it and add it to string
func hostnameToHashParam(host string, addslash bool) (string, map[string]bool) {
	var hashparam bytes.Buffer
	hashes := map[string]bool{}
	tld, icann := publicsuffix.PublicSuffix(host)
	if !icann {
		// private suffixes like cloudfront.net
		tld = ""
	}
	curhost := host
	for {
		if curhost == "" {
			// we've reached end of string
			break
		}
		if tld != "" && curhost == tld {
			// we've reached the TLD, don't hash it
			break
		}
		tohash := []byte(curhost)
		if addslash {
			tohash = append(tohash, '/')
		}
		sum := sha256.Sum256(tohash)
		hexhash := fmt.Sprintf("%X", sum)
		hashes[hexhash] = true
		hashparam.WriteString(fmt.Sprintf("%02X%02X%02X%02X/", sum[0], sum[1], sum[2], sum[3]))
		pos := strings.IndexByte(curhost, byte('.'))
		if pos < 0 {
			break
		}
		curhost = curhost[pos+1:]
	}
	return hashparam.String(), hashes
}

func (d *Dnsfilter) checkSafeSearch(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("SafeSearch HTTP lookup for %s", host)
	}

	if safeSearchCache == nil {
		safeSearchCache = gcache.New(defaultCacheSize).LRU().Expiration(defaultCacheTime).Build()
	}

	// Check cache. Return cached result if it was found
	cachedValue, isFound, err := getCachedReason(safeSearchCache, host)
	if isFound {
		atomic.AddUint64(&stats.Safesearch.CacheHits, 1)
		log.Tracef("%s: found in SafeSearch cache", host)
		return cachedValue, nil
	}

	if err != nil {
		return Result{}, err
	}

	safeHost, ok := d.SafeSearchDomain(host)
	if !ok {
		return Result{}, nil
	}

	res := Result{IsFiltered: true, Reason: FilteredSafeSearch}
	if ip := net.ParseIP(safeHost); ip != nil {
		res.IP = ip
		err = safeSearchCache.Set(host, res)
		if err != nil {
			return Result{}, nil
		}

		return res, nil
	}

	// TODO this address should be resolved with upstream that was configured in dnsforward
	addrs, err := net.LookupIP(safeHost)
	if err != nil {
		log.Tracef("SafeSearchDomain for %s was found but failed to lookup for %s cause %s", host, safeHost, err)
		return Result{}, err
	}

	for _, i := range addrs {
		if ipv4 := i.To4(); ipv4 != nil {
			res.IP = ipv4
			break
		}
	}

	if len(res.IP) == 0 {
		return Result{}, fmt.Errorf("no ipv4 addresses in safe search response for %s", safeHost)
	}

	// Cache result
	err = safeSearchCache.Set(host, res)
	if err != nil {
		return Result{}, nil
	}
	return res, nil
}

func (d *Dnsfilter) checkSafeBrowsing(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("SafeBrowsing HTTP lookup for %s", host)
	}

	format := func(hashparam string) string {
		url := fmt.Sprintf(defaultSafebrowsingURL, d.safeBrowsingServer, hashparam)
		return url
	}
	handleBody := func(body []byte, hashes map[string]bool) (Result, error) {
		result := Result{}
		scanner := bufio.NewScanner(strings.NewReader(string(body)))
		for scanner.Scan() {
			line := scanner.Text()
			splitted := strings.Split(line, ":")
			if len(splitted) < 3 {
				continue
			}
			hash := splitted[2]
			if _, ok := hashes[hash]; ok {
				// it's in the hash
				result.IsFiltered = true
				result.Reason = FilteredSafeBrowsing
				result.Rule = splitted[0]
				break
			}
		}

		if err := scanner.Err(); err != nil {
			// error, don't save cache
			return Result{}, err
		}
		return result, nil
	}
	if safebrowsingCache == nil {
		safebrowsingCache = gcache.New(defaultCacheSize).LRU().Expiration(defaultCacheTime).Build()
	}
	result, err := d.lookupCommon(host, &stats.Safebrowsing, safebrowsingCache, true, format, handleBody)
	return result, err
}

func (d *Dnsfilter) checkParental(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("Parental HTTP lookup for %s", host)
	}

	format := func(hashparam string) string {
		url := fmt.Sprintf(defaultParentalURL, d.parentalServer, hashparam, d.ParentalSensitivity)
		return url
	}
	handleBody := func(body []byte, hashes map[string]bool) (Result, error) {
		// parse json
		var m []struct {
			Blocked   bool   `json:"blocked"`
			ClientTTL int    `json:"clientTtl"`
			Reason    string `json:"reason"`
			Hash      string `json:"hash"`
		}
		err := json.Unmarshal(body, &m)
		if err != nil {
			// error, don't save cache
			log.Printf("Couldn't parse json '%s': %s", body, err)
			return Result{}, err
		}

		result := Result{}

		for i := range m {
			if !hashes[m[i].Hash] {
				continue
			}
			if m[i].Blocked {
				result.IsFiltered = true
				result.Reason = FilteredParental
				result.Rule = fmt.Sprintf("parental %s", m[i].Reason)
				break
			}
		}
		return result, nil
	}
	if parentalCache == nil {
		parentalCache = gcache.New(defaultCacheSize).LRU().Expiration(defaultCacheTime).Build()
	}
	result, err := d.lookupCommon(host, &stats.Parental, parentalCache, false, format, handleBody)
	return result, err
}

type formatHandler func(hashparam string) string
type bodyHandler func(body []byte, hashes map[string]bool) (Result, error)

// real implementation of lookup/check
func (d *Dnsfilter) lookupCommon(host string, lookupstats *LookupStats, cache gcache.Cache, hashparamNeedSlash bool, format formatHandler, handleBody bodyHandler) (Result, error) {
	// if host ends with a dot, trim it
	host = strings.ToLower(strings.Trim(host, "."))

	// check cache
	cachedValue, isFound, err := getCachedReason(cache, host)
	if isFound {
		atomic.AddUint64(&lookupstats.CacheHits, 1)
		log.Tracef("%s: found in the lookup cache", host)
		return cachedValue, nil
	}
	if err != nil {
		return Result{}, err
	}

	// convert hostname to hash parameters
	hashparam, hashes := hostnameToHashParam(host, hashparamNeedSlash)

	// format URL with our hashes
	url := format(hashparam)

	// do HTTP request
	atomic.AddUint64(&lookupstats.Requests, 1)
	atomic.AddInt64(&lookupstats.Pending, 1)
	updateMax(&lookupstats.Pending, &lookupstats.PendingMax)
	resp, err := d.client.Get(url)
	atomic.AddInt64(&lookupstats.Pending, -1)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		// error, don't save cache
		return Result{}, err
	}

	// get body text
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// error, don't save cache
		return Result{}, err
	}

	// handle status code
	switch {
	case resp.StatusCode == 204:
		// empty result, save cache
		err = cache.Set(host, Result{})
		if err != nil {
			return Result{}, err
		}
		return Result{}, nil
	case resp.StatusCode != 200:
		// error, don't save cache
		return Result{}, nil
	}

	result, err := handleBody(body, hashes)
	if err != nil {
		// error, don't save cache
		return Result{}, err
	}

	err = cache.Set(host, result)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

//
// Adding rule and matching against the rules
//

// Initialize urlfilter objects
func (d *Dnsfilter) initFiltering(filters map[int]string) error {
	var err error
	d.rulesStorage, err = urlfilter.NewRuleStorage(d.FilteringTempFilename)
	if err != nil {
		return err
	}

	d.filteringEngine = urlfilter.NewDNSEngine(filters, d.rulesStorage)
	return nil
}

// matchHost is a low-level way to check only if hostname is filtered by rules, skipping expensive safebrowsing and parental lookups
func (d *Dnsfilter) matchHost(host string, qtype uint16) (Result, error) {
	if d.filteringEngine == nil {
		return Result{}, nil
	}

	rules, ok := d.filteringEngine.Match(host)
	if !ok {
		return Result{}, nil
	}

	log.Tracef("%d rules matched for host '%s'", len(rules), host)

	for _, rule := range rules {

		log.Tracef("Found rule for host '%s': '%s'  list_id: %d",
			host, rule.Text(), rule.GetFilterListID())

		res := Result{}
		res.Reason = FilteredBlackList
		res.IsFiltered = true
		res.FilterID = int64(rule.GetFilterListID())
		res.Rule = rule.Text()

		if netRule, ok := rule.(*urlfilter.NetworkRule); ok {

			if netRule.Whitelist {
				res.Reason = NotFilteredWhiteList
				res.IsFiltered = false
			}
			return res, nil

		} else if hostRule, ok := rule.(*urlfilter.HostRule); ok {

			if qtype == dns.TypeA && hostRule.IP.To4() != nil {
				// either IPv4 or IPv4-mapped IPv6 address
				res.IP = hostRule.IP.To4()
				return res, nil
			} else if qtype == dns.TypeAAAA && hostRule.IP.To4() == nil {
				res.IP = hostRule.IP
				return res, nil
			}
			continue

		} else {
			log.Tracef("Rule type is unsupported: '%s'  list_id: %d",
				rule.Text(), rule.GetFilterListID())
		}
	}

	return Result{}, nil
}

//
// lifecycle helper functions
//

// Return TRUE if this host's IP should be cached
func (d *Dnsfilter) shouldBeInDialCache(host string) bool {
	return host == d.safeBrowsingServer ||
		host == d.parentalServer
}

// Search for an IP address by host name
func searchInDialCache(host string) string {
	rawValue, err := dialCache.Get(host)
	if err != nil {
		return ""
	}

	ip, _ := rawValue.(string)
	log.Debug("Found in cache: %s -> %s", host, ip)
	return ip
}

// Add "hostname" -> "IP address" entry to cache
func addToDialCache(host, ip string) {
	dialCache.Set(host, ip)
	log.Debug("Added to cache: %s -> %s", host, ip)
}

type dialFunctionType func(ctx context.Context, network, addr string) (net.Conn, error)

// Connect to a remote server resolving hostname using our own DNS server
func (d *Dnsfilter) createCustomDialContext(resolverAddr string) dialFunctionType {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		log.Tracef("network:%v  addr:%v", network, addr)

		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		dialer := &net.Dialer{
			Timeout: time.Minute * 5,
		}

		if net.ParseIP(host) != nil {
			con, err := dialer.DialContext(ctx, network, addr)
			return con, err
		}

		cache := d.shouldBeInDialCache(host)
		if cache {
			ip := searchInDialCache(host)
			if len(ip) != 0 {
				addr = fmt.Sprintf("%s:%s", ip, port)
				return dialer.DialContext(ctx, network, addr)
			}
		}

		r := upstream.NewResolver(resolverAddr, 30*time.Second)
		addrs, e := r.LookupIPAddr(ctx, host)
		log.Tracef("LookupIPAddr: %s: %v", host, addrs)
		if e != nil {
			return nil, e
		}

		var firstErr error
		firstErr = nil
		for _, a := range addrs {
			addr = fmt.Sprintf("%s:%s", a.String(), port)
			con, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}

			if cache {
				addToDialCache(host, a.String())
			}

			return con, err
		}
		return nil, firstErr
	}
}

// New creates properly initialized DNS Filter that is ready to be used
func New(c *Config, filters map[int]string) *Dnsfilter {
	d := new(Dnsfilter)

	// Customize the Transport to have larger connection pool,
	// We are not (re)using http.DefaultTransport because of race conditions found by tests
	d.transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          defaultHTTPMaxIdleConnections, // default 100
		MaxIdleConnsPerHost:   defaultHTTPMaxIdleConnections, // default 2
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if c != nil && len(c.ResolverAddress) != 0 {
		dialCache = gcache.New(maxDialCacheSize).LRU().Expiration(defaultCacheTime).Build()
		d.transport.DialContext = d.createCustomDialContext(c.ResolverAddress)
	}
	d.client = http.Client{
		Transport: d.transport,
		Timeout:   defaultHTTPTimeout,
	}
	d.safeBrowsingServer = defaultSafebrowsingServer
	d.parentalServer = defaultParentalServer
	if c != nil {
		d.Config = *c
	}

	if filters != nil {
		err := d.initFiltering(filters)
		if err != nil {
			log.Error("Can't initialize filtering subsystem: %s", err)
			d.Destroy()
			return nil
		}
	}

	return d
}

// Destroy is optional if you want to tidy up goroutines without waiting for them to die off
// right now it closes idle HTTP connections if there are any
func (d *Dnsfilter) Destroy() {
	if d != nil && d.transport != nil {
		d.transport.CloseIdleConnections()
	}

	if d.rulesStorage != nil {
		d.rulesStorage.Close()
		d.rulesStorage = nil
	}
}

//
// config manipulation helpers
//

// SetSafeBrowsingServer lets you optionally change hostname of safesearch lookup
func (d *Dnsfilter) SetSafeBrowsingServer(host string) {
	if len(host) == 0 {
		d.safeBrowsingServer = defaultSafebrowsingServer
	} else {
		d.safeBrowsingServer = host
	}
}

// SetHTTPTimeout lets you optionally change timeout during lookups
func (d *Dnsfilter) SetHTTPTimeout(t time.Duration) {
	d.client.Timeout = t
}

// ResetHTTPTimeout resets lookup timeouts
func (d *Dnsfilter) ResetHTTPTimeout() {
	d.client.Timeout = defaultHTTPTimeout
}

// SafeSearchDomain returns replacement address for search engine
func (d *Dnsfilter) SafeSearchDomain(host string) (string, bool) {
	if d.SafeSearchEnabled {
		val, ok := safeSearchDomains[host]
		return val, ok
	}
	return "", false
}

//
// stats
//

// GetStats return dns filtering stats since startup
func (d *Dnsfilter) GetStats() Stats {
	return stats
}
