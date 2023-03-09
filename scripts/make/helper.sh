#!/bin/sh

# Common script helpers
#
# This file contains common script helpers.  It should be sourced in scripts
# right after the initial environment processing.

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a remarkable change is made to this script.
#
# AdGuard-Project-Version: 2



# Deferred helpers

not_found_msg='
looks like a binary not found error.
make sure you have installed the linter binaries using:

	$ make go-tools
'
readonly not_found_msg

not_found() {
	if [ "$?" -eq '127' ]
	then
		# Code 127 is the exit status a shell uses when a command or a file is
		# not found, according to the Bash Hackers wiki.
		#
		# See https://wiki.bash-hackers.org/dict/terms/exit_status.
		echo "$not_found_msg" 1>&2
	fi
}
trap not_found EXIT



# Helpers

# run_linter runs the given linter with two additions:
#
# 1.  If the first argument is "-e", run_linter exits with a nonzero exit code
#     if there is anything in the command's combined output.
#
# 2.  In any case, run_linter adds the program's name to its combined output.
run_linter() (
	set +e

	if [ "$VERBOSE" -lt '2' ]
	then
		set +x
	fi

	cmd="${1:?run_linter: provide a command}"
	shift

	exit_on_output='0'
	if [ "$cmd" = '-e' ]
	then
		exit_on_output='1'
		cmd="${1:?run_linter: provide a command}"
		shift
	fi

	readonly cmd

	output="$( "$cmd" "$@" )"
	exitcode="$?"

	readonly output

	if [ "$output" != '' ]
	then
		echo "$output" | sed -e "s/^/${cmd}: /"

		if [ "$exitcode" -eq '0' ] && [ "$exit_on_output" -eq '1' ]
		then
			exitcode='1'
		fi
	fi

	return "$exitcode"
)
