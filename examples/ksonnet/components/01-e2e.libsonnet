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

local k = import "ksonnet.beta.2/k.libsonnet";
local c = k.core.v1.pod.mixin.spec.containersType;

local e2eConf = {
    jobName: "sonobuoy-e2e-job-{{.SessionID}}",
    pluginName: "e2e",
    annotations: {
        "sonobuoy-plugin": $.pluginName,
        "sonobuoy-driver": "Job",
        "sonobuoy-result-type": $.pluginName,
    },

    e2ec: {
        name: "e2e",
        image: "gcr.io/heptio-images/kube-conformance:latest",
    },
};


{
    pluginContainer(pluginConf, globalWorker,)::
        local e2eContainer =
            c.new(e2eConf.e2ec.name, e2eConf.e2ec.image) +
            c.imagePullPolicy("Always") +
            c.env([
                c.envType.new("E2E_FOCUS", "Pods should be submitted and removed"),
            ]) +
            c.volumeMounts([
                c.volumeMountsType.new("results", pluginConf.resultsDir),
            ]);
        e2eContainer,

    sonobuoyWorker(pluginConf, globalWorker,)::
        local sonobuoyWorker =
            c.new(globalWorker.name, globalWorker.image) +
            c.imagePullPolicy("Always") +
            c.command(globalWorker.command) +
            c.env([
                c.envType.fromFieldPath("NODE_NAME", "spec.nodeName"),
                c.envType.new("RESULTS_DIR", pluginConf.resultsDir),
                c.envType.new("MASTER_URL", "{{.MasterAddress}}"),
                c.envType.new("RESULT_TYPE", e2eConf.pluginName),
            ]) +
            c.volumeMounts([
                c.volumeMountsType.new("results", pluginConf.resultsDir),
            ]);
        sonobuoyWorker,

    pod(pluginConf, globalWorker, sonobuoyLabels, tolerations)::
        local e2epod =
            local p = k.core.v1.pod;
            p.new() +
            // Metaddata
            p.mixin.metadata.name(e2eConf.jobName) +
            p.mixin.metadata.annotations(e2eConf.annotations) +
            p.mixin.metadata.labels(sonobuoyLabels) +
            p.mixin.metadata.namespace(pluginConf.namespace) +
            // Spec
            p.mixin.spec.serviceAccountName(pluginConf.serviceAccountName) +
            p.mixin.spec.tolerations(tolerations) +
            p.mixin.spec.restartPolicy("Never") +
            p.mixin.spec.containers(
                [
                    $.pluginContainer(pluginConf, globalWorker),
                    $.sonobuoyWorker(pluginConf, globalWorker),
                ]
            ) +
            p.mixin.spec.volumes([
                p.mixin.spec.volumesType.fromEmptyDir("results"),
            ]);
        e2epod,
}
