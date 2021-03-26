// Package dnsfilter implements a DNS request and response filter.
package dnsfilter

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/util"
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

// FilteringSettings are custom filtering settings for a client.
type FilteringSettings struct {
	ClientName string
	ClientIP   net.IP
	ClientTags []string

	ServicesRules []ServiceEntry

	FilteringEnabled    bool
	SafeSearchEnabled   bool
	SafeBrowsingEnabled bool
	ParentalEnabled     bool
}

// Resolver is the interface for net.Resolver to simplify testing.
type Resolver interface {
	LookupIP(ctx context.Context, network, host string) (ips []net.IP, err error)
}

// Config allows you to configure DNS filtering with New() or just change variables directly.
type Config struct {
	ParentalEnabled     bool `yaml:"parental_enabled"`
	SafeSearchEnabled   bool `yaml:"safesearch_enabled"`
	SafeBrowsingEnabled bool `yaml:"safebrowsing_enabled"`

	SafeBrowsingCacheSize uint `yaml:"safebrowsing_cache_size"` // (in bytes)
	SafeSearchCacheSize   uint `yaml:"safesearch_cache_size"`   // (in bytes)
	ParentalCacheSize     uint `yaml:"parental_cache_size"`     // (in bytes)
	CacheTime             uint `yaml:"cache_time"`              // Element's TTL (in minutes)

	Rewrites []RewriteEntry `yaml:"rewrites"`

	// Names of services to block (globally).
	// Per-client settings can override this configuration.
	BlockedServices []string `yaml:"blocked_services"`

	// IP-hostname pairs taken from system configuration (e.g. /etc/hosts) files
	AutoHosts *util.AutoHosts `yaml:"-"`

	// Called when the configuration is changed by HTTP request
	ConfigModified func() `yaml:"-"`

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request)) `yaml:"-"`

	// CustomResolver is the resolver used by DNSFilter.
	CustomResolver Resolver
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
	allowFilters []Filter
	blockFilters []Filter
}

type hostChecker struct {
	check func(host string, qtype uint16, setts *FilteringSettings) (res Result, err error)
	name  string
}

// DNSFilter matches hostnames and DNS requests against filtering rules.
type DNSFilter struct {
	rulesStorage         *filterlist.RuleStorage
	filteringEngine      *urlfilter.DNSEngine
	rulesStorageAllow    *filterlist.RuleStorage
	filteringEngineAllow *urlfilter.DNSEngine
	engineLock           sync.RWMutex

	parentalServer       string // access via methods
	safeBrowsingServer   string // access via methods
	parentalUpstream     upstream.Upstream
	safeBrowsingUpstream upstream.Upstream

	Config   // for direct access by library users, even a = assignment
	confLock sync.RWMutex

	// Channel for passing data to filters-initializer goroutine
	filtersInitializerChan chan filtersInitializerParams
	filtersInitializerLock sync.Mutex

	// resolver only looks up the IP address of the host while safe search.
	//
	// TODO(e.burkov): Use upstream that configured in dnsforward instead.
	resolver Resolver

	hostCheckers []hostChecker
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
	// NotFilteredAllowList - the host is explicitly allowed
	NotFilteredAllowList
	// NotFilteredError is returned when there was an error during
	// checking.  Reserved, currently unused.
	NotFilteredError

	// reasons for filtering

	// FilteredBlockList - the host was matched to be advertising host
	FilteredBlockList
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

	// Rewritten is returned when there was a rewrite by a legacy DNS
	// rewrite rule.
	Rewritten

	// RewrittenAutoHosts is returned when there was a rewrite by autohosts
	// rules (/etc/hosts and so on).
	RewrittenAutoHosts

	// RewrittenRule is returned when a $dnsrewrite filter rule was applied.
	//
	// TODO(a.garipov): Remove Rewritten and RewrittenAutoHosts by merging
	// their functionality into RewrittenRule.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2499.
	RewrittenRule
)

