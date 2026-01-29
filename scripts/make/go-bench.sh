#!/bin/sh

verbose="${VERBOSE:-0}"
readonly verbose

# Verbosity levels:
#   0 = Don't print anything except for errors.
#   1 = Print commands, but not nested commands.
#   2 = Print everything.
if [ "$verbose" -gt '1' ]; then
	set -x
	v_flags='-v=1'
	x_flags='-x=1'
elif [ "$verbose" -gt '0' ]; then
	set -x
	v_flags='-v=1'
	x_flags='-x=0'
else
	set +x
	v_flags='-v=0'
	x_flags='-x=0'
fi
readonly v_flags x_flags

set -e -f -u

if [ "${RACE:-1}" -eq '0' ]; then
	race_flags='--race=0'
else
	race_flags='--race=1'
fi
readonly race_flags

benchtime_flags="${BENCHTIME_FLAGS:---benchtime=1x}"
count_flags='--count=2'
go="${GO:-go}"
shuffle_flags='--shuffle=on'
timeout_flags="${TIMEOUT_FLAGS:---timeout=30s}"
readonly benchtime_flags count_flags go shuffle_flags timeout_flags

env \
	GOMAXPROCS="${GOMAXPROCS:-2}" \
	"$go" test \
	"$benchtime_flags" \
	"$count_flags" \
	"$race_flags" \
	"$shuffle_flags" \
	"$timeout_flags" \
	"$v_flags" \
	"$x_flags" \
	--bench='.' \
	--benchmem \
	--run='^$' \
	work \
	;
