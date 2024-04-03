package filtering

import (
	"fmt"
	"sync/atomic"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/log"
)

// idGenerator generates filtering-list IDs in a way broadly compatible with the
// legacy approach of AdGuard Home.
//
// TODO(a.garipov): Get rid of this once we switch completely to the new
// rule-list architecture.
type idGenerator struct {
	current *atomic.Int32
}

// newIDGenerator returns a new ID generator initialized with the given seed
// value.
func newIDGenerator(seed int32) (g *idGenerator) {
	g = &idGenerator{
		current: &atomic.Int32{},
	}

	g.current.Store(seed)

	return g
}

// next returns the next ID from the generator.  It is safe for concurrent use.
func (g *idGenerator) next() (id rulelist.URLFilterID) {
	id32 := g.current.Add(1)
	if id32 < 0 {
		panic(fmt.Errorf("invalid current id value %d", id32))
	}

	return rulelist.URLFilterID(id32)
}

// fix ensures that flts all have unique IDs.
func (g *idGenerator) fix(flts []FilterYAML) {
	set := container.NewMapSet[rulelist.URLFilterID]()
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

		log.Info(
			"filtering: warning: filter at index %d has duplicate id %d; reassigning to %d",
			i,
			id,
			newID,
		)

		flts[i].ID = newID
		set.Add(newID)
	}
}
