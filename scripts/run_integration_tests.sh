#!/bin/bash

set -e

SCRIPTS_DIR="$( cd "$( dirname "$0" )" >/dev/null 2>&1 && pwd )"
DIR=$(cd $SCRIPTS_DIR; cd ..; pwd)

cluster="kind"
testImage="sonobuoy/testimage:v0.1"

if ! kind get clusters | grep -q "^$cluster$"; then
    kind create cluster --name $cluster
    # Although the cluster has been created, not all the pods in kube-system are created/available
    sleep 20
fi

# Build and load the test plugin image
make -C $DIR/test/integration/testImage
kind load docker-image --name $cluster $testImage

# Build and load the sonobuoy image and run integration tests
make -C $DIR KIND_CLUSTER=$cluster deploy_kind
KUBECONFIG="$(kind get kubeconfig-path --name="$cluster")" VERBOSE=true make -C $DIR int
