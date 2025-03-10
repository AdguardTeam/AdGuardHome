#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 7

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]; then
	set -x
	v_flags='-v=1'
	x_flags='-x=1'
elif [ "$verbose" -gt '0' ]; then
	set -x
	v_flags='-v=1'
	x_flags='-x=0'
else
	set +x
	v_flags='-v=0'
	x_flags='-x=0'
fi
readonly v_flags x_flags

set -e -f -u

# Reset GOARCH and GOOS to make sure we install the tools for the native
# architecture even when we're cross-compiling the main binary, and also to
# prevent the "cannot install cross-compiled binaries when GOBIN is set" error.
env \
	GOARCH="" \
	GOBIN="${PWD}/bin" \
	GOOS="" \
	GOWORK='off' \
	"${GO:-go}" install "$v_flags" "$x_flags" tool \
	;
