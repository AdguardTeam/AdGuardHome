package filtering

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/urlfilter/rules"
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
			ID: 1,
		}, {
			ID: 2,
		}, {
			ID: 3,
		}},
	}, {
		name: "two_dups",
		in: []FilterYAML{{
			ID: 1,
		}, {
			ID: 3,
		}, {
			ID: 1,
		}, {
			ID: 2,
		}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := newIDGenerator(1, testLogger)
			g.fix(tc.in)

			assertUniqueIDs(t, tc.in)
		})
	}
}

// assertUniqueIDs is a test helper that asserts that the IDs of filters are
// unique.
func assertUniqueIDs(tb testing.TB, flts []FilterYAML) {
	tb.Helper()

	uc := aghalg.UniqChecker[rules.ListID]{}
	for _, f := range flts {
		uc.Add(f.ID)
	}

	assert.NoError(tb, uc.Validate())
}
