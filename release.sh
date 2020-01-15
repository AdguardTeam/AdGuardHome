#!/usr/bin/env bash

set -eE
set -o pipefail
set -x

channel=${1:-release}
baseUrl="https://static.adguard.com/adguardhome/$channel"
dst=dist
version=`git describe --abbrev=4 --dirty --always --tags`

f() {
	make cleanfast; CGO_DISABLED=1 make
	if [[ $GOOS == darwin ]]; then
		zip $dst/AdGuardHome_MacOS.zip AdGuardHome README.md LICENSE.txt
	elif [[ $GOOS == windows ]]; then
		zip $dst/AdGuardHome_Windows_"$GOARCH".zip AdGuardHome.exe README.md LICENSE.txt
	else
		rm -rf dist/AdguardHome
		mkdir -p dist/AdGuardHome
		cp -pv {AdGuardHome,LICENSE.txt,README.md} dist/AdGuardHome/
		pushd dist
		if [[ $GOARCH == arm ]] && [[ $GOARM != 6 ]]; then
			tar zcvf AdGuardHome_"$GOOS"_armv"$GOARM".tar.gz AdGuardHome/
		else
			tar zcvf AdGuardHome_"$GOOS"_"$GOARCH".tar.gz AdGuardHome/
		fi
		popd
		rm -rf dist/AdguardHome
	fi
}

# Clean dist and build
make clean
rm -rf $dst

# Prepare the dist folder
mkdir -p $dst

# Prepare releases
CHANNEL=$channel GOOS=darwin GOARCH=amd64 f
CHANNEL=$channel GOOS=linux GOARCH=amd64 f
CHANNEL=$channel GOOS=linux GOARCH=386 GO386=387 f
CHANNEL=$channel GOOS=linux GOARCH=arm GOARM=5 f
CHANNEL=$channel GOOS=linux GOARCH=arm GOARM=6 f
CHANNEL=$channel GOOS=linux GOARCH=arm64 GOARM=6 f
CHANNEL=$channel GOOS=windows GOARCH=amd64 f
CHANNEL=$channel GOOS=windows GOARCH=386 f
CHANNEL=$channel GOOS=linux GOARCH=mipsle GOMIPS=softfloat f
CHANNEL=$channel GOOS=linux GOARCH=mips GOMIPS=softfloat f
CHANNEL=$channel GOOS=freebsd GOARCH=amd64 f

# Variables for CI
echo "version=$version" > $dst/version.txt

# Prepare the version.json file
echo "{" >> $dst/version.json
echo "  \"version\": \"$version\"," >> $dst/version.json
echo "  \"announcement\": \"AdGuard Home $version is now available!\"," >> $dst/version.json
echo "  \"announcement_url\": \"https://github.com/AdguardTeam/AdGuardHome/releases\"," >> $dst/version.json
echo "  \"download_windows_amd64\": \"$baseUrl/AdGuardHome_Windows_amd64.zip\"," >> $dst/version.json
echo "  \"download_windows_386\": \"$baseUrl/AdGuardHome_Windows_386.zip\"," >> $dst/version.json
echo "  \"download_darwin_amd64\": \"$baseUrl/AdGuardHome_MacOS.zip\"," >> $dst/version.json
echo "  \"download_linux_amd64\": \"$baseUrl/AdGuardHome_linux_amd64.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_386\": \"$baseUrl/AdGuardHome_linux_386.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_arm\": \"$baseUrl/AdGuardHome_linux_arm.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_armv5\": \"$baseUrl/AdGuardHome_linux_armv5.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_arm64\": \"$baseUrl/AdGuardHome_linux_arm64.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_mips\": \"$baseUrl/AdGuardHome_linux_mips.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_mipsle\": \"$baseUrl/AdGuardHome_linux_mipsle.tar.gz\"," >> $dst/version.json
echo "  \"download_freebsd_amd64\": \"$baseUrl/AdGuardHome_freebsd_amd64.tar.gz\"," >> $dst/version.json
echo "  \"selfupdate_min_version\": \"v0.0\"" >> $dst/version.json
echo "}" >> $dst/version.json
