// Package filtering implements a DNS request and response filter.
package filtering

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/mathutil"
	"github.com/AdguardTeam/golibs/syncutil"
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

// Settings are custom filtering settings for a client.
//
// TODO(s.chzhen):  Move to the client package.
type Settings struct {
	ClientName string
	ClientIP   netip.Addr
	ClientTags []string

	ServicesRules []ServiceEntry

	// BlockedServices is the configuration of blocked services of a client.  It
	// is nil if the client does not have any blocked services.
	BlockedServices *BlockedServices

	ProtectionEnabled   bool
	FilteringEnabled    bool
	SafeSearchEnabled   bool
	SafeBrowsingEnabled bool
	ParentalEnabled     bool

	// ClientSafeSearch is a client configured safe search.
	ClientSafeSearch SafeSearch
}

// Resolver is the interface for net.Resolver to simplify testing.
type Resolver interface {
	LookupIP(ctx context.Context, network, host string) (ips []net.IP, err error)
}

// Config allows you to configure DNS filtering with New() or just change variables directly.
type Config struct {
	// logger is used to log the operations of DNS filtering.  It must not be
	// nil.
	Logger *slog.Logger `yaml:"-"`

	// BlockingIPv4 is the IP address to be returned for a blocked A request.
	BlockingIPv4 netip.Addr `yaml:"blocking_ipv4"`

	// BlockingIPv6 is the IP address to be returned for a blocked AAAA request.
	BlockingIPv6 netip.Addr `yaml:"blocking_ipv6"`

	// SafeBrowsingChecker is the safe browsing hash-prefix checker.
	SafeBrowsingChecker Checker `yaml:"-"`

	// ParentControl is the parental control hash-prefix checker.
	ParentalControlChecker Checker `yaml:"-"`

	SafeSearch SafeSearch `yaml:"-"`

	// ApplyClientFiltering retrieves persistent client information using the
	// ClientID or client IP address, and applies it to the filtering settings.
	// It must not be nil.
	ApplyClientFiltering func(clientID string, cliAddr netip.Addr, setts *Settings) `yaml:"-"`

	// BlockedServices is the configuration of blocked services.
	// Per-client settings can override this configuration.
	BlockedServices *BlockedServices `yaml:"blocked_services"`

	// EtcHosts is a container of IP-hostname pairs taken from the operating
	// system configuration files (e.g. /etc/hosts).
	//
	// TODO(e.burkov):  Move it to dnsforward entirely.
	EtcHosts hostsfile.Storage `yaml:"-"`

	// ConfModifier is used to update the global configuration.  It must not be
	// nil.
	ConfModifier agh.ConfigModifier `yaml:"-"`

	// HTTPReg registers HTTP handlers.  It must not be nil.
	HTTPReg aghhttp.Registrar `yaml:"-"`

	// HTTPClient is the client to use for updating the remote filters.
	HTTPClient *http.Client `yaml:"-"`

	// filtersMu protects filter lists.
	filtersMu *sync.RWMutex

	// ProtectionDisabledUntil is the timestamp until when the protection is
	// disabled.
	ProtectionDisabledUntil *time.Time `yaml:"protection_disabled_until"`

	SafeSearchConf SafeSearchConfig `yaml:"safe_search"`

	// DataDir is used to store filters' contents.
	DataDir string `yaml:"-"`

	// BlockingMode defines the way how blocked responses are constructed.
	BlockingMode BlockingMode `yaml:"blocking_mode"`

	// ParentalBlockHost is the IP (or domain name) which is used to respond to
	// DNS requests blocked by parental control.
	ParentalBlockHost string `yaml:"parental_block_host"`

	// SafeBrowsingBlockHost is the IP (or domain name) which is used to respond
	// to DNS requests blocked by safe-browsing.
	SafeBrowsingBlockHost string `yaml:"safebrowsing_block_host"`

	// Rewrites is a list of legacy DNS rewrite records.
	Rewrites []*LegacyRewrite `yaml:"rewrites"`

	// Filters are the blocking filter lists.
	Filters []FilterYAML `yaml:"-"`

	// WhitelistFilters are the allowing filter lists.
	WhitelistFilters []FilterYAML `yaml:"-"`

	// UserRules is the global list of custom rules.
	UserRules []string `yaml:"-"`

	// SafeFSPatterns are the patterns for matching which local filtering-rule
	// files can be added.
	SafeFSPatterns []string `yaml:"safe_fs_patterns"`

	SafeBrowsingCacheSize uint `yaml:"safebrowsing_cache_size"` // (in bytes)
	SafeSearchCacheSize   uint `yaml:"safesearch_cache_size"`   // (in bytes)
	ParentalCacheSize     uint `yaml:"parental_cache_size"`     // (in bytes)
	// TODO(a.garipov): Use timeutil.Duration
	CacheTime uint `yaml:"cache_time"` // Element's TTL (in minutes)

	// enabled is used to be returned within Settings.
	//
	// It is of type uint32 to be accessed by atomic.
	//
	// TODO(e.burkov):  Use atomic.Bool in Go 1.19.
	enabled uint32

	// FiltersUpdateIntervalHours is the time period to update filters
	// (in hours).
	FiltersUpdateIntervalHours uint32 `yaml:"filters_update_interval"`

	// BlockedResponseTTL is the time-to-live value for blocked responses.  If
	// 0, then default value is used (3600).
	BlockedResponseTTL uint32 `yaml:"blocked_response_ttl"`

	// FilteringEnabled indicates whether or not use filter lists.
	FilteringEnabled bool `yaml:"filtering_enabled"`

	// RewritesEnabled indicates whether legacy rewrites are applied.
	RewritesEnabled bool `yaml:"rewrites_enabled"`

	ParentalEnabled     bool `yaml:"parental_enabled"`
	SafeBrowsingEnabled bool `yaml:"safebrowsing_enabled"`

	// ProtectionEnabled defines whether or not use any of filtering features.
	ProtectionEnabled bool `yaml:"protection_enabled"`
}

