#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 2

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]; then
	env
	set -x
	x_flags='-x=1'
elif [ "$verbose" -gt '0' ]; then
	set -x
	x_flags='-x=0'
else
	set +x
	x_flags='-x=0'
fi
readonly x_flags

set -e -f -u

go="${GO:-go}"
readonly go

"$go" mod download "$x_flags"
