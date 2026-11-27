#!/usr/bin/env bash
# Đóng gói module thành zip cho một kiến trúc / Package the module zip for one architecture.
#
#   ./pack.sh [arm64|armv7|amd64]     (mặc định / default: arm64)
#
# Biến môi trường / Environment:
#   AGH_VERSION  Tag AdGuardHome cần dùng, ví dụ v0.107.79 (mặc định: bản mới nhất)
#                AdGuardHome release tag to bundle, e.g. v0.107.79 (default: latest)
#   AGH_BINARY   Đường dẫn binary AdGuardHome có sẵn (ví dụ bản tự build từ fork này);
#                bỏ qua bước tải về.  Path to a prebuilt AdGuardHome binary; skips the download.
#
# Kết quả / Output: dist/AdGuardHome-for-Superuser_<version>_<arch>.zip
set -euo pipefail
cd "$(dirname "$0")"

arch="${1:-arm64}"
case "$arch" in
arm64) asset="linux_arm64" magisk_arch="arm64" ;;
armv7) asset="linux_armv7" magisk_arch="arm" ;;
amd64) asset="linux_amd64" magisk_arch="x64" ;;
*)
  echo "Usage: $0 [arm64|armv7|amd64]" >&2
  exit 1
  ;;
esac

for tool in zip tar; do
  command -v "$tool" >/dev/null || { echo "Missing tool: $tool" >&2; exit 1; }
done

version="$(sed -n 's/^version=//p' module/module.prop)"
[ -n "$version" ] || { echo "Cannot read version from module/module.prop" >&2; exit 1; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

if [ -n "${AGH_BINARY:-}" ]; then
  [ -f "$AGH_BINARY" ] || { echo "AGH_BINARY not found: $AGH_BINARY" >&2; exit 1; }
  binary="$AGH_BINARY"
  echo ">> Using prebuilt binary: $binary"
else
  agh_version="${AGH_VERSION:-latest}"
  if [ "$agh_version" = latest ]; then
    url="https://github.com/AdguardTeam/AdGuardHome/releases/latest/download/AdGuardHome_${asset}.tar.gz"
  else
    url="https://github.com/AdguardTeam/AdGuardHome/releases/download/${agh_version}/AdGuardHome_${asset}.tar.gz"
  fi
  mkdir -p cache
  tarball="cache/AdGuardHome_${agh_version}_${asset}.tar.gz"
  if [ ! -s "$tarball" ]; then
    echo ">> Downloading $url"
    curl -fL --retry 3 -o "$tarball.part" "$url"
    mv "$tarball.part" "$tarball"
  fi
  # Tarball chỉ gồm binary và vài tệp văn bản; giải nén hết rồi lấy đúng binary
  mkdir -p "$work/agh"
  tar -xzf "$tarball" -C "$work/agh"
  binary="$(find "$work/agh" -type f -name AdGuardHome | head -n 1)"
  [ -n "$binary" ] || { echo "AdGuardHome binary not found in $tarball" >&2; exit 1; }
fi

stage="$work/stage"
mkdir -p "$stage/bin"
cp -a module/. "$stage/"
cp "$binary" "$stage/bin/AdGuardHome"
chmod 0755 "$stage/bin/AdGuardHome"
printf '%s' "$magisk_arch" >"$stage/bin/ARCH"

mkdir -p dist
out="$PWD/dist/AdGuardHome-for-Superuser_${version}_${arch}.zip"
rm -f "$out"
(cd "$stage" && zip -r9 -X -q "$out" .)
echo ">> Built $out ($(du -h "$out" | cut -f1))"
