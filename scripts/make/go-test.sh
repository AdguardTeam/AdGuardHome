#!/bin/sh

verbose="${VERBOSE:-0}"

# Verbosity levels:
#   0 = Don't print anything except for errors.
#   1 = Print commands, but not nested commands.
#   2 = Print everything.
if [ "$verbose" -gt '1' ]
then
	set -x
	v_flags='-v'
	x_flags='-x'
elif [ "$verbose" -gt '0' ]
then
	set -x
	v_flags='-v'
	x_flags=''
else
	set +x
	v_flags=''
	x_flags=''
fi

set -e -f -u

race="${RACE:-1}"
if [ "$race" = '0' ]
then
	race_flags=''
else
	race_flags='--race'
fi

readonly go="${GO:-go}"
readonly timeout_flags="${TIMEOUT_FLAGS:---timeout 30s}"
readonly cover_flags='--coverprofile ./coverage.txt'
readonly count_flags='--count 1'

# Don't use quotes with flag variables because we want an empty space if
# those aren't set.
"$go" test $count_flags $cover_flags $race_flags $timeout_flags\
	$x_flags $v_flags ./...
