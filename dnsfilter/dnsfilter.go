package dnsfilter

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joomcode/errorx"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/bluele/gcache"
	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

const defaultHTTPTimeout = 5 * time.Minute
const defaultHTTPMaxIdleConnections = 100

const defaultSafebrowsingServer = "sb.adtidy.org"
const defaultSafebrowsingURL = "%s://%s/safebrowsing-lookup-hash.html?prefixes=%s"
const defaultParentalServer = "pctrl.adguard.com"
const defaultParentalURL = "%s://%s/check-parental-control-hash?prefixes=%s&sensitivity=%d"
const defaultParentalSensitivity = 13 // use "TEEN" by default
const maxDialCacheSize = 2            // the number of host names for safebrowsing and parental control

// ServiceEntry - blocked service array element
type ServiceEntry struct {
	Name  string
	Rules []*urlfilter.NetworkRule
}

// RequestFilteringSettings is custom filtering settings
type RequestFilteringSettings struct {
	FilteringEnabled    bool
	SafeSearchEnabled   bool
	SafeBrowsingEnabled bool
	ParentalEnabled     bool
	ServicesRules       []ServiceEntry
}

// RewriteEntry is a rewrite array element
type RewriteEntry struct {
	Domain string `yaml:"domain"`
	Answer string `yaml:"answer"` // IP address or canonical name
}

// Config allows you to configure DNS filtering with New() or just change variables directly.
type Config struct {
	ParentalSensitivity int    `yaml:"parental_sensitivity"` // must be either 3, 10, 13 or 17
	ParentalEnabled     bool   `yaml:"parental_enabled"`
	UsePlainHTTP        bool   `yaml:"-"` // use plain HTTP for requests to parental and safe browsing servers
	SafeSearchEnabled   bool   `yaml:"safesearch_enabled"`
	SafeBrowsingEnabled bool   `yaml:"safebrowsing_enabled"`
	ResolverAddress     string `yaml:"-"` // DNS server address

	SafeBrowsingCacheSize uint `yaml:"safebrowsing_cache_size"` // (in bytes)
	SafeSearchCacheSize   uint `yaml:"safesearch_cache_size"`   // (in bytes)
	ParentalCacheSize     uint `yaml:"parental_cache_size"`     // (in bytes)
	CacheTime             uint `yaml:"cache_time"`              // Element's TTL (in minutes)

	Rewrites []RewriteEntry `yaml:"rewrites"`

	// Called when the configuration is changed by HTTP request
	ConfigModified func() `yaml:"-"`

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request)) `yaml:"-"`
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

// Parameters to pass to filters-initializer goroutine
type filtersInitializerParams struct {
	filters map[int]string
}

// Dnsfilter holds added rules and performs hostname matches against the rules
type Dnsfilter struct {
	rulesStorage    *urlfilter.RuleStorage
	filteringEngine *urlfilter.DNSEngine
	engineLock      sync.RWMutex

	// HTTP lookups for safebrowsing and parental
	client    http.Client     // handle for http client -- single instance as recommended by docs
	transport *http.Transport // handle for http transport used by http client

	parentalServer     string // access via methods
	safeBrowsingServer string // access via methods

	Config   // for direct access by library users, even a = assignment
	confLock sync.RWMutex

	// Channel for passing data to filters-initializer goroutine
	filtersInitializerChan chan filtersInitializerParams
	filtersInitializerLock sync.Mutex
}

// Filter represents a filter list
type Filter struct {
	ID       int64  // auto-assigned when filter is added (see nextFilterID)
	Data     []byte `yaml:"-"` // List of rules divided by '\n'
	FilePath string `yaml:"-"` // Path to a filtering rules file
}

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
	// FilteredBlockedService - the host is blocked by "blocked services" settings
	FilteredBlockedService

	// ReasonRewrite - rewrite rule was applied
	ReasonRewrite
)