// BlockingMode is an enum of all allowed blocking modes.
type BlockingMode string

// Allowed blocking modes.
const (
	// BlockingModeCustomIP means respond with a custom IP address.
	BlockingModeCustomIP BlockingMode = "custom_ip"

	// BlockingModeDefault is the same as BlockingModeNullIP for
	// Adblock-style rules, but responds with the IP address specified in
	// the rule when blocked by an `/etc/hosts`-style rule.
	BlockingModeDefault BlockingMode = "default"

	// BlockingModeNullIP means respond with a zero IP address: "0.0.0.0"
	// for A requests and "::" for AAAA ones.
	BlockingModeNullIP BlockingMode = "null_ip"

	// BlockingModeNXDOMAIN means respond with the NXDOMAIN code.
	BlockingModeNXDOMAIN BlockingMode = "nxdomain"

	// BlockingModeREFUSED means respond with the REFUSED code.
	BlockingModeREFUSED BlockingMode = "refused"
)

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
	check func(host string, qtype uint16, setts *Settings) (res Result, err error)
	name  string
}

// Checker is used for safe browsing or parental control hash-prefix filtering.
type Checker interface {
	// Check returns true if request for the host should be blocked.
	Check(host string) (block bool, err error)
}

// DNSFilter matches hostnames and DNS requests against filtering rules.
type DNSFilter struct {
	// logger is used for logging the filtering process.
	logger *slog.Logger

	// idGen is used to generate IDs for package urlfilter.
	idGen *idGenerator

	// bufPool is a pool of buffers used for filtering-rule list parsing.
	bufPool *syncutil.Pool[[]byte]

	rulesStorage    *filterlist.RuleStorage
	filteringEngine *urlfilter.DNSEngine

	rulesStorageAllow    *filterlist.RuleStorage
	filteringEngineAllow *urlfilter.DNSEngine

	safeSearch SafeSearch

	// safeBrowsingChecker is the safe browsing hash-prefix checker.
	safeBrowsingChecker Checker

	// parentalControl is the parental control hash-prefix checker.
	parentalControlChecker Checker

	// applyClientFiltering retrieves persistent client information using the
	// ClientID or client IP address, and applies it to the filtering settings.
	//
	// TODO(s.chzhen):  Consider finding a better approach while taking an
	// import cycle into account.
	applyClientFiltering func(clientID string, cliAddr netip.Addr, setts *Settings)

	engineLock sync.RWMutex

	// confMu protects conf.
	confMu *sync.RWMutex

	// conf contains filtering parameters.
	conf *Config

	// done is the channel to signal to stop running filters updates loop.
	done chan struct{}

	// Channel for passing data to filters-initializer goroutine
	filtersInitializerChan chan filtersInitializerParams
	filtersInitializerLock sync.Mutex

	refreshLock *sync.Mutex

	hostCheckers []hostChecker

	safeFSPatterns []string
}

