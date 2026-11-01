package domainhash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMemAvailable(t *testing.T) {
	regular := []byte(
		"MemTotal:       32768000 kB\n" +
			"MemFree:         1048576 kB\n" +
			"MemAvailable:     8388608 kB\n" +
			"Buffers:          1024 kB\n",
	)

	testCases := []struct {
		name string
		data string
		want int64
		ok   bool
	}{{
		name: "regular",
		data: string(regular),
		want: 8388608 * 1024,
		ok:   true,
	}, {
		name: "no_field",
		data: "MemTotal:       32768000 kB\nMemFree:         1048576 kB\n",
		ok:   false,
	}, {
		name: "empty_value",
		data: "MemAvailable:\n",
		ok:   false,
	}, {
		name: "not_a_number",
		data: "MemAvailable:  unknown kB\n",
		ok:   false,
	}, {
		name: "zero",
		data: "MemAvailable: 0 kB\n",
		ok:   false,
	}, {
		name: "empty",
		data: "",
		ok:   false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			n, ok := parseMemAvailable([]byte(tc.data))
			assert.Equal(t, tc.ok, ok, "ok")
			if tc.ok {
				assert.Equal(t, tc.want, n, "n")
			}
		})
	}
}

func TestDetectMemBudget(t *testing.T) {
	// Smoke: a budget is always detected and never below the floor.
	// Do not compare two live DetectMemBudget calls for equality — each reads
	// /proc/meminfo MemAvailable, which can move between calls under load.
	budget := DetectMemBudget(0)
	require.GreaterOrEqual(t, budget, MinMemBudgetBytes, "budget")

	assert.Equal(t, MinMemBudgetBytes, DetectMemBudget(int64(1)<<62), "huge margin")
	require.GreaterOrEqual(t, DetectMemBudget(-1), MinMemBudgetBytes, "negative margin")
}
