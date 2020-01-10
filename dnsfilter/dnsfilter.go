package dnsfilter

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// ServiceEntry - blocked service array element
type ServiceEntry struct {
	Name  string
	Rules []*rules.NetworkRule
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
	rulesStorage    *filterlist.RuleStorage
	filteringEngine *urlfilter.DNSEngine
	engineLock      sync.RWMutex

	parentalServer       string // access via methods
	safeBrowsingServer   string // access via methods
	parentalUpstream     upstream.Upstream
	safeBrowsingUpstream upstream.Upstream

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
	if d.rulesStorage != nil {
		d.rulesStorage.Close()
	}
}

type dnsFilterContext struct {
	stats             Stats
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

// CheckHostRules tries to match the host against filtering rules only
func (d *Dnsfilter) CheckHostRules(host string, qtype uint16, setts *RequestFilteringSettings) (Result, error) {
	if !setts.FilteringEnabled {
		return Result{}, nil
	}

	return d.matchHost(host, qtype)
}

// CheckHost tries to match the host against filtering rules,
// then safebrowsing and parental if they are enabled
func (d *Dnsfilter) CheckHost(host string, qtype uint16, setts *RequestFilteringSettings) (Result, error) {
	// sometimes DNS clients will try to resolve ".", which is a request to get root servers
	if host == "" {
		return Result{Reason: NotFilteredNotFound}, nil
	}
	host = strings.ToLower(host)

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

	if setts.SafeSearchEnabled {
		result, err = d.checkSafeSearch(host)
		if err != nil {
			log.Info("SafeSearch: failed: %v", err)
			return Result{}, nil
		}

		if result.Reason.Matched() {
			return result, nil
		}
	}

	if setts.SafeBrowsingEnabled {
		result, err = d.checkSafeBrowsing(host)
		if err != nil {
			log.Info("SafeBrowsing: failed: %v", err)
			return Result{}, nil
		}
		if result.Reason.Matched() {
			return result, nil
		}
	}

	if setts.ParentalEnabled {
		result, err = d.checkParental(host)
		if err != nil {
			log.Printf("Parental: failed: %v", err)
			return Result{}, nil
		}
		if result.Reason.Matched() {
			return result, nil
		}
	}

	return Result{}, nil
}

// Return TRUE of host name matches a wildcard pattern
func matchDomainWildcard(host, wildcard string) bool {
	return len(wildcard) >= 2 &&
		wildcard[0] == '*' && wildcard[1] == '.' &&
		strings.HasSuffix(host, wildcard[1:])
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
			if !matchDomainWildcard(host, r.Domain) {
				continue
			}
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
			if !matchDomainWildcard(host, r.Domain) {
				continue
			}
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
	req := rules.NewRequestForHostname(host)
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
	listArray := []filterlist.RuleList{}
	for id, dataOrFilePath := range filters {
		var list filterlist.RuleList

		if id == 0 {
			list = &filterlist.StringRuleList{
				ID:             0,
				RulesText:      dataOrFilePath,
				IgnoreCosmetic: true,
			}

		} else if !fileExists(dataOrFilePath) {
			list = &filterlist.StringRuleList{
				ID:             id,
				IgnoreCosmetic: true,
			}

		} else if runtime.GOOS == "windows" {
			// On Windows we don't pass a file to urlfilter because
			//  it's difficult to update this file while it's being used.
			data, err := ioutil.ReadFile(dataOrFilePath)
			if err != nil {
				return fmt.Errorf("ioutil.ReadFile(): %s: %s", dataOrFilePath, err)
			}
			list = &filterlist.StringRuleList{
				ID:             id,
				RulesText:      string(data),
				IgnoreCosmetic: true,
			}

		} else {
			var err error
			list, err = filterlist.NewFileRuleList(id, dataOrFilePath, true)
			if err != nil {
				return fmt.Errorf("filterlist.NewFileRuleList(): %s: %s", dataOrFilePath, err)
			}
		}
		listArray = append(listArray, list)
	}

	rulesStorage, err := filterlist.NewRuleStorage(listArray)
	if err != nil {
		return fmt.Errorf("filterlist.NewRuleStorage(): %s", err)
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

	frules, ok := d.filteringEngine.Match(host)
	if !ok {
		return Result{}, nil
	}

	log.Tracef("%d rules matched for host '%s'", len(frules), host)

	for _, rule := range frules {

		log.Tracef("Found rule for host '%s': '%s'  list_id: %d",
			host, rule.Text(), rule.GetFilterListID())

		res := Result{}
		res.Reason = FilteredBlackList
		res.IsFiltered = true
		res.FilterID = int64(rule.GetFilterListID())
		res.Rule = rule.Text()

		if netRule, ok := rule.(*rules.NetworkRule); ok {

			if netRule.Whitelist {
				res.Reason = NotFilteredWhiteList
				res.IsFiltered = false
			}
			return res, nil

		} else if hostRule, ok := rule.(*rules.HostRule); ok {

			res.IP = net.IP{}
			if qtype == dns.TypeA && hostRule.IP.To4() != nil {
				// either IPv4 or IPv4-mapped IPv6 address
				res.IP = hostRule.IP.To4()

			} else if qtype == dns.TypeAAAA && hostRule.IP.To4() == nil {
				res.IP = hostRule.IP
			}
			return res, nil

		} else {
			log.Tracef("Rule type is unsupported: '%s'  list_id: %d",
				rule.Text(), rule.GetFilterListID())
		}
	}

	return Result{}, nil
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
	}

	d := new(Dnsfilter)

	err := d.initSecurityServices()
	if err != nil {
		log.Error("dnsfilter: initialize services: %s", err)
		return nil
	}

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
// stats
//

// GetStats return dns filtering stats since startup
func (d *Dnsfilter) GetStats() Stats {
	return gctx.stats
}
