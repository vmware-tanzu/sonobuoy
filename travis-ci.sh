#!/bin/bash

# Don't fail silently when a step doesn't succeed
set -e

make container deploy_kind

echo "|---- Creating new Sonobuoy run/waiting for results..."
./sonobuoy run --image-pull-policy=IfNotPresent -m Quick --wait

outFile=$(./sonobuoy retrieve)
results=$(./sonobuoy e2e $outFile)

echo $results

if echo $results | grep --quiet "failed tests: 0"; then
    echo "|---- Success!"
else
    echo "|---- Failure: E2E tests failed"
    exit 1
fi