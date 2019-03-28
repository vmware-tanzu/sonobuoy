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

if [ ! -z "$TRAVIS_TAG" ]; then

    if [ "$(./sonobuoy version)" != "$TRAVIS_TAG" ]; then
        echo "sonobuoy version does not match tagged version!" >&2
        echo "sonobuoy version is $(./sonobuoy version)" >&2
        echo "tag is $TRAVIS_TAG" >&2
        exit 1
    fi

    goreleaser
fi
