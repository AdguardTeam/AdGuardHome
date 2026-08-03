package filtering

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeRuleList writes data into a new file within a temporary directory and
// returns its path.
func writeRuleList(tb testing.TB, data string) (path string) {
	tb.Helper()

	path = filepath.Join(tb.TempDir(), "rules.txt")
	require.NoError(tb, os.WriteFile(path, []byte(data), 0o644))

	return path
}

func TestFiltersFingerprint(t *testing.T) {
	t.Parallel()

	const (
		blockID rules.ListID = 1
		allowID rules.ListID = 2
	)

	block := []Filter{{ID: blockID, Data: []byte("||example.com^\n")}}
	allow := []Filter{{ID: allowID, Data: []byte("@@||example.org^\n")}}

	base := filtersFingerprint(allow, block)

	t.Run("stable", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, base, filtersFingerprint(allow, block))
	})

	t.Run("data_changed", func(t *testing.T) {
		t.Parallel()

		other := []Filter{{ID: blockID, Data: []byte("||example.net^\n")}}
		assert.NotEqual(t, base, filtersFingerprint(allow, other))
	})

	t.Run("id_changed", func(t *testing.T) {
		t.Parallel()

		other := []Filter{{ID: blockID + 100, Data: []byte("||example.com^\n")}}
		assert.NotEqual(t, base, filtersFingerprint(allow, other))
	})

	t.Run("groups_swapped", func(t *testing.T) {
		t.Parallel()

		assert.NotEqual(t, base, filtersFingerprint(block, allow))
	})

	t.Run("list_added", func(t *testing.T) {
		t.Parallel()

		other := append(append([]Filter{}, block...), Filter{
			ID:   blockID + 1,
			Data: []byte("||example.net^\n"),
		})
		assert.NotEqual(t, base, filtersFingerprint(allow, other))
	})
}

// TestFiltersFingerprint_file makes sure that the fingerprint follows the
// contents of the files that the rule lists are read from, since rule-list
// updates only rewrite those files, leaving the [Filter] values intact.
func TestFiltersFingerprint_file(t *testing.T) {
	t.Parallel()

	path := writeRuleList(t, "||example.com^\n")
	flts := []Filter{{ID: 1, FilePath: path}}

	base := filtersFingerprint(nil, flts)
	assert.Equal(t, base, filtersFingerprint(nil, flts))

	// Set an explicit modification time, since some filesystems have a coarse
	// modification-time resolution.
	mtime := time.Now().Add(-1 * time.Hour)

	t.Run("contents_changed", func(t *testing.T) {
		require.NoError(t, os.WriteFile(path, []byte("||example.net^\n"), 0o644))
		require.NoError(t, os.Chtimes(path, mtime, mtime))

		assert.NotEqual(t, base, filtersFingerprint(nil, flts))
	})

	// The fingerprint follows the contents of the file and nothing else, so
	// metadata that a rewrite happens to preserve must not hide a change of the
	// rules, and metadata that changes on its own must not fake one.

	t.Run("same_metadata_different_bytes", func(t *testing.T) {
		prev := filtersFingerprint(nil, flts)

		// Replace the contents with different rules of exactly the same length
		// and restore the modification time, the way a restored backup would.
		const other = "||example.org^\n"
		require.Len(t, other, len("||example.net^\n"))

		require.NoError(t, os.WriteFile(path, []byte(other), 0o644))
		require.NoError(t, os.Chtimes(path, mtime, mtime))

		fi, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, int64(len(other)), fi.Size())
		require.True(t, fi.ModTime().Equal(mtime))

		assert.NotEqual(t, prev, filtersFingerprint(nil, flts))
	})

	t.Run("same_bytes_new_mtime", func(t *testing.T) {
		prev := filtersFingerprint(nil, flts)

		newTime := mtime.Add(1 * time.Minute)
		require.NoError(t, os.Chtimes(path, newTime, newTime))

		assert.Equal(t, prev, filtersFingerprint(nil, flts))
	})

	t.Run("removed", func(t *testing.T) {
		prev := filtersFingerprint(nil, flts)
		require.NoError(t, os.Remove(path))

		assert.NotEqual(t, prev, filtersFingerprint(nil, flts))
	})
}

// TestGCTarget_overlap makes sure that two rebuilds running at the same time
// don't restore each other's saved garbage-collection target.  The target
// belongs to the whole process, while each [DNSFilter] only tightens it for the
// duration of its own rebuild.
func TestGCTarget_overlap(t *testing.T) {
	t.Parallel()

	const (
		base    = 100
		percent = 20
	)

	// current is the target that a real [debug.SetGCPercent] would have left the
	// process at.
	current := base
	tgt := &gcTarget{
		set: func(p int) (prev int) {
			prev, current = current, p

			return prev
		},
	}

	// Two instances start rebuilding, the second while the first is running.
	restoreFirst := tgt.tighten(percent)
	assert.Equal(t, percent, current)

	restoreSecond := tgt.tighten(percent)
	assert.Equal(t, percent, current)

	// The first one finishes while the second is still running, so the target
	// must stay tightened.
	restoreFirst()
	assert.Equal(t, percent, current)

	// Only once the last one finishes is the original target restored.
	restoreSecond()
	assert.Equal(t, base, current)
}

