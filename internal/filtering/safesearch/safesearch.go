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

// DefaultSafeSearch is the default safesearch struct.
type DefaultSafeSearch struct {
	engine          *urlfilter.DNSEngine
	safeSearchCache cache.Cache
	resolver        filtering.Resolver
	cacheTime       time.Duration
}

// NewDefaultSafeSearch returns new safesearch struct.  CacheTime is an element
// TTL (in minutes).
func NewDefaultSafeSearch(
	conf filtering.SafeSearchConfig,
	cacheSize uint,
	cacheTime time.Duration,
) (ss *DefaultSafeSearch, err error) {
	engine, err := newEngine(filtering.SafeSearchListID, conf)
	if err != nil {
		return nil, err
	}

	var resolver filtering.Resolver = net.DefaultResolver
	if conf.CustomResolver != nil {
		resolver = conf.CustomResolver
	}

	return &DefaultSafeSearch{
		engine: engine,
		safeSearchCache: cache.New(cache.Config{
			EnableLRU: true,
			MaxSize:   cacheSize,
		}),
		cacheTime: cacheTime,
		resolver:  resolver,
	}, nil
}

// newEngine creates new engine for provided safe search configuration.
func newEngine(listID int, conf filtering.SafeSearchConfig) (engine *urlfilter.DNSEngine, err error) {
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
		return nil, fmt.Errorf("creating rule storage: %w", err)
	}

	engine = urlfilter.NewDNSEngine(rs)
	log.Info("safesearch: filter %d: reset %d rules", listID, engine.RulesCount)

	return engine, nil
}

// type check
var _ filtering.SafeSearch = (*DefaultSafeSearch)(nil)

// SearchHost implements the [filtering.SafeSearch] interface for *DefaultSafeSearch.
func (ss *DefaultSafeSearch) SearchHost(host string, qtype uint16) (res *rules.DNSRewrite) {
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

// CheckHost implements the [filtering.SafeSearch] interface for
// *DefaultSafeSearch.
func (ss *DefaultSafeSearch) CheckHost(
	host string,
	qtype uint16,
) (res filtering.Result, err error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("safesearch: lookup for %s", host)
	}

	// Check cache. Return cached result if it was found
	cachedValue, isFound := ss.getCachedResult(host)
	if isFound {
		log.Debug("safesearch: found in cache: %s", host)

		return cachedValue, nil
	}

	rewrite := ss.SearchHost(host, qtype)
	if rewrite == nil {
		return filtering.Result{}, nil
	}

	dRes, err := ss.newResult(rewrite, qtype)
	if err != nil {
		log.Debug("safesearch: failed to lookup addresses for %s: %s", host, err)

		return filtering.Result{}, err
	}

	if dRes != nil {
		res = *dRes
		ss.setCacheResult(host, res)

		return res, nil
	}

	return filtering.Result{}, fmt.Errorf("no ipv4 addresses in safe search response for %s", host)
}

// newResult creates Result object from rewrite rule.
func (ss *DefaultSafeSearch) newResult(
	rewrite *rules.DNSRewrite,
	qtype uint16,
) (res *filtering.Result, err error) {
	res = &filtering.Result{
		Rules: []*filtering.ResultRule{{
			FilterListID: filtering.SafeSearchListID,
		}},
		Reason:     filtering.FilteredSafeSearch,
		IsFiltered: true,
	}

	if rewrite.RRType == qtype && (qtype == dns.TypeA || qtype == dns.TypeAAAA) {
		ip, ok := rewrite.Value.(net.IP)
		if !ok || ip == nil {
			return nil, nil
		}

		res.Rules[0].IP = ip

		return res, nil
	}

	if rewrite.NewCNAME == "" {
		return nil, nil
	}

	ips, err := ss.resolver.LookupIP(context.Background(), "ip", rewrite.NewCNAME)
	if err != nil {
		return nil, err
	}

	for _, ip := range ips {
		if ip = ip.To4(); ip == nil {
			continue
		}

		res.Rules[0].IP = ip

		return res, nil
	}

	return nil, nil
}

// setCacheResult stores data in cache for host.
func (ss *DefaultSafeSearch) setCacheResult(host string, res filtering.Result) {
	expire := uint32(time.Now().Add(ss.cacheTime).Unix())
	exp := make([]byte, 4)
	binary.BigEndian.PutUint32(exp, expire)
	buf := bytes.NewBuffer(exp)

	err := gob.NewEncoder(buf).Encode(res)
	if err != nil {
		log.Error("safesearch: cache encoding: %s", err)

		return
	}

	val := buf.Bytes()
	_ = ss.safeSearchCache.Set([]byte(host), val)

	log.Debug("safesearch: stored in cache: %s (%d bytes)", host, len(val))
}

// getCachedResult returns stored data from cache for host.
func (ss *DefaultSafeSearch) getCachedResult(host string) (res filtering.Result, ok bool) {
	res = filtering.Result{}

	data := ss.safeSearchCache.Get([]byte(host))
	if data == nil {
		return res, false
	}

	exp := binary.BigEndian.Uint32(data[:4])
	if exp <= uint32(time.Now().Unix()) {
		ss.safeSearchCache.Del([]byte(host))

		return res, false
	}

	buf := bytes.NewBuffer(data[4:])

	err := gob.NewDecoder(buf).Decode(&res)
	if err != nil {
		log.Debug("safesearch: cache decoding: %s", err)

		return filtering.Result{}, false
	}

	return res, true
}
