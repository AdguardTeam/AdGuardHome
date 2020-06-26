#!/bin/bash
# Get admin tool port from configuration
bind_port=$(grep bind_port $SNAP_DATA/AdGuardHome.yaml | awk -F ' ' '{print $2}')

if [ -z "$bind_port" ]; then
	xdg-open http://localhost:3000
else
	xdg-open http://localhost:$bind_port
fi

