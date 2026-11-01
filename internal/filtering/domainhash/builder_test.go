package domainhash_test

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/domainhash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLine(t *testing.T) {
	testCases := []struct {
		name string
		line string
		want []string
		ok   bool
	}{{
		name: "hosts_zero",
		line: "0.0.0.0 example.com",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "hosts_loopback",
		line: "127.0.0.1\texample.com",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "hosts_ipv6_unspecified",
		line: ":: example.com",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "hosts_multiple",
		line: "0.0.0.0 example.com other.example.net",
		want: []string{"example.com", "other.example.net"},
		ok:   true,
	}, {
		name: "hosts_www_normalized",
		line: "0.0.0.0 example.com www.example.com",
		want: []string{"example.com", "example.com"},
		ok:   true,
	}, {
		name: "hosts_inline_comment",
		line: "0.0.0.0 example.com # ad server",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "hosts_normalized",
		line: "0.0.0.0 Example.COM",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "hosts_non_blocking_addr",
		line: "192.168.1.1 example.com",
		ok:   false,
	}, {
		name: "hosts_single_field",
		line: "0.0.0.0",
		ok:   false,
	}, {
		name: "hosts_comment",
		line: "# a comment",
		ok:   false,
	}, {
		name: "adblock_comment",
		line: "! A comment",
		ok:   false,
	}, {
		name: "adblock_section",
		line: "[/etc/hosts]",
		ok:   false,
	}, {
		name: "adblock_plain",
		line: "||example.com^",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "adblock_without_separator",
		line: "||example.com",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "adblock_normalized",
		line: "||WWW.Example.COM.^",
		want: []string{"example.com"},
		ok:   true,
	}, {
		name: "adblock_allowlist",
		line: "@@||example.com^",
		ok:   false,
	}, {
		name: "adblock_modifier",
		line: "||example.com^$third-party",
		ok:   false,
	}, {
		name: "adblock_regex",
		line: "/example\\.(com|net)/",
		ok:   false,
	}, {
		name: "adblock_cosmetic",
		line: "example.com##.ad",
		ok:   false,
	}, {
		name: "adblock_cosmetic_hide",
		line: "example.com#?#div:-abp-has(.ad)",
		ok:   false,
	}, {
		name: "adblock_wildcard",
		line: "||*.example.com^",
		ok:   false,
	}, {
		name: "adblock_path",
		line: "||example.com/ads/",
		ok:   false,
	}, {
		name: "bare_domain",
		line: "example.com",
		ok:   false,
	}, {
		name: "empty",
		line: "   ",
		ok:   false,
	}, {
		name: "bom",
		line: "\ufeff0.0.0.0 example.com",
		want: []string{"example.com"},
		ok:   true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domainhash.ParseLine(tc.line)
			assert.Equal(t, tc.ok, ok, "ok")
			if tc.ok {
				assert.Equal(t, tc.want, got, "domains")
			} else {
				assert.Empty(t, got, "domains")
			}
		})
	}
}

