#!/usr/bin/env bash

set -eE
set -o pipefail
set -x

version=`git describe --abbrev=4 --dirty --always --tags`

f() {
	make cleanfast; CGO_DISABLED=1 make
	if [[ $GOOS == darwin ]]; then
	    rm -f ../AdGuardHome_"$version"_MacOS.zip
	    zip ../AdGuardHome_"$version"_MacOS.zip AdGuardHome README.md LICENSE.TXT
	elif [[ $GOOS == windows ]]; then
	    rm -f ../AdGuardHome_"$version"_Windows.zip
	    zip ../AdGuardHome_"$version"_Windows.zip AdGuardHome.exe README.md LICENSE.TXT
	else
	    pushd ..
	    tar zcvf AdGuardHome_"$version"_"$GOOS"_"$GOARCH".tar.gz AdGuardHome/{AdGuardHome,LICENSE.TXT,README.md}
	    popd
	fi
}

#make clean
#make
GOOS=darwin GOARCH=amd64 f
GOOS=linux GOARCH=amd64 f
GOOS=linux GOARCH=386 f
GOOS=linux GOARCH=arm GOARM=6 f
GOOS=linux GOARCH=arm64 GOARM=6 f
GOOS=windows GOARCH=amd64 f
