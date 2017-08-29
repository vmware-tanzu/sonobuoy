# Sonobuoy Plugins

* [Overview][0]
* [Plugin Definition][1]
  * [Under the Hood][2]
  * [Example][3]
  * [Parameter Reference][4]
* [Available Plugins][5]

## Overview

In addition to querying API objects, Sonobuoy also supports a *Plugin model*. In this model, "worker" pods are dispatched into the cluster to collect data from each node, and use an aggregation URL to submit their results back to a waiting "master" pod. See the diagram below:

![sonobuoy plugins diagram][6]

There are two main components that specify plugin behavior:

1. **Plugin Selection**: A section in the main config (`config.json`) that declares *which* plugins should be used in the Sonobuoy run.

    *These configs are defined by the **end user**.*

1. **Plugin Definition**: A structured YAML definition that describes a plugin's features, method of launch, and other configurations.

    *This YAML is defined by the plugin **developer**, and can be taken as a given by the end user.*

The remainder of this document focuses on **Plugin Definition**.

## Plugin Definition

The *plugin definition* is a YAML file (raw or wrapped in a ConfigMap) that describes the core parts of your custom plugin. It should contain the following fields:

| Field | Description | Example Values |
| --- | --- | --- |
| `name` | A name that is used to identify the plugin (e.g. in the Plugin Selection described above). | "e2e" |
| `driver` | Sonobuoy implements *plugin drivers* that define different modes of operation.<br><br>(1) **"Job" driver**: The plugin will run on a single node (e.g. master).<br>(2) **"DaemonSet" driver**: The plugin runs on each cluster node.<br><br>You can find the implementations [here][7]. | "Job&#124;DaemonSet" |
| `resultType` | The name of the subdirectory that this plugin's results are saved in. With a `resultType` of "e2e", results are written into `plugins/e2e/...` (within the tarball output).<br><br>This value is typically the same as the plugin `name`. | "e2e" |
| `spec` | The Pod specification (e.g. network settings, container settings, volume definitions, etc.) | See [the parameter spec][4] below for reference. |

Sonobuoy searches for these definitions in three locations by default:

- `/etc/sonobuoy/plugins.d`
- `$HOME/.sonobuoy/plugins.d`
- `./plugins.d`

This search path can be overridden by the `PluginSearchPath` value of the main Sonobuoy config.

In the [quickstart example][12], the necessary YAML configs are wrapped in a ConfigMap, so that they can be mounted as files in the `/etc/sonobuoy/plugins.d` directory. *This is the recommended approach.*

### Under the Hood

While you can create a plugin without leveraging a Sonobuoy worker container, you will need to write your own aggregation code in that scenario. To make implementation easier, **we recommend using a two-container model**, as described below:

1. **A "Producer" Container**: A containerized way to perform the task that you want executed.

    *This is done by your own container, which does not need to be Sonobuoy-aware*.

1. **A "Consumer" Container**: A way of submitting results back to the aggregation URL, so that the master Sonobuoy instance can collect node-specific data.

    *This is done by a container running `sonobuoy worker`, which can handle the uploading for you.*

1. **A registry to store these compiled image(s)**. When they are ready to be used, the plugin's container image(s) should either be (1) uploaded to a public Docker registry or (2) a private registry, provided that the [appropriate image pull secrets][8] are created in advance.

To allow your two containers to interact (i.e. for the consumer to read and publish the results of the producer), you will need to ensure the following:
* The same `results` volume is mounted onto both containers.

* In the `results` directory (mounted above), the "producer" container writes the path of its output subdirectory to the `done` file.

    For instance, the `systemd_logs` plugin includes the following code in its [bash script][13]:

    ```
    echo -n "${RESULTS_DIR}/systemd_logs" >"${RESULTS_DIR}/done"
    ```




### Example

Below is the Pod `spec` component from the [`e2e` plugin definition][9]. It follows the two-container-paradigm. One container runs the e2e tests and writes to a temporary folder, and the other container (a Sonobuoy worker) reads those results and sends them back to the master Sonobuoy instance.

See [Parameter Reference][4] for descriptions of each of the important config keys.

