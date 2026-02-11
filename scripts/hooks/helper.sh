#!/bin/sh

# Common git hook script helpers
#
# This file contains common script helpers for git hooks.  It should be sourced
# in scripts right after the initial environment processing.

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 1

# Only show interactive prompts if there a terminal is attached to stdout.
# While this technically doesn't guarantee that reading from /dev/tty works,
# this should work reasonably well on all of our supported development systems
# and in most terminal emulators.
is_tty='0'
if [ -t '1' ]; then
	is_tty='1'
fi
readonly is_tty

# Helpers

# prompt is a helper that prompts the user for interactive input if that can be
# done.  If there is no terminal attached, it sleeps for two seconds, giving the
# programmer some time to react, and returns with a zero exit code.
prompt() {
	if [ "$is_tty" -eq '0' ]; then
		sleep 2

		return 0
	fi

	while true; do
		printf 'commit anyway? y/[n]: '
		read -r ans </dev/tty

		case "$ans" in
		'y' | 'Y')
			break
			;;
		'' | 'n' | 'N')
			exit 1
			;;
		*)
			continue
			;;
		esac
	done
}

# check_unstaged_changes helper checks for unstaged changes and untracked files.
# If any are found, the programmer will be warned, but the commit will not fail.
check_unstaged_changes() {
	# shellcheck disable=SC2016
	awk_prog='substr($2, 2, 1) != "." { print $9; } $1 == "?" { print $2; }'
	readonly awk_prog

	unstaged="$(git status --porcelain=2 | awk "$awk_prog")"
	readonly unstaged

	if [ "$unstaged" != '' ]; then
		printf 'WARNING: you have unstaged changes:\n\n%s\n\n' "$unstaged"
		prompt
	fi
}

# lint_staged_changes is a helper that runs all necessary linters, tests, etc.,
# based on the types of files that have been modified.
lint_staged_changes() {
	verbose="${VERBOSE:-0}"
	readonly verbose

	if [ "$(git diff --cached --name-only -- '*.md' || :)" != '' ]; then
		make VERBOSE="$verbose" md-lint
	fi

	if [ "$(git diff --cached --name-only -- '*.sh' || :)" != '' ]; then
		make VERBOSE="$verbose" sh-lint
	fi

	txt_diff="$(git diff --cached --name-only -- '*.md' '*.yaml' '*.yml' 'Makefile' '*.json' || :)"
	readonly txt_diff

	if [ "$txt_diff" != '' ]; then
		make VERBOSE="$verbose" txt-lint
	fi

	if [ "$(git diff --cached --name-only -- '*.go' '*.mod' 'Makefile' || :)" != '' ]; then
		make VERBOSE="$verbose" go-os-check go-lint go-test
	fi

}
