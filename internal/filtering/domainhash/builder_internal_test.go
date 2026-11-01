package domainhash

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainHashConfigBatchBytes(t *testing.T) {
	testCases := []struct {
		name   string
		budget int64
		want   int64
	}{{
		name:   "default",
		budget: 0,
		want:   defaultBatchBytes,
	}, {
		name:   "negative",
		budget: -1,
		want:   defaultBatchBytes,
	}, {
		name:   "small",
		budget: 1 << 10,
		want:   1 << 10,
	}, {
		name:   "large_is_capped",
		budget: 64 << 20,
		want:   defaultBatchBytes,
	}, {
		name:   "equal_to_default",
		budget: defaultBatchBytes,
		want:   defaultBatchBytes,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DomainHashConfig{MemBudgetBytes: tc.budget}
			assert.Equal(t, tc.want, cfg.batchBytes(), "batch bytes")
		})
	}
}

func TestBuilderDefaultConfig(t *testing.T) {
	b, err := NewBuilder(DomainHashConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		assert.NoError(t, b.Close(), "close")
	}()

	assert.Equal(t, DefaultHashBytes, b.cfg.HashBytes, "hash bytes")
	assert.Equal(t, DefaultMemBudgetBytes, b.cfg.MemBudgetBytes, "memory budget")

	requireNoError(t, b.AddDomain("example.com"))
	assert.Equal(t, int64(len("example.com")+recordOverhead), b.batchBytes, "batch bytes")
	assert.Equal(t, b.batchBytes, b.stats.PeakBatchBytes, "peak batch bytes")
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(fmt.Errorf("unexpected error: %w", err))
	}
}
