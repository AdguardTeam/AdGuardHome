#!/bin/sh

# AdGuard Home Version Generation Script
#
# This script generates versions based on the current git tree state.  The valid
# output formats are:
#
#  *  For release versions, "v0.123.4".  This version should be the one in the
#     current tag, and the script merely checks, that the current commit is
#     properly tagged.
#
#  *  For prerelease beta versions, "v0.123.4-b.5".  This version should be the
#     one in the current tag, and the script merely checks, that the current
#     commit is properly tagged.
#
#  *  For prerelease alpha versions (aka snapshots), "v0.123.4-a.6+a1b2c3d4".
#
# BUG(a.garipov): The script currently can't differentiate between beta tags and
# release tags if they are on the same commit, so the beta tag **must** be
# pushed and built **before** the release tag is pushed.
#
# TODO(a.garipov): The script currently doesn't handle release branches, so it
# must be modified once we have those.

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '0' ]; then
	set -x
fi

set -e -f -u

# bump_minor is an awk program that reads a minor release version, increments
# the minor part of it, and prints the next version.
#
# shellcheck disable=SC2016
bump_minor='/^v[0-9]+\.[0-9]+\.0$/ {
	print($1 "." $2 + 1 ".0");

	next;
}

{
	printf("invalid minor release version: \"%s\"\n", $0);

	exit 1;
}'
readonly bump_minor

# get_last_minor_zero returns the last new minor release.
get_last_minor_zero() {
	# List all tags.  Then, select those that fit the pattern of a new minor
	# release: a semver version with the patch part set to zero.
	#
	# Then, sort them first by the first field ("1"), starting with the
	# second character to skip the "v" prefix (".2"), and only spanning the
	# first field (",1").  The sort is numeric and reverse ("nr").
	#
	# Then, sort them by the second field ("2"), and only spanning the
	# second field (",2").  The sort is also numeric and reverse ("nr").
	#
	# Finally, get the top (that is, most recent) version.
	git tag \
		| grep -e 'v[0-9]\+\.[0-9]\+\.0$' \
		| sort -k 1.2,1nr -k 2,2nr -t '.' \
		| head -n 1 \
		;
}

channel="${CHANNEL:?please set CHANNEL}"
readonly channel

case "$channel" in
'development')
	# commit_number is the number of current commit within the branch.
	commit_number="$(git rev-list --count master..HEAD)"
	readonly commit_number

	# The development builds are described with a combination of unset semantic
	# version, the commit's number within the branch, and the commit hash, e.g.:
	#
	#   v0.0.0-dev.5-a1b2c3d4
	#
	version="v0.0.0-dev.${commit_number}+$(git rev-parse --short HEAD)"
	;;
'edge')
	# last_minor_zero is the last new minor release.
	last_minor_zero="$(get_last_minor_zero)"
	readonly last_minor_zero

	# num_commits_since_minor is the number of commits since the last new
	# minor release.  If the current commit is the new minor release,
	# num_commits_since_minor is zero.
	num_commits_since_minor="$(git rev-list --count "${last_minor_zero}..HEAD")"
	readonly num_commits_since_minor

	# next_minor is the next minor release version.
	next_minor="$(echo "$last_minor_zero" | awk -F '.' "$bump_minor")"
	readonly next_minor

	# Make this commit a prerelease version for the next minor release.  For
	# example, if the last minor release was v0.123.0, and the current
	# commit is the fifth since then, the version will look something like:
	#
	#   v0.124.0-a.5+a1b2c3d4
	#
	version="${next_minor}-a.${num_commits_since_minor}+$(git rev-parse --short HEAD)"
	;;
'beta' | 'release')
	# current_desc is the description of the current git commit.  If the
	# current commit is tagged, git describe will show the tag.
	current_desc="$(git describe)"
	readonly current_desc

	# last_tag is the most recent git tag.
	last_tag="$(git describe --abbrev=0)"
	readonly last_tag

	# Require an actual tag for the beta and final releases.
	if [ "$current_desc" != "$last_tag" ]; then
		echo 'need a tag' 1>&2

		exit 1
	fi

	version="$last_tag"
	;;
'candidate')
	# This pseudo-channel is used to set a proper versions into release
	# candidate builds.

	# last_tag is expected to be the latest release tag.
	last_tag="$(git describe --abbrev=0)"
	readonly last_tag

	# current_branch is the name of the branch currently checked out.
	current_branch="$(git rev-parse --abbrev-ref HEAD)"
	readonly current_branch

	# The branch should be named like:
	#
	#   rc-v12.34.56
	#
	if ! echo "$current_branch" | grep -E -e '^rc-v[0-9]+\.[0-9]+\.[0-9]+$' -q; then
		echo "invalid release candidate branch name '$current_branch'" 1>&2

		exit 1
	fi

	version="${current_branch#rc-}-rc.$(git rev-list --count "$last_tag"..HEAD)"
	;;
*)
	echo "invalid channel '$channel', supported values are \
		'development', 'edge', 'beta', 'release' and 'candidate'" 1>&2
	exit 1
	;;
esac

# Finally, make sure that we don't output invalid versions.
if ! echo "$version" | grep -E -e '^v[0-9]+\.[0-9]+\.[0-9]+(-(a|b|dev|rc)\.[0-9]+)?(\+[[:xdigit:]]+)?$' -q; then
	echo "generated an invalid version '$version'" 1>&2

	exit 1
fi

echo "$version"
