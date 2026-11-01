// Command bench-domainhash compares urlfilter vs domainhash build time,
// lookup latency, and process RSS for synthetic ||domain^ blocklists.
//
// Usage:
//
//	go run ./scripts/bench-domainhash -n 100000,500000,1000000
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/domainhash"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
)

func main() {
	sizesFlag := flag.String("n", "100000,500000,1000000", "comma-separated domain counts")
	lookups := flag.Int("lookups", 50_000, "lookup iterations per mode")
	budget := flag.Int64("budget", 16<<20, "domainhash builder memory budget in bytes")
	workdir := flag.String("dir", "", "work directory (default: temp)")
	flag.Parse()

	sizes := parseSizes(*sizesFlag)
	dir := *workdir
	if dir == "" {
		var err error
		dir, err = os.MkdirTemp("", "agh-bench-domainhash-")
		if err != nil {
			fatalf("temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(dir) }()
	}

	fmt.Printf("host=%s go=%s GOMAXPROCS=%d\n", hostSummary(), runtime.Version(), runtime.GOMAXPROCS(0))
	fmt.Printf("workdir=%s budget=%d KiB lookups=%d\n\n", dir, *budget>>10, *lookups)

	fmt.Println("## Build + steady-state memory (engines measured separately)")
	fmt.Println("| N | engine | build | RSS Δ | heap Δ | file |")
	fmt.Println("|---:|---|---:|---:|---:|---:|")

	type ready struct {
		n     int
		eng   *urlfilter.DNSEngine
		store *filterlist.RuleStorage
		idx   *domainhash.Index
	}
	var prepared []ready

	for _, n := range sizes {
		forceGC()
		rssBase := vmRSS()
		heapBase := heapAlloc()

		t0 := time.Now()
		eng, store := mustURLFilter(n)
		buildUF := time.Since(t0)
		forceGC()
		rssUF := vmRSS()
		heapUF := heapAlloc()

		fmt.Printf(
			"| %s | urlfilter | %s | %s | %s | — |\n",
			fmtInt(n),
			fmtDur(buildUF),
			fmtBytes(diff(rssUF, rssBase)),
			fmtBytes(diff(heapUF, heapBase)),
		)

		_ = store.Close()
		eng = nil
		store = nil
		forceGC()
		debugFreeOSMemory()

		forceGC()
		rssBase = vmRSS()
		heapBase = heapAlloc()

		t1 := time.Now()
		idx := mustDomainHash(dir, n, *budget)
		buildDH := time.Since(t1)
		forceGC()
		rssDH := vmRSS()
		heapDH := heapAlloc()
		fi, err := os.Stat(filepath.Join(dir, fmt.Sprintf("n%d", n), domainhash.FileName))
		fileSz := int64(0)
		if err == nil {
			fileSz = fi.Size()
		}

		fmt.Printf(
			"| %s | domainhash | %s | %s | %s | %s |\n",
			fmtInt(n),
			fmtDur(buildDH),
			fmtBytes(diff(rssDH, rssBase)),
			fmtBytes(diff(heapDH, heapBase)),
			fmtBytes(uint64(fileSz)),
		)

		// Rebuild urlfilter for the lookup section so both stay available.
		eng, store = mustURLFilter(n)
		prepared = append(prepared, ready{n: n, eng: eng, store: store, idx: idx})
	}

	fmt.Println()
	fmt.Println("## Lookup latency (warm)")
	fmt.Println("| N | engine | case | ns/op |")
	fmt.Println("|---:|---|---|---:|")

	for _, p := range prepared {
		hit := synthDomain(p.n / 2)
		miss := "miss.example.test"
		parent := "sub." + synthDomain(p.n/3)

		// Warm domainhash pages.
		for i := 0; i < p.n; i += 4096 / domainhash.DefaultHashBytes {
			_ = p.idx.ContainsDomain(synthDomain(i))
		}

		printLookup(p.n, "urlfilter", "hit", *lookups, func() {
			_, _ = p.eng.MatchRequest(&urlfilter.DNSRequest{Hostname: hit})
		})
		printLookup(p.n, "urlfilter", "miss", *lookups, func() {
			_, _ = p.eng.MatchRequest(&urlfilter.DNSRequest{Hostname: miss})
		})
		printLookup(p.n, "urlfilter", "parent_hit", *lookups, func() {
			_, _ = p.eng.MatchRequest(&urlfilter.DNSRequest{Hostname: parent})
		})
		printLookup(p.n, "domainhash", "hit", *lookups, func() {
			_ = p.idx.ContainsDomain(hit)
		})
		printLookup(p.n, "domainhash", "miss", *lookups, func() {
			_ = p.idx.ContainsDomain(miss)
		})
		printLookup(p.n, "domainhash", "parent_hit", *lookups, func() {
			_ = p.idx.ContainsDomain(parent)
		})
	}

	for _, p := range prepared {
		_ = p.store.Close()
		_ = p.idx.Close()
	}
}

func printLookup(n int, engine, cse string, iters int, fn func()) {
	for i := 0; i < iters/10; i++ {
		fn()
	}
	t0 := time.Now()
	for i := 0; i < iters; i++ {
		fn()
	}
	ns := float64(time.Since(t0).Nanoseconds()) / float64(iters)
	fmt.Printf("| %s | %s | %s | %.0f |\n", fmtInt(n), engine, cse, ns)
}

func mustURLFilter(n int) (eng *urlfilter.DNSEngine, store *filterlist.RuleStorage) {
	var b strings.Builder
	b.Grow(n * 40)
	for i := 0; i < n; i++ {
		b.WriteString("||")
		b.WriteString(synthDomain(i))
		b.WriteString("^\n")
	}
	rl := filterlist.NewBytes(&filterlist.BytesConfig{
		ID:             rules.ListID(1),
		RulesText:      []byte(b.String()),
		IgnoreCosmetic: true,
	})
	store, err := filterlist.NewRuleStorage([]filterlist.Interface{rl})
	if err != nil {
		fatalf("urlfilter storage: %v", err)
	}

	return urlfilter.NewDNSEngine(store), store
}

func mustDomainHash(root string, n int, budget int64) (idx *domainhash.Index) {
	dir := filepath.Join(root, fmt.Sprintf("n%d", n))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatalf("mkdir: %v", err)
	}
	b, err := domainhash.NewBuilder(domainhash.DomainHashConfig{
		Enabled:        true,
		HashBytes:      domainhash.DefaultHashBytes,
		DataDir:        dir,
		MemBudgetBytes: budget,
	})
	if err != nil {
		fatalf("builder: %v", err)
	}
	defer func() { _ = b.Close() }()

	for i := 0; i < n; i++ {
		if err = b.AddDomain(synthDomain(i)); err != nil {
			fatalf("AddDomain: %v", err)
		}
	}
	if _, err = b.Build(); err != nil {
		fatalf("Build: %v", err)
	}
	idx, err = domainhash.Open(b.Path(), domainhash.DefaultHashBytes)
	if err != nil {
		fatalf("Open: %v", err)
	}

	return idx
}

func synthDomain(i int) string {
	return fmt.Sprintf("d%06d.block.test", i)
}

func parseSizes(s string) (out []int) {
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 {
			fatalf("bad size %q", p)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		fatalf("no sizes")
	}

	return out
}

func vmRSS() (bytes uint64) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}

		return kb * 1024
	}

	return 0
}

func heapAlloc() (n uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.HeapAlloc
}

func forceGC() {
	runtime.GC()
	runtime.GC()
}

func debugFreeOSMemory() {
	debug.FreeOSMemory()
}

func diff(a, b uint64) (d uint64) {
	if a >= b {
		return a - b
	}

	return 0
}

func fmtBytes(n uint64) string {
	const (
		KiB = 1024
		MiB = 1024 * KiB
	)
	switch {
	case n >= MiB:
		return fmt.Sprintf("%.1f MiB", float64(n)/float64(MiB))
	case n >= KiB:
		return fmt.Sprintf("%.1f KiB", float64(n)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func fmtDur(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	default:
		return d.String()
	}
}

func fmtInt(n int) string {
	s := strconv.Itoa(n)
	if n < 1000 {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if i != 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}

	return b.String()
}

func hostSummary() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	model := runtime.GOARCH
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				model = strings.TrimSpace(parts[1])
			}
			break
		}
	}

	return model
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
