#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a remarkable change is made to this script.
#
# AdGuard-Project-Version: 3

verbose="${VERBOSE:-0}"
readonly verbose

set -e -f -u

if [ "$verbose" -gt '0' ]; then
	set -x
fi

# TODO(e.burkov):  Lint allmarkdown documents within this project.
markdownlint \
	./CHANGELOG.md \
	./openapi/CHANGELOG.md \
	;
