#!/bin/bash

set -e

SCRIPTS_DIR="$( cd "$( dirname "$0" )" >/dev/null 2>&1 && pwd )"
DIR=$(cd $SCRIPTS_DIR; cd ..; pwd)

cluster="kind"
testImage="sonobuoy/testimage:v0.1"

if ! kind get clusters | grep -q "^$cluster$"; then
    kind create cluster --name $cluster --config $DIR/kind-config.yaml
    # Although the cluster has been created, not all the pods in kube-system are created/available
    sleep 20
fi

# Build and load the test plugin image
make -C $DIR/test/integration/testImage
kind load docker-image --name $cluster $testImage

# Run integration tests
KO_FLAGS='--platform=linux/amd64' KUBECONFIG=${HOME}/.kube/config VERBOSE=true make -C $DIR int
