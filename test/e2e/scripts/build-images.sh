#!/usr/bin/env bash
#
# Build the Docker images the e2e suite needs, using a Linux/amd64 AdGuard Home
# binary built from THIS repository (not a GitHub release).
#
# Run from the repository root:  ./test/e2e/scripts/build-images.sh
#
# Env:
#   E2E_TARBALL   path to a prebuilt AdGuardHome_linux_amd64.tar.gz (skips `make`).
#   SYSTEMD=0     skip the privileged systemd image (install/* tests won't run).
#   CLIENT=0      skip the client image (only used by a couple of network tests).
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
e2e_dir="${repo_root}/test/e2e"
archive="${e2e_dir}/AdGuardHome_linux_amd64.tar.gz"

if [ -n "${E2E_TARBALL:-}" ]; then
  echo "Using provided tarball: ${E2E_TARBALL}"
  cp "${E2E_TARBALL}" "${archive}"
else
  echo "Building AdGuard Home (linux/amd64) from this checkout via make build-release..."
  ( cd "${repo_root}" && make OS=linux ARCH=amd64 SIGN=0 CHANNEL=development VERSION=v0.0.0-e2e build-release )
  cp "${repo_root}/dist/AdGuardHome_linux_amd64.tar.gz" "${archive}"
fi

# The binary is linux/amd64, so pin image builds to that platform (matters on
# arm64 hosts where the default would otherwise build an arm64 image).
echo "Building adguardhome-test:local (master image, binary embedded)..."
docker build --platform linux/amd64 -f "${e2e_dir}/runtime/image/Dockerfile" -t adguardhome-test:local "${e2e_dir}"

if [ "${SYSTEMD:-1}" != "0" ]; then
  echo "Building adguardhome-systemd:local (privileged host for install/* tests)..."
  docker build --platform linux/amd64 -f "${e2e_dir}/runtime/image/Dockerfile.systemd" -t adguardhome-systemd:local "${e2e_dir}"
fi

if [ "${CLIENT:-1}" != "0" ]; then
  echo "Building adguardhome-client:local..."
  docker build --platform linux/amd64 -f "${e2e_dir}/runtime/image/Dockerfile.client" -t adguardhome-client:local "${e2e_dir}"
fi

echo "Done. Images:"
docker images | grep -E 'adguardhome-(test|systemd|client)' || true
