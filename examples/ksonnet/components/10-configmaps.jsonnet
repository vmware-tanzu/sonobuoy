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
local kubecfg = import "kubecfg.libsonnet";
local configMap = k.core.v1.configMap;

local conf = {
    namespace: "heptio-sonobuoy",
    sonobuoyCfg: {
        name: "sonobuoy-config-cm",
    },
    pluginsCfg: {
        name: "sonobuoy-plugins-cm",
    },
    labels: {
        component: "sonobuoy",
    },
};

local sonobuoyConfigData = {
    Description: "EXAMPLE",
    Version: "v0.8.0",
    ResultsDir: "/tmp/sonobuoy",
    Resources: [
        "CertificateSigningRequests",
        "ClusterRoleBindings",
        "ClusterRoles",
        "ComponentStatuses",
        "Nodes",
        "PersistentVolumes",
        "PodSecurityPolicies",
        "ServerVersion",
        "StorageClasses",
        "ThirdPartyResources",
        "ConfigMaps",
        "DaemonSets",
        "Deployments",
        "Endpoints",
        "Events",
        "HorizontalPodAutoscalers",
        "Ingresses",
        "Jobs",
        "LimitRanges",
        "PersistentVolumeClaims",
        "Pods",
        "PodLogs",
        "PodDisruptionBudgets",
        "PodPresets",
        "PodTemplates",
        "ReplicaSets",
        "ReplicationControllers",
        "ResourceQuotas",
        "RoleBindings",
        "Roles",
        "ServerGroups",
        "ServiceAccounts",
        "Services",
        "StatefulSets"
    ],
    Filters: {
    LabelSelector: "",
    Namespaces: ".*"
    },
    Server: {
        advertiseaddress: "sonobuoy-master:8080",
        bindaddress: "0.0.0.0",
        bindport: 8080,
        timeoutseconds: 3600
    },
    PluginNamespace: "heptio-sonobuoy",
    Plugins: [
          {name: "systemd_logs"},
          {name: "e2e"}
    ]
};

local systemdlogsConfig = {
  name: "systemd_logs",
  driver: "DaemonSet",
  resultType: "systemd_logs",
  spec: {
    tolerations: [
      {
        key: "node-role.kubernetes.io/master",
        operator: "Exists",
        effect: "NoSchedule",
      },
      {
        key: "CriticalAddonsOnly",
        operator: "Exists",
      },
    ],
    hostNetwork: true,
    hostIPC: true,
    hostPID: true,
    dnsPolicy: "ClusterFirstWithHostNet",
    containers: [
      {
        name: "systemd-logs",
        command: [
          "sh",
          "-c",
          "/get_systemd_logs.sh && sleep 3600"
        ],
        env: [
          {
            name: "NODE_NAME",
            valueFrom: {
              fieldRef: {
                apiVersion: "v1",
                fieldPath: "spec.nodeName",
              },
            }
          },
          {
            name: "RESULTS_DIR",
            value: "/tmp/results",
          },
          {
            name: "CHROOT_DIR",
            value: "/node",
          },
        ],
        image: "gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest",
        imagePullPolicy: "Always",
        securityContext: {
          privileged: true,
        },
        volumeMounts: [
          {
            mountPath: "/node",
            name: "root",
          },
          {
            mountPath: "/tmp/results",
            name: "results",
          },
          {
            mountPath: "/etc/sonobuoy",
            name: "config",
          },
        ],
      },
      {
        name: "sonobuoy-worker",
        command: [
          "sh",
          "-c",
          "/sonobuoy worker single-node -v 5 --logtostderr && sleep 3600",
        ],
        env: [
          {
            name: "NODE_NAME",
            valueFrom: {
              fieldRef: {
                apiVersion: "v1",
                fieldPath: "spec.nodeName",
              },
            },
          },
          {
            name: "RESULTS_DIR",
            value: "/tmp/results",
          },
        ],
        image: "gcr.io/heptio-images/sonobuoy:latest",
        imagePullPolicy: "Always",
        securityContext: {
          privileged: true,
        },
        volumeMounts: [
          {
            mountPath: "/tmp/results",
            name: "results",
          },
          {
            mountPath: "/etc/sonobuoy",
            name: "config",
          },
        ],
      },
    ],
    volumes: [
      {
        name: "root",
        hostPath: {
          path: "/",
        },
      },
      {
        name: "results",
        emptyDir: {},
      },
      {
        name: "config",
        configMap: {
          name: "__SONOBUOY_CONFIGMAP__",
        },
      },
    ],
  },
};

local e2eConfig = {
  name: "e2e",
  driver: "Job",
  resultType: "e2e",
  spec: {
    serviceAccountName: "sonobuoy-serviceaccount",
    tolerations: [
      {
        key: "node-role.kubernetes.io/master",
        operator: "Exists",
        effect: "NoSchedule",
      },
      {
        key: "CriticalAddonsOnly",
        operator: "Exists",
      },
    ],
    restartPolicy: "Never",
    containers: [
      {
        name: "e2e",
        image: "gcr.io/heptio-images/kube-conformance:latest",
        imagePullPolicy: "Always",
        # NOTE: Full conformance can take a while depending on your cluster size.
        # As a result, only a single test is set atm to verify correctness.
        # Operators that want the complete test results can comment out the
        # env section.
        env: [
          {
            name: "E2E_FOCUS",
            value: "Pods should be submitted and removed",
          },
        ],
        volumeMounts: [
          {
            name: "results",
            mountPath: "/tmp/results",
          },
        ],
      },
      {
        name: "sonobuoy-worker",
        command: [
          "sh",
          "-c",
          "/sonobuoy worker global -v 5 --logtostderr",
        ],
        env: [
          {
            name: "NODE_NAME",
            valueFrom: {
              fieldRef: {
                apiVersion: "v1",
                fieldPath: "spec.nodeName",
              },
            },
          },
          {
            name: "RESULTS_DIR",
            value: "/tmp/results",
          },
        ],
        image: "gcr.io/heptio-images/sonobuoy:latest",
        imagePullPolicy: "Always",
        volumeMounts: [
          {
            name: "config",
            mountPath: "/etc/sonobuoy",
          },
          {
            name: "results",
            mountPath: "/tmp/results",
          },
        ],
      },
    ],
    volumes: [
      {
        name: "results",
        emptyDir: {},
      },
      {
        name: "config",
        configMap: {
          # This will be rewritten when the JobPlugin driver goes to launch the pod.
          name: "__SONOBUOY_CONFIGMAP__",
        },
      },
    ],
  },
};

local plugins = {
  "systemdlogs.yaml": kubecfg.manifestYaml(systemdlogsConfig),
  "e2e.yaml": kubecfg.manifestYaml(e2eConfig),
};

local sonobuoyConfig = configMap.new() +
    configMap.mixin.metadata.name(conf.sonobuoyCfg.name) +
    configMap.mixin.metadata.namespace(conf.namespace) +
    configMap.mixin.metadata.labels(conf.labels) +
    configMap.data({"config.json": kubecfg.manifestJson(sonobuoyConfigData)});

local pluginConfigs = configMap.new() +
    configMap.mixin.metadata.name(conf.pluginsCfg.name) +
    configMap.mixin.metadata.namespace(conf.namespace) +
    configMap.mixin.metadata.labels(conf.labels) +
    configMap.data(
        plugins
    );

k.core.v1.list.new([
    sonobuoyConfig,
    pluginConfigs,
])
