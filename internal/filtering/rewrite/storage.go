// Package rewrite implements DNS Rewrites storage and request matching.
package rewrite

import (
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
)

// DefaultStorage is the default storage for rewrite rules.
type DefaultStorage struct {
	// mu protects items.
	mu *sync.RWMutex

	// engine is the DNS filtering engine.
	engine *urlfilter.DNSEngine

	// ruleList is the filtering rule ruleList used by the engine.
	ruleList filterlist.RuleList

	// rewrites stores the rewrite entries from configuration.
	rewrites []*filtering.RewriteItem

	// urlFilterID is the synthetic integer identifier for the urlfilter engine.
	//
	// TODO(a.garipov): Change the type to a string in module urlfilter and
	// remove this crutch.
	urlFilterID int
}

// NewDefaultStorage returns new rewrites storage.  listID is used as an
// identifier of the underlying rules list.  rewrites must not be nil.
func NewDefaultStorage(rewrites []*filtering.RewriteItem) (s *DefaultStorage, err error) {
	s = &DefaultStorage{
		mu:          &sync.RWMutex{},
		urlFilterID: filtering.RewritesListID,
		rewrites:    rewrites,
	}

	err = s.resetRules()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// type check
var _ filtering.RewriteStorage = (*DefaultStorage)(nil)

// MatchRequest implements the [RewriteStorage] interface for *DefaultStorage.
func (s *DefaultStorage) MatchRequest(dReq *urlfilter.DNSRequest) (rws []*rules.DNSRewrite) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rrules := s.rewriteRulesForReq(dReq)
	if len(rrules) == 0 {
		return nil
	}

	// TODO(a.garipov): Check cnames for cycles on initialization.
	cnames := stringutil.NewSet()
	host := dReq.Hostname
	var lastCNAMERule *rules.NetworkRule
	for len(rrules) > 0 && rrules[0].DNSRewrite != nil && rrules[0].DNSRewrite.NewCNAME != "" {
		lastCNAMERule = rrules[0]
		lastDNSRewrite := lastCNAMERule.DNSRewrite
		rwAns := lastDNSRewrite.NewCNAME

		log.Debug("rewrite: cname for %s is %s", host, rwAns)

		if dReq.Hostname == rwAns {
			// A request for the hostname itself.
			// TODO(d.kolyshev): Check rewrite of a pattern onto itself.
			log.Debug("rewrite: request for hostname itself for %q", dReq.Hostname)

			return nil
		}

		if host == rwAns && isWildcard(lastCNAMERule.RuleText) {
			// An "*.example.com → sub.example.com" rewrite matching in a loop.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/4016.
			log.Debug("rewrite: cname wildcard loop for %q on %q", dReq.Hostname, rwAns)

			return []*rules.DNSRewrite{lastDNSRewrite}
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

		if drules == nil {
			break
		}

		rrules = drules
		host = rwAns
	}

	return s.collectDNSRewrites(rrules, lastCNAMERule, dReq.DNSType)
}

// collectDNSRewrites filters DNSRewrite by question type.
func (s *DefaultStorage) collectDNSRewrites(
	rewrites []*rules.NetworkRule,
	cnameRule *rules.NetworkRule,
	qtyp uint16,
) (rws []*rules.DNSRewrite) {
	if cnameRule != nil {
		rewrites = append([]*rules.NetworkRule{cnameRule}, rewrites...)
	}

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

// Add implements the [RewriteStorage] interface for *DefaultStorage.
func (s *DefaultStorage) Add(item *filtering.RewriteItem) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO(d.kolyshev): Handle duplicate items.
	s.rewrites = append(s.rewrites, item)

	return s.resetRules()
}

// Remove implements the [RewriteStorage] interface for *DefaultStorage.
func (s *DefaultStorage) Remove(item *filtering.RewriteItem) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	arr := []*filtering.RewriteItem{}

	// TODO(d.kolyshev): Use slices.IndexFunc + slices.Delete?
	for _, ent := range s.rewrites {
		if ent.Equal(item) {
			log.Debug("rewrite: removed element: %s -> %s", ent.Domain, ent.Answer)

			continue
		}

		arr = append(arr, ent)
	}
	s.rewrites = arr

	return s.resetRules()
}

// List implements the [RewriteStorage] interface for *DefaultStorage.
func (s *DefaultStorage) List() (items []*filtering.RewriteItem) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.rewrites)
}

// resetRules resets the filtering rules.
func (s *DefaultStorage) resetRules() (err error) {
	// TODO(a.garipov): Use strings.Builder.
	var rulesText []string
	for _, rewrite := range s.rewrites {
		rulesText = append(rulesText, toRule(rewrite))
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
	switch qt {
	case dns.TypeA:
		return dnsrr.RRType != dns.TypeAAAA
	case dns.TypeAAAA:
		return dnsrr.RRType != dns.TypeA
	default:
		return true
	}
}

// isWildcard returns true if pat is a wildcard domain pattern.
func isWildcard(pat string) (res bool) {
	return strings.HasPrefix(pat, "|*.")
}

// toRule converts rw to a filter rule.
func toRule(rw *filtering.RewriteItem) (res string) {
	if rw == nil {
		return ""
	}

	domain := strings.ToLower(rw.Domain)

	dType, exception := rewriteParams(rw)
	dTypeKey := dns.TypeToString[dType]
	if exception {
		return fmt.Sprintf("@@||%s^$dnstype=%s,dnsrewrite", domain, dTypeKey)
	}

	return fmt.Sprintf("|%s^$dnsrewrite=NOERROR;%s;%s", domain, dTypeKey, rw.Answer)
}

// RewriteParams returns dns request type and exception flag for rw.
func rewriteParams(rw *filtering.RewriteItem) (dType uint16, exception bool) {
	switch rw.Answer {
	case "AAAA":
		return dns.TypeAAAA, true
	case "A":
		return dns.TypeA, true
	default:
		// Go on.
	}

	addr, err := netip.ParseAddr(rw.Answer)
	if err != nil {
		// TODO(d.kolyshev): Validate rw.Answer as a domain name.
		return dns.TypeCNAME, false
	}

	if addr.Is4() {
		dType = dns.TypeA
	} else {
		dType = dns.TypeAAAA
	}

	return dType, false
}