// Filter represents a filter list
type Filter struct {
	// FilePath is the path to a filtering rules list file.
	FilePath string `yaml:"-"`

	// Data is the content of the file.
	Data []byte `yaml:"-"`

	// ID is automatically assigned when filter is added.
	ID rules.ListID `yaml:"id"`
}

// SetEnabled sets the status of the *DNSFilter.
func (d *DNSFilter) SetEnabled(enabled bool) {
	atomic.StoreUint32(&d.conf.enabled, mathutil.BoolToNumber[uint32](enabled))
}

// Settings returns filtering settings.
func (d *DNSFilter) Settings() (s *Settings) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	return &Settings{
		FilteringEnabled:    atomic.LoadUint32(&d.conf.enabled) != 0,
		SafeSearchEnabled:   d.conf.SafeSearchConf.Enabled,
		SafeBrowsingEnabled: d.conf.SafeBrowsingEnabled,
		ParentalEnabled:     d.conf.ParentalEnabled,
	}
}

// WriteDiskConfig - write configuration
func (d *DNSFilter) WriteDiskConfig(c *Config) {
	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()

		*c = *d.conf
		c.Rewrites = cloneRewrites(c.Rewrites)
	}()

	d.conf.filtersMu.RLock()
	defer d.conf.filtersMu.RUnlock()

	c.Filters = slices.Clone(d.conf.Filters)
	c.WhitelistFilters = slices.Clone(d.conf.WhitelistFilters)
	c.UserRules = slices.Clone(d.conf.UserRules)
}

// setFilters sets new filters, synchronously or asynchronously.  When filters
// are set asynchronously, the old filters continue working until the new
// filters are ready.
//
// In this case the caller must ensure that the old filter files are intact.
func (d *DNSFilter) setFilters(
	ctx context.Context,
	blockFilters []Filter,
	allowFilters []Filter,
	async bool,
) (err error) {
	if async {
		params := filtersInitializerParams{
			allowFilters: allowFilters,
			blockFilters: blockFilters,
		}

		d.filtersInitializerLock.Lock()
		defer d.filtersInitializerLock.Unlock()

		// Remove all pending tasks.
	removeLoop:
		for {
			select {
			case <-d.filtersInitializerChan:
				// Continue removing.
			default:
				break removeLoop
			}
		}

		d.filtersInitializerChan <- params

		return nil
	}

	return d.initFiltering(ctx, allowFilters, blockFilters)
}

// Close - close the object
func (d *DNSFilter) Close() {
	d.engineLock.Lock()
	defer d.engineLock.Unlock()

	if d.done != nil {
		d.done <- struct{}{}
	}

	d.reset(context.TODO())
}

func (d *DNSFilter) reset(ctx context.Context) {
	if d.rulesStorage != nil {
		if err := d.rulesStorage.Close(); err != nil {
			d.logger.ErrorContext(ctx, "closing rules storage", slogutil.KeyError, err)
		}
	}

	if d.rulesStorageAllow != nil {
		if err := d.rulesStorageAllow.Close(); err != nil {
			d.logger.ErrorContext(ctx, "closing allow rules storage", slogutil.KeyError, err)
		}
	}
}

// ProtectionStatus returns the status of protection and time until it's
// disabled if so.
func (d *DNSFilter) ProtectionStatus() (status bool, disabledUntil *time.Time) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	return d.conf.ProtectionEnabled, d.conf.ProtectionDisabledUntil
}

// SetProtectionStatus updates the status of protection and time until it's
// disabled.
func (d *DNSFilter) SetProtectionStatus(status bool, disabledUntil *time.Time) {
	d.confMu.Lock()
	defer d.confMu.Unlock()

	d.conf.ProtectionEnabled = status
	d.conf.ProtectionDisabledUntil = disabledUntil
}

// SetProtectionEnabled updates the status of protection.
func (d *DNSFilter) SetProtectionEnabled(status bool) {
	d.confMu.Lock()
	defer d.confMu.Unlock()

	d.conf.ProtectionEnabled = status
}

