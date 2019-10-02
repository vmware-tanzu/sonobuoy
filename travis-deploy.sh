#!/bin/bash

# Don't fail silently when a step doesn't succeed
set -e

if [ -z "$TRAVIS" ]; then
    echo "this script is intended to be run only on travis" >&2
    exit 1
fi

function goreleaser() {
    curl -sL https://git.io/goreleaser | bash
}

function image_push() {
    echo ${DOCKERHUB_TOKEN} | docker login --username sonobuoybot
    IMAGE_BRANCH="$BRANCH" make container push
}

if [ ! -z "$TRAVIS_TAG" ]; then
    if [ "$(./sonobuoy version --short)" != "$TRAVIS_TAG" ]; then
        echo "sonobuoy version does not match tagged version!" >&2
        echo "sonobuoy short version is $(./sonobuoy version --short)" >&2
        echo "tag is $TRAVIS_TAG" >&2
        echo "sonobuoy full version info is $(./sonobuoy version)" >&2
        exit 1
    fi

    goreleaser --skip-validate
    image_push
fi

if [ "$BRANCH" == "master" ]; then
    image_push
fi