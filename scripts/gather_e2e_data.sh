#!/bin/sh
set -ex

sonobuoy run -p ./gathere2e/gathere2e.yaml --wait
sonobuoy retrieve -x tmpoutput
cp -f tmpoutput/plugins/gathere2e/results/global/* ../cmd/sonobuoy/app/e2e/v2/testLists
sonobuoy delete
rm -rf tmpoutput

# ./minimize_e2e_data.sh
(
  cd ../cmd/sonobuoy/app/e2e/testLists/
  find . -regex '.*[0-9]$' -exec gzip -f {} \;
)