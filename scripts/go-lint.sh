#!/bin/sh

# Verbosity levels:
#   0 = Don't print anything except for errors.
#   1 = Print commands, but not nested commands.
#   2 = Print everything.
test "${VERBOSE:=0}" -gt '0' && set -x

# Set $EXITONERROR to zero to see all errors.
test "${EXITONERROR:=1}" = '0' && set +e || set -e

# We don't need glob expansions and we want to see errors about unset
# variables.
set -f -u

not_found_msg='
looks like a binary not found error.
make sure you have installed the linter binaries using:

	$ make go-install-tools
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

# blocklist_imports is a simple check against unwanted packages.
# Currently it only looks for package log which is replaced by our own
# package github.com/AdguardTeam/golibs/log.
blocklist_imports() {
	git grep -F -e '"log"' -- '*.go' || exit 0;
}

# underscores is a simple check against Go filenames with underscores.
underscores() {
	git ls-files '*_*.go' | { grep -F -e '_darwin.go' \
		-e '_freebsd.go' -e '_linux.go' -e '_others.go' \
		-e '_test.go' -e '_unix.go' -e '_windows.go' \
		-v || exit 0; }
}

# exit_on_output exits with a nonzero exit code if there is anything in
# the command's combined output.
exit_on_output() {
	test "$VERBOSE" -lt '2' && set +x

	cmd="$1"
	shift

	exitcode='0'
	output="$("$cmd" "$@" 2>&1)"
	if [ "$output" != '' ]
	then
		if [ "$*" != '' ]
		then
			echo "combined output of '$cmd $@':"
		else
			echo "combined output of '$cmd':"
		fi

		echo "$output"

		exitcode='1'
	fi

	test "$VERBOSE" -gt '0' && set -x

	return "$exitcode"
}

exit_on_output blocklist_imports

exit_on_output underscores

exit_on_output gofumpt --extra -l -s .

golint --set_exit_status ./...

"$GO" vet ./...

gocyclo --over 20 .

gosec --quiet .

ineffassign .

unparam ./...

git ls-files -- '*.go' '*.md' '*.yaml' '*.yml' | xargs misspell --error

looppointer ./...

nilness ./...

# TODO(a.garipov): Enable shadow after fixing all of the shadowing.
# shadow --strict ./...

# TODO(a.garipov): Enable errcheck fully after handling all errors,
# including the deferred ones, properly.  Also, perhaps, enable --blank.
# errcheck ./...
exit_on_output sh -c '
	errcheck --asserts ./... |\
		{ grep -e "defer" -e "_test\.go:" -v || exit 0; }
'

staticcheck ./...