func TestBuilderRules(t *testing.T) {
	cfg := domainhash.DomainHashConfig{
		HashBytes:      domainhash.DefaultHashBytes,
		DataDir:        t.TempDir(),
		MemBudgetBytes: 1 << 10,
	}

	b, err := domainhash.NewBuilder(cfg)
	require.NoError(t, err, "new")

	lines := []string{
		"# a comment",
		"0.0.0.0 ads.example.com",
		"127.0.0.1 ads.example.com www2.example.net",
		"||tracker.example.org^",
		"@@||allowed.example.org^",
		"||mod.example.org^$important",
		"||re.example.org^",
		"example.com##.ad",
		"192.168.0.1 router.example.com",
		"0.0.0.0 dup.example.com",
		"0.0.0.0 DUP.EXAMPLE.COM",
	}
	for _, l := range lines {
		_, err = b.AddLine(l)
		require.NoError(t, err, "add %q", l)
	}

	stats, err := b.Build()
	require.NoError(t, err, "build")

	// Five distinct domains are accepted and five lines are unsupported.
	require.Equal(t, 7, stats.Domains, "domains")
	require.Equal(t, 5, stats.Skipped, "skipped")
	require.Equal(t, 5, stats.Hashes, "hashes")
	require.Zero(t, stats.Collisions, "collisions")

	// The blocklist is exactly as large as the number of hashes it holds.
	fi, err := os.Stat(b.Path())
	require.NoError(t, err, "stat")
	require.Equal(
		t,
		int64(stats.Hashes*cfg.HashBytes),
		fi.Size(),
		"size",
	)

	idx, err := domainhash.Open(b.Path(), cfg.HashBytes)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, idx.Close(), "close")
	}()

	assert.True(t, idx.ContainsDomain("ads.example.com"), "ads.example.com")
	assert.True(t, idx.ContainsDomain("sub.ads.example.com"), "sub.ads.example.com")
	assert.True(t, idx.ContainsDomain("www2.example.net"), "www2.example.net")
	assert.True(t, idx.ContainsDomain("tracker.example.org"), "tracker.example.org")
	assert.True(t, idx.ContainsDomain("re.example.org"), "re.example.org")
	assert.True(t, idx.ContainsDomain("dup.example.com"), "dup.example.com")

	// Skipped rules must not end up in the blocklist.
	assert.False(t, idx.ContainsDomain("allowed.example.org"), "allowed.example.org")
	assert.False(t, idx.ContainsDomain("mod.example.org"), "mod.example.org")
	assert.False(t, idx.ContainsDomain("router.example.com"), "router.example.com")
}

// TestBuilderExternalSort checks that a tiny memory budget makes the builder
// flush many sorted runs and that the merged result is sorted and free of
// duplicates.
func TestBuilderExternalSort(t *testing.T) {
	const (
		hashBytes   = 4
		numDomains  = 5000
		budgetBytes = 1024
	)

	cfg := domainhash.DomainHashConfig{
		HashBytes:      hashBytes,
		DataDir:        t.TempDir(),
		MemBudgetBytes: budgetBytes,
	}

	b, err := domainhash.NewBuilder(cfg)
	require.NoError(t, err, "new")

	// Add the domains in reverse order and repeat every fiftieth one, so that
	// both the sorting and the deduplication have work to do.
	want := make(map[string]struct{}, numDomains)
	for i := numDomains - 1; i >= 0; i-- {
		d := fmt.Sprintf("host%05d.example.com", i)
		want[d] = struct{}{}

		require.NoError(t, b.AddDomain(d), "add %q", d)

		if i%50 == 0 {
			require.NoError(t, b.AddDomain(d), "add duplicate %q", d)
		}
	}

	stats, err := b.Build()
	require.NoError(t, err, "build")

	// Every domain is accepted, and so is every repeated one.
	require.Equal(t, numDomains+numDomains/50, stats.Domains, "domains")
	require.GreaterOrEqual(t, stats.Runs, 10, "runs")
	require.Equal(t, numDomains, stats.Hashes, "hashes")
	require.Zero(t, stats.Collisions, "collisions")

	hashes := readHashes(t, b.Path(), hashBytes)

	// The file is sorted in ascending order and has no duplicates.
	require.Len(t, hashes, stats.Hashes, "hashes")
	require.True(t, slices.IsSorted(hashes), "sorted")
	for i := 1; i < len(hashes); i++ {
		require.NotEqual(t, hashes[i-1], hashes[i], "duplicate at %d", i)
	}

	// Every domain of the input is in the blocklist.
	idx, err := domainhash.Open(b.Path(), hashBytes)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, idx.Close(), "close")
	}()

	for _, d := range []string{"host00000.example.com", "host02500.example.com", "host04999.example.com"} {
		require.True(t, idx.ContainsDomain(d), "%s", d)
	}

	assert.False(t, idx.ContainsDomain("host05000.example.com"), "host05000.example.com")
	assert.False(t, idx.ContainsDomain("host02500.example.net"), "host02500.example.net")
}

