#!/usr/bin/env bash
set -euo pipefail
pkill -TERM -x AdGuardHome 2>/dev/null || true
for _ in $(seq 1 20); do pgrep -x AdGuardHome >/dev/null || break; sleep 0.1; done
pkill -KILL -x AdGuardHome 2>/dev/null || true
rm -rf /opt/adguardhome/work/data
cp -f /pristine/AdGuardHome.yaml /opt/adguardhome/conf/AdGuardHome.yaml
# setsid + </dev/null detaches AGH into its own session so it survives this
# `docker exec` shell exiting; logs are routed to the container's stdout (PID 1).
setsid /opt/AdGuardHome/AdGuardHome --no-check-update \
  -c /opt/adguardhome/conf/AdGuardHome.yaml -w /opt/adguardhome/work \
  >/proc/1/fd/1 2>/proc/1/fd/2 </dev/null &
# Wait for the web API to answer.
for _ in $(seq 1 40); do
  if curl -fsS --max-time 5 -o /dev/null http://127.0.0.1:3000/; then exit 0; fi
  sleep 0.25
done
echo "agh-reset: AdGuard did not become ready" >&2
exit 1