// TODO(a.garipov): Resync with actual code names or replace completely
// in HTTP API v1.
var reasonNames = []string{
	NotFilteredNotFound:  "NotFilteredNotFound",
	NotFilteredAllowList: "NotFilteredWhiteList",
	NotFilteredError:     "NotFilteredError",

	FilteredBlockList:      "FilteredBlackList",
	FilteredSafeBrowsing:   "FilteredSafeBrowsing",
	FilteredParental:       "FilteredParental",
	FilteredInvalid:        "FilteredInvalid",
	FilteredSafeSearch:     "FilteredSafeSearch",
	FilteredBlockedService: "FilteredBlockedService",

	Rewritten:          "Rewrite",
	RewrittenAutoHosts: "RewriteEtcHosts",
	RewrittenRule:      "RewriteRule",
}

func (r Reason) String() string {
	if r < 0 || int(r) >= len(reasonNames) {
		return ""
	}

	return reasonNames[r]
}

// In returns true if reasons include r.
func (r Reason) In(reasons ...Reason) bool {
	for _, reason := range reasons {
		if r == reason {
			return true
		}
	}
	return false
}

// GetConfig - get configuration
func (d *DNSFilter) GetConfig() FilteringSettings {
	c := FilteringSettings{}
	// d.confLock.RLock()
	c.SafeSearchEnabled = d.Config.SafeSearchEnabled
	c.SafeBrowsingEnabled = d.Config.SafeBrowsingEnabled
	c.ParentalEnabled = d.Config.ParentalEnabled
	// d.confLock.RUnlock()
	return c
}

// WriteDiskConfig - write configuration
func (d *DNSFilter) WriteDiskConfig(c *Config) {
	d.confLock.Lock()
	*c = d.Config
	c.Rewrites = rewriteArrayDup(d.Config.Rewrites)
	// BlockedServices
	d.confLock.Unlock()
}