// TestBuilderCollisions checks that the builder reports distinct domains that
// share a truncated hash.
func TestBuilderCollisions(t *testing.T) {
	const (
		hashBytes   = domainhash.MinHashBytes
		numDomains  = 20000
		budgetBytes = 4096
	)

	cfg := domainhash.DomainHashConfig{
		HashBytes:      hashBytes,
		DataDir:        t.TempDir(),
		MemBudgetBytes: budgetBytes,
	}

	b, err := domainhash.NewBuilder(cfg)
	require.NoError(t, err, "new")

	for i := range numDomains {
		require.NoError(t, b.AddDomain(fmt.Sprintf("host%05d.test.example", i)), "add")
	}

	stats, err := b.Build()
	require.NoError(t, err, "build")

	require.Equal(t, numDomains, stats.Domains, "domains")
	require.Greater(t, stats.Collisions, 0, "collisions")

	// Every domain either becomes a hash of its own or collides with an
	// already written one.
	require.Equal(t, numDomains, stats.Hashes+stats.Collisions, "hashes+collisions")

	samples := b.CollisionSamples()
	require.NotEmpty(t, samples, "samples")
	require.LessOrEqual(t, len(samples), 16, "samples")

	for _, c := range samples {
		for _, d := range c.Domains {
			h, ok := domainhash.HashDomain(d, hashBytes)
			require.True(t, ok, "normalize %q", d)
			require.Equal(t, h, c.Hash, "hash of %q", d)
		}

		require.NotEqual(t, c.Domains[0], c.Domains[1], "domains")
	}

	// Colliding domains still match, since only the hash is stored.
	idx, err := domainhash.Open(b.Path(), hashBytes)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, idx.Close(), "close")
	}()

	require.Equal(t, stats.Hashes, idx.Len(), "len")

	for _, c := range samples {
		require.True(t, idx.ContainsDomain(c.Domains[0]), "%s", c.Domains[0])
		require.True(t, idx.ContainsDomain(c.Domains[1]), "%s", c.Domains[1])
	}
}

// TestBuilder100k checks that a hundred thousand domains are built with a
// bounded peak memory usage.
//
// The builder never holds more than the current batch in memory: as soon as the
// batch reaches the memory budget it is sorted and written out to a run file,
// and only the paths of the run files are kept, so the peak memory usage is one
// budget plus a single record no matter how many domains are added.  That peak
// is asserted exactly below through Stats.PeakBatchBytes; the heap sample is a
// loose safety net and is only logged unless it is wildly off.
func TestBuilder100k(t *testing.T) {
	const (
		hashBytes   = domainhash.DefaultHashBytes
		numDomains  = 100_000
		budgetBytes = 256 << 10

		// A single record is at most the hash, the longest domain, and the
		// record separator.
		maxRecordBytes = int64(8 + len("host099999.example49.com") + 1)

		// The whole process must stay far below a fully in-memory build of the
		// domain names plus their hashes would need on top of the baseline.
		softHeapLimit = 64 << 20
	)

	cfg := domainhash.DomainHashConfig{
		HashBytes:      hashBytes,
		DataDir:        t.TempDir(),
		MemBudgetBytes: budgetBytes,
	}

	b, err := domainhash.NewBuilder(cfg)
	require.NoError(t, err, "new")

	var baseline, peakHeap uint64
	{
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		baseline = m.HeapAlloc
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Sampling the heap stops the world, so do it sparingly.
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()

		var m runtime.MemStats
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
			}

			runtime.ReadMemStats(&m)
			if m.HeapAlloc > peakHeap {
				peakHeap = m.HeapAlloc
			}
		}
	}()

	start := time.Now()
	for i := range numDomains {
		d := fmt.Sprintf("host%06d.example%02d.com", i, i%50)
		require.NoError(t, b.AddDomain(d), "add %q", d)
	}

	stats, err := b.Build()
	close(stop)
	wg.Wait()

	require.NoError(t, err, "build")

	require.Equal(t, numDomains, stats.Domains, "domains")
	require.Equal(t, numDomains, stats.Hashes, "hashes")
	require.Greater(t, stats.Runs, 1, "runs")
	require.Zero(t, stats.Collisions, "collisions")

	// The in-memory batch never exceeds the budget by more than one record.
	require.LessOrEqual(
		t,
		stats.PeakBatchBytes,
		budgetBytes+maxRecordBytes,
		"peak batch bytes",
	)

	peak := int64(peakHeap) - int64(baseline)
	t.Logf(
		"built %d domains in %s, peak batch %d bytes, peak heap growth %d bytes",
		stats.Domains,
		time.Since(start).Round(time.Millisecond),
		stats.PeakBatchBytes,
		peak,
	)

	if peak > softHeapLimit {
		t.Errorf(
			"peak heap growth %d bytes exceeds the soft limit of %d bytes",
			peak,
			softHeapLimit,
		)
	}

	idx, err := domainhash.Open(b.Path(), hashBytes)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, idx.Close(), "close")
	}()

	require.Equal(t, stats.Hashes, idx.Len(), "len")
	require.True(t, idx.ContainsDomain("host000000.example00.com"), "first")
	require.True(t, idx.ContainsDomain("host099999.example49.com"), "last")
	require.True(t, idx.ContainsDomain("sub.host050000.example00.com"), "middle")
	require.False(t, idx.ContainsDomain("host100000.example00.com"), "absent")
}

