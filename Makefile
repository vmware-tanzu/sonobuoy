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

BINARY = sonobuoy
TARGET = sonobuoy
GOTARGET = github.com/heptio/$(TARGET)
REGISTRY ?= gcr.io/heptio-images
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
DOCKER ?= docker
LINUX_ARCH := amd64 arm64

GIT_VERSION ?= $(shell git describe --always --dirty)
IMAGE_VERSION ?= $(shell git describe --always --dirty)
IMAGE_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/\///g')
GIT_REF = $(shell git rev-parse --short=8 --verify HEAD)

ifneq ($(VERBOSE),)
VERBOSE_FLAG = -v
endif
BUILDMNT = /go/src/$(GOTARGET)
BUILD_IMAGE ?= golang:1.10-alpine

TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...
TEST_CMD = go test $(TESTARGS)
TEST = $(TEST_CMD) $(TEST_PKGS)

INT_TEST_PKGS ?= $(GOTARGET)/test/...
INT_TEST= $(TEST_CMD) $(INT_TEST_PKGS)

VET = go vet $(TEST_PKGS)

# Vendor this someday
GOLINT_FLAGS ?= -set_exit_status
LINT = golint $(GOLINT_FLAGS) $(TEST_PKGS)

WORKDIR ?= /sonobuoy
DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c

.PHONY: all container push clean test local-test local generate plugins int

all: container

local-test:
	$(TEST)

# Unit tests
test: sonobuoy vet
	$(DOCKER_BUILD) '$(TEST)'

# Integration tests
int: sonobuoy
	$(DOCKER_BUILD) '$(INT_TEST)'

lint:
	$(DOCKER_BUILD) '$(LINT)'

vet:
	$(DOCKER_BUILD) '$(VET)'

container: sonobuoy
	$(DOCKER) build \
		-t $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) \
		-t $(REGISTRY)/$(TARGET):$(IMAGE_BRANCH) \
		-t $(REGISTRY)/$(TARGET):$(GIT_REF) \
		.

build_sonobuoy:
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(SYSTEM) go build -o $(BINARY) $(VERBOSE_FLAG) -ldflags="-s -w -X github.com/heptio/sonobuoy/pkg/buildinfo.Version=$(GIT_VERSION)" $(GOTARGET)'

sonobuoy:
	for arch in $(LINUX_ARCH); do \
		mkdir -p build/linux/$$arch; \
		echo Building: linux/$$arch; \
		$(MAKE) build_sonobuoy SYSTEM="GOOS=linux GOARCH=$$arch" BINARY="build/linux/$$arch/sonobuoy"; \
	done
	@echo Building: host
	make build_sonobuoy

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
	rm -rf build
	$(DOCKER) rmi $(REGISTRY)/$(TARGET) || true
