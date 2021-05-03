#!/usr/bin/env bash

set -x

# Getting the scripts directory can be hard when dealing with sourcing bash files.
# Github actions has this env var set already and locally you can just source the
# build_func.sh yourself. This is just a best effort for local dev.
GITHUB_WORKSPACE=${GITHUB_WORKSPACE:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}
DIR="$(cd "$GITHUB_WORKSPACE" || exit; cd ..; pwd)"
VERSION="$1"
TOC_NAME="$(echo "${VERSION}"toc|sed s/\\./-/g)"

read -r -d '' CONFIG_VERSION_BLOCK << EOM
  docs_latest: v.*
  docs_versions:
    - master
EOM

read -r -d '' NEW_VERSION_BLOCK << EOM
  docs_latest: ${VERSION}
  docs_versions:
    - master
    - ${VERSION}
EOM

read -r -d '' OLD_SCOPE_BLOCK << EOM
  - scope:
      path: docs\/master
    values:
      version: master
      gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/master
      layout: \"docs\"
EOM

read -r -d '' NEW_SCOPE_BLOCK << EOM
  - scope:
      path: docs\/master
    values:
      version: master
      gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/master
      layout: \"docs\"
  - scope:
      path: docs\/${VERSION}
    values:
      version: ${VERSION}
      gh: https:\/\/github.com\/vmware-tanzu\/sonobuoy\/tree\/${VERSION}
      layout: \"docs\"
EOM

read -r -d '' MASTER_FRONTMATTER << EOM
---
version: master
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

OLD_TOC_BLOCK="master: master-toc"
read -r -d '' NEW_TOC_BLOCK << EOM
master: master-toc
${VERSION}: ${TOC_NAME}
EOM

OLD_FOOTER_BLOCK="cta_url: \/docs\/v.*"
NEW_FOOTER_BLOCK="cta_url: \/docs\/${VERSION}"

if [ -z "${VERSION}" ]
then
        echo "Should be called with version as first argument. No argument given, not creating docs."
else
    docker run --rm \
        -v "${DIR}":/root \
        debian:stretch-slim \
        /bin/sh -c \
        "rm -rf /root/site/content/docs/${VERSION} && \
        cp -r /root/site/content/docs/master /root/site/content/docs/${VERSION} && \
        sed -i 's/site\/docs\/master\///g' /root/site/content/docs/${VERSION}/_index.md && \
        sed -i 's/docs\/img/img/g' /root/site/content/docs/${VERSION}/_index.md && \
        sed -i 's/sonobuoy\/tree\/master/sonobuoy\/tree\/${VERSION}/g' /root/site/content/docs/${VERSION}/_index.md && \
        sed -i 's/sonobuoy.io\/docs\/master/sonobuoy.io\/docs\/${VERSION}/g' /root/site/content/docs/${VERSION}/_index.md && \
        cp /root/site/data/docs/master-toc.yml /root/site/data/docs/${TOC_NAME}.yml && \
        perl -i -0pe 's/${CONFIG_VERSION_BLOCK}/${NEW_VERSION_BLOCK}/' /root/site/config.yaml && \
        perl -i -0pe 's/${OLD_SCOPE_BLOCK}/${NEW_SCOPE_BLOCK}/' /root/site/config.yaml && \
        perl -i -0pe 's/${OLD_TOC_BLOCK}/${NEW_TOC_BLOCK}/' /root/site/data/docs/toc-mapping.yml && \
        perl -i -0pe 's/${MASTER_FRONTMATTER}/${RELEASE_FRONTMATTER}/' /root/site/content/docs/${VERSION}/index-frontmatter.yaml && \
        perl -i -0pe 's/${MASTER_FRONTMATTER}/${RELEASE_FRONTMATTER}/' /root/site/content/docs/${VERSION}/_index.md && \
        sed -i 's/${OLD_FOOTER_BLOCK}/${NEW_FOOTER_BLOCK}/' /root/site/config.yaml"
fi