func TestBuilderEmpty(t *testing.T) {
	cfg := domainhash.DomainHashConfig{DataDir: t.TempDir()}

	b, err := domainhash.NewBuilder(cfg)
	require.NoError(t, err, "new")

	stats, err := b.Build()
	require.NoError(t, err, "build")
	require.Zero(t, stats.Domains, "domains")
	require.Zero(t, stats.Hashes, "hashes")

	fi, err := os.Stat(b.Path())
	require.NoError(t, err, "stat")
	require.Zero(t, fi.Size(), "size")

	idx, err := domainhash.Open(b.Path(), 0)
	require.NoError(t, err, "open")
	require.Equal(t, 0, idx.Len(), "len")
	require.NoError(t, idx.Close(), "close")

	// The intermediate files are gone.
	entries, err := os.ReadDir(cfg.DataDir)
	require.NoError(t, err, "read dir")
	require.Len(t, entries, 1, "entries")
	require.Equal(t, domainhash.FileName, entries[0].Name(), "name")
}

func TestBuilderRebuild(t *testing.T) {
	dir := t.TempDir()
	cfg := domainhash.DomainHashConfig{
		HashBytes:      domainhash.DefaultHashBytes,
		DataDir:        dir,
		MemBudgetBytes: 1 << 10,
	}

	b, err := domainhash.NewBuilder(cfg)
	require.NoError(t, err, "new")
	for i := range 100 {
		require.NoError(t, b.AddDomain(fmt.Sprintf("host%03d.example.com", i)), "add")
	}

	_, err = b.Build()
	require.NoError(t, err, "build first")

	// A second builder atomically replaces the previous blocklist file.
	b, err = domainhash.NewBuilder(cfg)
	require.NoError(t, err, "new")
	require.NoError(t, b.AddDomain("only.example.com"), "add")

	stats, err := b.Build()
	require.NoError(t, err, "build second")
	require.Equal(t, 1, stats.Hashes, "hashes")

	idx, err := domainhash.Open(b.Path(), cfg.HashBytes)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, idx.Close(), "close")
	}()

	require.Equal(t, 1, idx.Len(), "len")
	assert.True(t, idx.ContainsDomain("only.example.com"), "only.example.com")
	assert.False(t, idx.ContainsDomain("host000.example.com"), "host000.example.com")
}

func TestBuilderClosed(t *testing.T) {
	b, err := domainhash.NewBuilder(domainhash.DomainHashConfig{DataDir: t.TempDir()})
	require.NoError(t, err, "new")

	require.NoError(t, b.AddDomain("example.com"), "add")
	_, err = b.Build()
	require.NoError(t, err, "build")

	require.NoError(t, b.Close(), "close again")
	require.ErrorIs(t, b.AddDomain("example.org"), domainhash.ErrBuilderClosed, "add")
	require.ErrorIs(t, b.MergeRuns(), domainhash.ErrBuilderClosed, "merge")
}

func TestBuilderBadConfig(t *testing.T) {
	b, err := domainhash.NewBuilder(domainhash.DomainHashConfig{})
	require.ErrorIs(t, err, domainhash.ErrBadConfig, "err")
	require.Nil(t, b, "builder")
}

// readHashes reads all the hashes of the blocklist file at path.
func readHashes(t *testing.T, path string, hashBytes int) (hashes []uint64) {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "read")
	require.Zero(t, len(data)%hashBytes, "size")

	hashes = make([]uint64, 0, len(data)/hashBytes)
	for i := 0; i < len(data); i += hashBytes {
		hashes = append(hashes, domainhash.GetHash(data[i:i+hashBytes]))
	}

	return hashes
}
