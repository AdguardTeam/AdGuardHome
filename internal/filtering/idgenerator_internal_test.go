package filtering

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/stretchr/testify/assert"
)

func TestIDGenerator_Fix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   []FilterYAML
	}{{
		name: "nil",
		in:   nil,
	}, {
		name: "empty",
		in:   []FilterYAML{},
	}, {
		name: "one_zero",
		in:   []FilterYAML{{}},
	}, {
		name: "two_zeros",
		in:   []FilterYAML{{}, {}},
	}, {
		name: "many_good",
		in: []FilterYAML{{
			Filter: Filter{
				ID: 1,
			},
		}, {
			Filter: Filter{
				ID: 2,
			},
		}, {
			Filter: Filter{
				ID: 3,
			},
		}},
	}, {
		name: "two_dups",
		in: []FilterYAML{{
			Filter: Filter{
				ID: 1,
			},
		}, {
			Filter: Filter{
				ID: 3,
			},
		}, {
			Filter: Filter{
				ID: 1,
			},
		}, {
			Filter: Filter{
				ID: 2,
			},
		}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := newIDGenerator(1)
			g.fix(tc.in)

			assertUniqueIDs(t, tc.in)
		})
	}
}

// assertUniqueIDs is a test helper that asserts that the IDs of filters are
// unique.
func assertUniqueIDs(t testing.TB, flts []FilterYAML) {
	t.Helper()

	uc := aghalg.UniqChecker[rulelist.URLFilterID]{}
	for _, f := range flts {
		uc.Add(f.ID)
	}

	assert.NoError(t, uc.Validate())
}
