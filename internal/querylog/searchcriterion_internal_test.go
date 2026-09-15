package querylog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchCriterion_Match_Negative(t *testing.T) {
	entry := &logEntry{
		QHost: "example.com",
		IP:    testClientIPv4,
	}

	testCases := []struct {
		name       string
		value      string
		negative   bool
		wantMatch  bool
	}{{
		name:      "positive_match",
		value:     "example.com",
		negative:  false,
		wantMatch: true,
	}, {
		name:      "positive_no_match",
		value:     "other.com",
		negative:  false,
		wantMatch: false,
	}, {
		name:      "negative_excludes_matching",
		value:     "example.com",
		negative:  true,
		wantMatch: false,
	}, {
		name:      "negative_includes_non_matching",
		value:     "other.com",
		negative:  true,
		wantMatch: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &searchCriterion{
				value:         tc.value,
				criterionType: ctTerm,
				negative:      tc.negative,
			}

			got := c.match(entry)
			assert.Equal(t, tc.wantMatch, got)

			// Verify that negative inverts the result.
			cWithoutNeg := &searchCriterion{
				value:         tc.value,
				criterionType: ctTerm,
			}
			withoutNegResult := cWithoutNeg.match(entry)

			if tc.negative {
				assert.Equal(t, !withoutNegResult, got,
					"negative should invert the match result",
				)
			} else {
				assert.Equal(t, withoutNegResult, got,
					"non-negative should match as-is",
				)
			}
		})
	}
}