// SetBlockingMode sets blocking mode properties.
func (d *DNSFilter) SetBlockingMode(mode BlockingMode, bIPv4, bIPv6 netip.Addr) {
	d.confMu.Lock()
	defer d.confMu.Unlock()

	d.conf.BlockingMode = mode
	if mode == BlockingModeCustomIP {
		d.conf.BlockingIPv4 = bIPv4
		d.conf.BlockingIPv6 = bIPv6
	}
}

// BlockingMode returns blocking mode properties.
func (d *DNSFilter) BlockingMode() (mode BlockingMode, bIPv4, bIPv6 netip.Addr) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	return d.conf.BlockingMode, d.conf.BlockingIPv4, d.conf.BlockingIPv6
}

// SetBlockedResponseTTL sets TTL for blocked responses.
func (d *DNSFilter) SetBlockedResponseTTL(ttl uint32) {
	d.confMu.Lock()
	defer d.confMu.Unlock()

	d.conf.BlockedResponseTTL = ttl
}

// BlockedResponseTTL returns TTL for blocked responses.
func (d *DNSFilter) BlockedResponseTTL() (ttl uint32) {
	d.confMu.Lock()
	defer d.confMu.Unlock()

	return d.conf.BlockedResponseTTL
}

// SafeBrowsingBlockHost returns a host for safe browsing blocked responses.
func (d *DNSFilter) SafeBrowsingBlockHost() (host string) {
	return d.conf.SafeBrowsingBlockHost
}

// ParentalBlockHost returns a host for parental protection blocked responses.
func (d *DNSFilter) ParentalBlockHost() (host string) {
	return d.conf.ParentalBlockHost
}

// Matched returns true if any match at all was found regardless of
// whether it was filtered or not.
func (r Reason) Matched() bool {
	return r != NotFilteredNotFound
}

// CheckHostRules tries to match the host against filtering rules only.
func (d *DNSFilter) CheckHostRules(host string, rrtype uint16, setts *Settings) (Result, error) {
	return d.matchHost(strings.ToLower(host), rrtype, setts)
}

// CheckHost tries to match the host against filtering rules, then safebrowsing
// and parental control rules, if they are enabled.
func (d *DNSFilter) CheckHost(
	host string,
	qtype uint16,
	setts *Settings,
) (res Result, err error) {
	// Sometimes clients try to resolve ".", which is a request to get root
	// servers.
	if host == "" {
		return Result{}, nil
	}

	host = strings.ToLower(host)

	if setts.FilteringEnabled {
		res = d.processRewrites(host, qtype)
		if res.Reason == Rewritten {
			return res, nil
		}
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

// processRewrites performs filtering based on the legacy rewrite records.
//
// Firstly, it finds CNAME rewrites for host.  If the CNAME is the same as host,
// this query isn't filtered.  If it's different, repeat the process for the new
// CNAME, breaking loops in the process.
//
// Secondly, it finds A or AAAA rewrites for host and, if found, sets res.IPList
// accordingly.  If the found rewrite has a special value of "A" or "AAAA", the
// result is an exception.
func (d *DNSFilter) processRewrites(host string, qtype uint16) (res Result) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	ctx := context.TODO()

	if !d.conf.RewritesEnabled {
		return Result{}
	}

	rewrites, matched := findRewrites(d.conf.Rewrites, host, qtype)
	if !matched {
		return Result{}
	}

	res.Reason = Rewritten

	return d.handleRewriteLoop(ctx, host, qtype, rewrites, matched, &res)
}

// handleRewriteLoop performs filtering rewrite processing based on the legacy
// rewrite records.  res must not be nil.
func (d *DNSFilter) handleRewriteLoop(
	ctx context.Context,
	host string,
	qtype uint16,
	rewrites []*LegacyRewrite,
	matched bool,
	res *Result,
) (resResult Result) {
	cnames := container.NewMapSet[string]()
	origHost := host

	for matched && len(rewrites) > 0 && rewrites[0].Type == dns.TypeCNAME {
		rw := rewrites[0]
		rwPat := rw.Domain
		rwAns := rw.Answer

		d.logger.DebugContext(ctx, "found rewrite", "host", host, "cname", rwAns)

		if origHost == rwAns || rwPat == rwAns {
			// Either a request for the hostname itself or a rewrite of
			// a pattern onto itself, both of which are an exception rules.
			// Return a not filtered result.
			return Result{}
		} else if host == rwAns && isWildcard(rwPat) {
			// An "*.example.com → sub.example.com" rewrite matching in a loop.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/4016.

			res.CanonName = host

			break
		}

		host = rwAns
		if cnames.Has(host) {
			d.logger.InfoContext(ctx, "cname loop", "host", host, "original", origHost)

			return *res
		}

		cnames.Add(host)
		res.CanonName = host
		rewrites, matched = findRewrites(d.conf.Rewrites, host, qtype)
	}

	d.setRewriteResult(ctx, res, host, rewrites, qtype)

	return *res
}

// matchBlockedServicesRules checks the host against the blocked services rules
// in settings, if any.  err is always nil, it is only there to make this a
// valid hostChecker function.
func (d *DNSFilter) matchBlockedServicesRules(
	host string,
	_ uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled {
		return Result{}, nil
	}

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
					// #nosec G115 -- The overflow is required for backwards
					// compatibility.
					FilterListID: rulelist.APIID(rule.GetFilterListID()),
					Text:         ruleText,
				}}

				d.logger.DebugContext(
					context.TODO(),
					"blocked services matched rule",
					"rule", ruleText,
					"host", host,
					"service", s.Name,
				)

				return res, nil
			}
		}
	}

	return res, nil
}

