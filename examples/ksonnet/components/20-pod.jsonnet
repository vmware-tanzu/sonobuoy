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
local values = std.extVar("values");
local conf(opts) = {
    namespace: "heptio-sonobuoy",
    selector: {
        run: "sonobuoy-master",
    },
    labels: $.selector {
        component: $.pod.name,
    },
    pod: {
        name: "sonobuoy",
        labels: $.labels {
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
      image: "gcr.io/heptio-images/sonobuoy:master",
      command: [
        "/bin/bash",
        "-c",
        std.join("", [
          "/sonobuoy master --no-exit=true -v ",
          if opts.debug then "5" else "3",
          " --logtostderr",
          if opts.debug then " --debug" else "",
        ]),
      ],
      imagePullPolicy: opts.pullPolicy,
      volumeMounts: [
        {
          name: $.volumes[0].name,
          mountPath: "/etc/sonobuoy",
        },
        {
          name: $.volumes[1].name,
          mountPath: "/plugins.d",
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
            configMap: { name: "sonobuoy-config-cm" },
        },
        {
            name: "sonobuoy-plugins-volume",
            configMap: { name: "sonobuoy-plugins-cm" },
        },
        {
            name: "output-volume",
            emptyDir: {},
        },
    ],
    name: "sonobuoy",
};


{
  objects(pullPolicy="Always", debug=false)::
    local opts = {
      pullPolicy: pullPolicy,
      debug: debug,
    };
    local myconf = conf(opts);

    local pod = k.core.v1.pod;
    local sonobuoyPod =
      pod.new() +
      pod.mixin.metadata.name(myconf.pod.name) +
      pod.mixin.metadata.namespace(myconf.namespace) +
      pod.mixin.metadata.labels(myconf.pod.labels) +
      pod.mixin.spec.restartPolicy(myconf.pod.restartPolicy) +
      pod.mixin.spec.serviceAccountName(myconf.pod.serviceAccountName) +
      pod.mixin.spec.containers([
        myconf.master +
        pod.mixin.spec.containersType.env([
          pod.mixin.spec.containersType.envType.fromFieldPath("SONOBUOY_ADVERTISE_IP", "status.podIP"),
        ]),
      ]) +
      pod.mixin.spec.volumes(myconf.volumes);

    local svc = k.core.v1.service;
    local sonobuoyService =
      svc.new(myconf.service.name, myconf.selector, [myconf.service.port],) +
      svc.mixin.metadata.namespace(myconf.namespace) +
      svc.mixin.metadata.labels(myconf.labels) +
      svc.mixin.spec.type(myconf.service.type);

    k.core.v1.list.new([sonobuoyPod, sonobuoyService]),
}
