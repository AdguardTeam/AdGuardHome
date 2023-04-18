#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a remarkable change is made to this script.
#
# AdGuard-Project-Version: 3

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

# Simple analyzers

# trailing_newlines is a simple check that makes sure that all plain-text files
# have a trailing newlines to make sure that all tools work correctly with them.
trailing_newlines() {
	nl="$( printf "\n" )"
	readonly nl

	# NOTE: Adjust for your project.
	git ls-files\
		':!*.png'\
		':!*.tar.gz'\
		':!*.zip'\
		| while read -r f
		do
			if [ "$( tail -c -1 "$f" )" != "$nl" ]
			then
				printf '%s: must have a trailing newline\n' "$f"
			fi
		done
}

run_linter -e trailing_newlines

git ls-files -- '*.md' '*.txt' '*.yaml' '*.yml' 'client/src/__locales/en.json'\
	| xargs misspell --error\
	| sed -e 's/^/misspell: /'
