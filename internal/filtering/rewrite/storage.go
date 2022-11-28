// Package rewrite implements DNS Rewrites storage and request matching.
package rewrite

import (
	"fmt"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"golang.org/x/exp/slices"
)

// Storage is a storage for rewrite rules.
type Storage interface {
	// MatchRequest finds a matching rule for the specified request.
	MatchRequest(dReq *urlfilter.DNSRequest) (res *urlfilter.DNSResult, matched bool)

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

	// urlFilterID is the synthetic integer identifier for the urlfilter engine.
	//
	// TODO(a.garipov): Change the type to a string in module urlfilter and
	// remove this crutch.
	urlFilterID int

	// rewrites stores the rewrite entries from configuration.
	rewrites []*Item
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
func (s *DefaultStorage) MatchRequest(dReq *urlfilter.DNSRequest) (res *urlfilter.DNSResult, matched bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.engine.MatchRequest(dReq)
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

	log.Info("filter %d: reset %d rules", s.urlFilterID, s.engine.RulesCount)

	return nil
}
