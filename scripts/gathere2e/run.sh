#!/bin/sh

# In order to sort by creatordate we need to clone part of object data, not use ls-remote.
git init repo
cd repo
git config extensions.partialClone true
git remote add origin https://github.com/kubernetes/kubernetes
git fetch --filter=blob:none --tags --depth=1 origin

# Consider recent tags
git tag -l --sort=-creatordate |
  grep -v "alpha\|beta\|rc" |
  head -n75|sort|xargs -t -I % sh -c \
  'echo % >> ./tmpversions.txt'

# Compare against the versions we already have data for
curl https://api.github.com/repos/vmware-tanzu/sonobuoy/contents/cmd/sonobuoy/app/e2e/testLists?ref=main | \
  jq -r '.[]|.name' | \
  cut -d'.' -f 1,2,3 > existingversions.txt

comm -13 existingversions.txt tmpversions.txt > newversions.txt

mkdir tmpplugins
cat newversions.txt|xargs -t -I % sh -c \
  'sonobuoy gen plugin e2e --plugin-env=e2e.E2E_EXTRA_ARGS= --plugin-env=e2e.E2E_DRYRUN=true --kubernetes-version=% | sed "s/plugin-name: e2e/plugin-name: e2e%/" > ./tmpplugins/p%.yaml'

sonobuoy run -p ./tmpplugins -n gathere2e --wait
sonobuoy retrieve -f output.tar.gz -n gathere2e
cat newversions.txt | xargs -t -I % sh -c \
  "sonobuoy results output.tar.gz -p e2e% --mode=detailed | jq .name -r | sort > ${SONOBUOY_RESULTS_DIR}/%"

sonobuoy delete -n gathere2e