# Conformance Testing - [1.7+][10]

* [Overview][0]
* [Integration with Sonobuoy][1]

## Overview

With such a [wide array][2] of Kubernetes distributions available, *conformance tests* help ensure that any self-described "Kubernetes" actually meets the minimal set of features. They are a subset of e2e (end-to-end) tests that should pass on *any* Kubernetes cluster.

A conformance-passing cluster gives administrators and users the following guarantees:
* **Best practices**: Your Kubernetes is properly configured. This is useful to know whether you are running a distribution out of the box or handling your own custom setup.

* **Predictability**: All of your cluster behavior is well-documented. (1) Available features in the official Kubernetes documentation can be taken as a given. (2) Unexpected bugs should be rare, as distribution-specific issues are weeded out during the conformance tests.

* **Interoperability**: Workloads from other conforming clusters can be easily ported into your cluster (or vice versa). This standardization of Kubernetes is a key advantage of open source software, and allows you to avoid vendor lock-in.

Kubernetes distributions may offer additional features beyond what is covered by Conformance Testing, but these are not expected to carry over in case of a switch.

<br>

> **NOTE:** Kubernetes documentation also describes the concept of [node conformance tests][3]. While useful, these tests are more component-focused than systemic. They only validate the behavior of a specific node, rather than cluster behavior as a whole.

<br>

See the [official documentation][4] of Kubernetes's existing conformance tests.

## Integration with Sonobuoy

Sonobuoy's [plugin architecture][5] enables you to integrate conformance test results into your reporting.  The e2e tests can be configured via the plugin mechanism and are set up by default to run the basic set of conformance tests against a local provider.

To customize the set of tests that will be run as part of the report, the following [environmental variables][6] can be set in the [plugin-specific YAML config][7]:

| Variable | Default Value | Description |
|---|---|---|
| `E2E_FOCUS` | "Conformance" | The test suite to run.<br><br>*NOTE*: Because the real conformance suite can take up to half an hour to run, the quickstart example's [e2e config][8] specifies just a single test, "Pods should be submitted and removed". |
| `E2E_SKIP` | "Alpha&#124;Disruptive&#124;Feature&#124;Flaky&#124;Kubectl" | Which subset of tests to skip |
| `E2E_PROVIDER` | "local" | The platform that the cluster is running on |

*NOTE: The length of time it takes to run conformance can vary based on the size of your cluster---the timeout can be adjusted in the [Server.timeoutseconds][9] field of the Sonobuoy `config.json`.*

[0]: #overview
[1]: #integration-with-sonobuoy
[2]: https://docs.google.com/spreadsheets/d/1LxSqBzjOxfGx3cmtZ4EbB_BGCxT_wlxW_xgHVVa23es/edit#gid=0
[3]: https://kubernetes.io/docs/admin/node-conformance/
[4]: https://github.com/kubernetes/community/blob/master/contributors/devel/e2e-tests.md#conformance-tests
[5]: plugins.md
[6]: https://github.com/heptio/sonobuoy/blob/master/build/Dockerfile
[7]: https://github.com/heptio/sonobuoy/blob/master/plugins.d/e2e.yaml
[8]: https://github.com/heptio/sonobuoy/blob/master/examples/quickstart/components/10-configmaps.yaml#L185
[9]: https://github.com/heptio/sonobuoy/blob/master/examples/quickstart/components/10-configmaps.yaml#L71
[10]: https://github.com/kubernetes/kubernetes/issues/49313
