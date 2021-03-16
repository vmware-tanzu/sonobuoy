# Sonobuoy Plugins
<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Sonobuoy Plugins](#sonobuoy-plugins)
    - [Overview](#overview)
    - [Plugin Definition](#plugin-definition)
        - [Writing your own plugin](#writing-your-own-plugin)
            - [The plugin definition file](#the-plugin-definition-file)
            - [Contract](#contract)
        - [Customizing PodSpec options](#customizing-podspec-options)
    - [Available Plugins](#available-plugins)

<!-- markdown-toc end -->

## Overview

In addition to querying API objects, Sonobuoy also supports a plugin model. In this model, worker pods are dispatched into the cluster to collect data from each node, and use an aggregation URL to submit their results back to a waiting aggregation pod. See the diagram below:

![sonobuoy plugins diagram][diagram]


[diagram]: /img/sonobuoy-plugins.png

Two main components specify plugin behavior:

1. **Plugin Selection**: A section in the main config (`config.json`) that declares which plugins to use in the Sonobuoy run.
  This can be generated or passed in with the `--config` flag to `sonobuoy run` or `sonobuoy gen`.

    These configs are defined by the end user.

2. **Plugin Definition**: A YAML document that defines metadata and a pod to produce a result.

    This YAML is defined by the plugin developer, and can be taken as a given by the end user.

## Plugin Definition

- `/etc/sonobuoy/plugins.d`
- `$HOME/.sonobuoy/plugins.d`
- `./plugins.d`

This search path can be overridden by the `PluginSearchPath` value of the Sonobuoy config.

### Writing your own plugin

#### The plugin definition file

``` yaml
---
sonobuoy-config:
  driver: Job        # Job or DaemonSet. Job runs once per run, Daemonset runs on every node per run.
  plugin-name: e2e   # The name of the plugin
  result-type: e2e   # The name of the "result type." Usually the name of the plugin.
spec:                # A kubernetes container spec
  env:
  - name: E2E_FOCUS
    value: Pods should be submitted and removed
  image: gcr.io/heptio-images/kube-conformance:latest
  imagePullPolicy: Always
  name: e2e
  volumeMounts:
  - mountPath: /tmp/results
    name: results
    readOnly: false
  - mountPath: /var/log/test
    name: test-volume
extra-volumes:
- name: test-volume
  hostPath:
    # directory location on host
    path: /data
```

#### Contract

A definition file defines a container that runs the tests. This container
can be anything you want, but must fulfil a contract.

After your container completes its work, it needs to signal to Sonobuoy that
it's done. This is done by writing out a filename to a results file. The default
value is `/tmp/results/done`, which you can configure with the `ResultsDir` value 
in the Sonobuoy config.

Sonobuoy waits for the `done` file to be present, then transmits the indicated
file back to the aggregator. The results file is opaque to Sonobuoy, and is
made available in the Sonobuoy results tarball in its original form.

If you need additional mounts besides the default `results` mount that Sonobuoy
always provides, you can define them in the `extra-volumes` field.

#### Choosing which plugins to run

All of the plugin definition files get mounted as files on the aggregator pod which runs them.

The aggregator loads all the plugins it finds, but a separate list controls which plugins actually get run. There is a separate `config.json` which gets mounted on the aggregator which sets the configuration options for the aggregator. It has a field `Plugins` which is an array of plugin names. The default value includes both the e2e and systemd-logs plugin:

```
"Plugins":[{"name":"e2e"},{"name":"systemd-logs"}]
```

If you want to prevent one of those plugins from  being run, simply remove that item from the list. Likewise, if you'd like to run your own custom plugin, you need to add it to this list (in addition to adding its definition file to the plugin configmap):

```
"Plugins":[{"name":"custom-plugin"},{"name":"systemd-logs"}]
```

In either case, you use the sonobuoy [gen][gen] flow to edit the YAML and start the run with `kubectl`.

### Customizing PodSpec options

By default, Sonobuoy will determine how to create and run the resources required for your plugin.
When creating your own plugins however, you may want additional control over how the plugin is run within your cluster.
To enable this, you can customize the [PodSpec][kubernetes-podspecs] used by Sonobuoy when creating the plugin's Pods or DaemonSets by supplying a `podSpec` object within your plugin defition.
The `podSpec` object corresponds directly to a Kubernetes [PodSpec][kubernetes-podspecs] so any fields that are available there can be set by your plugins.

If a `podSpec` is provided, Sonobuoy will use it as is, only adding what is necessary for Sonobuoy to run your plugin (such as a Sonobuoy worker container).
Sonobuoy will only ever _add_ to your `podSpec` definition, it will not remove or override settings within it.
If you don't need to provide any additional settings, you can omit this object and Sonobuoy will use the defaults.

#### Providing your own PodSpec
We recommend starting with the default `podSpec` used by Sonobuoy and then making any necessary modifications.
To view the default `podSpec`, you can use the flag `--show-default-podspec` with the `gen` and `gen plugin` commands.

When creating a new plugin, you can include the default `podSpec` in the generated definition as follows:

```
sonobuoy gen plugin --show-default-podspec -n my-plugin -i my-plugin:latest
```

This will produce the following plugin definition:

```yaml
podSpec:
  containers: []
  restartPolicy: Never
  serviceAccountName: sonobuoy-serviceaccount
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  - key: CriticalAddonsOnly
    operator: Exists
sonobuoy-config:
  driver: Job
  plugin-name: my-plugin
  result-type: my-plugin
spec:
  command:
  - ./run.sh
  image: my-plugin:latest
  name: plugin
  resources: {}
  volumeMounts:
  - mountPath: /tmp/results
    name: results
```

You are then free to make modifications to the `podSpec` object as necessary.

If you already have an existing plugin which you would like to customize, you can take the default `podSpec`, add it to your plugin definition and use it as the basis for customization.

> **NOTE:** The default `podSpec` differs for Job and DaemonSet plugins.
To be sure you are using the appropriate defaults as your starting point, be sure to provide the `--type` flag when using `sonobuoy gen plugin`.

You can also modify the `podSpec` from within a Sonobuoy manifest.
By providing the flag `--show-default-podspec` to `sonobuoy gen`, the default `podSpec` for each plugin will be included within the `sonobuoy-plugins-cm` ConfigMap in the manifest.

> **NOTE:** Modifications to the `podSpec` are only persisted within that generated manifest.
If you generate a new manifest by running `sonobuoy gen` again, you will need to reapply any changes made.
We recommend adding your desired customizations to the plugin definition itself.

## Available Plugins

The default Sonobuoy plugins are available in the `examples/plugins.d` directory in this repository.
Here's the current list:

| Plugin                    | Overview                                                                                     | Source Code Repository                              | Env Variables (Config)                                                                                    |
| ---                       | ---                                                                                          | ---                                                 | ---                                                                                                       |
| [`systemd_logs`][systemd] | Gather the latest system logs from each node, using systemd's `journalctl` command.          | [heptio/sonobuoy-plugin-systemd-logs][systemd-repo] | (1) `RESULTS_DIR`<br>(2)`CHROOT_DIR`<br>(3)`LOG_MINUTES`                                                  |
| [`e2e`][e2e]              | Run Kubernetes end-to-end tests (e.g. conformance) and gather the results.                   | [kubernetes/kubernetes][conformance]              | `E2E_*` variables configure the end-to-end tests. See the [conformance testing guide][guide] for details. |
| [`bulkhead`][bulkhead]    | Perform CIS Benchmark scans from each node using Aqua Security's [`kube-bench`][bench] tool. | [bgeesaman/sonobuoy-plugin-bulkhead][bulkhead]      | (1) `RESULTS_DIR`                                                                                         |



[gen]: gen.md
[systemd]: https://github.com/vmware-tanzu/sonobuoy/blob/master/examples/plugins.d/systemd_logs.yaml
[systemd-repo]: https://github.com/heptio/sonobuoy-plugin-systemd-logs
[e2e]: https://github.com/vmware-tanzu/sonobuoy/blob/master/examples/plugins.d/heptio-e2e.yaml
[conformance]: https://github.com/kubernetes/kubernetes/tree/master/cluster/images/conformance
[guide]: conformance-testing.md#integration-with-sonobuoy 
[bulkhead]: https://github.com/bgeesaman/sonobuoy-plugin-bulkhead/blob/master/examples/benchmark.yml
[bench]: https://github.com/bgeesaman/sonobuoy-plugin-bulkhead
