#!/usr/bin/env bash

# Splitting build_funcs apart from this is a small quality of life improvement
# for when testing these locally. You don't want to source the file to get the
# functions but also necessarily set these values which will exit the terminal.
set -ex
set -o pipefail

# Getting the scripts directory can be hard when dealing with sourcing bash files.
# Github actions has this env var set already and locally you can just source the
# build_func.sh yourself. This is just a best effort for local dev.
GITHUB_WORKSPACE=${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}

# shellcheck source=/dev/null
source "$GITHUB_WORKSPACE"/scripts/build_funcs.sh
