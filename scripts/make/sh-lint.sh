#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a remarkable change is made to this script.
#
# AdGuard-Project-Version: 3

verbose="${VERBOSE:-0}"
readonly verbose

# Don't use -f, because we use globs in this script.
set -e -u

if [ "$verbose" -gt '0' ]; then
	set -x
fi

# Source the common helpers, including not_found and run_linter.
. ./scripts/make/helper.sh

run_linter -e shfmt --binary-next-line -d -p -s \
	./scripts/hooks/* \
	./scripts/install.sh \
	./scripts/make/*.sh \
	./scripts/snap/*.sh \
	./snap/local/*.sh \
	;

shellcheck -e 'SC2250' -e 'SC2310' -f 'gcc' -o 'all' -x -- \
	./scripts/hooks/* \
	./scripts/install.sh \
	./scripts/make/*.sh \
	./scripts/snap/*.sh \
	./snap/local/*.sh \
	;
