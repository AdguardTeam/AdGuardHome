#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '0' ]
then
	set -x
	debug_flags='-D'
else
	set +x
	debug_flags=''
fi

set -e -f -u

# Require these to be set.  The channel value is validated later.
readonly channel="$CHANNEL"
readonly commit="$COMMIT"
readonly dist_dir="$DIST_DIR"

if [ "${VERSION:-}" = 'v0.0.0' -o "${VERSION:-}" = '' ]
then
	readonly version="$(sh ./scripts/make/version.sh)"
else
	readonly version="$VERSION"
fi

echo $version

# Allow users to use sudo.
readonly sudo_cmd="${SUDO:-}"

readonly docker_platforms="\
linux/386,\
linux/amd64,\
linux/arm/v6,\
linux/arm/v7,\
linux/arm64,\
linux/ppc64le"

readonly build_date="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

# Set DOCKER_IMAGE_NAME to 'adguard/adguard-home' if you want (and are
# allowed) to push to DockerHub.
readonly docker_image_name="${DOCKER_IMAGE_NAME:-adguardhome-dev}"

# Set DOCKER_OUTPUT to 'type=image,name=adguard/adguard-home,push=true'
# if you want (and are allowed) to push to DockerHub.
readonly docker_output="${DOCKER_OUTPUT:-type=image,name=${docker_image_name},push=false}"

case "$channel"
in
('release')
	readonly docker_image_full_name="${docker_image_name}:${version}"
	readonly docker_tags="--tag ${docker_image_name}:latest"
	;;
('beta')
	readonly docker_image_full_name="${docker_image_name}:${version}"
	readonly docker_tags="--tag ${docker_image_name}:beta"
	;;
('edge')
	# Don't set the version tag when pushing to the edge channel.
	readonly docker_image_full_name="${docker_image_name}:edge"
	readonly docker_tags=''
	;;
('development')
	readonly docker_image_full_name="${docker_image_name}"
	readonly docker_tags=''
	;;
(*)
	echo "invalid channel '$channel', supported values are\
		'development', 'edge', 'beta', and 'release'" 1>&2
	exit 1
	;;
esac

# Copy the binaries into a new directory under new names, so that it's
# eaiser to COPY them later.  DO NOT remove the trailing underscores.
# See scripts/make/Dockerfile.
readonly dist_docker="${dist_dir}/docker"
mkdir -p "$dist_docker"
cp "${dist_dir}/AdGuardHome_linux_386/AdGuardHome/AdGuardHome"\
	"${dist_docker}/AdGuardHome_linux_386_"
cp "${dist_dir}/AdGuardHome_linux_amd64/AdGuardHome/AdGuardHome"\
	"${dist_docker}/AdGuardHome_linux_amd64_"
cp "${dist_dir}/AdGuardHome_linux_arm64/AdGuardHome/AdGuardHome"\
	"${dist_docker}/AdGuardHome_linux_arm64_"
cp "${dist_dir}/AdGuardHome_linux_arm_6/AdGuardHome/AdGuardHome"\
	"${dist_docker}/AdGuardHome_linux_arm_v6"
cp "${dist_dir}/AdGuardHome_linux_arm_7/AdGuardHome/AdGuardHome"\
	"${dist_docker}/AdGuardHome_linux_arm_v7"
cp "${dist_dir}/AdGuardHome_linux_ppc64le/AdGuardHome/AdGuardHome"\
	"${dist_docker}/AdGuardHome_linux_ppc64le_"

# Don't use quotes with $docker_tags and $debug_flags because we want
# word splitting and or an empty space if tags are empty.
$sudo_cmd docker\
	$debug_flags\
	buildx build\
	--build-arg BUILD_DATE="$build_date"\
	--build-arg DIST_DIR="$dist_dir"\
	--build-arg VCS_REF="$commit"\
	--build-arg VERSION="$version"\
	--output "$docker_output"\
	--platform "$docker_platforms"\
	$docker_tags\
	-t "$docker_image_full_name"\
	-f ./scripts/make/Dockerfile\
	.
