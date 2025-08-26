#!/bin/sh

set -e -f -u -x

# This script syncs companies DB that we bundle with AdGuard Home.  The source
# for this database is https://github.com/AdguardTeam/companiesdb.
#
trackers_url='https://raw.githubusercontent.com/AdguardTeam/companiesdb/main/dist/trackers.json'
# TODO: Update output path to './client_v2/src/helpers/trackers/trackers.json' for new frontend migration
output='./client/src/helpers/trackers/trackers.json'
readonly trackers_url output

curl -o "$output" -v "$trackers_url"
