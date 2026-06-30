#!/usr/bin/env bash
set -euo pipefail
pkill -TERM -x AdGuardHome 2>/dev/null || true
for _ in $(seq 1 20); do pgrep -x AdGuardHome >/dev/null || break; sleep 0.1; done
pkill -KILL -x AdGuardHome 2>/dev/null || true
# Route logs to the container's stdout/stderr (PID 1) so they survive this
# `docker exec` shell exiting — mirrors agh-reset.sh.
setsid /opt/AdGuardHome/AdGuardHome --no-check-update \
  -c /opt/adguardhome/conf/AdGuardHome.yaml -w /opt/adguardhome/work \
  >/proc/1/fd/1 2>/proc/1/fd/2 </dev/null &
for _ in $(seq 1 40); do
  if curl -fsS --max-time 2 -o /dev/null "http://127.0.0.1:${AGH_WAIT_PORT:-3000}/"; then exit 0; fi
  sleep 0.25
done
echo "agh-apply: AdGuard did not become ready" >&2
exit 1
