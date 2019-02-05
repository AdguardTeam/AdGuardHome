#!/usr/bin/env bash

set -eE
set -o pipefail
set -x

DOCKERFILE="Dockerfile.travis"
if [ "${TRAVIS_BRANCH}" == "master" ]
then
  VERSION="latest"
else
  VERSION=`git describe --abbrev=4 --dirty --always --tags`
fi

build_image() {
    from="$(awk '$1 == toupper("FROM") { print $2 }' ${DOCKERFILE})"

    # See https://hub.docker.com/r/multiarch/alpine/tags
    case "${GOARCH}" in
        arm64)
           alpineArch='arm64-edge'
           imageArch='arm64'
           ;;
        arm)
           alpineArch='armhf-edge'
           imageArch='armhf'
           ;;
        386)
           alpineArch='i386-edge'
           imageArch='i386'
           ;;
        amd64)
           alpineArch='amd64-edge'
           ;;
        *)
           alpineArch='amd64-edge'
           ;;
    esac

    if [ "${GOOS}" == "linux" ] && [ "${GOARCH}" == "amd64" ]
    then
        image="adguard/adguardhome:${VERSION}"
    else
        image="adguard/adguardhome:${imageArch}-${VERSION}"
    fi

    make cleanfast; CGO_DISABLED=1 make

    docker pull "multiarch/alpine:${alpineArch}"
    docker tag "multiarch/alpine:${alpineArch}" "$from"
    docker build -t "${image}" -f ${DOCKERFILE} .
    docker push ${image}
    docker rmi "$from"
}

# prepare qemu
docker run --rm --privileged multiarch/qemu-user-static:register --reset

make clean

# Prepare releases
GOOS=linux GOARCH=amd64 build_image
GOOS=linux GOARCH=386 build_image
GOOS=linux GOARCH=arm GOARM=6 build_image
GOOS=linux GOARCH=arm64 GOARM=6 build_image
