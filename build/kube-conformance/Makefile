# Copyright 2017 Heptio Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Note the only reason we are creating this is because upstream
# does not yet publish a released e2e container
# https://github.com/kubernetes/kubernetes/issues/47920

TARGET = kube-conformance
GOTARGET = github.com/heptio/$(TARGET)
REGISTRY ?= gcr.io/heptio-images
KVER = v1.7.0
IMAGE = $(REGISTRY)/$(BIN)
DOCKER ?= docker
DIR := ${CURDIR}
TEST = go test $(TEST_PKGS) $(TESTARGS)

all: container

e2e.test: _cache/kubernetes/platforms/linux/amd64
	cp $</e2e.test $@
kubectl: _cache/kubernetes/platforms/linux/amd64
	cp $</kubectl $@

_cache/kubernetes/platforms/linux/amd64: _cache/kubernetes.tar.gz
	tar -C _cache -xzf $<
	cd _cache && KUBERNETES_DOWNLOAD_TESTS=true KUBERNETES_SKIP_CONFIRM=true ./kubernetes/cluster/get-kube-binaries.sh
	# Bump the timestamp of this directory to avoid remaking it
	touch $@

_cache/kubernetes.tar.gz: _cache
	curl -L -o $@ http://gcsweb.k8s.io/gcs/kubernetes-release/release/$(KVER)/kubernetes.tar.gz

_cache:
	mkdir -p _cache

container: e2e.test kubectl
	$(DOCKER) build -t $(REGISTRY)/$(TARGET):latest -t $(REGISTRY)/$(TARGET):$(KVER) .

push:
	gcloud docker -- push $(REGISTRY)/$(TARGET):latest $(REGISTRY)/$(TARGET):$(KVER)

.PHONY: all container

clean:
	rm -rf _cache
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):latest $(REGISTRY)/$(TARGET):$(KVER) || true
