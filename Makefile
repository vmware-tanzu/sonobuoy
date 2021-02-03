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
GOTARGET = github.com/vmware-tanzu/$(TARGET)
GOPATH = $(shell go env GOPATH)
REGISTRY ?= sonobuoy
IMAGE = $(REGISTRY)/$(TARGET)
DIR := ${CURDIR}
DOCKER ?= docker
LINUX_ARCH := amd64 arm64
WIN_ARCH := amd64
DOCKERFILE :=
KIND_CLUSTER = kind

KO ?= ko
KO_FLAGS ?= --platform=all

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
BUILD_IMAGE ?= golang:1.15-buster
AMD_IMAGE ?= gcr.io/distroless/static-debian10:latest
ARM_IMAGE ?= arm64v8/ubuntu:16.04
WIN_IMAGE ?= mcr.microsoft.com/windows/servercore:1809

TESTARGS ?= $(VERBOSE_FLAG) -timeout 60s
COVERARGS ?= -coverprofile=coverage.txt -covermode=atomic
TEST_PKGS ?= $(GOTARGET)/cmd/... $(GOTARGET)/pkg/...
TEST_CMD = GODEBUG=x509ignoreCN=0 go test $(TESTARGS)
TEST = $(TEST_CMD) $(COVERARGS) $(TEST_PKGS)

INT_TEST_PKGS ?= $(GOTARGET)/test/integration/...
INT_TEST= $(TEST_CMD) $(INT_TEST_PKGS) -tags=integration

STRESS_TEST_PKGS ?= $(GOTARGET)/test/stress/...
STRESS_TEST= $(TEST_CMD) $(STRESS_TEST_PKGS)

VET = go vet $(TEST_PKGS)

DOCKER_FLAGS =
DOCKER_BUILD ?= $(DOCKER) run --rm -v $(DIR):$(BUILDMNT) $(DOCKER_FLAGS) -w $(BUILDMNT) $(BUILD_IMAGE) /bin/sh -c
GO_BUILD ?= CGO_ENABLED=0 $(GO_SYSTEM_FLAGS) go build -o $(BINARY) $(VERBOSE_FLAG) -ldflags="-s -w -X $(GOTARGET)/pkg/buildinfo.Version=$(GIT_VERSION) -X $(GOTARGET)/pkg/buildinfo.GitSHA=$(GIT_REF_LONG)" $(GOTARGET)
PUSH_WINDOWS ?= false

# Kind images
K8S_PATH ?= $(GOPATH)/src/github.com/kubernetes/kubernetes
KIND_K8S_TAG ?= $(shell cd $(K8S_PATH) && git describe)

.PHONY: all build_container linux_containers windows_containers build_sonobuoy push push_images push_manifest clean clean_image test local-test local int lint stress vet pre native deploy_kind kind_images push_kind_images check-kind-env

all: linux_containers

local-test:
	$(TEST)

# Unit tests
test:
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(TEST)'

# Stress tests
stress:
	$(DOCKER_BUILD) 'CGO_ENABLED=0 $(STRESS_TEST)'

build_sonobuoy:
	$(DOCKER_BUILD) '$(GO_BUILD)'

# Integration tests
int: ARTIFACTS_DIR=/tmp/artifacts KUBECONFIG=$(KUBECONFIG) SONOBUOY_CLI=$(SONOBUOY_CLI)
int: TESTARGS=$(VERBOSE_FLAG) -timeout 3m
int: native
	KO_DOCKER_REPO=ko.local \
	KIND_CLUSTER_NAME=$(KIND_CLUSTER) \
	$(KO) publish --platform=linux/amd64 --tags $(GIT_VERSION) --base-import-paths github.com/vmware-tanzu/sonobuoy
	docker tag ko.local/sonobuoy:$(GIT_VERSION) sonobuoy/sonobuoy:$(GIT_VERSION)
	kind load docker-image sonobuoy/sonobuoy:$(GIT_VERSION)
	CGO_ENABLED=0 $(INT_TEST)

build_container:
	$(KO) publish $(KO_FLAGS) --base-import-paths \
		github.com/vmware-tanzu/sonobuoy

native:
	$(GO_BUILD)

deploy_kind:
	KO_DOCKER_REPO=kind.local \
	KIND_CLUSTER_NAME=$(KIND_CLUSTER) \
	$(KO) publish $(KO_FLAGS) --base-import-paths \
		github.com/vmware-tanzu/sonobuoy


# sonobuoy run --sonobuoy-image azd2k.azurecr.io/sonobuoy:latest
