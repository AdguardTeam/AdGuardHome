package querylog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsFold(t *testing.T) {
	testCases := []struct {
		name     string
		inS      string
		inSubstr string
		want     bool
	}{{
		name:     "empty",
		inS:      "",
		inSubstr: "",
		want:     true,
	}, {
		name:     "shorter",
		inS:      "a",
		inSubstr: "abc",
		want:     false,
	}, {
		name:     "same_len_true",
		inS:      "abc",
		inSubstr: "abc",
		want:     true,
	}, {
		name:     "same_len_true_fold",
		inS:      "abc",
		inSubstr: "aBc",
		want:     true,
	}, {
		name:     "same_len_false",
		inS:      "abc",
		inSubstr: "def",
		want:     false,
	}, {
		name:     "longer_true",
		inS:      "abcdedef",
		inSubstr: "def",
		want:     true,
	}, {
		name:     "longer_false",
		inS:      "abcded",
		inSubstr: "ghi",
		want:     false,
	}, {
		name:     "longer_true_fold",
		inS:      "abcdedef",
		inSubstr: "dEf",
		want:     true,
	}, {
		name:     "longer_false_fold",
		inS:      "abcded",
		inSubstr: "gHi",
		want:     false,
	}, {
		name:     "longer_true_cyr_fold",
		inS:      "абвгдедеё",
		inSubstr: "дЕЁ",
		want:     true,
	}, {
		name:     "longer_false_cyr_fold",
		inS:      "абвгдедеё",
		inSubstr: "жЗИ",
		want:     false,
	}, {
		name:     "no_letters_true",
		inS:      "1.2.3.4",
		inSubstr: "2.3.4",
		want:     true,
	}, {
		name:     "no_letters_false",
		inS:      "1.2.3.4",
		inSubstr: "2.3.5",
		want:     false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.want {
				assert.True(t, containsFold(tc.inS, tc.inSubstr))
			} else {
				assert.False(t, containsFold(tc.inS, tc.inSubstr))
			}
		})
	}
}

var sink bool

func BenchmarkContainsFold(b *testing.B) {
	const s = "aaahBbBhccchDDDeEehFfFhGGGhHhh"
	const substr = "HHH"

	// Compare our implementation of containsFold against a stupid solution
	// of calling strings.ToLower and strings.Contains.
	b.Run("containsfold", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			sink = containsFold(s, substr)
		}

		assert.True(b, sink)
	})

	b.Run("tolower_contains", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			sink = strings.Contains(strings.ToLower(s), strings.ToLower(substr))
		}

		assert.True(b, sink)
	})
}
