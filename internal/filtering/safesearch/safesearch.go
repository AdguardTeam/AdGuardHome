// Package safesearch implements safesearch host matching.
package safesearch

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/c2h5oh/datasize"
	"github.com/miekg/dns"
)

// Attribute keys and values for logging.
const (
	LogPrefix    = "safesearch"
	LogKeyClient = "client"
)

// Service is a enum with service names used as search providers.
type Service string

// Service enum members.
const (
	Bing       Service = "bing"
	DuckDuckGo Service = "duckduckgo"
	Ecosia     Service = "ecosia"
	Google     Service = "google"
	Pixabay    Service = "pixabay"
	Yandex     Service = "yandex"
	YouTube    Service = "youtube"
)

// isServiceProtected returns true if the service safe search is active.
func isServiceProtected(s filtering.SafeSearchConfig, service Service) (ok bool) {
	switch service {
	case Bing:
		return s.Bing
	case DuckDuckGo:
		return s.DuckDuckGo
	case Ecosia:
		return s.Ecosia
	case Google:
		return s.Google
	case Pixabay:
		return s.Pixabay
	case Yandex:
		return s.Yandex
	case YouTube:
		return s.YouTube
	default:
		panic(fmt.Errorf("safesearch: invalid sources: not found service %q", service))
	}
}

// DefaultConfig is the configuration structure for [Default].
type DefaultConfig struct {
	// Logger is used for logging the operation of the safe search filter.
	Logger *slog.Logger

	// ClientName is the name of the persistent client associated with the safe
	// search filter, if there is one.
	ClientName string

	// CacheSize is the size of the filter results cache.
	CacheSize uint

	// CacheTTL is the Time to Live duration for cached items.
	CacheTTL time.Duration

	// ServicesConfig contains safe search settings for services.  It must not
	// be nil.
	ServicesConfig filtering.SafeSearchConfig
}

// Default is the default safe search filter that uses filtering rules with the
// dnsrewrite modifier.
type Default struct {
	// logger is used for logging the operation of the safe search filter.
	logger *slog.Logger

	// mu protects engine.
	mu *sync.RWMutex

	// engine is the filtering engine that contains the DNS rewrite rules.
	// engine may be nil, which means that this safe search filter is disabled.
	engine *urlfilter.DNSEngine

	// cache stores safe search filtering results.
	cache cache.Cache

	// cacheTTL is the Time to Live duration for cached items.
	cacheTTL time.Duration
}

// NewDefault returns an initialized default safe search filter.  ctx is used
// to log the initial refresh.
func NewDefault(ctx context.Context, conf *DefaultConfig) (ss *Default, err error) {
	ss = &Default{
		logger: conf.Logger,
		mu:     &sync.RWMutex{},
		cache: cache.New(cache.Config{
			EnableLRU: true,
			MaxSize:   conf.CacheSize,
		}),
		cacheTTL: conf.CacheTTL,
	}

	// TODO(s.chzhen):  Move to [Default.InitialRefresh].
	err = ss.resetEngine(ctx, rulelist.URLFilterIDSafeSearch, conf.ServicesConfig)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	return ss, nil
}

// resetEngine creates new engine for provided safe search configuration and
// sets it in ss.
func (ss *Default) resetEngine(
	ctx context.Context,
	listID int,
	conf filtering.SafeSearchConfig,
) (err error) {
	if !conf.Enabled {
		ss.logger.DebugContext(ctx, "disabled")

		return nil
	}

	var sb strings.Builder
	for service, serviceRules := range safeSearchRules {
		if isServiceProtected(conf, service) {
			sb.WriteString(serviceRules)
		}
	}

	strList := &filterlist.StringRuleList{
		ID:             listID,
		RulesText:      sb.String(),
		IgnoreCosmetic: true,
	}

	rs, err := filterlist.NewRuleStorage([]filterlist.RuleList{strList})
	if err != nil {
		return fmt.Errorf("creating rule storage: %w", err)
	}

	ss.engine = urlfilter.NewDNSEngine(rs)

	ss.logger.InfoContext(ctx, "reset rules", "count", ss.engine.RulesCount)

	return nil
}

// type check
var _ filtering.SafeSearch = (*Default)(nil)

