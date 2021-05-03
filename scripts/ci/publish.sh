#!/bin/bash

# Don't fail silently when a step doesn't succeed
set -ex

if [ -z "$CIRCLECI" ]; then
    echo "this script is intended to be run only on CircleCI" >&2
    exit 1
fi

function image_push() {
    echo "${DOCKERHUB_TOKEN}" | docker login --username sonobuoybot --password-stdin
    make push
}

if [ -n "$CIRCLE_TAG" ]; then
    image_push
else
    echo "CIRCLE_TAG not set, not running goreleaser"
fi

if [ "$CIRCLE_BRANCH" == "master" ]; then
    image_push
else
    echo "CIRCLE_BRANCH not master, not pushing images"
fi
