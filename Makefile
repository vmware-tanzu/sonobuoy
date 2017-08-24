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
#
# Note the only reason we are creating this is because upstream
# does not yet publish a released e2e container
# https://github.com/kubernetes/kubernetes/issues/47920

EXAMPLE_FILES = $(wildcard examples/ksonnet/components/*.jsonnet)
EXAMPLE_OUTPUT = examples/quickstart/aggregate.yaml $(patsubst examples/ksonnet/components/%.jsonnet,examples/quickstart/components/%.yaml,$(EXAMPLE_FILES))
KSONNET_BUILD_IMAGE = ksonnet/ksonnet-lib:beta.2

TARGET = sonobuoy
GOTARGET = github.com/heptio/$(TARGET)
REGISTRY ?= gcr.io/heptio-images
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
DOCKER ?= docker

GIT_VERSION ?= $(shell git describe --always --dirty)
IMAGE_VERSION ?= $(shell git describe --always --dirty | sed 's/^v//')
IMAGE_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/\///g')
GIT_REF = $(shell git rev-parse --short=8 --verify HEAD)

BUILDMNT = /go/src/$(GOTARGET)
BUILD_IMAGE ?= golang:1.8
BUILDCMD = go build -o $(TARGET) -v -ldflags "-X github.com/heptio/sonobuoy/pkg/buildinfo.Version=$(GIT_VERSION) -X github.com/heptio/sonobuoy/pkg/buildinfo.DockerImage=$(REGISTRY)/$(TARGET):$(GIT_REF)"
BUILD = $(BUILDCMD) $(GOTARGET)/cmd/sonobuoy

TESTARGS ?= -v -timeout 60s
TEST = go test $(TEST_PKGS) $(TESTARGS)
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...

WORKDIR ?= /sonobuoy
RBAC_ENABLED ?= 1
KUBECFG_CMD = $(DOCKER) run \
  -v $(DIR):$(WORKDIR) \
	--workdir $(WORKDIR) \
	--rm \
	$(KSONNET_BUILD_IMAGE) \
	kubecfg show -o yaml -V RBAC_ENABLED=$(RBAC_ENABLED) -J $(WORKDIR) -o yaml $< > $@

.PHONY: all container push clean cbuild test local generate-examples

all: container

test:
	$(TEST)

local:
	$(BUILD)

container: cbuild
	$(DOCKER) build \
		-t $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) \
		-t $(REGISTRY)/$(TARGET):$(IMAGE_BRANCH) \
		-t $(REGISTRY)/$(TARGET):$(GIT_REF) \
		.

cbuild:
	$(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c '$(BUILD) && $(TEST)'

push:
	$(DOCKER) push $(REGISTRY)/$(TARGET):$(IMAGE_BRANCH)
	$(DOCKER) push $(REGISTRY)/$(TARGET):$(GIT_REF)
	if git describe --tags --exact-match >/dev/null 2>&1; \
	then \
		$(DOCKER) tag $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) $(REGISTRY)/$(TARGET):latest; \
		$(DOCKER) push $(REGISTRY)/$(TARGET):$(IMAGE_VERSION); \
		$(DOCKER) push $(REGISTRY)/$(TARGET):latest; \
	fi

clean:
	rm -f $(TARGET)
	$(DOCKER) rmi $(REGISTRY)/$(TARGET) || true
	find ./examples/ -type f -name '*.yaml' -delete

generate-examples: latest-ksonnet $(EXAMPLE_OUTPUT)

examples/quickstart/components/%.yaml: examples/ksonnet/components/%.jsonnet
	$(KUBECFG_CMD)

examples/quickstart/%.yaml: examples/ksonnet/%.jsonnet
	$(KUBECFG_CMD)

latest-ksonnet:
	$(DOCKER) pull $(KSONNET_BUILD_IMAGE)
