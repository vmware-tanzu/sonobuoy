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

FROM BASEIMAGE
MAINTAINER Timothy St. Clair "tstclair@heptio.com"

CMD1

ADD BINARY /sonobuoy
ADD scripts/run_master.sh /run_master.sh
ADD scripts/run_single_node_worker.sh /run_single_node_worker.sh
WORKDIR /
CMD ["/bin/sh", "-c", "/run_master.sh"]
