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

local heptioE2Econf = {
    jobName: "sonobuoy-heptio-e2e-job-{{.SessionID}}",
    pluginName: "heptio-e2e",
    annotations: {
        "sonobuoy-plugin": $.pluginName,
        "sonobuoy-driver": "Job",
        "sonobuoy-result-type": $.pluginName,
    },
    e2ec: {
        name: "heptio-e2e",
        image: "gcr.io/heptio-images/heptio-e2e:master",
    },
};

local heptioE2EContainer =
    c.new(heptioE2Econf.e2ec.name, heptioE2Econf.e2ec.image) +
    c.imagePullPolicy("Always") +
    c.volumeMounts([
        c.volumeMountsType.new("results", pluginConf.resultsDir),
    ]);

local heptioE2EWorker =
    c.new(globalWorker.name, globalWorker.image) +
    c.imagePullPolicy("Always") +
    c.command(globalWorker.command) +
    c.env([
        c.envType.fromFieldPath("NODE_NAME", "spec.nodeName"),
        c.envType.new("RESULTS_DIR", pluginConf.resultsDir),
        c.envType.new("MASTER_URL", "{{.MasterAddress}}"),
        c.envType.new("RESULT_TYPE", heptioE2Econf.pluginName),
    ]) +
    c.volumeMounts([
        c.volumeMountsType.new("results", pluginConf.resultsDir),
    ]);

local heptioe2ePod =
    local p = k.core.v1.pod;
    p.new() +
    // Metaddata
    p.mixin.metadata.name(heptioE2Econf.jobName) +
    p.mixin.metadata.annotations(heptioE2Econf.annotations) +
    p.mixin.metadata.labels(sonobuoyLabels) +
    p.mixin.metadata.namespace(pluginConf.namespace) +
    // Spec
    p.mixin.spec.serviceAccountName(pluginConf.serviceAccountName) +
    p.mixin.spec.tolerations(tolerations) +
    p.mixin.spec.restartPolicy("Never") +
    p.mixin.spec.containers([heptioE2EContainer, heptioE2EWorker]) +
    p.mixin.spec.volumes([
        p.mixin.spec.volumesType.fromEmptyDir("results"),
    ]);


heptioe2ePod
