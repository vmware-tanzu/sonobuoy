#!/usr/bin/env bash

set -x

# Getting the scripts directory can be hard when dealing with sourcing bash files.
# Github actions has this env var set already and locally you can just source the
# build_func.sh yourself. This is just a best effort for local dev.
GITHUB_WORKSPACE=${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}
DIR="$(cd "$GITHUB_WORKSPACE" || exit; cd ..; pwd)"

while getopts "bv:" opt; do
case $opt in
  v)
    VERSION="$OPTARG"
    TOC_NAME="$(echo "${VERSION}"toc|sed s/\\./-/g)"
    ;;
  b)
    BUMP=true
    ;;
  \?)
    echo "Invalid option: -$OPTARG" >&2
    exit 1
    ;;
  :)
    echo "Option -$OPTARG requires an argument." >&2
    exit 1
    ;;
esac
done

# Using local build for this portion rather than docer to avoid having to use a go build image, long build with go modules, etc.
# Refresh CLI docs to ensure the docs are up to date when copied for the next version.
echo "Refreshing CLI docs in main docs..."
source "${DIR}/scripts/build_funcs.sh"; update_cli_docs
echo "Done. Beginning to generate docs for specified version..."

if [ -z "${VERSION}" ]
then
  echo "-v requires argument to proceed making docs for the given version"
  exit 1
fi



read -r -d '' CONFIG_VERSION_BLOCK << EOM
  docs_latest: v.*
  docs_versions:
    - main
EOM

read -r -d '' NEW_VERSION_BLOCK << EOM
  docs_latest: ${VERSION}
  docs_versions:
    - main
    - ${VERSION}
EOM

read -r -d '' OLD_SCOPE_BLOCK << EOM
  - scope:
      path: docs\/main
    values:
      version: main
      gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/main
      layout: \"docs\"
EOM

read -r -d '' NEW_SCOPE_BLOCK << EOM
  - scope:
      path: docs\/main
    values:
      version: main
      gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/main
      layout: \"docs\"
  - scope:
      path: docs\/${VERSION}
    values:
      version: ${VERSION}
      gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/${VERSION}
      layout: \"docs\"
EOM

read -r -d '' MAIN_FRONTMATTER << EOM
---
version: main
cascade:
  layout: docs
  gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/.*
---
EOM

read -r -d '' RELEASE_FRONTMATTER << EOM
---
version: ${VERSION}
cascade:
  layout: docs
  gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/${VERSION}
---
EOM

OLD_TOC_BLOCK="main: main-toc"
read -r -d '' NEW_TOC_BLOCK << EOM
main: main-toc
${VERSION}: ${TOC_NAME}
EOM

OLD_FOOTER_BLOCK="cta_url: \/docs\/v.*"
NEW_FOOTER_BLOCK="cta_url: \/docs\/${VERSION}"

if [ $BUMP ]
then
  echo "-bump was triggered, modifying buildinfo.Version: $OPTARG" >&2
  docker run --rm \
      -v "${DIR}":/root \
      debian:stretch-slim \
      /bin/sh -c \
      "sed -i 's/var Version.*/var Version = \"${VERSION}\"/' /root/pkg/buildinfo/version.go"
fi

docker run --rm \
  -v "${DIR}":/root \
  debian:stretch-slim \
  /bin/sh -c \
  "rm -rf /root/site/content/docs/${VERSION} && \
  cp -r /root/site/content/docs/main /root/site/content/docs/${VERSION} && \
  sed -i 's/site\/docs\/main\///g' /root/site/content/docs/${VERSION}/_index.md && \
  sed -i 's/docs\/img/img/g' /root/site/content/docs/${VERSION}/_index.md && \
  sed -i 's/sonobuoy\/tree\/main/sonobuoy\/tree\/${VERSION}/g' /root/site/content/docs/${VERSION}/_index.md && \
  sed -i 's/sonobuoy.io\/docs\/main/sonobuoy.io\/docs\/${VERSION}/g' /root/site/content/docs/${VERSION}/_index.md && \
  cp /root/site/data/docs/main-toc.yml /root/site/data/docs/${TOC_NAME}.yml && \
  perl -i -0pe 's/${CONFIG_VERSION_BLOCK}/${NEW_VERSION_BLOCK}/' /root/site/config.yaml && \
  perl -i -0pe 's/${OLD_SCOPE_BLOCK}/${NEW_SCOPE_BLOCK}/' /root/site/config.yaml && \
  perl -i -0pe 's/${OLD_TOC_BLOCK}/${NEW_TOC_BLOCK}/' /root/site/data/docs/toc-mapping.yml && \
  perl -i -0pe 's/${MAIN_FRONTMATTER}/${RELEASE_FRONTMATTER}/' /root/site/content/docs/${VERSION}/index-frontmatter.yaml && \
  perl -i -0pe 's/${MAIN_FRONTMATTER}/${RELEASE_FRONTMATTER}/' /root/site/content/docs/${VERSION}/_index.md && \
  sed -i 's/${OLD_FOOTER_BLOCK}/${NEW_FOOTER_BLOCK}/' /root/site/config.yaml"