```
spec:
  tolerations:
  - key: node-role.kubernetes.io/master
    operator: Exists
    effect: NoSchedule
  - key: CriticalAddonsOnly
    operator: Exists

  restartPolicy: Never

  containers:
  # (1) This container runs the actual end-to-end tests.
  - name: e2e
    image: gcr.io/heptio-images/kube-conformance:latest
    imagePullPolicy: IfNotPresent
    env:
    - name: E2E_FOCUS
      value: "Pods should be submitted and removed"
    volumeMounts:
    - name: results
      mountPath: /tmp/results

  # (2) This container is the Sonobuoy worker that handles uploads.
  - name: job
    command:
    - sh
    - -c
    - /sonobuoy worker job -v 5 --logtostderr && sleep 3600

    env:
    - name: NODE_NAME
      valueFrom:
        fieldRef:
          apiVersion: v1
          fieldPath: spec.nodeName

    image: gcr.io/heptio-images/sonobuoy:latest
    imagePullPolicy: Always

    volumeMounts:
    - name: config
      mountPath: /etc/sonobuoy
    - name: results
      mountPath: /tmp/results

  volumes:
  - name: results
    emptyDir: {}

  - name: config
    configMap:
      name: __SONOBUOY_CONFIGMAP__
```

### Parameter Reference

| Field | Description |
| --- | --- |
| `tolerations` | A way of specifying node affinity (e.g.  if your plugin should be able to run on master). See [official Kubernetes documentation][10] for more details. |
| `restartPolicy` | Specifies whether your plugin should retry on failure.
| `container.image`, `container.imagePullPolicy` | What and how often a new image should be pulled. |
| `container.env` | Set environmental variables here. These variables can be used to configure plugin behavior.<br><br>For DaemonSet plugins (e.g. `systemdlogs`), the `sonobuoy worker` consumer container needs a `NODE_NAME` variable to know which node the results should be uploaded for.|
| `container.volumeMounts`, `volumes` | <br>It is important to set up volumes and mount them properly, so that the container(s) can:<br><br>(1) **Get necessary configs** from locations like `/etc/sonobuoy`. While the Sonobuoy master creates the ConfigMap, the inbuilt Sonobuoy drivers actually substitute it into the `__SONOBUOY_CONFIGMAP__` pseudo-template. The ConfigMap's name isn't predictable because it only lasts for one run.<br><br>(2) **Write results locally**, such that the Sonobuoy worker can find the files it needs to upload to the Sonobuoy master. Typically the same `emptyDir` `results` directory is shared by both the "plugin" container and the Sonobuoy worker container.<br><br> |

## Available Plugins

The current, default set of Sonobuoy plugins are available in the `plugins.d` directory within this repo. You can also use the list below as a reference:

| Plugin | Overview | Source Code Repository | Env Variables (Config) |
| --- | --- | --- | --- |
| [`systemd_logs`][11] | Gather the latest system logs from each node, using systemd's `journalctl` command. | [heptio/sonobuoy-plugin-systemd-logs][16] | (1) `RESULTS_DIR`<br>(2)`CHROOT_DIR`<br>(3)`LOG_MINUTES`|
| [`e2e`][9] | Run Kubernetes end-to-end tests (e.g. conformance) and gather the results. | [heptio/kube-conformance][17] | `E2E_*` variables configure the end-to-end tests. See the [conformance testing guide][15] for details. |

See the [`/build`][14] directory for the source code used to build these plugins (specifically, their "producer" containers).

[0]: #overview
[1]: #developer-plugin-definition
[2]: #under-the-hood
[3]: #example
[4]: #parameter-reference
[5]: available-plugins
[6]: img/sonobuoy-plugins.png
[7]: /pkg/plugin/driver
[8]: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
[9]: /plugins.d/e2e.yaml
[10]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#taints-and-tolerations-beta-feature
[11]: /plugins.d/systemdlogs.yaml
[12]: /examples/quickstart
[13]: https://github.com/heptio/sonobuoy-plugin-systemd-logs/blob/master/get_systemd_logs.sh
[14]: /build
[15]: conformance-testing.md#integration-with-sonobuoy
[16]: https://github.com/heptio/sonobuoy-plugin-systemd-logs
[17]: https://github.com/heptio/kube-conformance
