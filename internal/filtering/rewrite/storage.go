// Package rewrite implements DNS Rewrites storage and request matching.
package rewrite

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// Storage is a storage for rewrite rules.
type Storage interface {
	// MatchRequest returns matching dnsrewrites for the specified request.
	MatchRequest(dReq *urlfilter.DNSRequest) (rws []*rules.DNSRewrite)

	// Add adds item to the storage.
	Add(item *Item) (err error)

	// Remove deletes item from the storage.
	Remove(item *Item) (err error)

	// List returns all items from the storage.
	List() (items []*Item)
}

// Config is the configuration for DefaultStorage.
type Config struct {
	// logger is used for logging storage processes.  It must not be nil.
	Logger *slog.Logger

	// Rewrites stores the rewrite entries.  It must not be nil.
	Rewrites []*Item

	// ListID is used as an identifier of the underlying rules list.
	ListID rules.ListID
}

// DefaultStorage is the default storage for rewrite rules.
type DefaultStorage struct {
	// logger is used for logging storage processes.  It must not be nil.
	logger *slog.Logger

	// mu protects items.
	mu *sync.RWMutex

	// engine is the DNS filtering engine.
	engine *urlfilter.DNSEngine

	// ruleList is the filtering rule ruleList used by the engine.
	ruleList filterlist.Interface

	// rewrites stores the rewrite entries from configuration.
	rewrites []*Item

	// urlFilterID is the synthetic integer identifier for the urlfilter engine.
	urlFilterID rules.ListID
}

// NewDefaultStorage returns new rewrites storage.  conf must not be nil.
func NewDefaultStorage(conf *Config) (s *DefaultStorage, err error) {
	s = &DefaultStorage{
		logger:      conf.Logger,
		mu:          &sync.RWMutex{},
		urlFilterID: conf.ListID,
		rewrites:    conf.Rewrites,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.resetRules()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// type check
var _ Storage = (*DefaultStorage)(nil)

// MatchRequest implements the [Storage] interface for *DefaultStorage.
func (s *DefaultStorage) MatchRequest(dReq *urlfilter.DNSRequest) (rws []*rules.DNSRewrite) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.TODO()

	rewriteRules := s.rewriteRulesForReq(dReq)
	if len(rewriteRules) == 0 {
		return nil
	}

	resolvedRules, wildcardRewrite := s.resolveCNAMEChain(ctx, dReq, rewriteRules)
	if wildcardRewrite != nil {
		return []*rules.DNSRewrite{wildcardRewrite}
	}

	if resolvedRules == nil {
		return nil
	}

	return s.collectDNSRewrites(resolvedRules, dReq.DNSType)
}

// resolveCNAMEChain follows the CNAME chain for a DNS request, handling loops
// and special cases.  dReq must not be nil, and rewriteRules must not contain
// nil elements.
func (s *DefaultStorage) resolveCNAMEChain(
	ctx context.Context,
	dReq *urlfilter.DNSRequest,
	rewriteRules []*rules.NetworkRule,
) (resolvedRules []*rules.NetworkRule, wildcardRewrite *rules.DNSRewrite) {
	// TODO(a.garipov): Check cnames for cycles on initialization.
	cnames := container.NewMapSet[string]()
	host := dReq.Hostname
	for len(rewriteRules) > 0 &&
		rewriteRules[0].DNSRewrite != nil &&
		rewriteRules[0].DNSRewrite.NewCNAME != "" {
		rule := rewriteRules[0]
		rwAns := rule.DNSRewrite.NewCNAME

		s.logger.DebugContext(ctx, "cname found", "host", host, "cname", rwAns)

		if dReq.Hostname == rwAns {
			// A request for the hostname itself is an exception rule.
			// TODO(d.kolyshev): Check rewrite of a pattern onto itself.

			return nil, nil
		}

		if isSelfMatchingWildcard(host, rwAns, rule.Text()) {
			return nil, rule.DNSRewrite
		}

		if cnames.Has(rwAns) {
			s.logger.WarnContext(ctx, "rewrite cname loop", "host", dReq.Hostname, "rewrite", rwAns)

			return nil, nil
		}

		cnames.Add(rwAns)

		rewriteRulesForReq := s.rewriteRulesForReq(&urlfilter.DNSRequest{
			Hostname: rwAns,
			DNSType:  dReq.DNSType,
		})
		if rewriteRulesForReq != nil {
			rewriteRules = rewriteRulesForReq
		}

		host = rwAns
	}

	return rewriteRules, nil
}

// isSelfMatchingWildcard returns true when a wildcard rewrite matches its own
// result.
//
// For example, an "*.example.com → sub.example.com" rewrite matching in a loop.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/4016.
func isSelfMatchingWildcard(host, rwAns, ruleText string) (ok bool) {
	return host == rwAns && isWildcard(ruleText)
}

// collectDNSRewrites filters DNSRewrite by question type.
func (s *DefaultStorage) collectDNSRewrites(
	rewrites []*rules.NetworkRule,
	qtyp uint16,
) (rws []*rules.DNSRewrite) {
	for _, rewrite := range rewrites {
		dnsRewrite := rewrite.DNSRewrite
		if matchesQType(dnsRewrite, qtyp) {
			rws = append(rws, dnsRewrite)
		}
	}

	return rws
}

// rewriteRulesForReq returns matching dnsrewrite rules.
func (s *DefaultStorage) rewriteRulesForReq(dReq *urlfilter.DNSRequest) (rules []*rules.NetworkRule) {
	res, _ := s.engine.MatchRequest(dReq)

	return res.DNSRewrites()
}

// Add implements the [Storage] interface for *DefaultStorage.
func (s *DefaultStorage) Add(item *Item) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO(d.kolyshev): Handle duplicate items.
	s.rewrites = append(s.rewrites, item)

	return s.resetRules()
}

