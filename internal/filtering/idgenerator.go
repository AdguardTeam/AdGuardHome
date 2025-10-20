package filtering

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/urlfilter/rules"
)

// idGenerator generates filtering-list IDs in a way broadly compatible with the
// legacy approach of AdGuard Home.
//
// TODO(a.garipov): Get rid of this once we switch completely to the new
// rule-list architecture.
type idGenerator struct {
	current *atomic.Uint64
	logger  *slog.Logger
}

// newIDGenerator returns a new ID generator initialized with the given seed
// value.  l must not be nil.
func newIDGenerator(seed uint64, l *slog.Logger) (g *idGenerator) {
	g = &idGenerator{
		current: &atomic.Uint64{},
		logger:  l,
	}

	g.current.Store(seed)

	return g
}

// next returns the next ID from the generator.  It is safe for concurrent use.
func (g *idGenerator) next() (id rules.ListID) {
	id64 := g.current.Add(1)
	if id64 == 0 {
		panic(fmt.Errorf("invalid current id value %d", id64))
	}

	return rules.ListID(id64)
}

// fix ensures that flts all have unique IDs.
func (g *idGenerator) fix(flts []FilterYAML) {
	set := container.NewMapSet[rules.ListID]()
	for i, f := range flts {
		id := f.ID
		if id == 0 {
			id = g.next()
			flts[i].ID = id
		}

		if !set.Has(id) {
			set.Add(id)

			continue
		}

		newID := g.next()
		for set.Has(newID) {
			newID = g.next()
		}

		g.logger.WarnContext(
			context.TODO(),
			"filter has duplicate id; reassigning",
			"idx", i,
			"id", id,
			"new_id", newID,
		)

		flts[i].ID = newID
		set.Add(newID)
	}
}
