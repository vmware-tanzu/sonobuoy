# Configuration
* [Overview][0]
* [Data configuration][1]
  * [Sample JSON][2]
  * [Parameter reference][3]
* [Plugin configuration][4]
* [Kubernetes component definitions][5]
  * [Overview][6]
  * [RBAC][7]
  * [Pod][8]

## Overview

Sonobuoy's configuration can be split into three conceptual parts: (1) data configuration (2) plugin configuration (3) Kubernetes component definitions. Their file format depends on how Sonobuoy is run:
* **Standalone**: JSON files that live on a cluster node
* **Containerized**: YAML manifests that are pushed to the API Server using `kubectl`

Once the configs are loaded (in either case), Sonobuoy parses them and gathers data accordingly. If plugins are specified, Sonobuoy [submits worker pods to collect node-specific information][9].


| | Overview|Path on Cluster Node|[STANDALONE]<br>JSON example(s)|[CONTAINERIZED]<br>YAML manifest example(s)
|---|---|---|---|---|
|*Data configuration*| What Sonobuoy records, how, and where. |*ANY of the following*:<br>(1) `config.json` in the directory where `sonobuoy` is executed<br>(2) `/etc/sonobuoy/config.json`<br>(3) `$SONOBUOY_CONFIG`<br><br>|[`config.json`][10]|<br> [`examples/quickstart/components/10-configmaps.yaml`][11]<br><br>*The YAML file is basically a wrapper for the `config.json` file, which allows it to be properly mounted onto the cluster's Sonobuoy pod.* <br><br>
|*Plugin configuration*|Settings for each plugin integration.|*ANY of the following*:<br>(1) `/etc/sonobuoy/plugins.d`<br>(2) `$HOME/.sonobuoy/plugins.d`<br>(3) `./plugins.d`<br>(4) `PluginSearchPath` (override from the data configuration) <br><br>| There is a YAML config for each plugin:<br>(1) [`plugins.d/e2e.yaml`][16]<br>(2)[`plugins.d/systemdlogs.yaml`][17]|<br>[`examples/quickstart/components/10-configmaps.yaml`][11]<br><br>*Same comment about the YAML file as above.*
|*Kubernetes component definitions*|The various K8s objects that need to be defined for Sonobuoy to run as a containerized app.|N/A (manifest only)|N/A|The example splits this into two manifests:<br>(1) [`examples/quickstart/components/00-rbac.yaml`][12]<br>(2) [`examples/quickstart/components/20-pod.yaml`][13]|



*NOTE: The configuration for the containerized example is split into three manifests for the reader's sake---one covering the data and plugin configs, and two covering the Kubernetes component definitions. However, all of these specifications would still work if consolidated into one YAML file (as in [`examples/quickstart/aggregate.yaml`][18]), as long as any RBAC settings are defined first. *


## Data configuration

