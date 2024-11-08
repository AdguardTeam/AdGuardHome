#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '0' ]; then
	set -x
fi

set -e -f -u

# Function log is an echo wrapper that writes to stderr if the caller requested
# verbosity level greater than 0.  Otherwise, it does nothing.
#
# TODO(a.garipov): Add to helpers.sh and use more actively in scripts.
log() {
	if [ "$verbose" -gt '0' ]; then
		# Don't use quotes to get word splitting.
		echo "$1" 1>&2
	fi
}

version="$(./AdGuardHome_amd64 --version | cut -d ' ' -f 4)"
if [ "$version" = '' ]; then
	log 'empty version from ./AdGuardHome_amd64'

	exit 1
fi
readonly version

log "version '$version'"

for arch in \
	'amd64' \
	'arm64' \
	'armhf' \
	'i386'; do
	build_output="./AdGuardHome_${arch}"
	snap_output="./AdGuardHome_${arch}.snap"
	snap_dir="${snap_output}.dir"

	# Create the meta subdirectory and copy files there.
	mkdir -p "${snap_dir}/meta"
	cp "$build_output" "${snap_dir}/AdGuardHome"
	cp './snap/local/adguard-home-web.sh' "$snap_dir"
	cp -r './snap/gui' "${snap_dir}/meta/"

	# Create a snap.yaml file, setting the values.
	sed \
		-e 's/%VERSION%/'"$version"'/' \
		-e 's/%ARCH%/'"$arch"'/' \
		./snap/snap.tmpl.yaml \
		>"${snap_dir}/meta/snap.yaml"

	# TODO(a.garipov): The snapcraft tool will *always* write everything,
	# including errors, to stdout.  And there doesn't seem to be a way to change
	# that.  So, save the combined output, but only show it when snapcraft
	# actually fails.
	set +e
	snapcraft_output="$(snapcraft pack "$snap_dir" --output "$snap_output" 2>&1)"
	snapcraft_exit_code="$?"
	set -e

	if [ "$snapcraft_exit_code" -ne '0' ]; then
		log "$snapcraft_output"
		exit "$snapcraft_exit_code"
	fi

	log "$snap_output"

	rm -f -r "$snap_dir"
done
