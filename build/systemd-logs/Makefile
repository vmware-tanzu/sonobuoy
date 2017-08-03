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

TARGET = sonobuoy-plugin-systemd-logs
GOTARGET = github.com/heptio/$(TARGET)
REGISTRY ?= gcr.io/heptio-images
IMAGE = $(REGISTRY)/$(TARGET)
DOCKER ?= docker
DIR := ${CURDIR}
VERSION ?= v0.1

all: container

container: get_systemd_logs.sh
	$(DOCKER) build -t $(REGISTRY)/$(TARGET):latest -t $(REGISTRY)/$(TARGET):$(VERSION) .

push:
	gcloud docker -- push $(REGISTRY)/$(TARGET):latest $(REGISTRY)/$(TARGET):$(VERSION)

.PHONY: all container push

clean:
	$(DOCKER) rmi $(REGISTRY)/$(TARGET):latest $(REGISTRY)/$(TARGET):$(VERSION) || true
