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
channel="${CHANNEL:?please set CHANNEL}"
commit="${COMMIT:?please set COMMIT}"
dist_dir="${DIST_DIR:?please set DIST_DIR}"
readonly channel commit dist_dir

if [ "${VERSION:-}" = 'v0.0.0' ] || [ "${VERSION:-}" = '' ]
then
	version="$( sh ./scripts/make/version.sh )"
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

build_date="$( date -u +'%Y-%m-%dT%H:%M:%SZ' )"
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

case "$channel"
in
('release')
	docker_image_full_name="${docker_image_name}:${version}"
	docker_tags="--tag ${docker_image_name}:latest"
	;;
('beta')
	docker_image_full_name="${docker_image_name}:${version}"
	docker_tags="--tag ${docker_image_name}:beta"
	;;
('edge')
	# Don't set the version tag when pushing to the edge channel.
	docker_image_full_name="${docker_image_name}:edge"
	docker_tags=''
	;;
('development')
	docker_image_full_name="${docker_image_name}"
	docker_tags=''
	;;
(*)
	echo "invalid channel '$channel', supported values are\
		'development', 'edge', 'beta', and 'release'" 1>&2
	exit 1
	;;
esac
readonly docker_image_full_name docker_tags

# Copy the binaries into a new directory under new names, so that it's easier to
# COPY them later.  DO NOT remove the trailing underscores.  See file
# docker/Dockerfile.
dist_docker="${dist_dir}/docker"
readonly dist_docker

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

# Copy the helper scripts.  See file docker/Dockerfile.
dist_docker_scripts="${dist_docker}/scripts"
readonly dist_docker_scripts

mkdir -p "$dist_docker_scripts"
cp "./docker/dns-bind.awk"\
	"${dist_docker_scripts}/dns-bind.awk"
cp "./docker/web-bind.awk"\
	"${dist_docker_scripts}/web-bind.awk"
cp "./docker/healthcheck.sh"\
	"${dist_docker_scripts}/healthcheck.sh"

# Don't use quotes with $docker_tags and $debug_flags because we want word
# splitting and or an empty space if tags are empty.
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
	-f ./docker/Dockerfile\
	.
