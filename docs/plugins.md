# Sonobuoy Plugins
<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Sonobuoy Plugins](#sonobuoy-plugins)
    - [Overview](#overview)
    - [Plugin Definition](#plugin-definition)
        - [Writing your own plugin](#writing-your-own-plugin)
            - [The plugin definition file](#the-plugin-definition-file)
            - [Contract](#contract)
    - [Available Plugins](#available-plugins)

<!-- markdown-toc end -->

## Overview

In addition to querying API objects, Sonobuoy also supports a *Plugin model*. In this model, "worker" pods are dispatched into the cluster to collect data from each node, and use an aggregation URL to submit their results back to a waiting "aggregation" pod. See the diagram below:

![sonobuoy plugins diagram][diagram]


[diagram]: img/sonobuoy-plugins.png

There are two main components that specify plugin behavior:

1. **Plugin Selection**: A section in the main config (`config.json`) that declares *which* plugins should be used in the Sonobuoy run.
  This can be auto-generated or passed in via the `--config` flag to `sonobuoy run` or `sonobuoy gen`.

    *These configs are defined by the **end user**.*

2. **Plugin Definition**: A YAML document that defines some metadata and a pod to produce a result.

    *This YAML is defined by the plugin **developer**, and can be taken as a given by the end user.*

The remainder of this document focuses on **Plugin Definition**.

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
```

#### Contract

A definition file defines a container that will run the tests. This container
can be anything you want, but must fulfil a contract.

When your container has completed its work, it needs to signal to Sonobuoy that
it's done. This is done by writing out a filename to a results file (by default
`/tmp/results/done`, configurable by `ResultsDir` in the sonobuoy config) with a
filename leading to the results file.

Sonobuoy waits for this done file to be present, then transmits the indicated
file back to the aggregator. The results file is opaque to Sonobuoy, and will be
made available in the Sonobuoy results tarbal in its original form.

## Available Plugins

The current, default set of Sonobuoy plugins are available in the `examples/plugins.d` directory within this repo. You can also use the list below as a reference:

| Plugin                    | Overview                                                                                     | Source Code Repository                              | Env Variables (Config)                                                                                    |
| ---                       | ---                                                                                          | ---                                                 | ---                                                                                                       |
| [`systemd_logs`][systemd] | Gather the latest system logs from each node, using systemd's `journalctl` command.          | [heptio/sonobuoy-plugin-systemd-logs][systemd-repo] | (1) `RESULTS_DIR`<br>(2)`CHROOT_DIR`<br>(3)`LOG_MINUTES`                                                  |
| [`e2e`][e2e]              | Run Kubernetes end-to-end tests (e.g. conformance) and gather the results.                   | [heptio/kube-conformance][conformance]              | `E2E_*` variables configure the end-to-end tests. See the [conformance testing guide][guide] for details. |
| [`bulkhead`][bulkhead]    | Perform CIS Benchmark scans from each node using Aqua Security's [`kube-bench`][bench] tool. | [bgeesaman/sonobuoy-plugin-bulkhead][bulkhead]      | (1) `RESULTS_DIR`                                                                                         |



[systemd]: /examples/plugins.d/e2e.yaml
[e2e]: /examples/plugins.d/heptio-e2e.yaml
[conformance]: https://github.com/heptio/kube-conformance
[guide]: conformance-testing.md#integration-with-sonobuoy 
[bulkhead]: https://github.com/bgeesaman/sonobuoy-plugin-bulkhead
[bench]: https://github.com/bgeesaman/sonobuoy-plugin-bulkhead

