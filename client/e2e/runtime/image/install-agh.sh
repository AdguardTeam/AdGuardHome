#!/usr/bin/env bash
set -euo pipefail
VERSION="${1:?version required}"
case "${VERSION}" in
  *[!A-Za-z0-9._-]*) echo "Invalid version: ${VERSION}" >&2; exit 1 ;;
esac
if [ -f /tmp/AdGuardHome_linux_amd64.tar.gz ]; then
  tar -xz -f /tmp/AdGuardHome_linux_amd64.tar.gz -C /opt
else
  curl -fsSL --retry 3 --retry-connrefused --connect-timeout 10 --max-time 300 \
    "https://github.com/AdguardTeam/AdGuardHome/releases/download/${VERSION}/AdGuardHome_linux_amd64.tar.gz" \
    | tar -xz -C /opt
fi
test -x /opt/AdGuardHome/AdGuardHome
