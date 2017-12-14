local k = import "ksonnet.beta.2/k.libsonnet";
local c = k.core.v1.pod.mixin.spec.containersType;
local ds = k.extensions.v1beta1.daemonSet;

local conf = {
    sonobuoyImage: "gcr.io/heptio-images/sonobuoy:master",
};
local pluginConf = {
    resultsDir: "/tmp/results",
    namespace: "{{.Namespace}}",
    serviceAccountName: "sonobuoy-serviceaccount",
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

systemdDaemonSet
