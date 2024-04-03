// Package rewrite implements DNS Rewrites storage and request matching.
package rewrite

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/log"
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

// DefaultStorage is the default storage for rewrite rules.
type DefaultStorage struct {
	// mu protects items.
	mu *sync.RWMutex

	// engine is the DNS filtering engine.
	engine *urlfilter.DNSEngine

	// ruleList is the filtering rule ruleList used by the engine.
	ruleList filterlist.RuleList

	// rewrites stores the rewrite entries from configuration.
	rewrites []*Item

	// urlFilterID is the synthetic integer identifier for the urlfilter engine.
	//
	// TODO(a.garipov): Change the type to a string in module urlfilter and
	// remove this crutch.
	urlFilterID int
}

// NewDefaultStorage returns new rewrites storage.  listID is used as an
// identifier of the underlying rules list.  rewrites must not be nil.
func NewDefaultStorage(listID int, rewrites []*Item) (s *DefaultStorage, err error) {
	s = &DefaultStorage{
		mu:          &sync.RWMutex{},
		urlFilterID: listID,
		rewrites:    rewrites,
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

	rrules := s.rewriteRulesForReq(dReq)
	if len(rrules) == 0 {
		return nil
	}

	// TODO(a.garipov): Check cnames for cycles on initialization.
	cnames := container.NewMapSet[string]()
	host := dReq.Hostname
	for len(rrules) > 0 && rrules[0].DNSRewrite != nil && rrules[0].DNSRewrite.NewCNAME != "" {
		rule := rrules[0]
		rwAns := rule.DNSRewrite.NewCNAME

		log.Debug("rewrite: cname for %s is %s", host, rwAns)

		if dReq.Hostname == rwAns {
			// A request for the hostname itself is an exception rule.
			// TODO(d.kolyshev): Check rewrite of a pattern onto itself.

			return nil
		}

		if host == rwAns && isWildcard(rule.RuleText) {
			// An "*.example.com → sub.example.com" rewrite matching in a loop.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/4016.

			return []*rules.DNSRewrite{rule.DNSRewrite}
		}

		if cnames.Has(rwAns) {
			log.Info("rewrite: cname loop for %q on %q", dReq.Hostname, rwAns)

			return nil
		}

		cnames.Add(rwAns)

		drules := s.rewriteRulesForReq(&urlfilter.DNSRequest{
			Hostname: rwAns,
			DNSType:  dReq.DNSType,
		})
		if drules != nil {
			rrules = drules
		}

		host = rwAns
	}

	return s.collectDNSRewrites(rrules, dReq.DNSType)
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

	arr := []*Item{}

	// TODO(d.kolyshev): Use slices.IndexFunc + slices.Delete?
	for _, ent := range s.rewrites {
		if ent.equal(item) {
			log.Debug("rewrite: removed element: %s -> %s", ent.Domain, ent.Answer)

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

	strList := &filterlist.StringRuleList{
		ID:             s.urlFilterID,
		RulesText:      strings.Join(rulesText, "\n"),
		IgnoreCosmetic: true,
	}

	rs, err := filterlist.NewRuleStorage([]filterlist.RuleList{strList})
	if err != nil {
		return fmt.Errorf("creating list storage: %w", err)
	}

	s.ruleList = strList
	s.engine = urlfilter.NewDNSEngine(rs)

	log.Info("rewrite: filter %d: reset %d rules", s.urlFilterID, s.engine.RulesCount)

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
