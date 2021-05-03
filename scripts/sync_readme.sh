#!/usr/bin/env bash

set -x

# Getting the scripts directory can be hard when dealing with sourcing bash files.
# Github actions has this env var set already and locally you can just source the
# build_func.sh yourself. This is just a best effort for local dev.
GITHUB_WORKSPACE=${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}
DIR="$(cd "$GITHUB_WORKSPACE" || exit; cd ..; pwd)"
ROOT_README="$DIR/README.md"
MASTER_SITE_README="$DIR/site/content/docs/master/_index.md"
MODIFIED_README=/tmp/sonobuoy-readme-sync.md
FRONTMATTER="$DIR/site/content/docs/master/index-frontmatter.yaml"

# Users should update the README.md in the root of the repo and
# run this script to sync the README.md in the master docs. The reason
# the root version is taken as the source of truth is because it is more
# simple to remove prefixes from URLs and paths than to figure out
# where to add them.

# Following translations occur:
#  - use relative path to images
#  - use relative path to other pages (e.g. foo instead of sonobuoy.io/docs/foo)
#  - link to master docs instead of "docs" (which will go to the latest tagged version)
sed 's/site\/themes\/sonobuoy\/static\/img/img/' "$ROOT_README" |
sed 's/https:\/\/sonobuoy.io\/docs\///' |
sed 's/sonobuoy.io\/docs/sonobuoy.io\/docs\/master/' > "$MODIFIED_README"

cat "$FRONTMATTER" "$MODIFIED_README" > "$MASTER_SITE_README"
