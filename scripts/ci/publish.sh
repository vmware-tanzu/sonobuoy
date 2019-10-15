#!/bin/bash

# Don't fail silently when a step doesn't succeed
set -ex

if [ -z "$CIRCLECI" ]; then
    echo "this script is intended to be run only on CircleCI" >&2
    exit 1
fi
SONOBUOY_CLI="${SONOBUOY_CLI:-sonobuoy}"

function goreleaser() {
    curl -sL https://git.io/goreleaser | bash
}

function image_push() {
    echo ${DOCKERHUB_TOKEN} | docker login --username sonobuoybot --password-stdin
    IMAGE_BRANCH="$CIRCLE_BRANCH" make container push
}

if [ ! -z "$CIRCLE_TAG" ]; then
    if [ "$($SONOBUOY_CLI version --short)" != "$CIRCLE_TAG" ]; then
        echo "sonobuoy version does not match tagged version!" >&2
        echo "sonobuoy short version is $($SONOBUOY_CLI version --short)" >&2
        echo "tag is $CIRCLE_TAG" >&2
        echo "sonobuoy full version info is $($SONOBUOY_CLI version)" >&2
        exit 1
    fi

    goreleaser --skip-validate
    image_push
else
    echo "CIRCLE_TAG not set, not running goreleaser"
fi

if [ "$CIRCLE_BRANCH" == "master" ]; then
    image_push
else
    echo "CIRCLE_BRANCH not master, not pushing images"
fi