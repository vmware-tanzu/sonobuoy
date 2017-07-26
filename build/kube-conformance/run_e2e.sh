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

echo "/usr/local/bin/e2e.test --ginkgo.skip=\"${E2E_SKIP}\" --ginkgo.focus=\"${E2E_FOCUS}\" --provider=\"${E2E_PROVIDER}\" --report-dir=\"${RESULTS_DIR}\" --ginkgo.noColor=true"
/usr/local/bin/e2e.test --ginkgo.skip="${E2E_SKIP}" --ginkgo.focus="${E2E_FOCUS}" --provider="${E2E_PROVIDER}" --report-dir="${RESULTS_DIR}" --ginkgo.noColor=true | tee ${RESULTS_DIR}/e2e.log
# tar up the results for transmission back
cd ${RESULTS_DIR}
tar -czf e2e.tar.gz * 
# mark the done file as a termination notice.
echo -n ${RESULTS_DIR}/e2e.tar.gz > ${RESULTS_DIR}/done 
