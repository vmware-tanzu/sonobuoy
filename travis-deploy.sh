#!/bin/bash


if [ -z "$TRAVIS" ]; then
    echo "this script is intended to be run only on travis" >&2;
    exit 1
fi

if [ ! -z "$TRAVIS_TAG" ]; then
    curl -sL https://git.io/goreleaser | bash;
fi

openssl aes-256-cbc -K $encrypted_708bef23737d_key -iv $encrypted_708bef23737d_iv -in heptio-images-ee4b0474b93e.json.enc -out ./heptio-images-ee4b0474b93e.json -d
gcloud auth activate-service-account --key-file heptio-images-ee4b0474b93e.json

IMAGE_BRANCH="$TRAVIS_BRANCH" DOCKER="gcloud docker -- " make container push
