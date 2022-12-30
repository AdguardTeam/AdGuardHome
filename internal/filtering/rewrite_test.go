package filtering

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItem_equal(t *testing.T) {
	const (
		testDomain = "example.org"
		testAnswer = "1.1.1.1"
	)

	testItem := &RewriteItem{
		Domain: testDomain,
		Answer: testAnswer,
	}

	testCases := []struct {
		name  string
		left  *RewriteItem
		right *RewriteItem
		want  bool
	}{{
		name:  "nil_left",
		left:  nil,
		right: testItem,
		want:  false,
	}, {
		name:  "nil_right",
		left:  testItem,
		right: nil,
		want:  false,
	}, {
		name:  "nils",
		left:  nil,
		right: nil,
		want:  true,
	}, {
		name:  "equal",
		left:  testItem,
		right: testItem,
		want:  true,
	}, {
		name: "distinct",
		left: testItem,
		right: &RewriteItem{
			Domain: "other",
			Answer: "other",
		},
		want: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.left.Equal(tc.right)
			assert.Equal(t, tc.want, res)
		})
	}
}
