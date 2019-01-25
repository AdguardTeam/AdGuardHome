#!/usr/bin/env bash

set -eE
set -o pipefail
set -x

version=`git describe --abbrev=4 --dirty --always --tags`

f() {
	make cleanfast; CGO_DISABLED=1 make
	if [[ $GOOS == darwin ]]; then
	    zip dist/AdGuardHome_"$version"_MacOS.zip AdGuardHome README.md LICENSE.txt
	elif [[ $GOOS == windows ]]; then
	    zip dist/AdGuardHome_"$version"_Windows.zip AdGuardHome.exe README.md LICENSE.txt
	else
	    tar zcvf dist/AdGuardHome_"$version"_"$GOOS"_"$GOARCH".tar.gz AdGuardHome README.md LICENSE.txt
	fi
}

# Clean and rebuild both static and binary
make clean
make

# Prepare the dist folder
rm -rf dist
mkdir -p dist

# Prepare releases
GOOS=darwin GOARCH=amd64 f
GOOS=linux GOARCH=amd64 f
GOOS=linux GOARCH=386 f
GOOS=linux GOARCH=arm GOARM=6 f
GOOS=linux GOARCH=arm64 GOARM=6 f
GOOS=windows GOARCH=amd64 f