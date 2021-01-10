#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '0' ]
then
	set -x
fi

set -e -f -u

dist_dir="$DIST_DIR"
go="${GO:-go}"

# Set the GOPATH explicitly in case make clean is called from under sudo
# after a Docker build.
env PATH="$("$go" env GOPATH)/bin":"$PATH" packr clean

rm -f\
	./AdGuardHome\
	./AdGuardHome.exe\
	./coverage.txt\
	;

rm -f -r\
	./bin/\
	./build/\
	./build2/\
	./client/node_modules/\
	./client2/node_modules/\
	./data/\
	"./${dist_dir}/"\
	;
