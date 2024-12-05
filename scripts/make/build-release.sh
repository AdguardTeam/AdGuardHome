#!/bin/sh

# AdGuard Home Release Script
#
# The commentary in this file is written with the assumption that the reader
# only has superficial knowledge of the POSIX shell language and alike.
# Experienced readers may find it overly verbose.

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
set -e -f -u

# Function log is an echo wrapper that writes to stderr if the caller requested
# verbosity level greater than 0.  Otherwise, it does nothing.
log() {
	if [ "$verbose" -gt '0' ]; then
		# Don't use quotes to get word splitting.
		echo "$1" 1>&2
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
	signer_api_key="${SIGNER_API_KEY:?please set SIGNER_API_KEY or unset SIGN}"
	deploy_script_path="${DEPLOY_SCRIPT_PATH:?please set DEPLOY_SCRIPT_PATH or unset SIGN}"
else
	gpg_key_passphrase=''
	gpg_key=''
	signer_api_key=''
	deploy_script_path=''
fi
readonly gpg_key_passphrase gpg_key signer_api_key deploy_script_path

# The default distribution files directory is dist.
dist="${DIST_DIR:-dist}"
readonly dist

log "checking tools"

# Make sure we fail gracefully if one of the tools we need is missing.  Use
# alternatives when available.
use_shasum='0'
for tool in gpg gzip sed sha256sum tar zip; do
	if ! command -v "$tool" >/dev/null; then
		if [ "$tool" = 'sha256sum' ] && command -v 'shasum' >/dev/null; then
			# macOS doesn't have sha256sum installed by default, but it does
			# have shasum.
			log 'replacing sha256sum with shasum -a 256'
			use_shasum='1'
		else
			log "pieces don't fit, '$tool' not found"

			exit 1
		fi
	fi
done
readonly use_shasum

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

	if [ "$sign_os" != 'windows' ]; then
		gpg \
			--default-key "$gpg_key" \
			--detach-sig \
			--passphrase "$gpg_key_passphrase" \
			--pinentry-mode loopback -q "$sign_bin_path" \
			;

		return
	elif [ "$channel" = 'beta' ] || [ "$channel" = 'release' ]; then
		signed_bin_path="${sign_bin_path}.signed"

		env INPUT_FILE="$sign_bin_path" \
			OUTPUT_FILE="$signed_bin_path" \
			SIGNER_API_KEY="$signer_api_key" \
			"$deploy_script_path" sign-executable

		mv "$signed_bin_path" "$sign_bin_path"
	fi
}

# Function build builds the release for one platform.  It builds a binary and an
# archive.
build() {
	# Get the arguments.  Here and below, use the "build_" prefix for all
	# variables local to function build.
	build_dir="${dist}/${1}/AdGuardHome" \
		build_ar="$2" \
		build_os="$3" \
		build_arch="$4" \
		build_arm="$5" \
		build_mips="$6" \
		;

	# Use the ".exe" filename extension if we build a Windows release.
	if [ "$build_os" = 'windows' ]; then
		build_output="./${build_dir}/AdGuardHome.exe"
	else
		build_output="./${build_dir}/AdGuardHome"
	fi

	mkdir -p "./${build_dir}"

	# Build the binary.
	#
	# Set GOARM and GOMIPS to an empty string if $build_arm and $build_mips are
	# the zero value by removing the hyphen as if it's a prefix.
	env GOARCH="$build_arch" \
		GOARM="${build_arm#-}" \
		GOMIPS="${build_mips#-}" \
		GOOS="$os" \
		VERBOSE="$((verbose - 1))" \
		VERSION="$version" \
		OUT="$build_output" \
		sh ./scripts/make/go-build.sh

	log "$build_output"

	sign "$os" "$build_output"

	# Prepare the build directory for archiving.
	cp ./CHANGELOG.md ./LICENSE.txt ./README.md "$build_dir"

	# Make archives.  Windows and macOS prefer ZIP archives; the rest,
	# gzipped tarballs.
	case "$build_os" in
	'darwin' | 'windows')
		build_archive="./${dist}/${build_ar}.zip"
		# TODO(a.garipov): Find an option similar to the -C option of tar for
		# zip.
		(cd "${dist}/${1}" && zip -9 -q -r "../../${build_archive}" "./AdGuardHome")
		;;
	*)
		build_archive="./${dist}/${build_ar}.tar.gz"
		tar -C "./${dist}/${1}" -c -f - "./AdGuardHome" | gzip -9 - >"$build_archive"
		;;
	esac

	log "$build_archive"
}

