#!/bin/sh

readonly verbose="${VERBOSE:-0}"
if [ "$verbose" -gt '1' ]
then
	set -x
fi

set -e -f -u

readonly awk_program='/^v[0-9]+\.[0-9]+\.[0-9]+.*$/ {
	if (!$4) {
		# The last tag is a full release version, so bump the
		# minor release number and zero the patch release number
		# to get the next release.
		$2++;
		$3 = 0;
	}

	print($1 "." $2 "." $3);

	next;
}

{
	printf("invalid version: \"%s\"\n", $0);

	exit 1;
}'

readonly last_tag="$(git describe --abbrev=0)"
readonly current_desc="$(git describe)"

readonly channel="$CHANNEL"
case "$channel"
in
('development')
	echo 'v0.0.0'
	;;
('edge')
	next=$(echo $last_tag | awk -F '[.+-]' "$awk_program")
	echo "${next}-SNAPSHOT-$(git rev-parse --short HEAD)"
	;;
('beta'|'release')
	if [ "$current_desc" != "$last_tag" ]
	then
		echo 'need a tag' 1>&2

		exit 1
	fi

	echo "$last_tag"
	;;
(*)
	echo "invalid channel '$channel', supported values are\
		'development', 'edge', 'beta', and 'release'" 1>&2
	exit 1
	;;
esac
