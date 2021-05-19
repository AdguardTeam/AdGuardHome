#!/bin/sh

verbose="${VERBOSE:-0}"
readonly verbose

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
readonly v_flags x_flags

set -e -f -u

if [ "${RACE:-1}" -eq '0' ]
then
	race_flags=''
else
	race_flags='--race'
fi
readonly race_flags

go="${GO:-go}"
timeout_flags="${TIMEOUT_FLAGS:---timeout 30s}"
cover_flags='--coverprofile ./coverage.txt'
count_flags='--count 1'
readonly go timeout_flags cover_flags count_flags

# Don't use quotes with flag variables because we want an empty space if those
# aren't set.
"$go" test $count_flags $cover_flags $race_flags $timeout_flags $x_flags $v_flags ./...
