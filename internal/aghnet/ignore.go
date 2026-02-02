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

	// enabled determines whether ignoring is enabled.
	enabled bool
}

// NewIgnoreEngine creates a new instance of the IgnoreEngine and stores the
// list of rules for ignoring hostnames.  If enabled is set to false, hostnames
// will never be ignored.
func NewIgnoreEngine(ignored []string, enabled bool) (e *IgnoreEngine, err error) {
	ruleLists := []filterlist.Interface{
		filterlist.NewString(&filterlist.StringConfig{
			RulesText:      strings.ToLower(strings.Join(ignored, "\n")),
			IgnoreCosmetic: true,
		}),
	}
	ruleStorage, err := filterlist.NewRuleStorage(ruleLists)
	if err != nil {
		return nil, err
	}

	return &IgnoreEngine{
		engine:  urlfilter.NewDNSEngine(ruleStorage),
		ignored: ignored,
		enabled: enabled,
	}, nil
}

// Has returns true if IgnoreEngine matches the host.
func (e *IgnoreEngine) Has(host string) (ignore bool) {
	if e == nil || !e.enabled {
		return false
	}

	_, ignore = e.engine.Match(host)

	return ignore
}

// Values returns a copy of list of rules for ignoring hostnames.
func (e *IgnoreEngine) Values() (ignored []string) {
	return slices.Clone(e.ignored)
}

// IsEnabled returns true if hostnames ignoring is enabled.
func (e *IgnoreEngine) IsEnabled() (enabled bool) {
	return e.enabled
}
