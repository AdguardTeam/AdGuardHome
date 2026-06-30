#!/usr/bin/env bash
set -euo pipefail
# Seed the working config from the pristine copy on first boot.
cp -f /pristine/AdGuardHome.yaml /opt/adguardhome/conf/AdGuardHome.yaml
# Run AGH as a child (not PID 1) so agh-reset.sh can restart it in-place
# without killing the container.
# stdin from /dev/null; stdout/stderr are inherited and surface in `docker logs`.
setsid /opt/AdGuardHome/AdGuardHome --no-check-update \
  -c /opt/adguardhome/conf/AdGuardHome.yaml -w /opt/adguardhome/work </dev/null &
# Keep the container alive as PID 1.
exec tail -f /dev/null
