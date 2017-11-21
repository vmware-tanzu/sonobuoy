local k = import "ksonnet.beta.2/k.libsonnet";
local c = k.core.v1.pod.mixin.spec.containersType;

local conf = {
    sonobuoyImage: "gcr.io/heptio-images/sonobuoy:master",
};

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

local e2eContainer =
    c.new(e2eConf.e2ec.name, e2eConf.e2ec.image) +
    c.imagePullPolicy("Always") +
    c.env([
        c.envType.new("E2E_FOCUS", "Pods should be submitted and removed"),
    ]) +
    c.volumeMounts([
        c.volumeMountsType.new("results", pluginConf.resultsDir),
    ]);

local e2eworkerContainer =
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
    p.mixin.spec.containers([e2eContainer, e2eworkerContainer]) +
    p.mixin.spec.volumes([
        p.mixin.spec.volumesType.fromEmptyDir("results"),
    ]);


e2epod
