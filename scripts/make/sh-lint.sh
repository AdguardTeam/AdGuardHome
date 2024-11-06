#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a remarkable change is made to this script.
#
# AdGuard-Project-Version: 2

verbose="${VERBOSE:-0}"
readonly verbose

# Don't use -f, because we use globs in this script.
set -e -u

if [ "$verbose" -gt '0' ]
then
	set -x
fi

# NOTE: Adjust for your project.
#
# TODO(e.burkov):  Add build-docker.sh, build-release.sh and install.sh.
shellcheck -e 'SC2250' -f 'gcc' -o 'all' -x --\
	./scripts/hooks/*\
	./scripts/snap/*\
	./scripts/make/*\
	;
