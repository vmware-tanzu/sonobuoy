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

TARGET = sonobuoy
GOTARGET = github.com/heptio/$(TARGET)
BUILDMNT = /go/src/$(GOTARGET)
REGISTRY ?= gcr.io/heptio-images
VERSION ?= v0.8.0
TESTARGS ?= -v -timeout 60s
IMAGE = $(REGISTRY)/$(BIN)
BUILD_IMAGE ?= golang:1.8
TEST_PKGS ?= ./cmd/... ./pkg/...
DOCKER ?= docker
DIR := ${CURDIR}
BUILDCMD = go build -v -ldflags "-X github.com/heptio/sonobuoy/pkg/buildinfo.Version=$(VERSION) -X github.com/heptio/sonobuoy/pkg/buildinfo.DockerImage=$(REGISTRY)/$(TARGET)"
BUILD = $(BUILDCMD) ./cmd/sonobuoy
TEST = go test $(TEST_PKGS) $(TESTARGS)

local:
	$(BUILD)

test:
	$(TEST)

all: cbuild container

cbuild:
	$(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c '$(BUILD) && $(TEST)'

container: cbuild
	$(DOCKER) build -t $(REGISTRY)/$(TARGET):latest -t $(REGISTRY)/$(TARGET):$(VERSION) .
	$(MAKE) -C build/kube-conformance container
	$(MAKE) -C build/systemd-logs container

push:
	gcloud docker -- push $(REGISTRY)/$(TARGET):$(VERSION)
	$(MAKE) -C build/kube-conformance push
	$(MAKE) -C build/systemd-logs push

.PHONY: all local container cbuild push test

clean:
	$(MAKE) -C build/kube-conformance clean
	$(MAKE) -C build/systemd-logs clean
	rm -f $(TARGET)
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):latest $(REGISTRY)/$(TARGET):$(VERSION)

