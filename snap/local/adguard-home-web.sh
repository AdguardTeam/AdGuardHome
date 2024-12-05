#!/bin/sh

# shellcheck disable=SC2154
conf_file="${SNAP_DATA}/AdGuardHome.yaml"
readonly conf_file

if ! [ -f "$conf_file" ]; then
	xdg-open 'http://localhost:3000'

	exit
fi

# Get the admin interface port from the configuration.
#
# shellcheck disable=SC2016
awk_prog='/^[^[:space:]]/ { is_http = /^http:/ };/^[[:space:]]+address:/ { if (is_http) print $2 }'
readonly awk_prog

bind_port="$(awk "$awk_prog" "$conf_file" | awk -F ':' '{print $NF}')"
readonly bind_port

if [ "$bind_port" = '' ]; then
	xdg-open 'http://localhost:3000'
else
	xdg-open "http://localhost:${bind_port}"
fi
