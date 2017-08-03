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

TARGET = sonobuoy
GOTARGET = github.com/heptio/$(TARGET)
REGISTRY ?= gcr.io/heptio-images
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}/../..
VERSION ?= v0.8.0

DOCKER ?= docker

BUILDMNT = /go/src/$(GOTARGET)
BUILD_IMAGE ?= golang:1.8
BUILDCMD = go build -o $(TARGET) -v -ldflags "-X github.com/heptio/sonobuoy/pkg/buildinfo.Version=$(VERSION) -X github.com/heptio/sonobuoy/pkg/buildinfo.DockerImage=$(REGISTRY)/$(TARGET)"
BUILD = $(BUILDCMD) $(GOTARGET)/cmd/sonobuoy

TESTARGS ?= -v -timeout 60s
TEST = go test $(TEST_PKGS) $(TESTARGS)
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...

all: container

test:
	$(TEST)

local:
	$(BUILD)

container: cbuild
	$(DOCKER) build -t $(REGISTRY)/$(TARGET):latest -t $(REGISTRY)/$(TARGET):$(VERSION) .

cbuild:
	$(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT)/build/sonobuoy $(BUILD_IMAGE) /bin/sh -c '$(BUILD) && $(TEST)'

push:
	gcloud docker -- push $(REGISTRY)/$(TARGET):$(VERSION)

.PHONY: all container push

clean:
	rm -f $(TARGET)
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):latest $(REGISTRY)/$(TARGET):$(VERSION) || true
