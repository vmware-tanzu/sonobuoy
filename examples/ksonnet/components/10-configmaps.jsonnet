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
    Version: "v0.9.0",
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

local systemdconf = {
    pluginName: "systemd_logs",
    name: "sonobuoy-systemd-logs-config-{{.SessionID}}",
    annotations: {
        "sonobuoy-plugin": $.pluginName,
        "sonobuoy-driver": "DaemonSet",
        "sonobuoy-result-type": $.pluginName,
    },
    labels: {
        component: "sonobuoy",
        tier: "analysis",
        "sonobuoy-run": "{{.SessionID}}",
    },
    namespace: "{{.Namespace}}",

    serviceAccountName: "sonobuoy-serviceaccount",

    rootDir: "/node",
    resultsDir: "/tmp/results",
    selector: {
        "sonobuoy-run": "{{.SessionID}}",
    },
    systemd: {
        name: "systemd-logs",
        image: "gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest",
        command: ["sh", "-c", "/get_systemd_logs.sh && sleep 3600"],
    },
    workerContainer: {
        name: "sonobuoy-worker",
        image: "gcr.io/heptio-images/sonobuoy:master",
        command: ["sh", "-c", "/sonobuoy worker single-node -v 5 --logtostderr && sleep 3600"],
    },
};

local systemdContainer =
    c.new(systemdconf.systemd.name, systemdconf.systemd.image) +
    c.imagePullPolicy("Always") +
    c.command(systemdconf.systemd.command) +
    c.env([
        c.envType.fromFieldPath("NODE_NAME", "spec.nodeName"),
        c.envType.new("RESULTS_DIR", systemdconf.resultsDir),
        c.envType.new("CHROOT_DIR", systemdconf.rootDir),
    ]) +
    c.mixin.securityContext.privileged(true) +
    c.volumeMounts([
        c.volumeMountsType.new("results", systemdconf.resultsDir),
        c.volumeMountsType.new("root", systemdconf.rootDir),
    ]);

local systemdWorker =
    c.new(systemdconf.workerContainer.name, systemdconf.workerContainer.image) +
    c.imagePullPolicy("Always") +
    c.command(systemdconf.workerContainer.command) +
    c.env([
        c.envType.fromFieldPath("NODE_NAME", "spec.nodeName"),
        c.envType.new("RESULTS_DIR", systemdconf.resultsDir),
        c.envType.new("MASTER_URL", "{{.MasterAddress}}"),
        c.envType.new("RESULT_TYPE", systemdconf.pluginName),
    ]) +
    c.volumeMounts([
        c.volumeMountsType.new("results", systemdconf.resultsDir),
    ]);

local systemdDaemonSet =
    ds.new() +
    ds.mixin.metadata.name(systemdconf.name) +
    ds.mixin.metadata.annotations(systemdconf.annotations) +
    ds.mixin.metadata.labels(systemdconf.labels) +
    ds.mixin.metadata.namespace(systemdconf.namespace) +
    ds.mixin.spec.selector.matchLabels(systemdconf.selector) +
    ds.mixin.spec.template.metadata.labels(systemdconf.labels) +
    ds.mixin.spec.template.spec.containers([systemdContainer, systemdWorker]) +
    ds.mixin.spec.template.spec.volumes([
        ds.mixin.spec.template.spec.volumesType.fromEmptyDir("results"),
        ds.mixin.spec.template.spec.volumesType.fromHostPath("root", "/"),
    ]);


local pluginconf = {
    pluginName: "e2e",
    name: "sonobuoy-e2e-job-{{.SessionID}}",
    annotations: {
        "sonobuoy-plugin": $.pluginName,
        "sonobuoy-driver": "Job",
        "sonobuoy-result-type": $.pluginName,
    },
    labels: {
        component: "sonobuoy",
        tier: "analysis",
        "sonobuoy-run": "{{.SessionID}}",
    },
    namespace: "{{.Namespace}}",

    serviceAccountName: "sonobuoy-serviceaccount",

    resultsDir: "/tmp/results",

    e2ec: {
        name: $.pluginName,
        image: "gcr.io/heptio-images/kube-conformance:latest",
    },
    workerContainer: {
        name: "sonobuoy-worker",
        image: "gcr.io/heptio-images/sonobuoy:master",
        command: ["sh", "-c", "/sonobuoy worker global -v 5 --logtostderr"],
    },
};

local e2eContainer =
    c.new(pluginconf.e2ec.name, pluginconf.e2ec.image) +
    c.imagePullPolicy("Always") +
    c.env([
        c.envType.new("E2E_FOCUS", "Pods should be submitted and removed"),
    ]) +
    c.volumeMounts([
        c.volumeMountsType.new("results", pluginconf.resultsDir),
    ]);

local e2eWorker =
    c.new(pluginconf.workerContainer.name, pluginconf.workerContainer.image) +
    c.imagePullPolicy("Always") +
    c.command(pluginconf.workerContainer.command) +
    c.env([
        c.envType.fromFieldPath("NODE_NAME", "spec.nodeName"),
        c.envType.new("RESULTS_DIR", pluginconf.resultsDir),
        c.envType.new("MASTER_URL", "{{.MasterAddress}}"),
        c.envType.new("RESULT_TYPE", pluginconf.pluginName),
    ]) +
    c.volumeMounts([
        c.volumeMountsType.new("results", pluginconf.resultsDir),
    ]);

local e2epod =
    local p = k.core.v1.pod;
    p.new() +
    // Metaddata
    p.mixin.metadata.name(pluginconf.pluginName) +
    p.mixin.metadata.annotations(pluginconf.annotations) +
    p.mixin.metadata.labels(pluginconf.labels) +
    p.mixin.metadata.namespace(pluginconf.namespace) +
    // Spec
    p.mixin.spec.serviceAccountName(pluginconf.serviceAccountName) +
    p.mixin.spec.tolerations(tolerations) +
    p.mixin.spec.restartPolicy("Never") +
    p.mixin.spec.containers([e2eContainer, e2eWorker]) +
    p.mixin.spec.volumes([
        p.mixin.spec.volumesType.fromEmptyDir("results"),
    ]);

local plugins = {
  "systemd_logs.tmpl": kubecfg.manifestYaml(systemdDaemonSet),
  "e2e.tmpl": kubecfg.manifestYaml(e2epod),
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
