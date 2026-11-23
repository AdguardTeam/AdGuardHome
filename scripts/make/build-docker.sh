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

# newest_release prints the newest release version that matches the extended
# regular expression in the first argument, considering both the git tags in
# git_tags and the version being built.  Sort the versions numerically and in
# reverse by the major part, skipping the "v" prefix, then by the minor and
# patch parts.
newest_release() {
	printf '%s\n' "$git_tags" "$version" \
		| grep -E -x -e "$1" \
		| sort -k 1.2,1nr -k 2,2nr -k 3,3nr -t '.' \
		| head -n 1 \
		;
}

case "$channel" in
'release')
	# The minor and major tags, for example v1.2 and v1 for v1.2.3, point to
	# the newest release of their line.  They require a version of the form
	# vMAJOR.MINOR.PATCH.
	if ! printf '%s\n' "$version" | grep -E -q -x 'v[0-9]+\.[0-9]+\.[0-9]+'; then
		echo "invalid release version '$version', expected vMAJOR.MINOR.PATCH" 1>&2
		exit 1
	fi

	version_minor="${version%.*}"
	version_major="${version_minor%.*}"
	readonly version_minor version_major

	docker_version_tag="--tag=${docker_image_name}:${version}"

	# Don't move a line's tag back when an older release of that line, for
	# example a patch for an older minor line, is built after a newer one.
	#
	# List the tags in an assignment, so that the build fails if they can't be
	# listed.
	git_tags="$(git tag)"
	major_num="${version_major#v}"
	minor_num="${version_minor##*.}"
	newest_in_major="$(newest_release "v${major_num}\\.[0-9]+\\.[0-9]+")"
	newest_in_minor="$(newest_release "v${major_num}\\.${minor_num}\\.[0-9]+")"
	readonly git_tags major_num minor_num newest_in_major newest_in_minor

	docker_minor_tag=''
	if [ "$newest_in_minor" = "$version" ]; then
		docker_minor_tag="--tag=${docker_image_name}:${version_minor}"
	else
		echo "not moving tag ${version_minor}: ${newest_in_minor} is newer" 1>&2
	fi

	docker_major_tag=''
	if [ "$newest_in_major" = "$version" ]; then
		docker_major_tag="--tag=${docker_image_name}:${version_major}"
	else
		echo "not moving tag ${version_major}: ${newest_in_major} is newer" 1>&2
	fi

	docker_channel_tag="--tag=${docker_image_name}:latest"
	;;
'beta')
	docker_version_tag="--tag=${docker_image_name}:${version}"
	docker_minor_tag=''
	docker_major_tag=''
	docker_channel_tag="--tag=${docker_image_name}:beta"
	;;
'edge')
	# Set the version tags to empty strings when pushing to the edge channel.
	docker_version_tag=''
	docker_minor_tag=''
	docker_major_tag=''
	docker_channel_tag="--tag=${docker_image_name}:edge"
	;;
'development')
	# Set all tags to empty strings for development builds.
	docker_version_tag=''
	docker_minor_tag=''
	docker_major_tag=''
	docker_channel_tag=''
	;;
*)
	echo "invalid channel '$channel', supported values are\
		'development', 'edge', 'beta', and 'release'" 1>&2
	exit 1
	;;
esac
readonly docker_version_tag docker_minor_tag docker_major_tag docker_channel_tag

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

	# Append the tags, if any, each as a separate argument.
	for tag in \
		"$docker_channel_tag" \
		"$docker_version_tag" \
		"$docker_minor_tag" \
		"$docker_major_tag"; do
		if [ "$tag" != '' ]; then
			set -- "$@" "$tag"
		fi
	done

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