// CheckHost implements the [filtering.SafeSearch] interface for *Default.
func (ss *Default) CheckHost(
	ctx context.Context,
	host string,
	qtype rules.RRType,
) (res filtering.Result, err error) {
	start := time.Now()
	defer func() {
		ss.logger.DebugContext(ctx, "lookup finished", "host", host, "elapsed", time.Since(start))
	}()

	switch qtype {
	case dns.TypeA, dns.TypeAAAA, dns.TypeHTTPS:
		// Go on.
	default:
		return filtering.Result{}, nil
	}

	// Check cache. Return cached result if it was found
	cachedValue, isFound := ss.getCachedResult(ctx, host, qtype)
	if isFound {
		ss.logger.DebugContext(ctx, "found in cache", "host", host)

		return cachedValue, nil
	}

	rewrite := ss.searchHost(host, qtype)
	if rewrite == nil {
		return filtering.Result{}, nil
	}

	fltRes, err := ss.newResult(rewrite, qtype)
	if err != nil {
		ss.logger.ErrorContext(ctx, "looking up addresses", "host", host, slogutil.KeyError, err)

		return filtering.Result{}, err
	}

	res = *fltRes

	// TODO(a.garipov): Consider switch back to resolving CNAME records IPs and
	// saving results to cache.
	ss.setCacheResult(ctx, host, qtype, res)

	return res, nil
}

// searchHost looks up DNS rewrites in the internal DNS filtering engine.
func (ss *Default) searchHost(host string, qtype rules.RRType) (res *rules.DNSRewrite) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if ss.engine == nil {
		return nil
	}

	r, _ := ss.engine.MatchRequest(&urlfilter.DNSRequest{
		Hostname: strings.ToLower(host),
		DNSType:  qtype,
	})

	rewritesRules := r.DNSRewrites()
	if len(rewritesRules) > 0 {
		return rewritesRules[0].DNSRewrite
	}

	return nil
}

// newResult creates Result object from rewrite rule.  qtype must be either
// [dns.TypeA] or [dns.TypeAAAA], or [dns.TypeHTTPS].  If err is nil, res is
// never nil, so that the empty result is converted into a NODATA response.
func (ss *Default) newResult(
	rewrite *rules.DNSRewrite,
	qtype rules.RRType,
) (res *filtering.Result, err error) {
	res = &filtering.Result{
		Reason:     filtering.FilteredSafeSearch,
		IsFiltered: true,
	}

	if rewrite.RRType == qtype {
		ip, ok := rewrite.Value.(netip.Addr)
		if !ok || ip == (netip.Addr{}) {
			return nil, fmt.Errorf("expected ip rewrite value, got %T(%[1]v)", rewrite.Value)
		}

		res.Rules = []*filtering.ResultRule{{
			FilterListID: rulelist.URLFilterIDSafeSearch,
			IP:           ip,
		}}

		return res, nil
	}

	res.CanonName = rewrite.NewCNAME

	return res, nil
}

// setCacheResult stores data in cache for host.  qtype is expected to be either
// [dns.TypeA] or [dns.TypeAAAA].
func (ss *Default) setCacheResult(
	ctx context.Context,
	host string,
	qtype rules.RRType,
	res filtering.Result,
) {
	expire := uint32(time.Now().Add(ss.cacheTTL).Unix())
	exp := make([]byte, 4)
	binary.BigEndian.PutUint32(exp, expire)
	buf := bytes.NewBuffer(exp)

	err := gob.NewEncoder(buf).Encode(res)
	if err != nil {
		ss.logger.ErrorContext(ctx, "cache encoding", slogutil.KeyError, err)

		return
	}

	val := buf.Bytes()
	_ = ss.cache.Set([]byte(dns.Type(qtype).String()+" "+host), val)

	ss.logger.DebugContext(
		ctx,
		"stored in cache",
		"host", host,
		"entry_size", datasize.ByteSize(len(val)),
	)
}

// getCachedResult returns stored data from cache for host.  qtype is expected
// to be either [dns.TypeA] or [dns.TypeAAAA].
func (ss *Default) getCachedResult(
	ctx context.Context,
	host string,
	qtype rules.RRType,
) (res filtering.Result, ok bool) {
	res = filtering.Result{}

	data := ss.cache.Get([]byte(dns.Type(qtype).String() + " " + host))
	if data == nil {
		return res, false
	}

	exp := binary.BigEndian.Uint32(data[:4])
	if exp <= uint32(time.Now().Unix()) {
		ss.cache.Del([]byte(host))

		return res, false
	}

	buf := bytes.NewBuffer(data[4:])

	err := gob.NewDecoder(buf).Decode(&res)
	if err != nil {
		ss.logger.ErrorContext(ctx, "cache decoding", slogutil.KeyError, err)

		return filtering.Result{}, false
	}

	return res, true
}

// Update implements the [filtering.SafeSearch] interface for *Default.  Update
// ignores the CustomResolver and Enabled fields.
func (ss *Default) Update(ctx context.Context, conf filtering.SafeSearchConfig) (err error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	err = ss.resetEngine(ctx, rulelist.URLFilterIDSafeSearch, conf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	ss.cache.Clear()

	return nil
}
