#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 3

verbose="${VERBOSE:-0}"
readonly verbose

set -e -f -u

if [ "$verbose" -gt '0' ]; then
	set -x
fi

# TODO(e.burkov):  Add README.md and possibly AGHTechDoc.md.
markdownlint \
	./CHANGELOG.md \
	./CONTRIBUTING.md \
	./HACKING.md \
	./SECURITY.md \
	./internal/next/changelog.md \
	./internal/dhcpd/*.md \
	./openapi/*.md \
	./scripts/*.md \
	;
