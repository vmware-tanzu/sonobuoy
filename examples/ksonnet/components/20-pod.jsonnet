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

local conf = {
    namespace: "heptio-sonobuoy",
    selector: {
        run: "sonobuoy-master",
    },
    labels: $.selector + {
        component: $.pod.name,
    },
    pod: {
        name: "sonobuoy",
        labels: $.labels + {
            tier: "analysis",
        },
        restartPolicy: "Never",
        serviceAccountName: "sonobuoy-serviceaccount",
    },
    service: {
        name: "sonobuoy-master",
        port: {
            port: 8080,
            protocol: "TCP",
            targetPort: 8080,
        },
        type: "ClusterIP",
    },
    master: {
        name: "kube-sonobuoy",
        command: ["/bin/bash", "-c", "/sonobuoy master --no-exit=true -v 3 --logtostderr"],
        image: "gcr.io/heptio-images/sonobuoy:latest",
        imagePullPolicy: "Always",
        volumeMounts: [
            {
                name: $.volumes[0].name,
                mountPath: "/etc/sonobuoy",
            },
            {
                name: $.volumes[1].name,
                mountPath: "/etc/sonobuoy/plugins.d",
            },
            {
                name: $.volumes[2].name,
                mountPath: "/tmp/sonobuoy",
            },
        ],
    },
    volumes: [
        {
            name: "sonobuoy-config-volume",
            configMap: {name: "sonobuoy-config-cm"},
        },
        {
            name: "sonobuoy-plugins-volume",
            configMap: {name: "sonobuoy-plugins-cm"},
        },
        {
            name: "output-volume",
            emptyDir: {},
        },
    ],
    name: "sonobuoy",
};

local sonobuoyPod = local pod = k.core.v1.pod;
    pod.new() +
    pod.mixin.metadata.name(conf.pod.name) +
    pod.mixin.metadata.namespace(conf.namespace) +
    pod.mixin.metadata.labels(conf.pod.labels) +
    pod.mixin.spec.restartPolicy(conf.pod.restartPolicy) +
    pod.mixin.spec.serviceAccountName(conf.pod.serviceAccountName) +
    pod.mixin.spec.containers([
        conf.master +
            pod.mixin.spec.containersType.env([
               pod.mixin.spec.containersType.envType.fromFieldPath("SONOBUOY_ADVERTISE_IP", "status.podIP")
            ])
    ]) +
    pod.mixin.spec.volumes(conf.volumes);

local sonobuoyService = local svc = k.core.v1.service;
    svc.new(conf.service.name, conf.selector, [conf.service.port],) +
    svc.mixin.metadata.namespace(conf.namespace) +
    svc.mixin.metadata.labels(conf.labels) +
    svc.mixin.spec.type(conf.service.type);

k.core.v1.list.new([sonobuoyPod, sonobuoyService])
