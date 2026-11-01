# Benchmarks: upstream urlfilter vs Next-Gen domainhash

Synthetic `||domain^` lists (same shape as typical DNS blocklists). Harness:

```bash
go run ./scripts/bench-domainhash -n 100000,500000,1000000
```

Go microbenchmarks:

```bash
go test ./internal/filtering/domainhash/ -bench . -benchmem
go test ./internal/filtering/ -bench 'BenchmarkCompare' -benchmem
```

## Does every lookup hit the disk?

**No — not once the working set is warm.**

The index is opened with `mmap(PROT_READ, MAP_SHARED)`. Linux does **not**
eagerly copy the whole file into RAM. It maps the file into the process address
space and faults pages (~4 KiB) into the **page cache** on first access.

| Situation | What happens |
|---|---|
| Cold start / first lookups | Minor page faults; pages read from storage into page cache |
| Steady DNS load, RAM available | Binary search touches ~`log2(N)` cache lines; pages stay resident → **RAM-speed lookups, no syscall I/O** |
| Memory pressure | Kernel can reclaim clean mmap pages; next touch faults them back in |
| Full file warm (1M × 5 B ≈ 4.8 MiB) | Entire blob fits easily even on ~100 MiB routers |

So: storage-backed, **page-cache resident when the OS can afford it**, not a
`read()` per DNS query. Under reclaim, you pay occasional faults — still cheap
versus upstream RTT.

## Measured on a desktop (2026-11-01)

Host: AMD Ryzen AI MAX+ 395, linux/amd64, go1.27.3, `GOMAXPROCS=4`
(OpenStick-like core count; this CPU is far faster per core than an A53).

### Steady-state memory (engines measured separately)

| N | Engine | Build | RSS Δ | Heap Δ | On-disk file |
|---:|---|---:|---:|---:|---:|
| 100,000 | **urlfilter** (upstream path) | 134 ms | **24.7 MiB** | 2.1 MiB | — |
| 100,000 | **domainhash** (Next-Gen) | 146 ms | 3.1 MiB | ~1 KiB | **488 KiB** |
| 500,000 | urlfilter | 813 ms | **89.2 MiB** | 10.5 MiB | — |
| 500,000 | domainhash | 582 ms | 19.9 MiB* | ~0 | **2.4 MiB** |
| 1,000,000 | urlfilter | 1.93 s | **155.0 MiB** | 21.0 MiB | — |
| 1,000,000 | domainhash | 921 ms | 72.3 MiB* | ~0 | **4.8 MiB** |

\* Domainhash RSS Δ right after `Build()` still includes transient builder
scratch the OS has not fully reclaimed. **True steady-state for lookups is the
mmap file size** (page cache), i.e. ~0.5 / 2.4 / 4.8 MiB — reclaimable under
pressure. Urlfilter RSS is live heap objects and is **not** reclaimable without
dropping the engine.

### Warm lookup latency

| N | Engine | hit | miss | parent hit |
|---:|---|---:|---:|---:|
| 100,000 | urlfilter | 2455 ns | 850 ns | 2895 ns |
| 100,000 | domainhash | **396 ns** | **588 ns** | **535 ns** |
| 500,000 | urlfilter | 3694 ns | 541 ns | 5112 ns |
| 500,000 | domainhash | **299 ns** | **587 ns** | **499 ns** |
| 1,000,000 | urlfilter | 4706 ns | 839 ns | 8310 ns |
| 1,000,000 | domainhash | **319 ns** | **543 ns** | **553 ns** |

Domainhash stays sub-microsecond and almost flat with N (`O(log N)`). Urlfilter
hit/parent cost grows with list size (shortcut index + rule objects).

## OpenStick estimates (4× Cortex-A53 @ ~1 GHz)

Not yet measured on-device. Scaled from the `GOMAXPROCS=4` desktop numbers with
a **~12×** single-thread factor (A53 @ 1 GHz vs this Zen desktop for
Go/integer work). Domainhash build is still mostly single-threaded today, so
extra A53 cores help little until parallel hash workers land.

| N | domainhash build (estimate) | domainhash warm lookup (estimate) | urlfilter RSS (order of magnitude) | domainhash steady file |
|---:|---:|---:|---:|---:|
| 100,000 | **~1.5–2.5 s** | ~5–8 µs | ~25 MiB | 0.5 MiB |
| 500,000 | **~6–9 s** | ~5–8 µs | ~90 MiB | 2.4 MiB |
| 1,000,000 | **~10–15 s** | ~5–10 µs | ~150–200+ MiB | 4.8 MiB |

Notes:

- Lookup estimates remain **≪ typical DNS upstream RTT** (milliseconds).
- Urlfilter at 500k–1M is why ~100 MiB routers OOM on rebuild (old+new engine
  ≈ 2× RSS, plus `GOGC` headroom → ~3× peak). Domainhash rebuild peak is
  bounded by `MemAvailable − margin` batching.
- Re-run on the OpenStick to replace estimates:

  ```bash
  GOMAXPROCS=4 go run ./scripts/bench-domainhash -n 100000,500000,1000000
  ```

## How to reproduce

```bash
# Comparative RSS + lookup table (used above)
go run ./scripts/bench-domainhash -n 100000,500000,1000000 -lookups 20000

# Package microbenchmarks
go test ./internal/filtering/domainhash/ -bench BenchmarkBuilder -benchtime=3x -benchmem
go test ./internal/filtering/domainhash/ -bench BenchmarkContainsDomain -benchmem
go test ./internal/filtering/ -run TestCompareSteadyStateMemory -v
```
