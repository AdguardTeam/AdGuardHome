#!/bin/sh

# AdGuard Home Build Script
#
# The commentary in this file is written with the assumption that the
# reader only has superficial knowledge of the POSIX shell language and
# alike.  Experienced readers may find it overly verbose.

# The default verbosity level is 0.  Show every command that is run and
# every package that is processed if the caller requested verbosity
# level greater than 0.  Also show subcommands if the requested
# verbosity level is greater than 1.  Otherwise, do nothing.
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

# Exit the script if a pipeline fails (-e), prevent accidental filename
# expansion (-f), and consider undefined variables as errors (-u).
set -e -f -u

# Allow users to set the Go version.
go="${GO:-go}"

# Require the channel to be set and validate the value.
channel="$CHANNEL"
case "$channel"
in
('development'|'edge'|'beta'|'release')
	# All is well, go on.
	;;
(*)
	echo "invalid channel '$channel', supported values are\
		'development', 'edge', 'beta', and 'release'" 1>&2
	exit 1
	;;
esac

# Require the version to be set.
#
# TODO(a.garipov): Additional validation?
version="$VERSION"

# Set date and time of the current build.
buildtime="$(date -u +%FT%TZ%z)"

# Set the linker flags accordingly: set the release channel and the
# current version as well as goarm and gomips variable values, if the
# variables are set and are not empty.
readonly version_pkg='github.com/AdguardTeam/AdGuardHome/internal/version'
ldflags="-s -w"
ldflags="${ldflags} -X ${version_pkg}.version=${version}"
ldflags="${ldflags} -X ${version_pkg}.channel=${channel}"
ldflags="${ldflags} -X ${version_pkg}.buildtime=${buildtime}"
if [ "${GOARM:-}" != '' ]
then
	ldflags="${ldflags} -X ${version_pkg}.goarm=${GOARM}"
elif [ "${GOMIPS:-}" != '' ]
then
	ldflags="${ldflags} -X ${version_pkg}.gomips=${GOMIPS}"
fi

# Allow users to limit the build's parallelism.
readonly parallelism="${PARALLELISM:-}"
if [ "$parallelism" != '' ]
then
	readonly par_flags="-p ${parallelism}"
else
	readonly par_flags=''
fi

# Allow users to specify a different output name.
readonly out="${OUT:-}"
if [ "$out" != '' ]
then
	readonly out_flags="-o ${out}"
else
	readonly out_flags=''
fi

# Don't use cgo.  Use modules.
export CGO_ENABLED='0' GO111MODULE='on'

readonly build_flags="${BUILD_FLAGS:-$out_flags $par_flags\
	$v_flags $x_flags}"

# Don't use quotes with flag variables to get word splitting.
"$go" generate $v_flags $x_flags ./main.go

# Don't use quotes with flag variables to get word splitting.
"$go" build --ldflags "$ldflags" $build_flags
