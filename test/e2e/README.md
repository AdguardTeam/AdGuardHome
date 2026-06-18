# AdGuard Home — black-box e2e suite

A curated [Playwright](https://playwright.dev/) + [testcontainers](https://node.testcontainers.org/)
suite that exercises a real AdGuard Home binary (built from this repository) as a
black box: DNS resolution, filtering, query log, settings and the web UI.

Every test maps 1:1 to a catalogue test case and is titled `<caseId> — <name>`
(e.g. `4085 — Upstream DNS servers: DoH`). The suite contains only high-signal
cases that fail on a real product regression — **84 tests** across three projects.

## Layout

```
test/e2e/
  runtime/        container fixtures + the Docker image definitions (runtime/image)
  shared/         reusable API / DNS / query-log helpers
  <domain>/       the spec files (dnsSettings, customRules, ui, install, …)
  playwright.config.ts
  scripts/build-images.sh
```

## Requirements

- A Docker daemon reachable from the host (the suite boots one AGH container per worker).
- Node 20+.
- For the `install` project only: a Linux Docker host (privileged systemd container).

## Run

```sh
# 1. Build the AGH binary from this checkout and bake the test images.
./test/e2e/scripts/build-images.sh

# 2. Install deps and run.
cd test/e2e
npm ci
npx playwright install --with-deps chromium
npx playwright test                       # all projects
npx playwright test --project=integration # DNS / filtering / settings (non-privileged)
npx playwright test --project=ui          # web UI (non-privileged, chromium)
npx playwright test --project=install     # service install/runtime (Linux + privileged)
```

`build-images.sh` runs `make build-release` for linux/amd64; pass
`E2E_TARBALL=/path/to/AdGuardHome_linux_amd64.tar.gz` to reuse an existing build,
or `SYSTEMD=0 CLIENT=0` to skip images you don't need.

## Projects

| Project | Tests | Container | Notes |
|---|---|---|---|
| `integration` | DNS, filtering, rewrites, blocklists/allowlists, clients, settings | master image (unprivileged) | mock upstreams via `host.docker.internal` |
| `ui` | dashboard, query log, clients | master image + chromium (unprivileged) | |
| `install` | service install/uninstall/start/stop/status, logs, `--web-addr` | systemd image (**privileged, Linux only**) | |
