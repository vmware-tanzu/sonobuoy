#!/bin/bash

if [ -z "$TRAVIS" ]; then
    echo "this script is intended to be run only on travis" >&2;
    exit 1
fi


if [ ! -z "$TRAVIS_TAG" ]; then
    curl -sL https://git.io/goreleaser | bash;
fi

IMAGE_BRANCH="$TRAVIS_BRANCH" DOCKER="gcloud docker -- " make container push
TRAVIS
