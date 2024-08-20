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
	./scripts/make/clean.sh\
	./scripts/make/go-bench.sh\
	./scripts/make/go-build.sh\
	./scripts/make/go-deps.sh\
	./scripts/make/go-fuzz.sh\
	./scripts/make/go-lint.sh\
	./scripts/make/go-test.sh\
	./scripts/make/go-tools.sh\
	./scripts/make/go-upd-tools.sh\
	./scripts/make/helper.sh\
	./scripts/make/md-lint.sh\
	./scripts/make/sh-lint.sh\
	./scripts/make/txt-lint.sh\
	./scripts/make/version.sh\
	;