var reasonNames = []string{
	"NotFilteredNotFound",
	"NotFilteredWhiteList",
	"NotFilteredError",

	"FilteredBlackList",
	"FilteredSafeBrowsing",
	"FilteredParental",
	"FilteredInvalid",
	"FilteredSafeSearch",
	"FilteredBlockedService",

	"Rewrite",
}

func (r Reason) String() string {
	if uint(r) >= uint(len(reasonNames)) {
		return ""
	}
	return reasonNames[r]
}

// GetConfig - get configuration
func (d *Dnsfilter) GetConfig() RequestFilteringSettings {
	c := RequestFilteringSettings{}
	// d.confLock.RLock()
	c.SafeSearchEnabled = d.Config.SafeSearchEnabled
	c.SafeBrowsingEnabled = d.Config.SafeBrowsingEnabled
	c.ParentalEnabled = d.Config.ParentalEnabled
	// d.confLock.RUnlock()
	return c
}

// WriteDiskConfig - write configuration
func (d *Dnsfilter) WriteDiskConfig(c *Config) {
	*c = d.Config
}

// SetFilters - set new filters (synchronously or asynchronously)
// When filters are set asynchronously, the old filters continue working until the new filters are ready.
//  In this case the caller must ensure that the old filter files are intact.
func (d *Dnsfilter) SetFilters(filters map[int]string, async bool) error {
	if async {
		params := filtersInitializerParams{
			filters: filters,
		}

		d.filtersInitializerLock.Lock() // prevent multiple writers from adding more than 1 task
		// remove all pending tasks
		stop := false
		for !stop {
			select {
			case <-d.filtersInitializerChan:
				//
			default:
				stop = true
			}
		}

		d.filtersInitializerChan <- params
		d.filtersInitializerLock.Unlock()
		return nil
	}

	err := d.initFiltering(filters)
	if err != nil {
		log.Error("Can't initialize filtering subsystem: %s", err)
		return err
	}

	return nil
}

// Starts initializing new filters by signal from channel
func (d *Dnsfilter) filtersInitializer() {
	for {
		params := <-d.filtersInitializerChan
		err := d.initFiltering(params.filters)
		if err != nil {
			log.Error("Can't initialize filtering subsystem: %s", err)
			continue
		}
	}
}

// Close - close the object
func (d *Dnsfilter) Close() {
	if d != nil && d.transport != nil {
		d.transport.CloseIdleConnections()
	}
	if d.rulesStorage != nil {
		d.rulesStorage.Close()
	}
}

type dnsFilterContext struct {
	stats             Stats
	dialCache         gcache.Cache // "host" -> "IP" cache for safebrowsing and parental control servers
	safebrowsingCache cache.Cache
	parentalCache     cache.Cache
	safeSearchCache   cache.Cache
}

var gctx dnsFilterContext // global dnsfilter context

// Result holds state of hostname check
type Result struct {
	IsFiltered bool   `json:",omitempty"` // True if the host name is filtered
	Reason     Reason `json:",omitempty"` // Reason for blocking / unblocking
	Rule       string `json:",omitempty"` // Original rule text
	IP         net.IP `json:",omitempty"` // Not nil only in the case of a hosts file syntax
	FilterID   int64  `json:",omitempty"` // Filter ID the rule belongs to

	// for ReasonRewrite:
	CanonName string   `json:",omitempty"` // CNAME value
	IPList    []net.IP `json:",omitempty"` // list of IP addresses

	// for FilteredBlockedService:
	ServiceName string `json:",omitempty"` // Name of the blocked service
}

// Matched can be used to see if any match at all was found, no matter filtered or not
func (r Reason) Matched() bool {
	return r != NotFilteredNotFound
}

