// Package safesearch implements safesearch host matching.
package safesearch

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// Service is a enum with service names used as search providers.
type Service string

// Service enum members.
const (
	Bing       Service = "bing"
	DuckDuckGo Service = "duckduckgo"
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

// Default is the default safe search filter that uses filtering rules with the
// dnsrewrite modifier.
type Default struct {
	// mu protects engine.
	mu *sync.RWMutex

	// engine is the filtering engine that contains the DNS rewrite rules.
	// engine may be nil, which means that this safe search filter is disabled.
	engine *urlfilter.DNSEngine

	cache     cache.Cache
	resolver  filtering.Resolver
	logPrefix string
	cacheTTL  time.Duration
}

// NewDefault returns an initialized default safe search filter.  name is used
// for logging.
func NewDefault(
	conf filtering.SafeSearchConfig,
	name string,
	cacheSize uint,
	cacheTTL time.Duration,
) (ss *Default, err error) {
	var resolver filtering.Resolver = net.DefaultResolver
	if conf.CustomResolver != nil {
		resolver = conf.CustomResolver
	}

	ss = &Default{
		mu: &sync.RWMutex{},

		cache: cache.New(cache.Config{
			EnableLRU: true,
			MaxSize:   cacheSize,
		}),
		resolver: resolver,
		// Use %s, because the client safe-search names already contain double
		// quotes.
		logPrefix: fmt.Sprintf("safesearch %s: ", name),
		cacheTTL:  cacheTTL,
	}

	err = ss.resetEngine(filtering.SafeSearchListID, conf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	return ss, nil
}

// log is a helper for logging  that includes the name of the safe search
// filter.  level must be one of [log.DEBUG], [log.INFO], and [log.ERROR].
func (ss *Default) log(level log.Level, msg string, args ...any) {
	switch level {
	case log.DEBUG:
		log.Debug(ss.logPrefix+msg, args...)
	case log.INFO:
		log.Info(ss.logPrefix+msg, args...)
	case log.ERROR:
		log.Error(ss.logPrefix+msg, args...)
	default:
		panic(fmt.Errorf("safesearch: unsupported logging level %d", level))
	}
}

// resetEngine creates new engine for provided safe search configuration and
// sets it in ss.
func (ss *Default) resetEngine(
	listID int,
	conf filtering.SafeSearchConfig,
) (err error) {
	if !conf.Enabled {
		ss.log(log.INFO, "disabled")

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

	ss.log(log.INFO, "reset %d rules", ss.engine.RulesCount)

	return nil
}

// type check
var _ filtering.SafeSearch = (*Default)(nil)

// CheckHost implements the [filtering.SafeSearch] interface for *Default.
func (ss *Default) CheckHost(host string, qtype rules.RRType) (res filtering.Result, err error) {
	start := time.Now()
	defer func() {
		ss.log(log.DEBUG, "lookup for %q finished in %s", host, time.Since(start))
	}()

	if qtype != dns.TypeA && qtype != dns.TypeAAAA {
		return filtering.Result{}, fmt.Errorf("unsupported question type %s", dns.Type(qtype))
	}

	// Check cache. Return cached result if it was found
	cachedValue, isFound := ss.getCachedResult(host, qtype)
	if isFound {
		ss.log(log.DEBUG, "found in cache: %q", host)

		return cachedValue, nil
	}

	rewrite := ss.searchHost(host, qtype)
	if rewrite == nil {
		return filtering.Result{}, nil
	}

	fltRes, err := ss.newResult(rewrite, qtype)
	if err != nil {
		ss.log(log.DEBUG, "looking up addresses for %q: %s", host, err)

		return filtering.Result{}, err
	}

	res = *fltRes
	ss.setCacheResult(host, qtype, res)

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
// [dns.TypeA] or [dns.TypeAAAA].  If err is nil, res is never nil, so that the
// empty result is converted into a NODATA response.
//
// TODO(a.garipov): Use the main rewrite result mechanism used in
// [dnsforward.Server.filterDNSRequest].
func (ss *Default) newResult(
	rewrite *rules.DNSRewrite,
	qtype rules.RRType,
) (res *filtering.Result, err error) {
	res = &filtering.Result{
		Rules: []*filtering.ResultRule{{
			FilterListID: filtering.SafeSearchListID,
		}},
		Reason:     filtering.FilteredSafeSearch,
		IsFiltered: true,
	}

	if rewrite.RRType == qtype {
		v := rewrite.Value
		ip, ok := v.(net.IP)
		if !ok || ip == nil {
			return nil, fmt.Errorf("expected ip rewrite value, got %T(%[1]v)", v)
		}

		res.Rules[0].IP = ip

		return res, nil
	}

	host := rewrite.NewCNAME
	if host == "" {
		return res, nil
	}

	ss.log(log.DEBUG, "resolving %q", host)

	ips, err := ss.resolver.LookupIP(context.Background(), qtypeToProto(qtype), host)
	if err != nil {
		return nil, fmt.Errorf("resolving cname: %w", err)
	}

	ss.log(log.DEBUG, "resolved %s", ips)

	for _, ip := range ips {
		// TODO(a.garipov): Remove this filtering once the resolver we use
		// actually learns about network.
		ip = fitToProto(ip, qtype)
		if ip == nil {
			continue
		}

		res.Rules[0].IP = ip
	}

	return res, nil
}

// qtypeToProto returns "ip4" for [dns.TypeA] and "ip6" for [dns.TypeAAAA].
// It panics for other types.
func qtypeToProto(qtype rules.RRType) (proto string) {
	switch qtype {
	case dns.TypeA:
		return "ip4"
	case dns.TypeAAAA:
		return "ip6"
	default:
		panic(fmt.Errorf("safesearch: unsupported question type %s", dns.Type(qtype)))
	}
}

// fitToProto returns a non-nil IP address if ip is the correct protocol version
// for qtype.  qtype is expected to be either [dns.TypeA] or [dns.TypeAAAA].
func fitToProto(ip net.IP, qtype rules.RRType) (res net.IP) {
	ip4 := ip.To4()
	if qtype == dns.TypeA {
		return ip4
	}

	if ip4 == nil {
		return ip
	}

	return nil
}

// setCacheResult stores data in cache for host.  qtype is expected to be either
// [dns.TypeA] or [dns.TypeAAAA].
func (ss *Default) setCacheResult(host string, qtype rules.RRType, res filtering.Result) {
	expire := uint32(time.Now().Add(ss.cacheTTL).Unix())
	exp := make([]byte, 4)
	binary.BigEndian.PutUint32(exp, expire)
	buf := bytes.NewBuffer(exp)

	err := gob.NewEncoder(buf).Encode(res)
	if err != nil {
		ss.log(log.ERROR, "cache encoding: %s", err)

		return
	}

	val := buf.Bytes()
	_ = ss.cache.Set([]byte(dns.Type(qtype).String()+" "+host), val)

	ss.log(log.DEBUG, "stored in cache: %q, %d bytes", host, len(val))
}

// getCachedResult returns stored data from cache for host.  qtype is expected
// to be either [dns.TypeA] or [dns.TypeAAAA].
func (ss *Default) getCachedResult(
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
		ss.log(log.ERROR, "cache decoding: %s", err)

		return filtering.Result{}, false
	}

	return res, true
}

// Update implements the [filtering.SafeSearch] interface for *Default.  Update
// ignores the CustomResolver and Enabled fields.
func (ss *Default) Update(conf filtering.SafeSearchConfig) (err error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	err = ss.resetEngine(filtering.SafeSearchListID, conf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	ss.cache.Clear()

	return nil
}
