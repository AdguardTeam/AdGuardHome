#!/usr/bin/env bash

set -eE
set -o pipefail
set -x

BUILDER_IMAGE="adguard/snapcraft:1.0"
SNAPCRAFT_TMPL="packaging/snap/snapcraft.yaml"
SNAP_NAME="adguard-home"
LAUNCHPAD_CREDENTIALS_DIR=".local/share/snapcraft/provider/launchpad"

if [[ -z ${VERSION} ]]; then
    VERSION=`git describe --abbrev=4 --dirty --always --tags`
    echo "VERSION env variable is not set, getting it from git: ${VERSION}"
fi

# If bash is interactive, set `-it` parameter for docker run
INTERACTIVE=""
if [ -t 0 ] ; then
    INTERACTIVE="-it"
fi

function usage() {
    cat <<EOF
    Usage: ${0##*/} command [options]

    Please note that in order for the builds to work properly, you need to setup some env variables.

    These are necessary for "remote-build' command.
    Read this doc on how to generate them: https://uci.readthedocs.io/en/latest/oauth.html

        * LAUNCHPAD_KEY -- launchpad CI key
        * LAUNCHPAD_ACCESS_TOKEN -- launchpad access token
        * LAUNCHPAD_ACCESS_SECRET -- launchpad access secret

    These are necessary for snapcraft publish command to work.
    They can be exported using "snapcraft export-login"

        * SNAPCRAFT_MACAROON
        * SNAPCRAFT_UBUNTU_DISCHARGE
        * SNAPCRAFT_EMAIL

    Examples:
        ${0##*/} build-docker - builds snaps using remote-build inside a Docker environment
        ${0##*/} build - builds snaps using remote-build
        ${0##*/} publish-docker-beta - publishes snaps to the beta channel using Docker environment
        ${0##*/} publish-docker-release - publishes snaps to the release channel using Docker environment
        ${0##*/} publish-beta - publishes snaps to the beta channel
        ${0##*/} publish-release - publishes snaps to the release channel
        ${0##*/} cleanup - clean up temporary files that were created by the builds
EOF
    exit 1
}

#######################################
# helper functions
#######################################

function prepare() {
    if [ -z "${LAUNCHPAD_KEY}" ] || [ -z "${LAUNCHPAD_ACCESS_TOKEN}" ] || [ -z "${LAUNCHPAD_ACCESS_SECRET}" ]; then
        echo "Launchpad oauth tokens are not set, exiting"
        usage
        exit 1
    fi

    if [ -z "${SNAPCRAFT_MACAROON}" ] || [ -z "${SNAPCRAFT_UBUNTU_DISCHARGE}" ] || [ -z "${SNAPCRAFT_EMAIL}" ]; then
        echo "Snapcraft auth params are not set, exiting"
        usage
        exit 1
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
    rm -f snapcraft.yaml.bak
}

build_snap() {
    # prepare credentials
    prepare

    # copy them to the directory where snapcraft will be able to read them
    mkdir -p ~/${LAUNCHPAD_CREDENTIALS_DIR}
    cp -f snapcraft_login ~/${LAUNCHPAD_CREDENTIALS_DIR}/credentials
    chmod 600 ~/${LAUNCHPAD_CREDENTIALS_DIR}/credentials

    # run the build
    snapcraft remote-build --build-on=${ARCH} --launchpad-accept-public-upload

    # remove the credentials - we don't need them anymore
    rm -rf ~/${LAUNCHPAD_CREDENTIALS_DIR}

    # remove version from the file name
    rename_snap_file

    # cleanup credentials
    cleanup
}

build_snap_docker() {
    # prepare credentials
    prepare

    docker run ${INTERACTIVE} --rm  \
        -v $(pwd):/build \
        -v $(pwd)/launchpad_credentials:/root/${LAUNCHPAD_CREDENTIALS_DIR}/credentials:ro \
        ${BUILDER_IMAGE} \
        snapcraft remote-build --build-on=${ARCH} --launchpad-accept-public-upload

    # remove version from the file name
    rename_snap_file

    # cleanup credentials
    cleanup
}

rename_snap_file() {
    # In order to make working with snaps easier later on
    # we remove version from the file name

    # Check that the snap file exists
    snapFile="${SNAP_NAME}_${VERSION}_${ARCH}.snap"
    if [ ! -f ${snapFile} ]; then
       echo "Snap file ${snapFile} not found!"
       exit 1
    fi

    mv -f ${snapFile} "${SNAP_NAME}_${ARCH}.snap"
}

publish_snap() {
    # prepare credentials
    prepare

    # Check that the snap file exists
    snapFile="${SNAP_NAME}_${ARCH}.snap"
    if [ ! -f ${snapFile} ]; then
       echo "Snap file ${snapFile} not found!"
       exit 1
    fi

    # Login if necessary
    snapcraft login --with=snapcraft_login

    # Push to the channel
    snapcraft push --release=${CHANNEL} ${snapFile}

    # cleanup credentials
    cleanup
}

publish_snap_docker() {
    # prepare credentials
    prepare

    # Check that the snap file exists
    snapFile="${SNAP_NAME}_${ARCH}.snap"
    if [ ! -f ${snapFile} ]; then
       echo "Snap file ${snapFile} not found!"
       exit 1
    fi

    # Login and publish the snap
    docker run ${INTERACTIVE} --rm \
        -v $(pwd):/build \
        ${BUILDER_IMAGE} \
        sh -c "snapcraft login --with=/build/snapcraft_login && snapcraft push --release=${CHANNEL} /build/${snapFile}"

    # cleanup credentials
    cleanup
}

#######################################
# main functions
#######################################

build() {
    if [[ -n "$1" ]]; then
        echo "ARCH is set to $1"
        ARCH=$1 build_snap
    else
        ARCH=i386 build_snap
        ARCH=arm64 build_snap
        ARCH=armhf build_snap
        ARCH=amd64 build_snap
    fi
}

build_docker() {
    if [[ -n "$1" ]]; then
        echo "ARCH is set to $1"
        ARCH=$1 build_snap_docker
    else
        ARCH=i386 build_snap_docker
        ARCH=arm64 build_snap_docker
        ARCH=armhf build_snap_docker
        ARCH=amd64 build_snap_docker
    fi
}

publish_docker() {
    if [[ -z $1 ]]; then
        echo "No channel specified"
        exit 1
    fi
    CHANNEL="${1}"
    if [ "$CHANNEL" != "stable" ] && [ "$CHANNEL" != "beta" ]; then
        echo "$CHANNEL is an invalid value for the update channel!"
        exit 1
    fi

    ARCH=i386 publish_snap_docker
    ARCH=arm64 publish_snap_docker
    ARCH=armhf publish_snap_docker
    ARCH=amd64 publish_snap_docker
}

publish() {
    if [[ -z $1 ]]; then
        echo "No channel specified"
        exit 1
    fi
    CHANNEL="${1}"
    if [ "$CHANNEL" != "stable" ] && [ "$CHANNEL" != "beta" ]; then
        echo "$CHANNEL is an invalid value for the update channel!"
        exit 1
    fi

    ARCH=i386 publish_snap
    ARCH=arm64 publish_snap
    ARCH=armhf publish_snap
    ARCH=amd64 publish_snap
}

cleanup() {
    rm -f launchpad_credentials
    rm -f snapcraft.yaml
    rm -f snapcraft.yaml.bak
    rm -f snapcraft_login
    git checkout snapcraft.yaml
}

#######################################
# main
#######################################
if [[ -z $1 || $1 == "--help" || $1 == "-h" ]]; then
    usage
fi

case "$1" in
"build-docker") build_docker $2 ;;
"build") build $2 ;;
"publish-docker-beta") publish_docker beta ;;
"publish-docker-release") publish_docker stable ;;
"publish-beta") publish beta ;;
"publish-release") publish stable ;;
"prepare") prepare ;;
"cleanup") cleanup ;;
*) usage ;;
esac

exit 0