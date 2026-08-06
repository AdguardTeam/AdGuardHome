#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '0' ]; then
	set -x
else
	set +x
fi

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

# Make sure that those are built using something like:
#	make ARCH='386 amd64 arm arm64 ppc64le' OS=linux VERBOSE=1 build-release
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

# Set DOCKER_PUSH to '1' if you want (and are allowed) to push to DockerHub.
docker_push="${DOCKER_PUSH:-0}"
readonly docker_push

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

# docker_build_opt_tag is a function that wraps the call of docker build command
# with optionally --tag flags.
docker_build_opt_tag() {
	if [ "$sudo_cmd" != '' ]; then
		set -- "$sudo_cmd"
	fi

	# Set the initial parameters.
	set -- \
		"$@" \
		docker \
		buildx \
		build \
		--build-arg BUILD_DATE="$build_date" \
		--build-arg DIST_DIR="$dist_dir" \
		--build-arg VCS_REF="$commit" \
		--build-arg VERSION="$version" \
		--platform "$docker_platforms" \
		--progress 'plain' \
		;

	# Append the channel tag, if any.
	if [ "$docker_channel_tag" != '' ]; then
		set -- "$@" "$docker_channel_tag"
	fi

	# Append the version tag, if any.
	if [ "$docker_version_tag" != '' ]; then
		set -- "$@" "$docker_version_tag"
	fi

	# Push to DockerHub, if requested.
	if [ "$docker_push" -eq 1 ]; then
		set -- "$@" '--push'
	fi

	# Append the rest.
	set -- \
		"$@" \
		-f \
		./docker/build.Dockerfile \
		. \
		;

	# Call the command with the assembled parameters.
	"$@"
}

docker_build_opt_tag
