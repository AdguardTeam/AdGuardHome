#!/bin/sh

# AdGuard Home Release Script
#
# The commentary in this file is written with the assumption that the reader
# only has superficial knowledge of the POSIX shell language and alike.
# Experienced readers may find it overly verbose.
#
# It builds the artifacts for the specified platforms and signs the ones that
# can be signed in place.
#
# The default verbosity level is 0.  Show log messages if the caller requested
# verbosity level greater than 0.  Show the environment and every command that
# is run if the verbosity level is greater than 1.  Otherwise, print nothing.
#
# The level of verbosity for the build script is the same minus one level.  See
# below in build().
verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]; then
	env
	set -x
fi

# By default, sign the packages, but allow users to skip that step.
sign="${SIGN:-1}"
readonly sign

# Exit the script if a pipeline fails (-e), prevent accidental filename
# expansion (-f), and consider undefined variables as errors (-u).
#
# TODO(d.kolyshev): Use set -o 'pipefail' when GitHub supports it.
set -e -f -u

# Function log is an echo wrapper that writes to stderr if the caller requested
# verbosity level greater than 0.  Otherwise, it does nothing.
log() {
	if [ "$verbose" -gt '0' ]; then
		printf '%s\n' "$1" 1>&2
	fi
}

log 'starting to build AdGuard Home release'

# Require the channel to be set.  Additional validation is performed later by
# go-build.sh.
channel="${CHANNEL:?please set CHANNEL}"
readonly channel

# Check VERSION against the default value from the Makefile.  If it is that, use
# the version calculation script.
version="${VERSION:-}"
if [ "$version" = 'v0.0.0' ] || [ "$version" = '' ]; then
	version="$(sh ./scripts/make/version.sh)"
fi
readonly version

log "channel '$channel'"
log "version '$version'"

# Check architecture and OS limiters.  Add spaces to the local versions for
# better pattern matching.
if [ "${ARCH:-}" != '' ]; then
	log "arches: '$ARCH'"
	arches=" $ARCH "
else
	arches=''
fi
readonly arches

if [ "${OS:-}" != '' ]; then
	log "oses: '$OS'"
	oses=" $OS "
else
	oses=''
fi
readonly oses

# Require the gpg key and passphrase to be set if the signing is required.
if [ "$sign" -eq '1' ]; then
	gpg_key_passphrase="${GPG_KEY_PASSPHRASE:?please set GPG_KEY_PASSPHRASE or unset SIGN}"
	gpg_key="${GPG_KEY:?please set GPG_KEY or unset SIGN}"
else
	gpg_key_passphrase=''
	gpg_key=''
fi
readonly gpg_key_passphrase gpg_key

# The default distribution files directory is dist.
dist="${DIST_DIR:-dist}"
readonly dist

log 'checking tools'

# Make sure we fail gracefully if the tool we need is missing.
if ! command -v 'gpg' >/dev/null; then
	log "pieces don't fit, 'gpg' not found"

	exit 1
fi

# Data section.  Arrange data into space-separated tables for read -r to read.
# Use a hyphen for missing values.

#    os  arch      arm mips
platforms="\
darwin   amd64     -   -
darwin   arm64     -   -
freebsd  386       -   -
freebsd  amd64     -   -
freebsd  arm       5   -
freebsd  arm       6   -
freebsd  arm       7   -
freebsd  arm64     -   -
linux    386       -   -
linux    amd64     -   -
linux    arm       5   -
linux    arm       6   -
linux    arm       7   -
linux    arm64     -   -
linux    mips      -   softfloat
linux    mips64    -   softfloat
linux    mips64le  -   softfloat
linux    mipsle    -   softfloat
linux    ppc64le   -   -
linux    riscv64   -   -
openbsd  amd64     -   -
openbsd  arm64     -   -
windows  386       -   -
windows  amd64     -   -
windows  arm64     -   -"
readonly platforms

# Function sign signs the specified build as intended by the target operating
# system.
sign() {
	# Only sign if needed.
	if [ "$sign" -ne '1' ]; then
		return
	fi

	# Get the arguments.  Here and below, use the "sign_" prefix for all
	# variables local to function sign.
	sign_os="$1"
	sign_bin_path="$2"

	log "signing $sign_bin_path"

	if [ "$sign_os" != 'windows' ]; then
		gpg \
			--default-key "$gpg_key" \
			--detach-sig \
			--passphrase "$gpg_key_passphrase" \
			--pinentry-mode loopback -q "$sign_bin_path" \
			;
	fi
}

# Function build builds the release for one platform.  It builds a binary and
# prepares its directory for packing.
build() {
	# Get the arguments.  Here and below, use the "build_" prefix for all
	# variables local to function build.
	build_dir="${dist}/${1}/AdGuardHome" \
		build_os="$2" \
		build_arch="$3" \
		build_arm="$4" \
		build_mips="$5" \
		;

	# Use the ".exe" filename extension if we build a Windows release.
	if [ "$build_os" = 'windows' ]; then
		build_output="./${build_dir}/AdGuardHome.exe"
	else
		build_output="./${build_dir}/AdGuardHome"
	fi

	mkdir -p "./${build_dir}"

	# Prepare the build directory for archiving.
	cp ./CHANGELOG.md ./LICENSE.txt ./README.md "$build_dir"

	# Build the binary.
	#
	# Set GOARM and GOMIPS to an empty string if $build_arm and $build_mips are
	# the zero value by removing the hyphen as if it's a prefix.
	env \
		GOARCH="$build_arch" \
		GOARM="${build_arm#-}" \
		GOMIPS="${build_mips#-}" \
		GOOS="$build_os" \
		VERBOSE="$((verbose - 1))" \
		VERSION="$version" \
		OUT="$build_output" \
		sh ./scripts/make/go-build.sh

	log "$build_output"

	sign "$build_os" "$build_output"
}

log 'starting builds'

# Go over all platforms defined in the space-separated table above, tweak the
# values where necessary, and feed to build.
echo "$platforms" | while read -r os arch arm mips; do
	# See if the architecture or the OS is in the allowlist.  To do so, try
	# removing everything that matches the pattern (well, a prefix, but that
	# doesn't matter here) containing the arch or the OS.
	#
	# For example, when $arches is " amd64 arm64 " and $arch is "amd64",
	# then the pattern to remove is "* amd64 *", so the whole string becomes
	# empty.  On the other hand, if $arch is "windows", then the pattern is
	# "* windows *", which doesn't match, so nothing is removed.
	#
	# See https://stackoverflow.com/a/43912605/1892060.
	#
	# shellcheck disable=SC2295
	if [ "${arches##* $arch *}" != '' ]; then
		log "$arch excluded, continuing"

		continue
	elif [ "${oses##* $os *}" != '' ]; then
		log "$os excluded, continuing"

		continue
	fi

	case "$arch" in
	arm)
		dir="AdGuardHome_${os}_${arch}_${arm}"
		;;
	mips*)
		dir="AdGuardHome_${os}_${arch}_${mips}"
		;;
	*)
		dir="AdGuardHome_${os}_${arch}"
		;;
	esac

	build "$dir" "$os" "$arch" "$arm" "$mips"
done

log 'finished'
