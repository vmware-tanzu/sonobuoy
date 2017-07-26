##########################################################################
# Copyright 2017 Heptio Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM buildpack-deps:jessie-scm
MAINTAINER Timothy St. Clair "tstclair@heptio.com"  

RUN apt-get update && apt-get -y --no-install-recommends install \
    ca-certificates \
    && rm -rf /var/cache/apt/* \
    && rm -rf /var/lib/apt/lists/*
COPY e2e.test /usr/local/bin/
COPY kubectl /usr/local/bin/
COPY run_e2e.sh /run_e2e.sh

ENV E2E_FOCUS="Conformance"
# NOTE: kubectl tests are temporarily disabled due to the fact that they do not use the in-cluster 
# configuration atm.  Fixes will be made upstream to resolve.
ENV E2E_SKIP="Alpha|Disruptive|Feature|Flaky|Kubectl"
ENV E2E_PROVIDER="local"
ENV RESULTS_DIR="/tmp/results"

CMD ["/bin/sh", "-c", "/run_e2e.sh"]
