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
	    zip dist/AdGuardHome_"$version"_Windows_"$GOARCH".zip AdGuardHome.exe README.md LICENSE.txt
	else
	    rm -rf dist/AdguardHome
	    mkdir -p dist/AdGuardHome
	    cp -pv {AdGuardHome,LICENSE.txt,README.md} dist/AdGuardHome/
	    pushd dist
	    tar zcvf AdGuardHome_"$version"_"$GOOS"_"$GOARCH".tar.gz AdGuardHome/{AdGuardHome,LICENSE.txt,README.md}
	    popd
	    rm -rf dist/AdguardHome
	fi
}

# Clean dist and build
make clean
rm -rf dist

# Prepare the dist folder
mkdir -p dist

# Prepare releases
GOOS=darwin GOARCH=amd64 f
GOOS=linux GOARCH=amd64 f
GOOS=linux GOARCH=386 f
GOOS=linux GOARCH=arm GOARM=6 f
GOOS=linux GOARCH=arm64 GOARM=6 f
GOOS=windows GOARCH=amd64 f
GOOS=windows GOARCH=386 f
GOOS=linux GOARCH=mipsle GOMIPS=softfloat f
GOOS=linux GOARCH=mips GOMIPS=softfloat f