#!/bin/sh

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]
then
	env
	set -x
	x_flags='-x'
elif [ "$verbose" -gt '0' ]
then
	set -x
	x_flags=''
else
	set +x
	x_flags=''
fi
readonly x_flags

set -e -f -u

go="${GO:-go}"
readonly go

# Don't use quotes with flag variables because we want an empty space if those
# aren't set.
"$go" mod download $x_flags
