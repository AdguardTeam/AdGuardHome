#!/bin/sh

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '0' ]
then
	set -x
fi

set -e -f -u

dist_dir="${DIST_DIR:?please set DIST_DIR}"
sudo_cmd="${SUDO:-}"
readonly dist_dir sudo_cmd

$sudo_cmd rm -f\
	./AdGuardHome\
	./AdGuardHome.exe\
	./coverage.txt\
	;

$sudo_cmd rm -f -r\
	./bin/\
	./build/static/\
	./client/node_modules/\
	./data/\
	"./${dist_dir}/"\
	;
