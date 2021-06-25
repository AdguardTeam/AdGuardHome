#!/bin/sh

verbose="${VERBOSE:-0}"

# Set verbosity.
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



# Deferred Helpers

not_found_msg='
looks like a binary not found error.
make sure you have installed the linter binaries using:

	$ make go-tools
'
readonly not_found_msg

# TODO(a.garipov): Put it into a separate script and source it both here and in
# txt-lint.sh?
not_found() {
	if [ "$?" -eq '127' ]
	then
		# Code 127 is the exit status a shell uses when a command or
		# a file is not found, according to the Bash Hackers wiki.
		#
		# See https://wiki.bash-hackers.org/dict/terms/exit_status.
		echo "$not_found_msg" 1>&2
	fi
}
trap not_found EXIT



# Warnings

go_min_version='go1.16'
go_version_msg="
warning: your go version is different from the recommended minimal one (${go_min_version}).
if you have the version installed, please set the GO environment variable.
for example:

	export GO='${go_min_version}'
"
readonly go_min_version go_version_msg

case "$( "$GO" version )"
in
('go version'*"$go_min_version"*)
	# Go on.
	;;
(*)
	echo "$go_version_msg" 1>&2
	;;
esac



# Simple Analyzers

# blocklist_imports is a simple check against unwanted packages.  Package
# io/ioutil is soft-deprecated.  Packages errors and log are replaced by our own
# packages in the github.com/AdguardTeam/golibs module.
blocklist_imports() {
	git grep -F -e '"errors"' -e '"io/ioutil"' -e '"log"' -- '*.go' || exit 0;
}

# method_const is a simple check against the usage of some raw strings and
# numbers where one should use named constants.
method_const() {
	git grep -F -e '"GET"' -e '"POST"' -- '*.go' || exit 0;
}

# underscores is a simple check against Go filenames with underscores.
underscores() {
	git ls-files '*_*.go' | {
		grep -F\
		-e '_big.go'\
		-e '_bsd.go'\
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

# TODO(a.garipov): Add an analyser to look for `fallthrough`, `goto`, and `new`?



# Helpers

# exit_on_output exits with a nonzero exit code if there is anything in the
# command's combined output.
exit_on_output() (
	set +e

	if [ "$VERBOSE" -lt '2' ]
	then
		set +x
	fi

	cmd="$1"
	shift

	output="$( "$cmd" "$@" 2>&1 )"
	exitcode="$?"
	if [ "$exitcode" != '0' ]
	then
		echo "'$cmd' failed with code $exitcode"
	fi

	if [ "$output" != '' ]
	then
		if [ "$*" != '' ]
		then
			echo "combined output of '$cmd $*':"
		else
			echo "combined output of '$cmd':"
		fi

		echo "$output"

		if [ "$exitcode" -eq '0' ]
		then
			exitcode='1'
		fi
	fi

	return "$exitcode"
)



# Constants

go_files='./main.go ./internal/'
readonly go_files



# Checks

exit_on_output blocklist_imports

exit_on_output method_const

exit_on_output underscores

exit_on_output gofumpt --extra -l -s .

golint --set_exit_status ./...

"$GO" vet ./...

# Apply more lax standards to the code we haven't properly refactored yet.
gocyclo --over 17 ./internal/dhcpd/ ./internal/dnsforward/\
	./internal/filtering/ ./internal/home/ ./internal/querylog/\
	./internal/stats/ ./internal/updater/

# Apply stricter standards to new or vetted code
gocyclo --over 10 ./internal/aghio/ ./internal/aghnet/ ./internal/aghos/\
	./internal/aghstrings/ ./internal/aghtest/ ./internal/tools/\
	./internal/version/ ./main.go

gosec --quiet $go_files

ineffassign ./...

unparam ./...

git ls-files -- '*.go' '*.mod' '*.sh' 'Makefile' | xargs misspell --error

looppointer ./...

nilness ./...

exit_on_output shadow --strict ./...

# TODO(a.garipov): Enable --blank?
errcheck --asserts ./...

staticcheck ./...
