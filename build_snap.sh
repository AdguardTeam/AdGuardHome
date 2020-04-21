#!/usr/bin/env bash

set -eE
set -o pipefail
set -x

BUILDER_IMAGE="adguard/snapcraft:1.0"
SNAPCRAFT_TMPL="packaging/snap/snapcraft.yaml"
SNAP_NAME="adguardhometest"
VERSION=`git describe --abbrev=4 --dirty --always --tags`

if [[ "${TRAVIS_BRANCH}" == "master" ]]
then
  CHANNEL="edge"
else
  CHANNEL="release"
fi

# Launchpad oauth tokens data is necessary to run snapcraft remote-build
#
# Here's an instruction on how to generate launchpad OAuth tokens:
# https://uci.readthedocs.io/en/latest/oauth.html
#
# Launchpad credentials are necessary to run snapcraft remote-build command
echo "[1]
consumer_key = ${LAUNCHPAD_KEY}
consumer_secret =
access_token = ${LAUNCHPAD_ACCESS_TOKEN}
access_secret = ${LAUNCHPAD_ACCESS_SECRET}
" > launchpad_credentials

# Snapcraft login data
# It can be exported using snapcraft export-login command
echo "[login.ubuntu.com]
macaroon = ${SNAPCRAFT_MACAROON}
unbound_discharge = ${SNAPCRAFT_UBUNTU_DISCHARGE}
email = ${SNAPCRAFT_EMAIL}" > snapcraft_login

# Prepare the snap configuration
cp ${SNAPCRAFT_TMPL} ./snapcraft.yaml
sed -i.bak 's/dev_version/'"${VERSION}"'/g' ./snapcraft.yaml

build_snap() {
    # Run the build
    docker run -it -v $(pwd):/build \
        -v $(pwd)/launchpad_credentials:/root/.local/share/snapcraft/provider/launchpad/credentials:ro \
        ${BUILDER_IMAGE} \
        snapcraft remote-build --build-on=${ARCH} --launchpad-accept-public-upload
}

publish_snap() {
    # Check that the snap file exists
    snapFile="${SNAP_NAME}_${VERSION}_${ARCH}.snap"
    if [ ! -f ${snapFile} ]; then
       echo "Snap file ${snapFile} not found!"
       exit 1
    fi

    # Login and publish the snap
    docker run -it -v $(pwd):/build \
        ${BUILDER_IMAGE} \
        sh -c "snapcraft login --with=/build/snapcraft_login && snapcraft push --release=${CHANNEL} /build/${snapFile}"
}

# Build snaps
ARCH=i386 build_snap
ARCH=arm64 build_snap
ARCH=armhf build_snap
ARCH=amd64 build_snap

# Publish snaps
ARCH=i386 publish_snap
ARCH=arm64 publish_snap
ARCH=armhf publish_snap
ARCH=amd64 publish_snap

# Clean up
rm launchpad_credentials
rm snapcraft.yaml
rm snapcraft.yaml.bak
rm snapcraft_login