// Remove implements the [Storage] interface for *DefaultStorage.
func (s *DefaultStorage) Remove(item *Item) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.TODO()

	arr := []*Item{}

	// TODO(d.kolyshev): Use slices.IndexFunc + slices.Delete?
	for _, ent := range s.rewrites {
		if ent.equal(item) {
			s.logger.DebugContext(ctx, "removed element", "domain", ent.Domain, "ans", ent.Answer)

			continue
		}

		arr = append(arr, ent)
	}
	s.rewrites = arr

	return s.resetRules()
}

// List implements the [Storage] interface for *DefaultStorage.
func (s *DefaultStorage) List() (items []*Item) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.rewrites)
}

// resetRules resets the filtering rules.
func (s *DefaultStorage) resetRules() (err error) {
	// TODO(a.garipov): Use strings.Builder.
	var rulesText []string
	for _, rewrite := range s.rewrites {
		rulesText = append(rulesText, rewrite.toRule())
	}

	strList := filterlist.NewString(&filterlist.StringConfig{
		ID:             s.urlFilterID,
		RulesText:      strings.Join(rulesText, "\n"),
		IgnoreCosmetic: true,
	})

	rs, err := filterlist.NewRuleStorage([]filterlist.Interface{strList})
	if err != nil {
		return fmt.Errorf("creating list storage: %w", err)
	}

	s.ruleList = strList
	s.engine = urlfilter.NewDNSEngine(rs)

	s.logger.InfoContext(
		context.TODO(),
		"reset rules",
		"filter", s.urlFilterID,
		"count", s.engine.RulesCount,
	)

	return nil
}

// matchesQType returns true if dnsrewrite matches the question type qt.
func matchesQType(dnsrr *rules.DNSRewrite, qt uint16) (ok bool) {
	// Add CNAMEs, since they match for all types requests.
	if dnsrr.RRType == dns.TypeCNAME {
		return true
	}

	// Reject types other than A and AAAA.
	if qt != dns.TypeA && qt != dns.TypeAAAA {
		return false
	}

	return dnsrr.RRType == qt
}

// isWildcard returns true if pat is a wildcard domain pattern.
func isWildcard(pat string) (res bool) {
	return strings.HasPrefix(pat, "|*.")
}
