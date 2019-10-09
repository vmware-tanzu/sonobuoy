#!/bin/bash

CI_SCRIPTS_DIR="$( cd "$( dirname "$0" )" >/dev/null 2>&1 && pwd )"
DIR=$(cd $CI_SCRIPTS_DIR; cd ../..; pwd)

tarball=$1

set +e
results=$($DIR/sonobuoy e2e $tarball)
e2eCode=$?
echo $results
set -e

if [ $e2eCode -ne 0 ]; then
    echo "Error getting results from tarball"
    $DIR/sonobuoy status
    $DIR/sonobuoy logs
    mkdir results; tar xzf $tarball -C results

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
