#!/usr/bin/env bash
set -euo pipefail
V="${1:?version}"
arch="$(dpkg --print-architecture)"
case "${arch}" in amd64|arm64) a="${arch}" ;; *) echo "unsupported: ${arch}" >&2; exit 1 ;; esac
curl -fsSL "https://github.com/ameshkov/dnslookup/releases/download/${V}/dnslookup-linux-${a}-${V}.tar.gz" \
  | tar -xz --strip-components=1 -C /usr/local/bin "linux-${a}/dnslookup"
chmod +x /usr/local/bin/dnslookup
