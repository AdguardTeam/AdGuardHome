#!/bin/sh

verbose="${VERBOSE:-0}"

# Verbosity levels:
#   0 = Don't print anything except for errors.
#   1 = Print commands, but not nested commands.
#   2 = Print everything.
if [ "$verbose" -gt '0' ]
then
	set -x
fi

# Set $EXIT_ON_ERROR to zero to see all errors.
if [ "${EXIT_ON_ERROR:-1}" = '0' ]
then
	set +e
else
	set -e
fi

# We don't need glob expansions and we want to see errors about unset
# variables.
set -f -u



# Deferred Helpers

not_found_msg='
looks like a binary not found error.
make sure you have installed the linter binaries using:

	$ make go-tools
'

not_found() {
	if [ "$?" = '127' ]
	then
		# Code 127 is the exit status a shell uses when
		# a command or a file is not found, according to the
		# Bash Hackers wiki.
		#
		# See https://wiki.bash-hackers.org/dict/terms/exit_status.
		echo "$not_found_msg" 1>&2
	fi
}
trap not_found EXIT



# Simple Analyzers

# blocklist_imports is a simple check against unwanted packages.
# Currently it only looks for package log which is replaced by our own
# package github.com/AdguardTeam/golibs/log.
blocklist_imports() {
	git grep -F -e '"log"' -- '*.go' || exit 0;
}

# method_const is a simple check against the usage of some raw strings
# and numbers where one should use named constants.
method_const() {
	git grep -F -e '"GET"' -e '"POST"' -- '*.go' || exit 0;
}

# underscores is a simple check against Go filenames with underscores.
underscores() {
	git ls-files '*_*.go' | {
		grep -F\
		-e '_big.go'\
		-e '_darwin.go'\
		-e '_freebsd.go'\
		-e '_linux.go'\
		-e '_little.go'\
		-e '_others.go'\
		-e '_test.go'\
		-e '_unix.go'\
		-e '_windows.go' \
		-v\
		|| exit 0
	}
}



# Helpers

# exit_on_output exits with a nonzero exit code if there is anything in
# the command's combined output.
exit_on_output() (
	set +e

	if [ "$VERBOSE" -lt '2' ]
	then
		set +x
	fi

	cmd="$1"
	shift

	output="$("$cmd" "$@" 2>&1)"
	exitcode="$?"
	if [ "$exitcode" != '0' ]
	then
		echo "'$cmd' failed with code $exitcode"
	fi

	if [ "$output" != '' ]
	then
		if [ "$*" != '' ]
		then
			echo "combined output of '$cmd $@':"
		else
			echo "combined output of '$cmd':"
		fi

		echo "$output"

		if [ "$exitcode" = '0' ]
		then
			exitcode='1'
		fi
	fi

	return "$exitcode"
)



# Constants

readonly go_files='./main.go ./tools.go ./internal/'



# Checks

exit_on_output blocklist_imports

exit_on_output method_const

exit_on_output underscores

exit_on_output gofumpt --extra -l -s .

golint --set_exit_status ./...

"$GO" vet ./...

# Here and below, don't use quotes to get word splitting.
gocyclo --over 17 $go_files

gosec --quiet $go_files

ineffassign ./...

unparam ./...

git ls-files -- '*.go' '*.md' '*.mod' '*.sh' '*.yaml' '*.yml'\
	'Makefile'\
	| xargs misspell --error

looppointer ./...

nilness ./...

exit_on_output shadow --strict ./...

# TODO(a.garipov): Enable errcheck fully after handling all errors,
# including the deferred and generated ones, properly.  Also, perhaps,
# enable --blank.
#
# errcheck ./...
exit_on_output sh -c '
	errcheck --asserts --ignoregenerated ./... |\
		{ grep -e "defer" -v || exit 0; }
'

staticcheck ./...
