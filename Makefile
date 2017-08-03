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

local:
	$(MAKE) -C build/sonobuoy local

test:
	$(MAKE) -C build/sonobuoy test

all: cbuild container

cbuild:
	$(MAKE) -C build/sonobuoy cbuild

container: cbuild
	$(MAKE) -C build/sonobuoy container
	$(MAKE) -C build/kube-conformance container
	$(MAKE) -C build/systemd-logs container

push:
	$(MAKE) -C build/sonobuoy push
	$(MAKE) -C build/kube-conformance push
	$(MAKE) -C build/systemd-logs push

.PHONY: all local container cbuild push test

clean:
	$(MAKE) -C build/sonobuoy clean
	$(MAKE) -C build/kube-conformance clean
	$(MAKE) -C build/systemd-logs clean
