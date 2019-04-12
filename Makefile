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
EMPTY :=
SPACE := $(EMPTY) $(EMPTY)
COMMA := $(EMPTY),$(EMPTY)

BINARY = sonobuoy
TARGET = sonobuoy
GOTARGET = github.com/heptio/$(TARGET)
GOPATH = $(shell go env GOPATH)
REGISTRY ?= gcr.io/heptio-images
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
DOCKER ?= docker
LINUX_ARCH := amd64 arm64
DOCKERFILE :=
PLATFORMS := $(subst $(SPACE),$(COMMA),$(foreach arch,$(LINUX_ARCH),linux/$(arch)))

# SECRET is used when pushing the manifest.
SECRET := $(shell echo "https://gcr.io" | docker-credential-gcr get | jq '.Secret')

# Not used for pushing images, just for local building on other GOOS. Defaults to
# grabbing from the local go env but can be set manually to avoid that requirement.
HOST_GOOS ?= $(shell go env GOOS)
HOST_GOARCH ?= $(shell go env GOARCH)
GO_SYSTEM_FLAGS ?= GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH)

# --tags allows detecting non-annotated tags as well as annotated ones
GIT_VERSION ?= $(shell git describe --always --dirty --tags)
IMAGE_VERSION ?= $(shell git describe --always --dirty --tags)
IMAGE_TAG := $(shell echo $(IMAGE_VERSION) | cut -d. -f1,2)
IMAGE_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD | sed 's/\///g')
GIT_REF_SHORT = $(shell git rev-parse --short=8 --verify HEAD)
GIT_REF_LONG = $(shell git rev-parse --verify HEAD)

ifneq ($(VERBOSE),)
VERBOSE_FLAG = -v
endif
BUILDMNT = /go/src/$(GOTARGET)
BUILD_IMAGE ?= golang:1.12.1-stretch
AMD_IMAGE ?= debian:stretch-slim
ARM_IMAGE ?= arm64v8/ubuntu:16.04

TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
COVERARGS ?= -coverprofile=coverage.txt -covermode=atomic
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...
TEST_CMD = go test $(TESTARGS)
TEST = $(TEST_CMD) $(COVERARGS) $(TEST_PKGS)

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
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(TEST)'

# Integration tests
int: sonobuoy
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(INT_TEST)'

lint:
	$(DOCKER_BUILD) '$(LINT)'

vet:
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(VET)'

pre:
	wget https://github.com/estesp/manifest-tool/releases/download/v0.9.0/manifest-tool-linux-amd64 \
	  -O manifest-tool && \
	 chmod +x ./manifest-tool

build_container:
	$(DOCKER) build \
       -t $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) \
       -t $(REGISTRY)/$(TARGET):$(IMAGE_TAG) \
       -t $(REGISTRY)/$(TARGET):$(IMAGE_BRANCH) \
       -t $(REGISTRY)/$(TARGET):$(GIT_REF_SHORT) \
       -f $(DOCKERFILE) \
		.

container: sonobuoy
	for arch in $(LINUX_ARCH); do \
		if [ $$arch = amd64 ]; then \
			sed -e 's|BASEIMAGE|$(AMD_IMAGE)|g' \
			-e 's|CMD1||g' \
			-e 's|BINARY|build/linux/amd64/sonobuoy|g' Dockerfile > Dockerfile-$$arch; \
			$(MAKE) build_container DOCKERFILE=Dockerfile-$$arch; \
			$(MAKE) build_container DOCKERFILE="Dockerfile-$$arch" TARGET="sonobuoy-$$arch"; \
	elif [ $$arch = arm64 ]; then \
			sed -e 's|BASEIMAGE|$(ARM_IMAGE)|g' \
			-e 's|CMD1||g' \
			-e 's|BINARY|build/linux/arm64/sonobuoy|g' Dockerfile > Dockerfile-$$arch; \
			$(MAKE) build_container DOCKERFILE="Dockerfile-$$arch" TARGET="sonobuoy-$$arch"; \
		else \
			echo "ARCH unknown"; \
        fi \
	done

build_sonobuoy:
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(GO_SYSTEM_FLAGS) go build -o $(BINARY) $(VERBOSE_FLAG) -ldflags="-s -w -X $(GOTARGET)/pkg/buildinfo.Version=$(GIT_VERSION) -X $(GOTARGET)/pkg/buildinfo.GitSHA=$(GIT_REF_LONG)" $(GOTARGET)'

sonobuoy:
	for arch in $(LINUX_ARCH); do \
		mkdir -p build/linux/$$arch; \
		echo Building: linux/$$arch; \
		$(MAKE) build_sonobuoy GO_SYSTEM_FLAGS="GOOS=linux GOARCH=$$arch" BINARY="build/linux/$$arch/sonobuoy"; \
	done
	@echo Building: host
	$(MAKE) build_sonobuoy

push_images:
	$(DOCKER) push $(REGISTRY)/$(TARGET):$(IMAGE_BRANCH)
	$(DOCKER) push $(REGISTRY)/$(TARGET):$(GIT_REF_SHORT)
	if git describe --tags --exact-match >/dev/null 2>&1; \
	then \
		$(DOCKER) tag $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) $(REGISTRY)/$(TARGET):$(IMAGE_TAG); \
		$(DOCKER) tag $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) $(REGISTRY)/$(TARGET):latest; \
		$(DOCKER) push $(REGISTRY)/$(TARGET):$(IMAGE_VERSION); \
		$(DOCKER) push $(REGISTRY)/$(TARGET):$(IMAGE_TAG); \
		$(DOCKER) push $(REGISTRY)/$(TARGET):latest; \
	fi

push_manifest:
	./manifest-tool --username oauth2accesstoken --password $(SECRET) push from-args --platforms $(PLATFORMS) --template $(REGISTRY)/$(TARGET)-ARCH:$(VERSION) --target $(REGISTRY)/$(TARGET):$(VERSION)

push: pre container
	for arch in $(LINUX_ARCH); do \
		$(MAKE) push_images TARGET="sonobuoy-$$arch"; \
	done

	$(MAKE) push_manifest VERSION=$(IMAGE_BRANCH) TARGET="sonobuoy"
	$(MAKE) push_manifest VERSION=$(GIT_REF_SHORT) TARGET="sonobuoy"

	if git describe --tags --exact-match >/dev/null 2>&1; \
	then \
		$(MAKE) push_manifest VERSION=$(IMAGE_VERSION) TARGET="sonobuoy"; \
		$(MAKE) push_manifest VERSION=$(IMAGE_TAG) TARGET="sonobuoy"; \
		$(MAKE) push_manifest VERSION=latest TARGET="sonobuoy"; \
	fi

clean_image:
	$(DOCKER) rmi -f `$(DOCKER) images $(REGISTRY)/$(TARGET) -a -q` || true

clean:
	rm -f $(TARGET)
	rm -f Dockerfile-*
	rm -rf build

	for arch in $(LINUX_ARCH); do \
		$(MAKE) clean_image TARGET=$(TARGET)-$$arch; \
	done

deploy_kind:
	kind load docker-image $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) || true