// SetFilters - set new filters (synchronously or asynchronously)
// When filters are set asynchronously, the old filters continue working until the new filters are ready.
//  In this case the caller must ensure that the old filter files are intact.
func (d *DNSFilter) SetFilters(blockFilters, allowFilters []Filter, async bool) error {
	if async {
		params := filtersInitializerParams{
			allowFilters: allowFilters,
			blockFilters: blockFilters,
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

	err := d.initFiltering(allowFilters, blockFilters)
	if err != nil {
		log.Error("Can't initialize filtering subsystem: %s", err)
		return err
	}

	return nil
}

// Starts initializing new filters by signal from channel
func (d *DNSFilter) filtersInitializer() {
	for {
		params := <-d.filtersInitializerChan
		err := d.initFiltering(params.allowFilters, params.blockFilters)
		if err != nil {
			log.Error("Can't initialize filtering subsystem: %s", err)
			continue
		}
	}
}

// Close - close the object
func (d *DNSFilter) Close() {
	d.engineLock.Lock()
	defer d.engineLock.Unlock()
	d.reset()
}

func (d *DNSFilter) reset() {
	var err error

	if d.rulesStorage != nil {
		err = d.rulesStorage.Close()
		if err != nil {
			log.Error("dnsfilter: rulesStorage.Close: %s", err)
		}
	}

	if d.rulesStorageAllow != nil {
		err = d.rulesStorageAllow.Close()
		if err != nil {
			log.Error("dnsfilter: rulesStorageAllow.Close: %s", err)
		}
	}
}

type dnsFilterContext struct {
	safebrowsingCache cache.Cache
	parentalCache     cache.Cache
	safeSearchCache   cache.Cache
}

var gctx dnsFilterContext // global dnsfilter context

// ResultRule contains information about applied rules.
type ResultRule struct {
	// FilterListID is the ID of the rule's filter list.
	FilterListID int64 `json:",omitempty"`
	// Text is the text of the rule.
	Text string `json:",omitempty"`
	// IP is the host IP.  It is nil unless the rule uses the
	// /etc/hosts syntax or the reason is FilteredSafeSearch.
	IP net.IP `json:",omitempty"`
}

// Result contains the result of a request check.
//
// All fields transitively have omitempty tags so that the query log
// doesn't become too large.
//
// TODO(a.garipov): Clarify relationships between fields.  Perhaps
// replace with a sum type or an interface?
type Result struct {
	// IsFiltered is true if the request is filtered.
	IsFiltered bool `json:",omitempty"`

	// Reason is the reason for blocking or unblocking the request.
	Reason Reason `json:",omitempty"`

	// Rules are applied rules.  If Rules are not empty, each rule
	// is not nil.
	Rules []*ResultRule `json:",omitempty"`

	// ReverseHosts is the reverse lookup rewrite result.  It is
	// empty unless Reason is set to RewrittenAutoHosts.
	ReverseHosts []string `json:",omitempty"`

	// IPList is the lookup rewrite result.  It is empty unless
	// Reason is set to RewrittenAutoHosts or Rewritten.
	IPList []net.IP `json:",omitempty"`

	// CanonName is the CNAME value from the lookup rewrite result.
	// It is empty unless Reason is set to Rewritten or RewrittenRule.
	CanonName string `json:",omitempty"`

	// ServiceName is the name of the blocked service.  It is empty
	// unless Reason is set to FilteredBlockedService.
	ServiceName string `json:",omitempty"`

	// DNSRewriteResult is the $dnsrewrite filter rule result.
	DNSRewriteResult *DNSRewriteResult `json:",omitempty"`
}

// Matched returns true if any match at all was found regardless of
// whether it was filtered or not.
func (r Reason) Matched() bool {
	return r != NotFilteredNotFound
}

// CheckHostRules tries to match the host against filtering rules only.
func (d *DNSFilter) CheckHostRules(host string, qtype uint16, setts *FilteringSettings) (Result, error) {
	if !setts.FilteringEnabled {
		return Result{}, nil
	}

	return d.matchHost(host, qtype, setts)
}

// CheckHost tries to match the host against filtering rules, then safebrowsing
// and parental control rules, if they are enabled.
func (d *DNSFilter) CheckHost(
	host string,
	qtype uint16,
	setts *FilteringSettings,
) (res Result, err error) {
	// Sometimes clients try to resolve ".", which is a request to get root
	// servers.
	if host == "" {
		return Result{Reason: NotFilteredNotFound}, nil
	}

	host = strings.ToLower(host)

	res = d.processRewrites(host, qtype)
	if res.Reason == Rewritten {
		return res, nil
	}

	for _, hc := range d.hostCheckers {
		res, err = hc.check(host, qtype, setts)
		if err != nil {
			return Result{}, fmt.Errorf("%s: %w", hc.name, err)
		}

		if res.Reason.Matched() {
			return res, nil
		}
	}

	return Result{}, nil
}

// checkAutoHosts compares the host against our autohosts table.  The err is
// always nil, it is only there to make this a valid hostChecker function.
func (d *DNSFilter) checkAutoHosts(
	host string,
	qtype uint16,
	_ *FilteringSettings,
) (res Result, err error) {
	if d.Config.AutoHosts == nil {
		return Result{}, nil
	}

	ips := d.Config.AutoHosts.Process(host, qtype)
	if ips != nil {
		res = Result{
			Reason: RewrittenAutoHosts,
			IPList: ips,
		}

		return res, nil
	}

	revHosts := d.Config.AutoHosts.ProcessReverse(host, qtype)
	if len(revHosts) != 0 {
		res = Result{
			Reason: RewrittenAutoHosts,
		}

		// TODO(a.garipov): Optimize this with a buffer.
		res.ReverseHosts = make([]string, len(revHosts))
		for i := range revHosts {
			res.ReverseHosts[i] = revHosts[i] + "."
		}

		return res, nil
	}

	return Result{}, nil
}

// Process rewrites table
// . Find CNAME for a domain name (exact match or by wildcard)
//  . if found and CNAME equals to domain name - this is an exception;  exit
//  . if found, set domain name to canonical name
//  . repeat for the new domain name (Note: we return only the last CNAME)
// . Find A or AAAA record for a domain name (exact match or by wildcard)
//  . if found, set IP addresses (IPv4 or IPv6 depending on qtype) in Result.IPList array
func (d *DNSFilter) processRewrites(host string, qtype uint16) (res Result) {
	d.confLock.RLock()
	defer d.confLock.RUnlock()

	rr := findRewrites(d.Rewrites, host)
	if len(rr) != 0 {
		res.Reason = Rewritten
	}

	cnames := map[string]bool{}
	origHost := host
	for len(rr) != 0 && rr[0].Type == dns.TypeCNAME {
		log.Debug("Rewrite: CNAME for %s is %s", host, rr[0].Answer)

		if host == rr[0].Answer { // "host == CNAME" is an exception
			res.Reason = NotFilteredNotFound

			return res
		}

		host = rr[0].Answer
		_, ok := cnames[host]
		if ok {
			log.Info("Rewrite: breaking CNAME redirection loop: %s.  Question: %s", host, origHost)
			return res
		}
		cnames[host] = false
		res.CanonName = rr[0].Answer
		rr = findRewrites(d.Rewrites, host)
	}

	for _, r := range rr {
		if (r.Type == dns.TypeA && qtype == dns.TypeA) ||
			(r.Type == dns.TypeAAAA && qtype == dns.TypeAAAA) {

			if r.IP == nil { // IP exception
				res.Reason = 0
				return res
			}

			res.IPList = append(res.IPList, r.IP)
			log.Debug("Rewrite: A/AAAA for %s is %s", host, r.IP)
		}
	}

	return res
}

// matchBlockedServicesRules checks the host against the blocked services rules
// in settings, if any.  The err is always nil, it is only there to make this
// a valid hostChecker function.
func matchBlockedServicesRules(
	host string,
	_ uint16,
	setts *FilteringSettings,
) (res Result, err error) {
	svcs := setts.ServicesRules
	if len(svcs) == 0 {
		return Result{}, nil
	}

	req := rules.NewRequestForHostname(host)
	for _, s := range svcs {
		for _, rule := range s.Rules {
			if rule.Match(req) {
				res.Reason = FilteredBlockedService
				res.IsFiltered = true
				res.ServiceName = s.Name

				ruleText := rule.Text()
				res.Rules = []*ResultRule{{
					FilterListID: int64(rule.GetFilterListID()),
					Text:         ruleText,
				}}

				log.Debug("blocked services: matched rule: %s  host: %s  service: %s",
					ruleText, host, s.Name)

				return res, nil
			}
		}
	}

	return res, nil
}

//
// Adding rule and matching against the rules
//

// fileExists returns true if file exists.
func fileExists(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil
}

func createFilteringEngine(filters []Filter) (*filterlist.RuleStorage, *urlfilter.DNSEngine, error) {
	listArray := []filterlist.RuleList{}
	for _, f := range filters {
		var list filterlist.RuleList

		if f.ID == 0 {
			list = &filterlist.StringRuleList{
				ID:             0,
				RulesText:      string(f.Data),
				IgnoreCosmetic: true,
			}
		} else if !fileExists(f.FilePath) {
			list = &filterlist.StringRuleList{
				ID:             int(f.ID),
				IgnoreCosmetic: true,
			}
		} else if runtime.GOOS == "windows" {
			// On Windows we don't pass a file to urlfilter because
			// it's difficult to update this file while it's being
			// used.
			data, err := ioutil.ReadFile(f.FilePath)
			if err != nil {
				return nil, nil, fmt.Errorf("ioutil.ReadFile(): %s: %w", f.FilePath, err)
			}
			list = &filterlist.StringRuleList{
				ID:             int(f.ID),
				RulesText:      string(data),
				IgnoreCosmetic: true,
			}

		} else {
			var err error
			list, err = filterlist.NewFileRuleList(int(f.ID), f.FilePath, true)
			if err != nil {
				return nil, nil, fmt.Errorf("filterlist.NewFileRuleList(): %s: %w", f.FilePath, err)
			}
		}
		listArray = append(listArray, list)
	}

	rulesStorage, err := filterlist.NewRuleStorage(listArray)
	if err != nil {
		return nil, nil, fmt.Errorf("filterlist.NewRuleStorage(): %w", err)
	}
	filteringEngine := urlfilter.NewDNSEngine(rulesStorage)
	return rulesStorage, filteringEngine, nil
}

// Initialize urlfilter objects.
func (d *DNSFilter) initFiltering(allowFilters, blockFilters []Filter) error {
	rulesStorage, filteringEngine, err := createFilteringEngine(blockFilters)
	if err != nil {
		return err
	}
	rulesStorageAllow, filteringEngineAllow, err := createFilteringEngine(allowFilters)
	if err != nil {
		return err
	}

	d.engineLock.Lock()
	d.reset()
	d.rulesStorage = rulesStorage
	d.filteringEngine = filteringEngine
	d.rulesStorageAllow = rulesStorageAllow
	d.filteringEngineAllow = filteringEngineAllow
	d.engineLock.Unlock()

	// Make sure that the OS reclaims memory as soon as possible
	debug.FreeOSMemory()
	log.Debug("initialized filtering engine")

	return nil
}

// matchHostProcessAllowList processes the allowlist logic of host
// matching.
func (d *DNSFilter) matchHostProcessAllowList(host string, dnsres urlfilter.DNSResult) (res Result, err error) {
	var rule rules.Rule
	if dnsres.NetworkRule != nil {
		rule = dnsres.NetworkRule
	} else if len(dnsres.HostRulesV4) > 0 {
		rule = dnsres.HostRulesV4[0]
	} else if len(dnsres.HostRulesV6) > 0 {
		rule = dnsres.HostRulesV6[0]
	}

	if rule == nil {
		return Result{}, fmt.Errorf("invalid dns result: rules are empty")
	}

	log.Debug("Filtering: found allowlist rule for host %q: %q  list_id: %d",
		host, rule.Text(), rule.GetFilterListID())

	return makeResult(rule, NotFilteredAllowList), nil
}

// matchHostProcessDNSResult processes the matched DNS filtering result.
func (d *DNSFilter) matchHostProcessDNSResult(
	qtype uint16,
	dnsres urlfilter.DNSResult,
) (res Result) {
	if dnsres.NetworkRule != nil {
		reason := FilteredBlockList
		if dnsres.NetworkRule.Whitelist {
			reason = NotFilteredAllowList
		}

		return makeResult(dnsres.NetworkRule, reason)
	}

	if qtype == dns.TypeA && dnsres.HostRulesV4 != nil {
		rule := dnsres.HostRulesV4[0]
		res = makeResult(rule, FilteredBlockList)
		res.Rules[0].IP = rule.IP.To4()

		return res
	}

	if qtype == dns.TypeAAAA && dnsres.HostRulesV6 != nil {
		rule := dnsres.HostRulesV6[0]
		res = makeResult(rule, FilteredBlockList)
		res.Rules[0].IP = rule.IP.To16()

		return res
	}

	if dnsres.HostRulesV4 != nil || dnsres.HostRulesV6 != nil {
		// Question type doesn't match the host rules.  Return the first
		// matched host rule, but without an IP address.
		var rule rules.Rule
		if dnsres.HostRulesV4 != nil {
			rule = dnsres.HostRulesV4[0]
		} else if dnsres.HostRulesV6 != nil {
			rule = dnsres.HostRulesV6[0]
		}

		res = makeResult(rule, FilteredBlockList)
		res.Rules[0].IP = net.IP{}

		return res
	}

	return Result{}
}

// matchHost is a low-level way to check only if hostname is filtered by rules,
// skipping expensive safebrowsing and parental lookups.
func (d *DNSFilter) matchHost(
	host string,
	qtype uint16,
	setts *FilteringSettings,
) (res Result, err error) {
	if !setts.FilteringEnabled {
		return Result{}, nil
	}

	d.engineLock.RLock()
	// Keep in mind that this lock must be held no just when calling Match()
	//  but also while using the rules returned by it.
	defer d.engineLock.RUnlock()

	ureq := urlfilter.DNSRequest{
		Hostname:         host,
		SortedClientTags: setts.ClientTags,
		// TODO(e.burkov): Wait for urlfilter update to pass net.IP.
		ClientIP:   setts.ClientIP.String(),
		ClientName: setts.ClientName,
		DNSType:    qtype,
	}

	if d.filteringEngineAllow != nil {
		dnsres, ok := d.filteringEngineAllow.MatchRequest(ureq)
		if ok {
			return d.matchHostProcessAllowList(host, dnsres)
		}
	}

	if d.filteringEngine == nil {
		return Result{}, nil
	}

	dnsres, ok := d.filteringEngine.MatchRequest(ureq)

	// Check DNS rewrites first, because the API there is a bit awkward.
	if dnsr := dnsres.DNSRewrites(); len(dnsr) > 0 {
		res = d.processDNSRewrites(dnsr)
		if res.Reason == RewrittenRule && res.CanonName == host {
			// A rewrite of a host to itself.  Go on and try
			// matching other things.
		} else {
			return res, nil
		}
	} else if !ok {
		return Result{}, nil
	}

	res = d.matchHostProcessDNSResult(qtype, dnsres)
	if len(res.Rules) > 0 {
		r := res.Rules[0]
		log.Debug(
			"filtering: found rule %q for host %q, filter list id: %d",
			r.Text,
			host,
			r.FilterListID,
		)
	}

	return res, nil
}

// makeResult returns a properly constructed Result.
func makeResult(rule rules.Rule, reason Reason) Result {
	res := Result{
		Reason: reason,
		Rules: []*ResultRule{{
			FilterListID: int64(rule.GetFilterListID()),
			Text:         rule.Text(),
		}},
	}

	if reason == FilteredBlockList {
		res.IsFiltered = true
	}

	return res
}

// InitModule manually initializes blocked services map.
func InitModule() {
	initBlockedServices()
}

// New creates properly initialized DNS Filter that is ready to be used.
func New(c *Config, blockFilters []Filter) *DNSFilter {
	var resolver Resolver = net.DefaultResolver
	if c != nil {
		cacheConf := cache.Config{
			EnableLRU: true,
		}

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

		if c.CustomResolver != nil {
			resolver = c.CustomResolver
		}
	}

	d := &DNSFilter{
		resolver: resolver,
	}

	d.hostCheckers = []hostChecker{{
		check: d.checkAutoHosts,
		name:  "autohosts",
	}, {
		check: d.matchHost,
		name:  "filtering",
	}, {
		check: matchBlockedServicesRules,
		name:  "blocked services",
	}, {
		check: d.checkSafeBrowsing,
		name:  "safe browsing",
	}, {
		check: d.checkParental,
		name:  "parental",
	}, {
		check: d.checkSafeSearch,
		name:  "safe search",
	}}

	err := d.initSecurityServices()
	if err != nil {
		log.Error("dnsfilter: initialize services: %s", err)
		return nil
	}

	if c != nil {
		d.Config = *c
		d.prepareRewrites()
	}

	bsvcs := []string{}
	for _, s := range d.BlockedServices {
		if !BlockedSvcKnown(s) {
			log.Debug("skipping unknown blocked-service %q", s)
			continue
		}
		bsvcs = append(bsvcs, s)
	}
	d.BlockedServices = bsvcs

	if blockFilters != nil {
		err = d.initFiltering(nil, blockFilters)
		if err != nil {
			log.Error("Can't initialize filtering subsystem: %s", err)
			d.Close()
			return nil
		}
	}

	return d
}

// Start - start the module:
// . start async filtering initializer goroutine
// . register web handlers
func (d *DNSFilter) Start() {
	d.filtersInitializerChan = make(chan filtersInitializerParams, 1)
	go d.filtersInitializer()

	if d.Config.HTTPRegister != nil { // for tests
		d.registerSecurityHandlers()
		d.registerRewritesHandlers()
		d.registerBlockedServicesHandlers()
	}
}
