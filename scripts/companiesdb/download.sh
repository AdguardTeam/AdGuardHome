#!/bin/sh

set -e -f -u -x

# This script syncs companies DB that we bundle with AdGuard Home.  The source
# for this database is https://github.com/AdguardTeam/companiesdb.

whotracksme='https://raw.githubusercontent.com/AdguardTeam/companiesdb/main/dist/whotracksme.json'
adguard='https://raw.githubusercontent.com/AdguardTeam/companiesdb/main/dist/adguard.json'
base_path='../../client/src/helpers/trackers'
readonly whotracksme adguard base_path

curl "$whotracksme" --output "${base_path}/whotracksme.json"
curl "$adguard" --output "${base_path}/adguard.json"
