#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a remarkable change is made to this script.
#
# AdGuard-Project-Version: 2

verbose="${VERBOSE:-0}"
readonly verbose

set -e -f -u

if [ "$verbose" -gt '0' ]
then
	set -x
fi

# NOTE: Adjust for your project.
# markdownlint\
# 	./README.md\
# 	;

# TODO(e.burkov):  Lint markdown documents within this project.
