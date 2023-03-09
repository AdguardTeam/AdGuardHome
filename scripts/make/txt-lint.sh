#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a remarkable change is made to this script.
#
# AdGuard-Project-Version: 2

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '0' ]
then
	set -x
fi

# Set $EXIT_ON_ERROR to zero to see all errors.
if [ "${EXIT_ON_ERROR:-1}" -eq '0' ]
then
	set +e
else
	set -e
fi

# We don't need glob expansions and we want to see errors about unset variables.
set -f -u

# Source the common helpers, including not_found.
. ./scripts/make/helper.sh

git ls-files -- '*.md' '*.yaml' '*.yml' 'client/src/__locales/en.json'\
	| xargs misspell --error\
	| sed -e 's/^/misspell: /'
