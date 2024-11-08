#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '0' ]; then
	set -x
fi

set -e -f -u

# Function log is an echo wrapper that writes to stderr if the caller requested
# verbosity level greater than 0.  Otherwise, it does nothing.
log() {
	if [ "$verbose" -gt '0' ]; then
		# Don't use quotes to get word splitting.
		echo "$1" 1>&2
	fi
}

# Do not set a new lowercase variable, because the snapcraft tool expects the
# uppercase form.
if [ "${SNAPCRAFT_STORE_CREDENTIALS:-}" = '' ]; then
	log 'please set SNAPCRAFT_STORE_CREDENTIALS'

	exit 1
fi
export SNAPCRAFT_STORE_CREDENTIALS

snapcraft_channel="${SNAPCRAFT_CHANNEL:?please set SNAPCRAFT_CHANNEL}"
readonly snapcraft_channel

# Allow developers to overwrite the command, e.g. for testing.
snapcraft_cmd="${SNAPCRAFT_CMD:-snapcraft}"
readonly snapcraft_cmd

default_timeout='90s'
kill_timeout='120s'
readonly default_timeout kill_timeout

for arch in \
	'amd64' \
	'arm64' \
	'armhf' \
	'i386'; do
	snap_file="./AdGuardHome_${arch}.snap"

	# Catch the exit code and the combined output to later inspect it.
	set +e
	snapcraft_output="$(
		# Use timeout(1) to force snapcraft to quit after a certain time.  There
		# seems to be no environment variable or flag to force this behavior.
		timeout \
			--preserve-status \
			-k "$kill_timeout" \
			-v "$default_timeout" \
			"$snapcraft_cmd" upload \
			--release="${snapcraft_channel}" \
			--quiet \
			"${snap_file}" \
			2>&1
	)"
	snapcraft_exit_code="$?"
	set -e

	if [ "$snapcraft_exit_code" -eq '0' ]; then
		log "successful upload: ${snapcraft_output}"

		continue
	fi

	# Skip the ones that were failed by a duplicate upload error.
	case "$snapcraft_output" in
	*'A file with this exact same content has already been uploaded'* | \
		*'Error checking upload uniqueness'*)

		log "warning: duplicate upload, skipping"
		log "snapcraft upload error: ${snapcraft_output}"

		continue
		;;
	*)
		echo "unexpected snapcraft upload error: ${snapcraft_output}"

		return "$snapcraft_exit_code"
		;;
	esac
done
