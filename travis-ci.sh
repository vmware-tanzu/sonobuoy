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

./scripts/sync_readme.sh
git_status=$(git status -s)
if [ -n "$git_status" ]; then
    echo $git_status
    echo "scripts/sync_readme.sh modified the git status. If updating the README.md, update the root README.md and run that script."
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

    echo "Full contents of tarball:"
    find results 

    echo "Printing data on the following files:"
    find results/plugins -type f
    find results/plugins -type f \
      -exec echo Printing file info and contents of {} \; \
      -exec ls -lah {} \; \
      -exec cat {} \; \
      -exec echo \; \
      -exec echo \;
    echo "[Exit code of find was: $?]"

    echo "Printing data on the following files:"
    find results/podlogs -type f
    find results/podlogs -type f \
      -exec echo Printing file info and contents of {} \; \
      -exec ls -lah {} \; \
      -exec cat {} \; \
      -exec echo \; \
      -exec echo \;
    echo "[Exit code of find was: $?]"
    exit $e2eCode
fi

if echo $results | grep --quiet "failed tests: 0"; then
    echo "|---- Success!"
else
    echo "|---- Failure: E2E tests failed"
    exit 1
fi