#!/bin/sh
set -ex

# Note: I've experienced flakiness here before where the files came over but had 0 data; so I think it was a small race
# between some of these commands somehow. If necessary, you can run them manually and confirm they have the expected data.
sonobuoy run -p ./gathere2e/gathere2e.yaml --wait
sonobuoy retrieve -x tmpoutput
cp -f tmpoutput/plugins/gathere2e/results/global/* ../cmd/sonobuoy/app/e2e/testLists
(
  cd ../cmd/sonobuoy/app/e2e/testLists/
  find . -regex '.*[0-9]$' -exec gzip -f {} \;
)

# Uncomment if you want cleanup to automatically occur rather than be manual.
#sonobuoy delete
#rm -rf tmpoutput
