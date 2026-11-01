package filtering

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/domainhash"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
)

// synthDomain mirrors the domainhash package helper for comparative benches.
func synthDomain(i int) (d string) {
	return fmt.Sprintf("d%06d.block.test", i)
}

// adblockRules builds plain ||domain^ rules.  Those become urlfilter
// NetworkRules — the heavy representation AdGuard Home uses for typical
// blocklists — and they match subdomains, matching domainhash parent walks.
func adblockRules(n int) (text string) {
	var b strings.Builder
	b.Grow(n * 40)
	for i := 0; i < n; i++ {
		b.WriteString("||")
		b.WriteString(synthDomain(i))
		b.WriteString("^\n")
	}

	return b.String()
}

func heapInUse() (n uint64) {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.HeapInuse
}

func buildURLFilter(tb testing.TB, n int) (eng *urlfilter.DNSEngine, storage *filterlist.RuleStorage) {
	tb.Helper()

	text := adblockRules(n)
	rl := filterlist.NewBytes(&filterlist.BytesConfig{
		ID:             rules.ListID(1),
		RulesText:      []byte(text),
		IgnoreCosmetic: true,
	})
	storage, err := filterlist.NewRuleStorage([]filterlist.Interface{rl})
	if err != nil {
		tb.Fatalf("NewRuleStorage: %v", err)
	}
	eng = urlfilter.NewDNSEngine(storage)

	return eng, storage
}

func buildDomainHash(tb testing.TB, n int) (idx *domainhash.Index) {
	tb.Helper()

	dir := tb.TempDir()
	b, err := domainhash.NewBuilder(domainhash.DomainHashConfig{
		Enabled:        true,
		HashBytes:      domainhash.DefaultHashBytes,
		DataDir:        dir,
		MemBudgetBytes: 16 << 20,
	})
	if err != nil {
		tb.Fatalf("NewBuilder: %v", err)
	}
	defer func() { _ = b.Close() }()

	for i := 0; i < n; i++ {
		if err = b.AddDomain(synthDomain(i)); err != nil {
			tb.Fatalf("AddDomain: %v", err)
		}
	}
	if _, err = b.Build(); err != nil {
		tb.Fatalf("Build: %v", err)
	}
	idx, err = domainhash.Open(b.Path(), domainhash.DefaultHashBytes)
	if err != nil {
		tb.Fatalf("Open: %v", err)
	}

	return idx
}

func BenchmarkCompareBuild(b *testing.B) {
	sizes := []int{100_000, 500_000, 1_000_000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("urlfilter/n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				eng, storage := buildURLFilter(b, n)
				_ = eng
				_ = storage.Close()
			}
		})

		b.Run(fmt.Sprintf("domainhash/n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				idx := buildDomainHash(b, n)
				_ = idx.Close()
			}
		})
	}
}

func BenchmarkCompareLookup(b *testing.B) {
	sizes := []int{100_000, 500_000, 1_000_000}

	for _, n := range sizes {
		hit := synthDomain(n / 2)
		miss := "miss.example.test"
		parent := "sub." + synthDomain(n/3)

		b.Run(fmt.Sprintf("urlfilter/n=%d", n), func(b *testing.B) {
			eng, storage := buildURLFilter(b, n)
			b.Cleanup(func() { _ = storage.Close() })

			reqHit := &urlfilter.DNSRequest{Hostname: hit}
			reqMiss := &urlfilter.DNSRequest{Hostname: miss}
			reqParent := &urlfilter.DNSRequest{Hostname: parent}

			b.Run("hit", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, ok := eng.MatchRequest(reqHit)
					if !ok {
						b.Fatal("expected hit")
					}
				}
			})
			b.Run("miss", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, ok := eng.MatchRequest(reqMiss)
					if ok {
						b.Fatal("expected miss")
					}
				}
			})
			b.Run("parent_hit", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, ok := eng.MatchRequest(reqParent)
					if !ok {
						b.Fatal("expected parent hit")
					}
				}
			})
		})

		b.Run(fmt.Sprintf("domainhash/n=%d", n), func(b *testing.B) {
			idx := buildDomainHash(b, n)
			b.Cleanup(func() { _ = idx.Close() })

			// Warm page cache.
			for i := 0; i < n; i += 4096 / domainhash.DefaultHashBytes {
				_ = idx.ContainsDomain(synthDomain(i))
			}

			b.Run("hit", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if !idx.ContainsDomain(hit) {
						b.Fatal("expected hit")
					}
				}
			})
			b.Run("miss", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if idx.ContainsDomain(miss) {
						b.Fatal("expected miss")
					}
				}
			})
			b.Run("parent_hit", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if !idx.ContainsDomain(parent) {
						b.Fatal("expected parent hit")
					}
				}
			})
		})
	}
}

func TestCompareSteadyStateMemory(t *testing.T) {
	sizes := []int{100_000, 500_000, 1_000_000}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			before := heapInUse()
			eng, storage := buildURLFilter(t, n)
			afterUF := heapInUse()
			ufBytes := afterUF - before
			if afterUF < before {
				ufBytes = afterUF // GC noise; still report absolute in-use
			}

			beforeDH := heapInUse()
			idx := buildDomainHash(t, n)
			afterDH := heapInUse()
			dhBytes := afterDH - beforeDH
			if afterDH < beforeDH {
				dhBytes = 0
			}

			t.Logf(
				"n=%d urlfilter_heap_delta=%d MiB (inuse=%d) domainhash_heap_delta=%d KiB (inuse=%d) file=%d KiB",
				n,
				ufBytes>>20,
				afterUF>>20,
				dhBytes>>10,
				afterDH>>20,
				(n*domainhash.DefaultHashBytes)>>10,
			)

			_ = eng
			_ = storage.Close()
			_ = idx.Close()

			// Soft expectation: domain-hash heap delta stays tiny; urlfilter
			// grows with N.  HeapInuse deltas are noisy — only assert the
			// domainhash side stays well under the on-disk blob size.
			fileBytes := uint64(n * domainhash.DefaultHashBytes)
			if dhBytes > fileBytes+32<<20 {
				t.Fatalf("domainhash heap delta %d exceeds file+32MiB (%d)", dhBytes, fileBytes)
			}
			_ = ufBytes
		})
	}
}