//
// Adding rule and matching against the rules
//

func newRuleStorage(filters []Filter) (rs *filterlist.RuleStorage, err error) {
	lists := make([]filterlist.Interface, 0, len(filters))
	for _, f := range filters {
		var rl filterlist.Interface
		var skip bool
		rl, skip, err = ruleListFromFilter(f)
		if skip {
			continue
		}

		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return nil, err
		}

		lists = append(lists, rl)
	}

	rs, err = filterlist.NewRuleStorage(lists)
	if err != nil {
		return nil, fmt.Errorf("creating rule storage: %w", err)
	}

	return rs, nil
}

// ruleListFromFilter returns a rule list from a Filter.
func ruleListFromFilter(f Filter) (rl filterlist.Interface, skip bool, err error) {
	if len(f.Data) != 0 {
		return filterlist.NewBytes(&filterlist.BytesConfig{
			ID:             f.ID,
			RulesText:      f.Data,
			IgnoreCosmetic: true,
		}), false, nil
	}

	if f.FilePath == "" {
		return nil, true, nil
	}

	if runtime.GOOS == "windows" {
		// On Windows we don't pass a file to urlfilter because it's
		// difficult to update this file while it's being used.
		var data []byte
		data, err = os.ReadFile(f.FilePath)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, true, nil
		} else if err != nil {
			return nil, false, fmt.Errorf("reading filter content: %w", err)
		}

		return filterlist.NewBytes(&filterlist.BytesConfig{
			ID:             f.ID,
			RulesText:      data,
			IgnoreCosmetic: true,
		}), false, nil
	}

	rl, err = filterlist.NewFile(&filterlist.FileConfig{
		ID:             f.ID,
		Path:           f.FilePath,
		IgnoreCosmetic: true,
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, true, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("creating file rule list with %q: %w", f.FilePath, err)
	}

	return rl, false, nil
}

// Initialize urlfilter objects.
func (d *DNSFilter) initFiltering(ctx context.Context, allowFilters, blockFilters []Filter) (err error) {
	rulesStorage, err := newRuleStorage(blockFilters)
	if err != nil {
		return err
	}

	rulesStorageAllow, err := newRuleStorage(allowFilters)
	if err != nil {
		return err
	}

	filteringEngine := urlfilter.NewDNSEngine(rulesStorage)
	filteringEngineAllow := urlfilter.NewDNSEngine(rulesStorageAllow)

	func() {
		d.engineLock.Lock()
		defer d.engineLock.Unlock()

		d.reset(ctx)
		d.rulesStorage = rulesStorage
		d.filteringEngine = filteringEngine
		d.rulesStorageAllow = rulesStorageAllow
		d.filteringEngineAllow = filteringEngineAllow
	}()

	// Make sure that the OS reclaims memory as soon as possible.
	debug.FreeOSMemory()

	d.logger.DebugContext(ctx, "initialized filtering engine")

	return nil
}