### Sample JSON
See [Parameter Reference][3] for a more detailed description of each setting.
```json
{
    "UUID": "12345-something-unique",
    "Description": "EXAMPLE",
    "Version": "v0.3.0",
    "Kubeconfig": "~/.kube/config",
    "ResultsDir": "./results",
    "Resources": [
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
        "Secrets",
        "ServerGroups",
        "ServiceAccounts",
        "Services",
        "StatefulSets"
    ],
    "Filters": {
        "LabelSelector": "",
        "Namespaces": ".*"
    },
    "Limits": {
        "PodLogs": {
            "LimitTime": "24h"
        }
    },
    "Server": {
        "advertiseaddress": "",
        "bindaddress": "0.0.0.0",
        "bindport": 8080,
        "timeoutseconds": 300
    },
    "Plugins": [
        {"name": "e2e"},
        {"name": "systemd_logs"}
    ],
    "PluginSearchPath": "override-path"
}
```
### Parameter reference
| Key  | Type | Default | Meaning  |
|---|---|---|---|
|  UUID | String |  A randomly generated UUID | Used to name the directory where Sonobuoy-collected data is saved (e.g. `sonobuoy_<UUID>`)  |
| Description | String | "DEFAULT" | Metadata for keeping track of configs and their use cases |
| Version | String | `buildInfo.version` (comes from the compilation of the Docker image) | The Sonobuoy version that the config uses for its schema |
| Kubeconfig | String | ""<br><br>*NOT necessary for containerized Sonobuoy, which will default to the in-cluster config* | Allows Sonobuoy to communicate with and gather info from the Kubernetes cluster |
| ResultsDir | String | "./results" | The directory in which Sonobuoy writes its results. Customizable with `RESULTS_DIR` environment variable. |
| Resources | String Array | An array containing all possible resources | *See the [sample JSON][2] above for a list of all available resource types.*<br><br>Indicates to Sonobuoy what type of data it should be recording |
| Filters.LabelSelector | String | "" | Uses standard Kubernetes [label selector syntax][14] to filter which resource objects are recorded |
| Filters.Namespaces | String | ".*" | Uses regex on namespaces to filter which resource objects are recorded |
| Limits.PodLogs.LimitTime | String | "" | Limits how far back in time to gather Pod Logs, leave blank for no limit (e.g. "24h", "60m". See https://golang.org/pkg/time/#ParseDuration for details.) |
| Limits.PodLogs.LimitSize | String | "" | Limits the size of Pod Logs to gather, per container, leave blank for no limit (e.g. "10 MB", "1 GB", etc.) |
| Server.advertiseaddress | String | `$SONOBUOY_ADVERTISE_IP` &#124;&#124; the current server's `os.Hostname()`| *Only used if Sonobuoy dispatches agent pods to collect node-specific information*<br><br>The IP address that remote Sonobuoy agents send information back to, in order for disparate data to be aggregated into a single report |
| Server.bindaddress | String | "0.0.0.0" | *See `Server.advertiseaddress` for context.*<br><br>If data aggregation is required, an HTTP server is started to handle the worker requests. This is the address that server binds to. |
| Server.bindport | Int | 8080 | The port for the HTTP server mentioned in *Server.bindaddress*. |
| Server.timeoutseconds | Int | 300 (5 min) | *See `Server.advertiseaddress` for context.*<br><br>This determines how long the master Sonobuoy pod should wait to hear back from the dispatched agents. |
| Plugins | Array of plugin descriptions: `{"name": <PLUGIN_NAME>}` | `[]` | The list of Sonobuoy plugins enabled for custom data collection. See the [plugins reference][9] for details.|
| PluginSearchPath | String Array | `"./plugins.d", "/etc/sonobuoy/plugins.d", "~/sonobuoy/plugins.d"` | The paths where Sonobuoy should look for its plugin configs

## Plugin configuration

If you have the appropriate plugin definition (third-party or your own), you can easily opt into it by adding a *Plugin Selection* to the main Sonobuoy config's "Plugin" section.

The following code snippet, for example, selects the `e2e` and `systemd_logs` plugins:

```
    "Plugins": [
        {"name": "e2e"},
        {"name": "systemd_logs"}
    ]
```

For more details on creating custom plugins, see the [plugin reference][9].

## Kubernetes component definitions

### Overview

*This section of the configuration is only applicable when running Sonobuoy as a containerized pod.*

While the other configuration sections control Sonobuoy's data collection, this part defines the Kubernetes resources required to actually run Sonobuoy on your cluster. In the [`examples/quickstart/components`][19] example, it is split into two manifests:
1. `00-rbac.yaml`: This sets up gives Sonobuoy the necessary permissions to query the API server.
2. `20-pod.yaml`: This sets up the Sonobuoy Pod and associated Service.

### RBAC

If you are using the provided manifests as a basis for your own, *you will not need to change this file.* The ServiceAccount, ClusterRoleBinding, and ClusterRole declarations should be left as they are.


### Pod



However, ensure that your pod declaration has addressed the following aspects, which may be unique to your cluster:
* **Image Name**
  * The provided example configs pull from Heptio's registry of pre-built Sonobuoy container images (*gcr.io/heptio-images/sonobuoy*).
  * To use custom-built images from your private registry, make sure to change the `image` and `imagePullPolicy` config fields.


* **Config Volume Mount**
  * To mount the main Sonobuoy config on the Sonobuoy pod, a ConfigMap is recommended. This is used in the example.
  * Ensure that `mountPath` matches one of the paths enumerated in [the chart above][0].


* **Output Volume Mount**
  * Sonobuoy output should be persisted by writing to [Persistent Volume Claims (PVC)][15].
  * However, for the sake of prototyping (as in the example), you can mount via `emptyDir`. This allows you to later inspect Sonobuoy's results by copying them to your local machine with `kubectl cp`.


* **Advertise IP (env variable)**
  * It's recommended to keep this the same as in the example config, where Kubernetes auto-populates the environmental variable `$SONOBUOY_ADVERTISE_IP` with the IP of the assigned pod.

[0]: #Overview
[1]: #data-configuration
[2]: #sample-json
[3]: #parameter-reference
[4]: #plugin-configuration
[5]: #kubernetes-component-definitions
[6]: #overview-1
[7]: #rbac
[8]: #pod
[9]: plugins.md
[10]: /config.json
[11]: /examples/quickstart/components/10-configmaps.yaml
[12]: /examples/quickstart/components/00-rbac.yaml
[13]: /examples/quickstart/components/20-pod.yaml
[14]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
[15]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-persistent-volume-storage/
[16]: /plugins.d/e2e.yaml
[17]: plugins.d/systemdlogs.yaml
[18]: /examples/quickstart/aggregate.yaml
[19]: /examples/quickstart/components
