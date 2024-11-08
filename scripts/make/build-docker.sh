#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '0' ]; then
	set -x
	debug_flags='--debug=1'
else
	set +x
	debug_flags='--debug=0'
fi
readonly debug_flags

set -e -f -u

# Require these to be set.  The channel value is validated later.
channel="${CHANNEL:?please set CHANNEL}"
commit="${REVISION:?please set REVISION}"
dist_dir="${DIST_DIR:?please set DIST_DIR}"
readonly channel commit dist_dir

if [ "${VERSION:-}" = 'v0.0.0' ] || [ "${VERSION:-}" = '' ]; then
	version="$(sh ./scripts/make/version.sh)"
else
	version="$VERSION"
fi
readonly version

# Allow users to use sudo.
sudo_cmd="${SUDO:-}"
readonly sudo_cmd

docker_platforms="\
linux/386,\
linux/amd64,\
linux/arm/v6,\
linux/arm/v7,\
linux/arm64,\
linux/ppc64le"
readonly docker_platforms

build_date="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
readonly build_date

# Set DOCKER_IMAGE_NAME to 'adguard/adguard-home' if you want (and are allowed)
# to push to DockerHub.
docker_image_name="${DOCKER_IMAGE_NAME:-adguardhome-dev}"
readonly docker_image_name

# Set DOCKER_OUTPUT to 'type=image,name=adguard/adguard-home,push=true' if you
# want (and are allowed) to push to DockerHub.
#
# If you want to inspect the resulting image using commands like "docker image
# ls", change type to docker and also set docker_platforms to a single platform.
#
# See https://github.com/docker/buildx/issues/166.
docker_output="${DOCKER_OUTPUT:-type=image,name=${docker_image_name},push=false}"
readonly docker_output

case "$channel" in
'release')
	docker_version_tag="--tag=${docker_image_name}:${version}"
	docker_channel_tag="--tag=${docker_image_name}:latest"
	;;
'beta')
	docker_version_tag="--tag=${docker_image_name}:${version}"
	docker_channel_tag="--tag=${docker_image_name}:beta"
	;;
'edge')
	# Set the version tag to an empty string when pushing to the edge channel.
	docker_version_tag=''
	docker_channel_tag="--tag=${docker_image_name}:edge"
	;;
'development')
	# Set both tags to an empty string for development builds.
	docker_version_tag=''
	docker_channel_tag=''
	;;
*)
	echo "invalid channel '$channel', supported values are\
		'development', 'edge', 'beta', and 'release'" 1>&2
	exit 1
	;;
esac
readonly docker_version_tag docker_channel_tag

# Copy the binaries into a new directory under new names, so that it's easier to
# COPY them later.  DO NOT remove the trailing underscores.  See file
# docker/Dockerfile.
dist_docker="${dist_dir}/docker"
readonly dist_docker

mkdir -p "$dist_docker"
cp "${dist_dir}/AdGuardHome_linux_386/AdGuardHome/AdGuardHome" \
	"${dist_docker}/AdGuardHome_linux_386_"
cp "${dist_dir}/AdGuardHome_linux_amd64/AdGuardHome/AdGuardHome" \
	"${dist_docker}/AdGuardHome_linux_amd64_"
cp "${dist_dir}/AdGuardHome_linux_arm64/AdGuardHome/AdGuardHome" \
	"${dist_docker}/AdGuardHome_linux_arm64_"
cp "${dist_dir}/AdGuardHome_linux_arm_6/AdGuardHome/AdGuardHome" \
	"${dist_docker}/AdGuardHome_linux_arm_v6"
cp "${dist_dir}/AdGuardHome_linux_arm_7/AdGuardHome/AdGuardHome" \
	"${dist_docker}/AdGuardHome_linux_arm_v7"
cp "${dist_dir}/AdGuardHome_linux_ppc64le/AdGuardHome/AdGuardHome" \
	"${dist_docker}/AdGuardHome_linux_ppc64le_"

# Don't use quotes with $docker_version_tag and $docker_channel_tag, because we
# want word splitting and or an empty space if tags are empty.
#
# TODO(a.garipov): Once flag --tag of docker buildx build supports commas, use
# them instead.
#
# shellcheck disable=SC2086
$sudo_cmd docker "$debug_flags" \
	buildx build \
	--build-arg BUILD_DATE="$build_date" \
	--build-arg DIST_DIR="$dist_dir" \
	--build-arg VCS_REF="$commit" \
	--build-arg VERSION="$version" \
	--output "$docker_output" \
	--platform "$docker_platforms" \
	$docker_version_tag $docker_channel_tag -f ./docker/Dockerfile .