// CheckHost tries to match host against rules, then safebrowsing and parental if they are enabled
func (d *Dnsfilter) CheckHost(host string, qtype uint16, setts *RequestFilteringSettings) (Result, error) {
	// sometimes DNS clients will try to resolve ".", which is a request to get root servers
	if host == "" {
		return Result{Reason: NotFilteredNotFound}, nil
	}
	host = strings.ToLower(host)
	// prevent recursion
	if host == d.parentalServer || host == d.safeBrowsingServer {
		return Result{}, nil
	}

	var result Result
	var err error

	result = d.processRewrites(host, qtype)
	if result.Reason == ReasonRewrite {
		return result, nil
	}

	// try filter lists first
	if setts.FilteringEnabled {
		result, err = d.matchHost(host, qtype)
		if err != nil {
			return result, err
		}
		if result.Reason.Matched() {
			return result, nil
		}
	}

	if len(setts.ServicesRules) != 0 {
		result = matchBlockedServicesRules(host, setts.ServicesRules)
		if result.Reason.Matched() {
			return result, nil
		}
	}

	// check safeSearch if no match
	if setts.SafeSearchEnabled {
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
	if setts.SafeBrowsingEnabled {
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
	if setts.ParentalEnabled {
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

// Process rewrites table
// . Find CNAME for a domain name
//  . if found, set domain name to canonical name
// . Find A or AAAA record for a domain name
//  . if found, return IP addresses
func (d *Dnsfilter) processRewrites(host string, qtype uint16) Result {
	var res Result

	d.confLock.RLock()
	defer d.confLock.RUnlock()

	for _, r := range d.Rewrites {
		if r.Domain != host {
			continue
		}

		ip := net.ParseIP(r.Answer)
		if ip == nil {
			log.Debug("Rewrite: CNAME for %s is %s", host, r.Answer)
			host = r.Answer
			res.CanonName = r.Answer
			res.Reason = ReasonRewrite
			break
		}
	}

	for _, r := range d.Rewrites {
		if r.Domain != host {
			continue
		}

		ip := net.ParseIP(r.Answer)
		if ip == nil {
			continue
		}
		ip4 := ip.To4()

		if qtype == dns.TypeA && ip4 != nil {
			res.IPList = append(res.IPList, ip4)
			log.Debug("Rewrite: A for %s is %s", host, ip4)

		} else if qtype == dns.TypeAAAA && ip4 == nil {
			res.IPList = append(res.IPList, ip)
			log.Debug("Rewrite: AAAA for %s is %s", host, ip)
		}
	}

	if len(res.IPList) != 0 {
		res.Reason = ReasonRewrite
	}

	return res
}

func matchBlockedServicesRules(host string, svcs []ServiceEntry) Result {
	req := urlfilter.NewRequestForHostname(host)
	res := Result{}

	for _, s := range svcs {
		for _, rule := range s.Rules {
			if rule.Match(req) {
				res.Reason = FilteredBlockedService
				res.IsFiltered = true
				res.ServiceName = s.Name
				res.Rule = rule.Text()
				log.Debug("Blocked Services: matched rule: %s  host: %s  service: %s",
					res.Rule, host, s.Name)
				return res
			}
		}
	}
	return res
}

/*
expire byte[4]
res Result
*/
func (d *Dnsfilter) setCacheResult(cache cache.Cache, host string, res Result) {
	var buf bytes.Buffer

	expire := uint(time.Now().Unix()) + d.Config.CacheTime*60
	var exp []byte
	exp = make([]byte, 4)
	binary.BigEndian.PutUint32(exp, uint32(expire))
	_, _ = buf.Write(exp)

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(res)
	if err != nil {
		log.Error("gob.Encode(): %s", err)
		return
	}
	_ = cache.Set([]byte(host), buf.Bytes())
	log.Debug("Stored in cache %p: %s", cache, host)
}

func getCachedResult(cache cache.Cache, host string) (Result, bool) {
	data := cache.Get([]byte(host))
	if data == nil {
		return Result{}, false
	}

	exp := int(binary.BigEndian.Uint32(data[:4]))
	if exp <= int(time.Now().Unix()) {
		cache.Del([]byte(host))
		return Result{}, false
	}

	var buf bytes.Buffer
	buf.Write(data[4:])
	dec := gob.NewDecoder(&buf)
	r := Result{}
	err := dec.Decode(&r)
	if err != nil {
		log.Debug("gob.Decode(): %s", err)
		return Result{}, false
	}

	return r, true
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

	// Check cache. Return cached result if it was found
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, host)
	if isFound {
		atomic.AddUint64(&gctx.stats.Safesearch.CacheHits, 1)
		log.Tracef("%s: found in SafeSearch cache", host)
		return cachedValue, nil
	}

	safeHost, ok := d.SafeSearchDomain(host)
	if !ok {
		return Result{}, nil
	}

	res := Result{IsFiltered: true, Reason: FilteredSafeSearch}
	if ip := net.ParseIP(safeHost); ip != nil {
		res.IP = ip
		d.setCacheResult(gctx.safeSearchCache, host, res)
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
	d.setCacheResult(gctx.safeSearchCache, host, res)
	return res, nil
}

func (d *Dnsfilter) checkSafeBrowsing(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("SafeBrowsing HTTP lookup for %s", host)
	}

	format := func(hashparam string) string {
		schema := "https"
		if d.UsePlainHTTP {
			schema = "http"
		}
		url := fmt.Sprintf(defaultSafebrowsingURL, schema, d.safeBrowsingServer, hashparam)
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

	// check cache
	cachedValue, isFound := getCachedResult(gctx.safebrowsingCache, host)
	if isFound {
		atomic.AddUint64(&gctx.stats.Safebrowsing.CacheHits, 1)
		log.Tracef("%s: found in the lookup cache %p", host, gctx.safebrowsingCache)
		return cachedValue, nil
	}

	result, err := d.lookupCommon(host, &gctx.stats.Safebrowsing, true, format, handleBody)

	if err == nil {
		d.setCacheResult(gctx.safebrowsingCache, host, result)
	}

	return result, err
}

func (d *Dnsfilter) checkParental(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("Parental HTTP lookup for %s", host)
	}

	format := func(hashparam string) string {
		schema := "https"
		if d.UsePlainHTTP {
			schema = "http"
		}
		sensitivity := d.ParentalSensitivity
		if sensitivity == 0 {
			sensitivity = defaultParentalSensitivity
		}
		url := fmt.Sprintf(defaultParentalURL, schema, d.parentalServer, hashparam, sensitivity)
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

	// check cache
	cachedValue, isFound := getCachedResult(gctx.parentalCache, host)
	if isFound {
		atomic.AddUint64(&gctx.stats.Parental.CacheHits, 1)
		log.Tracef("%s: found in the lookup cache %p", host, gctx.parentalCache)
		return cachedValue, nil
	}

	result, err := d.lookupCommon(host, &gctx.stats.Parental, false, format, handleBody)

	if err == nil {
		d.setCacheResult(gctx.parentalCache, host, result)
	}

	return result, err
}

type formatHandler func(hashparam string) string
type bodyHandler func(body []byte, hashes map[string]bool) (Result, error)

// real implementation of lookup/check
func (d *Dnsfilter) lookupCommon(host string, lookupstats *LookupStats, hashparamNeedSlash bool, format formatHandler, handleBody bodyHandler) (Result, error) {
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
		return Result{}, nil
	case resp.StatusCode != 200:
		return Result{}, fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	result, err := handleBody(body, hashes)
	if err != nil {
		return Result{}, err
	}

	return result, nil
}

//
// Adding rule and matching against the rules
//

// Return TRUE if file exists
func fileExists(fn string) bool {
	_, err := os.Stat(fn)
	if err != nil {
		return false
	}
	return true
}

// Initialize urlfilter objects
func (d *Dnsfilter) initFiltering(filters map[int]string) error {
	listArray := []urlfilter.RuleList{}
	for id, dataOrFilePath := range filters {
		var list urlfilter.RuleList

		if id == 0 {
			list = &urlfilter.StringRuleList{
				ID:             0,
				RulesText:      dataOrFilePath,
				IgnoreCosmetic: false,
			}

		} else if !fileExists(dataOrFilePath) {
			list = &urlfilter.StringRuleList{
				ID:             id,
				IgnoreCosmetic: false,
			}

		} else {
			var err error
			list, err = urlfilter.NewFileRuleList(id, dataOrFilePath, false)
			if err != nil {
				return fmt.Errorf("urlfilter.NewFileRuleList(): %s: %s", dataOrFilePath, err)
			}
		}
		listArray = append(listArray, list)
	}

	rulesStorage, err := urlfilter.NewRuleStorage(listArray)
	if err != nil {
		return fmt.Errorf("urlfilter.NewRuleStorage(): %s", err)
	}
	filteringEngine := urlfilter.NewDNSEngine(rulesStorage)

	d.engineLock.Lock()
	if d.rulesStorage != nil {
		d.rulesStorage.Close()
	}
	d.rulesStorage = rulesStorage
	d.filteringEngine = filteringEngine
	d.engineLock.Unlock()
	log.Debug("initialized filtering engine")

	return nil
}

// matchHost is a low-level way to check only if hostname is filtered by rules, skipping expensive safebrowsing and parental lookups
func (d *Dnsfilter) matchHost(host string, qtype uint16) (Result, error) {
	d.engineLock.RLock()
	defer d.engineLock.RUnlock()
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

			} else if qtype == dns.TypeAAAA {
				ip4 := hostRule.IP.To4()
				if ip4 == nil {
					res.IP = hostRule.IP
					return res, nil
				}
				if bytes.Equal(ip4, []byte{0, 0, 0, 0}) {
					// send IP="::" response for a rule "0.0.0.0 blockdomain"
					res.IP = net.IPv6zero
					return res, nil
				}
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
	rawValue, err := gctx.dialCache.Get(host)
	if err != nil {
		return ""
	}

	ip, _ := rawValue.(string)
	log.Debug("Found in cache: %s -> %s", host, ip)
	return ip
}

// Add "hostname" -> "IP address" entry to cache
func addToDialCache(host, ip string) {
	err := gctx.dialCache.Set(host, ip)
	if err != nil {
		log.Debug("dialCache.Set: %s", err)
	}
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

		if len(addrs) == 0 {
			return nil, fmt.Errorf("couldn't lookup host: %s", host)
		}

		var dialErrs []error
		for _, a := range addrs {
			addr = fmt.Sprintf("%s:%s", a.String(), port)
			con, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				dialErrs = append(dialErrs, err)
				continue
			}

			if cache {
				addToDialCache(host, a.String())
			}

			return con, err
		}
		return nil, errorx.DecorateMany(fmt.Sprintf("couldn't dial to %s", addr), dialErrs...)
	}
}

// New creates properly initialized DNS Filter that is ready to be used
func New(c *Config, filters map[int]string) *Dnsfilter {

	if c != nil {
		cacheConf := cache.Config{
			EnableLRU: true,
		}

		// initialize objects only once

		if gctx.safebrowsingCache == nil {
			cacheConf.MaxSize = c.SafeBrowsingCacheSize
			gctx.safebrowsingCache = cache.New(cacheConf)
		}

		if gctx.safeSearchCache == nil {
			cacheConf.MaxSize = c.SafeSearchCacheSize
			gctx.safeSearchCache = cache.New(cacheConf)
		}

		if gctx.parentalCache == nil {
			cacheConf.MaxSize = c.ParentalCacheSize
			gctx.parentalCache = cache.New(cacheConf)
		}

		if len(c.ResolverAddress) != 0 && gctx.dialCache == nil {
			dur := time.Duration(c.CacheTime) * time.Minute
			gctx.dialCache = gcache.New(maxDialCacheSize).LRU().Expiration(dur).Build()
		}
	}

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
			d.Close()
			return nil
		}
	}

	d.filtersInitializerChan = make(chan filtersInitializerParams, 1)
	go d.filtersInitializer()

	if d.Config.HTTPRegister != nil { // for tests
		d.registerSecurityHandlers()
		d.registerRewritesHandlers()
	}
	return d
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
	return gctx.stats
}
