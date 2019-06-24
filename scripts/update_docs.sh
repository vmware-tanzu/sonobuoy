#!/bin/bash

set -x

SCRIPTS_DIR="$( cd "$( dirname "$0" )" >/dev/null 2>&1 && pwd )"
DIR=$(cd $SCRIPTS_DIR; cd ..; pwd)
VERSION=$1
TOC_NAME=$(echo ${VERSION}toc|sed s/\\./-/g)

read -r -d '' CONFIG_VERSION_BLOCK << EOM
latest: v.*
versions:
- master
EOM

read -r -d '' NEW_VERSION_BLOCK << EOM
latest: ${VERSION}
versions:
- master
- ${VERSION}
EOM

read -r -d '' OLD_SCOPE_BLOCK << EOM
  - scope:
      path: docs\/master
    values:
      version: master
      gh: https:\/\/github.com\/heptio\/sonobuoy\/tree\/master
      layout: \"docs\"
EOM

read -r -d '' NEW_SCOPE_BLOCK << EOM
  - scope:
      path: docs\/master
    values:
      version: master
      gh: https:\/\/github.com\/heptio\/sonobuoy\/tree\/master
      layout: \"docs\"
  - scope:
      path: docs\/${VERSION}
    values:
      version: ${VERSION}
      gh: https:\/\/github.com\/heptio\/sonobuoy\/tree\/${VERSION}
      layout: \"docs\"
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
        -v ${DIR}:/root \
        debian:stretch-slim \
        /bin/sh -c \
        "rm -rf /root/site/docs/${VERSION} && \
        cp -r /root/site/docs/master /root/site/docs/${VERSION} && \
        cp /root/README.md /root/site/docs/${VERSION}/README.md && \
        sed -i 's/site\/docs\/master\///g' /root/site/docs/${VERSION}/README.md && \
        sed -i 's/docs\/img/img/g' /root/site/docs/${VERSION}/README.md && \
        sed -i 's/sonobuoy\/tree\/master/sonobuoy\/tree\/${VERSION}/g' /root/site/docs/${VERSION}/README.md && \
        cp /root/site/_data/master-toc.yml /root/site/_data/${TOC_NAME}.yml && \
        perl -i -0pe 's/${CONFIG_VERSION_BLOCK}/${NEW_VERSION_BLOCK}/' /root/site/_config.yml && \
        perl -i -0pe 's/${OLD_SCOPE_BLOCK}/${NEW_SCOPE_BLOCK}/' /root/site/_config.yml && \
        perl -i -0pe 's/${OLD_TOC_BLOCK}/${NEW_TOC_BLOCK}/' /root/site/_data/toc-mapping.yml && \
        sed -i 's/\/docs\/v.*\"/\/docs\/${VERSION}\"/' /root/site/_includes/site-header.html && \
        sed -i 's/${OLD_FOOTER_BLOCK}/${NEW_FOOTER_BLOCK}/' /root/site/_config.yml"
fi