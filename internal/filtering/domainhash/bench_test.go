package domainhash_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/domainhash"
)

// synthDomain returns a deterministic synthetic domain for index i.
func synthDomain(i int) (d string) {
	return fmt.Sprintf("d%06d.block.test", i)
}

func buildIndex(tb testing.TB, n, hashBytes int, budget int64) (idx *domainhash.Index, path string) {
	tb.Helper()

	dir := tb.TempDir()
	b, err := domainhash.NewBuilder(domainhash.DomainHashConfig{
		Enabled:        true,
		HashBytes:      hashBytes,
		DataDir:        dir,
		MemBudgetBytes: budget,
	})
	if err != nil {
		tb.Fatalf("NewBuilder: %v", err)
	}
	defer func() { _ = b.Close() }()

	for i := 0; i < n; i++ {
		if err = b.AddDomain(synthDomain(i)); err != nil {
			tb.Fatalf("AddDomain(%d): %v", i, err)
		}
	}

	if _, err = b.Build(); err != nil {
		tb.Fatalf("Build: %v", err)
	}

	path = b.Path()
	idx, err = domainhash.Open(path, hashBytes)
	if err != nil {
		tb.Fatalf("Open: %v", err)
	}

	return idx, path
}

func BenchmarkBuilder(b *testing.B) {
	sizes := []int{100_000, 500_000, 1_000_000}
	budgets := []int64{1 << 20, 4 << 20, 16 << 20}

	for _, n := range sizes {
		for _, budget := range budgets {
			b.Run(fmt.Sprintf("n=%d/budget=%dKiB", n, budget>>10), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					dir := b.TempDir()
					builder, err := domainhash.NewBuilder(domainhash.DomainHashConfig{
						Enabled:        true,
						HashBytes:      domainhash.DefaultHashBytes,
						DataDir:        dir,
						MemBudgetBytes: budget,
					})
					if err != nil {
						b.Fatal(err)
					}

					for i := 0; i < n; i++ {
						if err = builder.AddDomain(synthDomain(i)); err != nil {
							_ = builder.Close()
							b.Fatal(err)
						}
					}

					stats, err := builder.Build()
					_ = builder.Close()
					if err != nil {
						b.Fatal(err)
					}

					b.ReportMetric(float64(stats.PeakBatchBytes), "peak-batch-bytes")
					b.ReportMetric(float64(stats.Runs), "runs")
					b.ReportMetric(float64(stats.Hashes), "hashes")
				}
			})
		}
	}
}

func BenchmarkContainsDomain(b *testing.B) {
	sizes := []int{100_000, 500_000, 1_000_000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			idx, _ := buildIndex(b, n, domainhash.DefaultHashBytes, 16<<20)
			b.Cleanup(func() { _ = idx.Close() })

			// Touch every page once so the benchmark measures warm page-cache
			// lookups, not cold major faults.
			for i := 0; i < n; i += 4096 / domainhash.DefaultHashBytes {
				_ = idx.ContainsDomain(synthDomain(i))
			}
			_ = idx.ContainsDomain(synthDomain(n / 2))
			_ = idx.ContainsDomain("miss.example.test")

			hit := synthDomain(n / 2)
			miss := "miss.example.test"

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
				// Domain itself is not in the list; parent synthDomain(n/2)
				// style: use a subdomain of a blocked name.
				parent := synthDomain(n / 3)
				host := "sub." + parent
				b.ReportAllocs()
				for b.Loop() {
					if !idx.ContainsDomain(host) {
						b.Fatal("expected parent hit")
					}
				}
			})
		})
	}
}

func BenchmarkOpen(b *testing.B) {
	const n = 1_000_000
	dir := b.TempDir()
	builder, err := domainhash.NewBuilder(domainhash.DomainHashConfig{
		Enabled:        true,
		DataDir:        dir,
		MemBudgetBytes: 16 << 20,
	})
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < n; i++ {
		if err = builder.AddDomain(synthDomain(i)); err != nil {
			b.Fatal(err)
		}
	}
	if _, err = builder.Build(); err != nil {
		b.Fatal(err)
	}
	path := builder.Path()
	_ = builder.Close()

	b.ReportAllocs()
	for b.Loop() {
		idx, openErr := domainhash.Open(path, domainhash.DefaultHashBytes)
		if openErr != nil {
			b.Fatal(openErr)
		}
		_ = idx.Close()
	}

	b.ReportMetric(float64((n*domainhash.DefaultHashBytes)>>20), "file-MiB")
	_ = filepath.Base(path)
	runtime.KeepAlive(path)
}