log "starting builds"

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
		ar="AdGuardHome_${os}_${arch}v${arm}"
		;;
	mips*)
		dir="AdGuardHome_${os}_${arch}_${mips}"
		ar="$dir"
		;;
	*)
		dir="AdGuardHome_${os}_${arch}"
		ar="$dir"
		;;
	esac

	build "$dir" "$ar" "$os" "$arch" "$arm" "$mips"
done

log "packing frontend"

build_archive="./${dist}/AdGuardHome_frontend.tar.gz"
tar -c -f - ./build | gzip -9 - >"$build_archive"
log "$build_archive"

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
#
# TODO(a.garipov): Consider calculating these as the build goes.
(
	set +f

	cd "./${dist}"

	: >./checksums.txt

	for archive in ./*.zip ./*.tar.gz; do
		# Make sure that we don't try to calculate a checksum for a glob pattern
		# that matched no files.
		if [ ! -f "$archive" ]; then
			continue
		fi

		calculate_checksums "$archive" >>./checksums.txt
	done
)

log "writing versions"

echo "version=$version" >"./${dist}/version.txt"

# Create the version.json file.

version_download_url="https://static.adtidy.org/adguardhome/${channel}"
version_json="./${dist}/version.json"
readonly version_download_url version_json

# If the channel is edge, point users to the "Platforms" page on the Wiki,
# because the direct links to the edge packages are listed there.
if [ "$channel" = 'edge' ]; then
	announcement_url='https://github.com/AdguardTeam/AdGuardHome/wiki/Platforms'
else
	announcement_url="https://github.com/AdguardTeam/AdGuardHome/releases/tag/${version}"
fi
readonly announcement_url

# TODO(a.garipov): Remove "selfupdate_min_version" in future versions.
rm -f "$version_json"
echo "{
  \"version\": \"${version}\",
  \"announcement\": \"AdGuard Home ${version} is now available!\",
  \"announcement_url\": \"${announcement_url}\",
  \"selfupdate_min_version\": \"0.0\",
" >>"$version_json"

# Add the MIPS* object keys without the "softfloat" part to mitigate the
# consequences of #5373.
#
# TODO(a.garipov): Remove this around fall 2023.
echo "
  \"download_linux_mips64\": \"${version_download_url}/AdGuardHome_linux_mips64_softfloat.tar.gz\",
  \"download_linux_mips64le\": \"${version_download_url}/AdGuardHome_linux_mips64le_softfloat.tar.gz\",
  \"download_linux_mipsle\": \"${version_download_url}/AdGuardHome_linux_mipsle_softfloat.tar.gz\",
" >>"$version_json"

# Same as with checksums above, don't use ls, because files matching one of the
# patterns may be absent.
ar_files="$(find "./${dist}" ! -name "${dist}" -prune \( -name '*.tar.gz' -o -name '*.zip' \))"
ar_files_len="$(echo "$ar_files" | wc -l)"
readonly ar_files ar_files_len

i='1'
# Don't use quotes to get word splitting.
for f in $ar_files; do
	platform="$f"

	# Remove the prefix.
	platform="${platform#"./${dist}/AdGuardHome_"}"

	# Remove the filename extensions.
	platform="${platform%.zip}"
	platform="${platform%.tar.gz}"

	# Use the filename's base path.
	filename="${f#"./${dist}/"}"

	if [ "$i" -eq "$ar_files_len" ]; then
		echo "  \"download_${platform}\": \"${version_download_url}/${filename}\"" >>"$version_json"
	else
		echo "  \"download_${platform}\": \"${version_download_url}/${filename}\"," >>"$version_json"
	fi

	i="$((i + 1))"
done

echo '}' >>"$version_json"

log "finished"
