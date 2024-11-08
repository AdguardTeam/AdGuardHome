#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 6

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

count_flags='--count=2'
cover_flags='--coverprofile=./cover.out'
go="${GO:-go}"
shuffle_flags='--shuffle=on'
timeout_flags="${TIMEOUT_FLAGS:---timeout=90s}"
readonly count_flags cover_flags go shuffle_flags timeout_flags

go_test() {
	"$go" test \
		"$count_flags" \
		"$cover_flags" \
		"$race_flags" \
		"$shuffle_flags" \
		"$timeout_flags" \
		"$v_flags" \
		"$x_flags" \
		./...
}

test_reports_dir="${TEST_REPORTS_DIR:-}"
readonly test_reports_dir

if [ "$test_reports_dir" = '' ]; then
	go_test

	exit "$?"
fi

mkdir -p "$test_reports_dir"

# NOTE:  The pipe ignoring the exit code here is intentional, as go-junit-report
# will set the exit code to be saved.
go_test 2>&1 \
	| tee "${test_reports_dir}/test-output.txt"

# Don't fail on errors in exporting, because TEST_REPORTS_DIR is generally only
# not empty in CI, and so the exit code must be preserved to exit with it later.
set +e
go-junit-report \
	--in "${test_reports_dir}/test-output.txt" \
	--set-exit-code \
	>"${test_reports_dir}/test-report.xml"
printf '%s\n' "$?" \
	>"${test_reports_dir}/test-exit-code.txt"
