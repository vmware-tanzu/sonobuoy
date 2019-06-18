#!/bin/bash

# Don't fail silently when a step doesn't succeed
set -e

# Early detect/fail if deps not correct.
./dep ensure
git_status=$(git status -s)
if [ -n "$git_status" ]; then
    echo $git_status
    echo "dep ensure modified the git status; did you need add/remove a dependency?"
    exit 1
fi

make container deploy_kind

echo "|---- Creating new Sonobuoy run/waiting for results..."
./sonobuoy run --image-pull-policy=IfNotPresent -m Quick --wait

outFile=$(./sonobuoy retrieve)

set +e
results=$(./sonobuoy e2e $outFile)
e2eCode=$?
echo $results
set -e

if [ $e2eCode -ne 0 ]; then
    echo "Error getting results from tarball"
    ./sonobuoy status
    ./sonobuoy logs
    mkdir results; tar xzf $outFile -C results
    find results
    find results/plugins -exec cat {} \;
    exit $e2eCode
fi

if echo $results | grep --quiet "failed tests: 0"; then
    echo "|---- Success!"
else
    echo "|---- Failure: E2E tests failed"
    exit 1
fi