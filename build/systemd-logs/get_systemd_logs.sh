#!/bin/bash
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
set -o errexit
set -o pipefail
set -o nounset

RESULTS_DIR="${RESULTS_DIR:-/tmp/results}"
CHROOT_DIR="${CHROOT_DIR:-/node}"
LOG_MINUTES="${LOG_MINUTES:-60}"

chroot "${CHROOT_DIR}" journalctl -o json -a --no-pager --since "${LOG_MINUTES} minutes ago" >"${RESULTS_DIR}/systemd_logs"
echo -n "${RESULTS_DIR}/systemd_logs" >"${RESULTS_DIR}/done"
