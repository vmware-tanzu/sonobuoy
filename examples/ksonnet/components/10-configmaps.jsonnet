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
local c = k.core.v1.pod.mixin.spec.containersType;
local ds = k.extensions.v1beta1.daemonSet;

local e2e = import "01-e2e.libsonnet";

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
    resultsDir: "/tmp/results",
    sonobuoyImage: "gcr.io/heptio-images/sonobuoy:master",
};

local sonobuoyConfigData = {
    Description: "EXAMPLE",
    Version: "v0.10.0",
    ResultsDir: "/tmp/sonobuoy",
    Resources: [
        "CertificateSigningRequests",
        "ClusterRoleBindings",
        "ClusterRoles",
        "ComponentStatuses",
        "CustomResourceDefinitions",
        "Nodes",
        "PersistentVolumes",
        "PodSecurityPolicies",
        "ServerVersion",
        "StorageClasses",
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
        "PodDisruptionBudgets",
        "PodTemplates",
        "ReplicaSets",
        "ReplicationControllers",
        "ResourceQuotas",
        "RoleBindings",
        "Roles",
        "ServerGroups",
        "ServiceAccounts",
        "Services",
        "StatefulSets",
    ],
    Filters: {
    LabelSelector: "",
    Namespaces: ".*",
    },
    Server: {
        advertiseaddress: "sonobuoy-master:8080",
        bindaddress: "0.0.0.0",
        bindport: 8080,
        timeoutseconds: 5400,
    },
    PluginNamespace: "heptio-sonobuoy",
    Plugins: [
          { name: "systemd_logs" },
          { name: "e2e" },
    ],
};

local tolerations = [
    {
        key: "node-role.kubernetes.io/master",
        operator: "Exists",
        effect: "NoSchedule",
    },
    {
        key: "CriticalAddonsOnly",
        operator: "Exists",
    },
];

# Plugin configuration
local pluginConf = {
    resultsDir: "/tmp/results",
    namespace: "{{.Namespace}}",
    serviceAccountName: "sonobuoy-serviceaccount",
};
local globalWorker = {
    name: "sonobuoy-worker",
    image: conf.sonobuoyImage,
    command: ["sh", "-c", "/sonobuoy worker global -v 5 --logtostderr"],
};
local singleNodeWorker = {
    name: "sonobuoy-worker",
    image: conf.sonobuoyImage,
    command: ["sh", "-c", "/sonobuoy worker single-node -v 5 --logtostderr && sleep 3600"],
};
local sonobuoyLabels = {
    component: "sonobuoy",
    tier: "analysis",
    "sonobuoy-run": "{{.SessionID}}",
};

## systemd_logs plugin
local systemdconf = {
    name: "systemd-logs",
    pluginName: "systemd_logs",
    annotations: {
        "sonobuoy-plugin": $.pluginName,
        "sonobuoy-driver": "DaemonSet",
        "sonobuoy-result-type": $.pluginName,
    },
    rootDir: "/node",
    selector: {
        "sonobuoy-run": "{{.SessionID}}",
    },
    systemd: {
        name: "sonobuoy-systemd-logs-config-{{.SessionID}}",
        image: "gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest",
        command: ["sh", "-c", "/get_systemd_logs.sh && sleep 3600"],
    },
};

local systemdContainer =
    c.new(systemdconf.systemd.name, systemdconf.systemd.image) +
    c.imagePullPolicy("Always") +
    c.command(systemdconf.systemd.command) +
    c.env([
        c.envType.fromFieldPath("NODE_NAME", "spec.nodeName"),
        c.envType.new("RESULTS_DIR", pluginConf.resultsDir),
        c.envType.new("CHROOT_DIR", systemdconf.rootDir),
    ]) +
    c.mixin.securityContext.privileged(true) +
    c.volumeMounts([
        c.volumeMountsType.new("results", pluginConf.resultsDir),
        c.volumeMountsType.new("root", systemdconf.rootDir),
    ]);

local systemdWorkerContainer =
    c.new(singleNodeWorker.name, singleNodeWorker.image) +
    c.imagePullPolicy("Always") +
    c.command(singleNodeWorker.command) +
    c.env([
        c.envType.fromFieldPath("NODE_NAME", "spec.nodeName"),
        c.envType.new("RESULTS_DIR", pluginConf.resultsDir),
        c.envType.new("MASTER_URL", "{{.MasterAddress}}"),
        c.envType.new("RESULT_TYPE", systemdconf.pluginName),
    ]) +
    c.volumeMounts([
        c.volumeMountsType.new("results", pluginConf.resultsDir),
    ]);

local systemdDaemonSet =
    ds.new() +
    ds.mixin.metadata.name(systemdconf.name) +
    ds.mixin.metadata.annotations(systemdconf.annotations) +
    ds.mixin.metadata.labels(sonobuoyLabels) +
    ds.mixin.metadata.namespace(pluginConf.namespace) +
    ds.mixin.spec.selector.matchLabels(systemdconf.selector) +
    ds.mixin.spec.template.metadata.labels(sonobuoyLabels) +
    ds.mixin.spec.template.spec.containers([systemdContainer, systemdWorkerContainer]) +
    ds.mixin.spec.template.spec.tolerations(tolerations) +
    ds.mixin.spec.template.spec.hostIpc(true) +
    ds.mixin.spec.template.spec.hostNetwork(true) +
    ds.mixin.spec.template.spec.hostPid(true) +
    ds.mixin.spec.template.spec.dnsPolicy("ClusterFirstWithHostNet") +
    ds.mixin.spec.template.spec.volumes([
        ds.mixin.spec.template.spec.volumesType.fromEmptyDir("results"),
        ds.mixin.spec.template.spec.volumesType.fromHostPath("root", "/"),
    ]);

local plugins = {
  "systemd_logs.tmpl": kubecfg.manifestYaml(systemdDaemonSet),
  "e2e.tmpl": kubecfg.manifestYaml(e2e.pod(pluginConf, globalWorker, sonobuoyLabels, tolerations)),
};

local sonobuoyConfig = configMap.new() +
    configMap.mixin.metadata.name(conf.sonobuoyCfg.name) +
    configMap.mixin.metadata.namespace(conf.namespace) +
    configMap.mixin.metadata.labels(conf.labels) +
    configMap.data({ "config.json": kubecfg.manifestJson(sonobuoyConfigData) });

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
