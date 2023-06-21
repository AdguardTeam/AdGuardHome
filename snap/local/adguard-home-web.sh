#!/bin/sh

# Get the admin interface port from the configuration.
bind_port="$( grep -e 'bind_port' "${SNAP_DATA}/AdGuardHome.yaml" | awk -F ' ' '{print $2}' )"
readonly bind_port

if [ "$bind_port" = '' ]
then
	xdg-open 'http://localhost:3000'
else
	xdg-open "http://localhost:${bind_port}"
fi
