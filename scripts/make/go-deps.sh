#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '1' ]
then
	env
	set -x
	readonly v_flags='-v'
	readonly x_flags='-x'
elif [ "$verbose" -gt '0' ]
then
	set -x
	readonly v_flags='-v'
	readonly x_flags=''
else
	set +x
	readonly v_flags=''
	readonly x_flags=''
fi

set -e -f -u

go="${GO:-go}"

# Don't use quotes with flag variables because we want an empty space if
# those aren't set.
"$go" mod download $x_flags

# Reset GOARCH and GOOS to make sure we install the tools for the native
# architecture even when we're cross-compiling the main binary, and also
# to prevent the "cannot install cross-compiled binaries when GOBIN is
# set" error.
env\
	GOARCH=""\
	GOOS=""\
	GOBIN="${PWD}/bin"\
	"$go" install $v_flags $x_flags\
	github.com/gobuffalo/packr/packr
