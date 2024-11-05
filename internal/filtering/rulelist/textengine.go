package rulelist

import (
	"fmt"
	"strings"
	"sync"

	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
)

// TextEngine is a single DNS filter based on a list of rules in text form.
type TextEngine struct {
	// mu protects engine and storage.
	mu *sync.RWMutex

	// engine is the filtering engine.
	engine *urlfilter.DNSEngine

	// storage is the filtering-rule storage.  It is saved here to close it.
	storage *filterlist.RuleStorage

	// name is the human-readable name of the engine.
	name string
}

// TextEngineConfig is the configuration for a rule-list filtering engine
// created from a filtering rule text.
type TextEngineConfig struct {
	// name is the human-readable name of the engine; see [EngineNameAllow] and
	// similar constants.
	Name string

	// Rules is the text of the filtering rules for this engine.
	Rules []string

	// ID is the ID to use inside a URL-filter engine.
	ID URLFilterID
}

// NewTextEngine returns a new rule-list filtering engine that uses rules
// directly.  The engine is ready to use and should not be refreshed.
func NewTextEngine(c *TextEngineConfig) (e *TextEngine, err error) {
	text := strings.Join(c.Rules, "\n")
	storage, err := filterlist.NewRuleStorage([]filterlist.RuleList{
		&filterlist.StringRuleList{
			RulesText:      text,
			ID:             c.ID,
			IgnoreCosmetic: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating rule storage: %w", err)
	}

	engine := urlfilter.NewDNSEngine(storage)

	return &TextEngine{
		mu:      &sync.RWMutex{},
		engine:  engine,
		storage: storage,
		name:    c.Name,
	}, nil
}

// FilterRequest returns the result of filtering req using the DNS filtering
// engine.
func (e *TextEngine) FilterRequest(
	req *urlfilter.DNSRequest,
) (res *urlfilter.DNSResult, hasMatched bool) {
	var engine *urlfilter.DNSEngine

	func() {
		e.mu.RLock()
		defer e.mu.RUnlock()

		engine = e.engine
	}()

	return engine.MatchRequest(req)
}

// Close closes the underlying rule list engine as well as the rule lists.
func (e *TextEngine) Close() (err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.storage == nil {
		return nil
	}

	err = e.storage.Close()
	if err != nil {
		return fmt.Errorf("closing text engine %q: %w", e.name, err)
	}

	return nil
}
