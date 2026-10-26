#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 3

verbose="${VERBOSE:-0}"
readonly verbose

# Don't use -f, because we use globs in this script.
set -e -o 'pipefail' -u

if [ "$verbose" -gt '0' ]; then
	set -x
fi

markdownlint \
	./CHANGELOG.md \
	./CONTRIBUTING.md \
	./HACKING.md \
	./README.md \
	./SECURITY.md \
	./internal/next/changelog.md \
	./internal/dhcpd/*.md \
	./openapi/*.md \
	./scripts/*.md \
	;
