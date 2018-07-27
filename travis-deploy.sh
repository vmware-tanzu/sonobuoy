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

function gcr_push() {
    openssl aes-256-cbc -K $encrypted_708bef23737d_key -iv $encrypted_708bef23737d_iv -in heptio-images-ee4b0474b93e.json.enc -out ./heptio-images-ee4b0474b93e.json -d
    gcloud auth activate-service-account --key-file heptio-images-ee4b0474b93e.json
    IMAGE_BRANCH="$TRAVIS_BRANCH" DOCKER="gcloud docker -- " make container push
}

if [ ! -z "$TRAVIS_TAG" ]; then

    if [ "$(./sonobuoy version)" != "$TRAVIS_TAG" ]; then
        echo "sonobuoy version does not match tagged version!" >&2
        echo "sonobuoy version is $(./sonobuoy version)" >&2
        echo "tag is $TRAVIS_TAG" >&2
        exit 1
    fi

    goreleaser
    gcr_push
fi

if [ "$TRAVIS_BRANCH" == "master" ]; then
    gcr_push
fi

