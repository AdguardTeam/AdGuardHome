#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '0' ]; then
	set -x
fi

set -e -f -u

channel="${CHANNEL:?please set CHANNEL}"
readonly channel

printf '%s %s\n' \
	'386' 'i386' \
	'amd64' 'amd64' \
	'armv7' 'armhf' \
	'arm64' 'arm64' \
	| while read -r arch snap_arch; do
		release_url="https://static.adtidy.org/adguardhome/${channel}/AdGuardHome_linux_${arch}.tar.gz"
		output="./AdGuardHome_linux_${arch}.tar.gz"

		curl -o "$output" -v "$release_url"
		tar -f "$output" -v -x -z
		cp ./AdGuardHome/AdGuardHome "./AdGuardHome_${snap_arch}"
		rm -f -r "$output" ./AdGuardHome
	done
