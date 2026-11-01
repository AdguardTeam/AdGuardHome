package domainhash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachSuffix(t *testing.T) {
	testCases := []struct {
		name   string
		domain string
		want   []string
	}{{
		name:   "full",
		domain: "ads.tracker.example.com",
		want:   []string{"ads.tracker.example.com", "tracker.example.com", "example.com"},
	}, {
		name:   "www_stripped_by_normalization",
		domain: "example.com",
		want:   []string{"example.com"},
	}, {
		name:   "single_label",
		domain: "com",
		want:   []string{"com"},
	}, {
		name:   "empty",
		domain: "",
		want:   nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			forEachSuffix([]byte(tc.domain), func(s []byte) (cont bool) {
				got = append(got, string(s))

				return true
			})

			assert.Equal(t, tc.want, got, "suffixes")
		})
	}
}

func TestForEachSuffixStop(t *testing.T) {
	var got []string
	forEachSuffix([]byte("ads.tracker.example.com"), func(s []byte) (cont bool) {
		got = append(got, string(s))

		return len(got) < 2
	})

	require.Equal(t, []string{"ads.tracker.example.com", "tracker.example.com"}, got, "suffixes")
}