// TestGCTarget_stricter makes sure that a target stricter than the one used for
// rebuilds is left alone, including a disabled garbage collector.
func TestGCTarget_stricter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		base int
	}{{
		name: "stricter",
		base: 10,
	}, {
		name: "disabled",
		base: -1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			current := tc.base
			tgt := &gcTarget{
				set: func(p int) (prev int) {
					prev, current = current, p

					return prev
				},
			}

			restore := tgt.tighten(20)
			assert.Equal(t, tc.base, current)

			restore()
			assert.Equal(t, tc.base, current)
		})
	}
}

// newFilterForTest returns a *DNSFilter with an initialized filtering engine
// built from blockFilters.
func newFilterForTest(tb testing.TB, blockFilters []Filter) (d *DNSFilter) {
	tb.Helper()

	d, err := New(&Config{
		Logger:  testLogger,
		DataDir: tb.TempDir(),
	}, blockFilters)
	require.NoError(tb, err)
	tb.Cleanup(d.Close)

	return d
}

// TestDNSFilter_initFiltering_unchanged makes sure that the engines are not
// rebuilt when nothing that they are built from has changed.  Rebuilding them
// temporarily doubles the memory taken by the rules.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/8297.
func TestDNSFilter_initFiltering_unchanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	blockFilters := []Filter{{ID: 1, Data: []byte("||example.com^\n")}}

	d := newFilterForTest(t, blockFilters)

	engine := d.filteringEngine
	require.NotNil(t, engine)

	t.Run("no_rebuild", func(t *testing.T) {
		require.NoError(t, d.initFiltering(ctx, nil, blockFilters))

		assert.Same(t, engine, d.filteringEngine)
	})

	t.Run("rebuild_on_change", func(t *testing.T) {
		changed := []Filter{{ID: 1, Data: []byte("||example.net^\n")}}
		require.NoError(t, d.initFiltering(ctx, nil, changed))

		assert.NotSame(t, engine, d.filteringEngine)
	})
}

// TestDNSFilter_initFiltering_error makes sure that a failed rebuild leaves
// the previous engines in place and does not prevent the following ones.
func TestDNSFilter_initFiltering_error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	blockFilters := []Filter{{ID: 1, Data: []byte("||example.com^\n")}}

	d := newFilterForTest(t, blockFilters)

	engine := d.filteringEngine
	require.NotNil(t, engine)

	// Rule lists with duplicate IDs are rejected by the rule storage.
	bad := []Filter{
		{ID: 1, Data: []byte("||example.net^\n")},
		{ID: 1, Data: []byte("||example.org^\n")},
	}

	require.Error(t, d.initFiltering(ctx, nil, bad))
	assert.Same(t, engine, d.filteringEngine)

	// The failed attempt must not have been recorded as the current state.
	changed := []Filter{{ID: 1, Data: []byte("||example.net^\n")}}
	require.NoError(t, d.initFiltering(ctx, nil, changed))
	assert.NotSame(t, engine, d.filteringEngine)
}

// benchRuleList writes a rule list of n unique blocking rules into a file
// within a temporary directory and returns its path.
func benchRuleList(tb testing.TB, n int, suffix string) (path string) {
	tb.Helper()

	buf := &strings.Builder{}
	for i := range n {
		fmt.Fprintf(buf, "||host%d-%s.example^\n", i, suffix)
	}

	path = filepath.Join(tb.TempDir(), "rules.txt")
	require.NoError(tb, os.WriteFile(path, []byte(buf.String()), 0o644))

	return path
}

// BenchmarkDNSFilter_initFiltering measures rebuilding the filtering engines
// from rule lists of a realistic size, which is the operation that briefly
// doubles the memory taken by the rules.
//
// The "unchanged" case is the one the fingerprint is meant to skip: every
// filtering-related HTTP API request reaches initFiltering, including the ones
// that set a property to the value it already had.  Compare the two with:
//
//	go test -bench BenchmarkDNSFilter_initFiltering -benchmem ./internal/filtering/
//
// Allocated bytes per operation are the figure to watch.  They stay
// proportional to the size of the lists for a real rebuild, and drop to
// approximately zero once a rebuild is skipped.
//
// This benchmark is deliberately self-contained, so that it can be run on the
// constrained hardware these rebuilds are reported to run out of memory on
// without having to obtain a particular blocklist first.
func BenchmarkDNSFilter_initFiltering(b *testing.B) {
	ctx := context.Background()

	for _, n := range []int{10_000, 100_000} {
		firstPath := benchRuleList(b, n, "a")
		secondPath := benchRuleList(b, n, "b")

		first := []Filter{{ID: 1, FilePath: firstPath}}
		second := []Filter{{ID: 1, FilePath: secondPath}}

		b.Run(fmt.Sprintf("changed_%d", n), func(b *testing.B) {
			d := newFilterForTest(b, first)

			b.ReportAllocs()
			b.ResetTimer()

			for i := range b.N {
				// Alternate the lists so that every iteration has work to do.
				flts := first
				if i%2 == 0 {
					flts = second
				}

				require.NoError(b, d.initFiltering(ctx, nil, flts))
			}
		})

		b.Run(fmt.Sprintf("unchanged_%d", n), func(b *testing.B) {
			d := newFilterForTest(b, first)
			require.NoError(b, d.initFiltering(ctx, nil, first))

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				require.NoError(b, d.initFiltering(ctx, nil, first))
			}
		})
	}
}