// hostRules is a helper that converts a slice of host rules into a slice of the
// rules.Rule interface values.
func hostRulesToRules(netRules []*rules.HostRule) (res []rules.Rule) {
	if netRules == nil {
		return nil
	}

	res = make([]rules.Rule, len(netRules))
	for i, nr := range netRules {
		res[i] = nr
	}

	return res
}

// matchHostProcessAllowList processes the allowlist logic of host matching.
func (d *DNSFilter) matchHostProcessAllowList(
	ctx context.Context,
	host string,
	dnsres *urlfilter.DNSResult,
) (res Result, err error) {
	var matchedRules []rules.Rule
	if dnsres.NetworkRule != nil {
		matchedRules = []rules.Rule{dnsres.NetworkRule}
	} else if len(dnsres.HostRulesV4) > 0 {
		matchedRules = hostRulesToRules(dnsres.HostRulesV4)
	} else if len(dnsres.HostRulesV6) > 0 {
		matchedRules = hostRulesToRules(dnsres.HostRulesV6)
	}

	if len(matchedRules) == 0 {
		return Result{}, fmt.Errorf("invalid dns result: rules are empty")
	}

	d.logger.DebugContext(
		ctx,
		"allowlist rules for host",
		"host", host,
		"rules", matchedRules,
	)

	return makeResult(matchedRules, NotFilteredAllowList), nil
}

// matchHostProcessDNSResult processes the matched DNS filtering result.
func (d *DNSFilter) matchHostProcessDNSResult(
	qtype uint16,
	dnsres *urlfilter.DNSResult,
) (res Result) {
	if dnsres.NetworkRule != nil {
		reason := FilteredBlockList
		if dnsres.NetworkRule.Whitelist {
			reason = NotFilteredAllowList
		}

		return makeResult([]rules.Rule{dnsres.NetworkRule}, reason)
	}

	if result, ok := resultFromHostRules(qtype, dnsres); ok {
		return result
	}

	return hostResultForOtherQType(dnsres)
}

// resultFromHostRules handles the HostRulesV4/HostRulesV6 case for
// [matchHostProcessDNSResult].  dnsres must not be nil.
func resultFromHostRules(qtype uint16, dnsres *urlfilter.DNSResult) (res Result, ok bool) {
	if qtype == dns.TypeA && dnsres.HostRulesV4 != nil {
		res = makeResult(hostRulesToRules(dnsres.HostRulesV4), FilteredBlockList)
		for i, hr := range dnsres.HostRulesV4 {
			res.Rules[i].IP = hr.IP
		}

		return res, true
	}

	if qtype == dns.TypeAAAA && dnsres.HostRulesV6 != nil {
		res = makeResult(hostRulesToRules(dnsres.HostRulesV6), FilteredBlockList)
		for i, hr := range dnsres.HostRulesV6 {
			res.Rules[i].IP = hr.IP
		}

		return res, true
	}

	return Result{}, false
}

// hostResultForOtherQType returns a result based on the host rules in dnsres,
// if any.  dnsres.HostRulesV4 take precedence over dnsres.HostRulesV6.
func hostResultForOtherQType(dnsres *urlfilter.DNSResult) (res Result) {
	if len(dnsres.HostRulesV4) != 0 {
		return makeResult([]rules.Rule{dnsres.HostRulesV4[0]}, FilteredBlockList)
	}

	if len(dnsres.HostRulesV6) != 0 {
		return makeResult([]rules.Rule{dnsres.HostRulesV6[0]}, FilteredBlockList)
	}

	return Result{}
}

