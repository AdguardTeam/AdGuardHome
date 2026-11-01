# AdGuard Home Next-Gen

Fork of [AdGuardHome](https://github.com/AdguardTeam/AdGuardHome) focused on
**memory-efficient DNS blocklists** for OpenStick-class and consumer-router
hardware (≈100–512 MiB RAM), targeting **500k–1M** threat-intelligence domains.

Base tag: `v1.0.0-b.1`.

## Observed OpenStick behaviour (current AGH, small lists)

Podman RSS for AGH on the OpenStick sawtooths roughly **8–40 MiB** on a ~1–2 h
cadence (rapid rise, rapid drop; other containers flat). That is **not** the
million-entry OOM failure mode yet — it is normal Go process breathing:

| Layer | What it is |
|---|---|
| Floor (~8–10 MiB) | Live set: binary + small rule engine + idle caches |
| Rise | DNS/query caches fill; Go heap grows under `GOGC=100` until ~2× live |
| Cliff | GC + background scavenger return pages → cgroup/Podman RSS drops |

Filter-list rebuilds (`filters_update_interval`, default **24 h**) and
`debug.FreeOSMemory()` after engine init are a *different*, rarer spike. They
do not explain a 1–2 h period. Default DNS cache alone is configured at
**4 MiB** (`dns.cache_size`); safebrowsing/parental/safesearch caches add up to
~1 MiB each when enabled.

Cheap knobs for this chart (no fork required):

1. `GOMEMLIMIT=48MiB` (or similar) — hard ceiling; preferred over raw `GOGC`.
2. `GOGC=50` — earlier collection, lower peak, more CPU.
3. Shrink `dns.cache_size`; disable unused safebrowsing/parental if not needed.

The Next-Gen work below targets the **other** problem: loading 500k–1M threat
lists without a 3× urlfilter rebuild peak.

## Problem

Upstream AGH builds a full `urlfilter.DNSEngine` for every filter list. Each
`NetworkRule` is ~296 bytes plus shortcut-index overhead. A 2.1M-rule HaGeZi
Threat Intelligence rebuild peaks around **800+ MiB** because:

1. The previous engine must stay alive while the new one is built (~2× live).
2. Default `GOGC=100` lets the heap grow another ~2× before collection (~3× peak).

Upstream PR [#8498](https://github.com/AdguardTeam/AdGuardHome/pull/8498) reduces
that peak (skip unchanged rebuilds, serialize rebuilds, tighten GOGC, collect
before/after) but **steady-state still stores every rule as a rich object**. That
cannot fit 1M threat domains on a ~100 MiB router budget.

## Approach: dual-path filtering

Inspired by [esp32-c3-adblock](https://github.com/M-Abozaid/esp32-c3-adblock):
store simple blocked domains as **sorted truncated FNV-1a hashes on disk** and
binary-search them. Keep `urlfilter` only for rules a hash table cannot express.

```
filter update
  ├─ stream lists
  ├─ simple hosts / ||domain^  ──▶ batch-hash ──▶ external merge-sort ──▶ blocklist.bin
  └─ complex Adblock rules     ──▶ compact urlfilter DNSEngine

DNS query
  ├─ domainhash.Index (mmap + binary search, parent suffixes)
  │     hit  → block
  └─ urlfilter (allowlist + complex block rules)
```

### mmap / I/O behaviour

The blocklist file is **`mmap`'d**, not `read()` per query. Linux faults pages
into the **page cache** on first touch; afterwards warm lookups are RAM-speed
binary searches over that cache. The kernel may reclaim clean pages under
memory pressure and fault them back later. A 1M×5-byte index is **~4.8 MiB** —
small enough to stay resident on OpenStick-class devices when RAM allows.

Full numbers: **[BENCHMARKS.md](BENCHMARKS.md)**.

### Memory math (1M domains)

| Representation | Steady-state |
|---|---|
| urlfilter `NetworkRule` (~296 B + index) | hundreds of MiB |
| 40-bit hash blob | **~5 MiB** on disk / mmap |
| 64-bit hash blob | **~8 MiB** |

40-bit birthday collisions ≈ 1 at ~537k domains (acceptable over-block for ads;
use 64-bit for stricter threat-intel).

### Rebuild (user requirements)

1. **Batch size** = `MemAvailable − margin` (Linux `/proc/meminfo`), not a fixed
   constant. Default margin 32 MiB on OpenStick, 16 MiB on routers.
2. Hash runs are **written under the data dir** (prefer configured `data/` over
   `/tmp` on devices where `/tmp` is tmpfs and counts against RAM).
3. Atomic rename of the final blob; old mmap stays valid until swap + `Close`.
4. Port [#8498](https://github.com/AdguardTeam/AdGuardHome/pull/8498) mitigations
   for whatever remains on the urlfilter path.

## Hashing / build speed (OpenStick ≈ 4× A53 @ 1 GHz)

**Verdict: raw FNV is cheap; end-to-end build is sort/merge + I/O.**

Desktop (`GOMAXPROCS=4`): domainhash build ≈ **0.15 s / 0.6 s / 0.9 s** for
100k / 500k / 1M. OpenStick estimate (×~12 single-thread factor):
**~2 s / ~7–9 s / ~10–15 s**. Warm lookups stay ~µs — negligible vs DNS RTT.

Upstream urlfilter rebuild cost is dominated by parsing into `NetworkRule`,
building `ShortcutIndex`, and GC over millions of heap objects — and its
**RSS stays ~25 / 90 / 155 MiB** for those sizes versus a **~0.5 / 2.4 / 4.8 MiB**
mmap file for domainhash.

Worth doing next (cheap):

- parallel hash workers across cores while filling a batch
- parallel run merge when run count is high

Not worth early investment:

- hand-written NEON FNV kernels — build time is not FNV-bound today
- switching hash family for speed alone — prefer **64-bit width** if collision
  policy matters

See **[BENCHMARKS.md](BENCHMARKS.md)** for the full tables.

## Feature flag

`dns.domain_hash_index` (working name):

- `enabled` (default true on Next-Gen builds)
- `hash_bytes` (5 or 8)
- `mem_margin_mb`
- `data_dir` (defaults beside filter files)

When enabled, simple domain rules are diverted out of `urlfilter` during engine
build. Complex rules and allowlists keep the existing path.

## Non-goals (v1)

- Full Adblock syntax in the hash index (`$client`, regex, `/path`, cosmetics)
- Replacing SafeBrowsing `hashprefix` (different protocol)
- Windows file-locking parity beyond "read into builder then close"

## Verification targets

- Build + lookup 1M domains on a machine with `GOMEMLIMIT=96MiB` without OOM
- Rebuild peak ≪ 2× steady-state of the hash blob + small complex engine
- OpenStick deploy: HaGeZi TI / similar 500k–1M list usable with DNS latency
  dominated by upstream RTT, not index search (`O(log n)` comparisons)

## OpenStick deploy

Live device: `192.168.1.174` (see homelab `hosts-openstick.ini`), aarch64,
Podman quadlet `adguardhome.service` (`MemoryMax=150M`).

```bash
# CI builds linux/arm64 artifacts (.github/workflows/next-gen.yml)
# Local build + deploy:
make CLIENT_DIR=client_v2 GOTOOLCHAIN=local js-deps js-build
mkdir -p dist/AdGuardHome_linux_arm64/AdGuardHome
env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 CHANNEL=development \
  VERSION=v1.0.0-b.1-nextgen-$(git rev-parse --short HEAD) \
  OUT=dist/AdGuardHome_linux_arm64/AdGuardHome/AdGuardHome \
  sh ./scripts/make/go-build.sh
./scripts/deploy-openstick.sh
```

The deploy script bind-mounts `AdGuardHome.nextgen` over the stock
`adguard/adguardhome:v1.0.0-b.1` image binary and restarts the quadlet.

### First live result (2026-11-01)

| Check | Result |
|---|---|
| Version | `v1.0.0-b.1-nextgen-ebc23755` |
| domainhash (baseline lists) | **185,994** hashes, 0 collisions |
| After +Normal/Pro/Ultimate/TIF | **2,184,149** hashes / 2.71M domains diverted, **3** collisions |
| On-disk blob | `blocklist.bin` **0.89 MiB → 11 MiB** |
| Complex leftovers | 7 filter files still in urlfilter |
| Service memory | steady **~38 → ~47 MiB**; rebuild peak **~122–139 MiB** (cap 250 MiB) |
| DNS | `doubleclick.net` → `0.0.0.0` on `192.168.1.174:53` |

Scale table: [BENCHMARKS.md](BENCHMARKS.md#openstick-live-scale-up-2026-11-01).

Optional knobs for the 8–40 MiB sawtooth on small lists:

```
GOMEMLIMIT=48MiB
GOGC=50
```

```yaml
dns:
  cache_size: 1048576   # 1 MiB
filtering:
  domain_hash_enabled: true
  domain_hash_bytes: 5  # 40-bit; use 8 for stricter TI collision policy
```
