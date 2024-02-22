package aghnet

import (
	"slices"
	"strings"

	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
)

// IgnoreEngine contains the list of rules for ignoring hostnames and matches
// them.
//
// TODO(s.chzhen):  Move all urlfilter stuff to aghfilter.
type IgnoreEngine struct {
	// engine is the filtering engine that can match rules for ignoring
	// hostnames.
	engine *urlfilter.DNSEngine

	// ignored is the list of rules for ignoring hostnames.
	ignored []string
}

// NewIgnoreEngine creates a new instance of the IgnoreEngine and stores the
// list of rules for ignoring hostnames.
func NewIgnoreEngine(ignored []string) (e *IgnoreEngine, err error) {
	ruleList := &filterlist.StringRuleList{
		RulesText:      strings.ToLower(strings.Join(ignored, "\n")),
		IgnoreCosmetic: true,
	}
	ruleStorage, err := filterlist.NewRuleStorage([]filterlist.RuleList{ruleList})
	if err != nil {
		return nil, err
	}

	return &IgnoreEngine{
		engine:  urlfilter.NewDNSEngine(ruleStorage),
		ignored: ignored,
	}, nil
}

// Has returns true if IgnoreEngine matches the host.
func (e *IgnoreEngine) Has(host string) (ignore bool) {
	if e == nil {
		return false
	}

	_, ignore = e.engine.Match(host)

	return ignore
}

// Values returns a copy of list of rules for ignoring hostnames.
func (e *IgnoreEngine) Values() (ignored []string) {
	return slices.Clone(e.ignored)
}