// matchHost is a low-level way to check only if host is filtered by rules,
// skipping expensive safebrowsing and parental lookups.
func (d *DNSFilter) matchHost(
	host string,
	rrtype uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.FilteringEnabled {
		return Result{}, nil
	}

	ctx := context.TODO()

	// TODO(f.setrakov): Reuse client tags and identifiers.
	ufReq := &urlfilter.DNSRequest{
		Hostname:          host,
		ClientTags:        container.NewSortedSliceSet(setts.ClientTags...),
		ClientIP:          setts.ClientIP,
		ClientIdentifiers: container.NewSortedSliceSet(setts.ClientName),
		DNSType:           rrtype,
	}

	d.engineLock.RLock()
	// Keep in mind that this lock must be held no just when calling Match() but
	// also while using the rules returned by it.
	//
	// TODO(e.burkov):  Inspect if the above is true.
	defer d.engineLock.RUnlock()

	if setts.ProtectionEnabled && d.filteringEngineAllow != nil {
		dnsres, ok := d.filteringEngineAllow.MatchRequest(ufReq)
		if ok {
			return d.matchHostProcessAllowList(ctx, host, dnsres)
		}
	}

	if d.filteringEngine == nil {
		return Result{}, nil
	}

	dnsres, matchedEngine := d.filteringEngine.MatchRequest(ufReq)

	// Check DNS rewrites first, because the API there is a bit awkward.
	dnsRWRes := d.processDNSResultRewrites(dnsres, host)
	if dnsRWRes.Reason != NotFilteredNotFound {
		return dnsRWRes, nil
	} else if !matchedEngine {
		return Result{}, nil
	}

	if !setts.ProtectionEnabled {
		// Don't check non-dnsrewrite filtering results.
		return Result{}, nil
	}

	res = d.matchHostProcessDNSResult(rrtype, dnsres)
	for _, r := range res.Rules {
		d.logger.DebugContext(
			ctx,
			"found rule for host",
			"host", host,
			"rule", r.Text,
			"filter_list_id", r.FilterListID,
		)
	}

	return res, nil
}

// makeResult returns a properly constructed Result.
func makeResult(matchedRules []rules.Rule, reason Reason) (res Result) {
	resRules := make([]*ResultRule, len(matchedRules))
	for i, mr := range matchedRules {
		resRules[i] = NewResultRule(mr)
	}

	return Result{
		Rules:      resRules,
		Reason:     reason,
		IsFiltered: reason == FilteredBlockList,
	}
}

// InitModule manually initializes blocked services map.  l must not be nil.
func InitModule(ctx context.Context, l *slog.Logger) {
	initBlockedServices(ctx, l)
}

