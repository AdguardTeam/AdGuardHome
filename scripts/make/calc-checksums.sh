#!/bin/sh

# AdGuard Home Checksums Script
#
# The commentary in this file is written with the assumption that the reader
# only has superficial knowledge of the POSIX shell language and alike.
# Experienced readers may find it overly verbose.
#
# It calculates SHA-256 checksums for the archives in the distribution
# directory and writes them into checksums.txt.

# The default verbosity level is 0.  Show log messages if the caller requested
# verbosity level greater than 0.  Show the environment and every command that
# is run if the verbosity level is greater than 1.  Otherwise, print nothing.
verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]; then
	env
	set -x
fi

# Don't use -f, because we use globs in this script.  Exit the script if a
# pipeline fails (-e) and consider undefined variables as errors (-u).
#
# TODO(a.garipov): Use set -o 'pipefail' when the image supports it.
set -e -u

# Function log is an echo wrapper that writes to stderr if the caller requested
# verbosity level greater than 0.  Otherwise, it does nothing.
log() {
	if [ "$verbose" -gt '0' ]; then
		# Don't use quotes to get word splitting.
		printf '%s\n' "$1" 1>&2
	fi
}

# The default distribution files directory is dist.
dist="${DIST_DIR:-dist}"
readonly dist

log "checking tools"

# Make sure we fail gracefully if the SHA-256 tool we need is missing.  Use
# shasum as an alternative when available.
use_shasum='0'
if ! command -v 'sha256sum' >/dev/null; then
	if command -v 'shasum' >/dev/null; then
		# macOS doesn't have sha256sum installed by default, but it does
		# have shasum.
		log 'replacing sha256sum with shasum -a 256'
		use_shasum='1'
	else
		log "pieces don't fit, 'sha256sum' not found"

		exit 1
	fi
fi
readonly use_shasum

log "calculating checksums"

# calculate_checksums uses the previously detected SHA-256 tool to calculate
# checksums.  Do not use find with -exec, since shasum requires arguments.
calculate_checksums() {
	if [ "$use_shasum" -eq '0' ]; then
		sha256sum "$@"
	else
		shasum -a 256 "$@"
	fi
}

# Calculate the checksums of the files in a subshell with a different working
# directory.  Don't use ls, because files matching one of the patterns may be
# absent, which will make ls return with a non-zero status code.
(
	cd "./${dist}"

	: >./checksums.txt

	for archive in ./*.zip ./*.tar.gz ./*.msi; do
		# Make sure that we don't try to calculate a checksum for a glob pattern
		# that matched no files.
		if [ ! -f "$archive" ]; then
			continue
		fi

		calculate_checksums "$archive" >>./checksums.txt
	done
)

log "finished"