// New creates properly initialized DNS Filter that is ready to be used.  c must
// be non-nil.
func New(c *Config, blockFilters []Filter) (d *DNSFilter, err error) {
	ctx := context.TODO()

	d = &DNSFilter{
		logger: c.Logger,
		// #nosec G115 -- The Unix epoch time is highly unlikely to be negative.
		idGen:                  newIDGenerator(uint64(time.Now().Unix()), c.Logger),
		bufPool:                syncutil.NewSlicePool[byte](rulelist.DefaultRuleBufSize),
		safeSearch:             c.SafeSearch,
		refreshLock:            &sync.Mutex{},
		safeBrowsingChecker:    c.SafeBrowsingChecker,
		parentalControlChecker: c.ParentalControlChecker,
		applyClientFiltering:   c.ApplyClientFiltering,
		confMu:                 &sync.RWMutex{},
	}

	err = d.validateSafeFSPatterns(c.SafeFSPatterns)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	d.hostCheckers = []hostChecker{{
		check: d.matchSysHosts,
		name:  "hosts container",
	}, {
		check: d.matchHost,
		name:  "filtering",
	}, {
		check: d.matchBlockedServicesRules,
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

	defer func() { err = errors.Annotate(err, "filtering: %w") }()

	d.conf = c
	d.conf.filtersMu = &sync.RWMutex{}

	err = d.prepareRewrites(ctx)
	if err != nil {
		return nil, fmt.Errorf("rewrites: preparing: %w", err)
	}

	if d.conf.BlockedServices != nil {
		d.conf.BlockedServices.FilterUnknownIDs(ctx, d.logger)
		err = d.conf.BlockedServices.Validate()
		if err != nil {
			return nil, fmt.Errorf("initializing blocked services: %w", err)
		}
	}

	if blockFilters != nil {
		err = d.initFiltering(ctx, nil, blockFilters)
		if err != nil {
			d.Close()

			return nil, fmt.Errorf("initializing filtering subsystem: %w", err)
		}
	}

	err = os.MkdirAll(filepath.Join(d.conf.DataDir, filterDir), aghos.DefaultPermDir)
	if err != nil {
		d.Close()

		return nil, fmt.Errorf("making filtering directory: %w", err)
	}

	d.loadFilters(ctx, d.conf.Filters)
	d.loadFilters(ctx, d.conf.WhitelistFilters)

	d.conf.Filters = deduplicateFilters(d.conf.Filters)
	d.conf.WhitelistFilters = deduplicateFilters(d.conf.WhitelistFilters)

	d.idGen.fix(d.conf.Filters)
	d.idGen.fix(d.conf.WhitelistFilters)

	return d, nil
}

// validateSafeFSPatterns validates and stores patterns for local filtering‑rule
// files.
func (d *DNSFilter) validateSafeFSPatterns(patterns []string) (err error) {
	for i, p := range patterns {
		// Use Match to validate the patterns here.
		_, err = filepath.Match(p, "test")
		if err != nil {
			return fmt.Errorf("safe_fs_patterns: at index %d: %w", i, err)
		}

		d.safeFSPatterns = append(d.safeFSPatterns, p)
	}

	return nil
}

// Start registers web handlers and starts filters updates loop.
func (d *DNSFilter) Start() {
	d.filtersInitializerChan = make(chan filtersInitializerParams, 1)
	d.done = make(chan struct{}, 1)

	d.RegisterFilteringHandlers()

	go d.updatesLoop(context.TODO())
}

// updatesLoop initializes new filters and checks for filters updates in a loop.
func (d *DNSFilter) updatesLoop(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, d.logger)

	ivl := time.Second * 5
	t := time.NewTimer(ivl)

	for {
		select {
		case params := <-d.filtersInitializerChan:
			err := d.initFiltering(ctx, params.allowFilters, params.blockFilters)
			if err != nil {
				d.logger.ErrorContext(ctx, "initializing", slogutil.KeyError, err)

				continue
			}
		case <-t.C:
			ivl = d.periodicallyRefreshFilters(ivl)
			t.Reset(ivl)
		case <-d.done:
			t.Stop()

			return
		}
	}
}

// periodicallyRefreshFilters checks for filters updates and returns time
// interval for the next update.
func (d *DNSFilter) periodicallyRefreshFilters(ivl time.Duration) (nextIvl time.Duration) {
	const maxInterval = time.Hour

	if d.conf.FiltersUpdateIntervalHours == 0 {
		return ivl
	}

	isNetErr, ok := false, false
	_, isNetErr, ok = d.tryRefreshFilters(true, true, false)

	if ok && !isNetErr {
		ivl = maxInterval
	} else if isNetErr {
		ivl *= 2
		ivl = max(ivl, maxInterval)
	}

	return ivl
}

// Safe browsing and parental control methods.

// TODO(a.garipov): Unify with checkParental.
func (d *DNSFilter) checkSafeBrowsing(
	host string,
	_ uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled || !setts.SafeBrowsingEnabled {
		return Result{}, nil
	}

	ctx := context.TODO()
	if d.logger.Enabled(ctx, slogutil.LevelDebug) {
		startTime := time.Now()
		defer func() {
			elapsed := time.Since(startTime)
			d.logger.DebugContext(ctx, "safebrowsing lookup", "host", host, "elapsed", elapsed)
		}()
	}

	res = Result{
		Rules: []*ResultRule{{
			Text:         "adguard-malware-shavar",
			FilterListID: rulelist.APIIDSafeBrowsing,
		}},
		Reason:     FilteredSafeBrowsing,
		IsFiltered: true,
	}

	block, err := d.safeBrowsingChecker.Check(host)
	if !block || err != nil {
		return Result{}, err
	}

	return res, nil
}

// TODO(a.garipov): Unify with checkSafeBrowsing.
func (d *DNSFilter) checkParental(
	host string,
	_ uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled || !setts.ParentalEnabled {
		return Result{}, nil
	}

	ctx := context.TODO()
	if d.logger.Enabled(ctx, slogutil.LevelDebug) {
		startTime := time.Now()
		defer func() {
			elapsed := time.Since(startTime)
			d.logger.DebugContext(ctx, "parental lookup", "host", host, "elapsed", elapsed)
		}()
	}

	res = Result{
		Rules: []*ResultRule{{
			Text:         "parental CATEGORY_BLACKLISTED",
			FilterListID: rulelist.APIIDParentalControl,
		}},
		Reason:     FilteredParental,
		IsFiltered: true,
	}

	block, err := d.parentalControlChecker.Check(host)
	if !block || err != nil {
		return Result{}, err
	}

	return res, nil
}
