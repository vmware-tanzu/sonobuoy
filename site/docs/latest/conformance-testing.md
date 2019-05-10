# Conformance Testing - [1.8+][6]

## Overview

With such a [wide array][2] of Kubernetes distributions available, *conformance tests* help ensure that a Kubernetes cluster meets the minimal set of features. They are a subset of end-to-end (e2e) tests that should pass on any Kubernetes cluster.

A conformance-passing cluster provides the following guarantees:

* **Best practices**: Your Kubernetes is properly configured. This is useful to know whether you are running a distribution out of the box or handling your own custom setup.

* **Predictability**: All your cluster behavior is well-documented. Available features in the official Kubernetes documentation can be taken as a given. Unexpected bugs should be rare, because distribution-specific issues are weeded out during the conformance tests.

* **Interoperability**: Workloads from other conforming clusters can be ported into your cluster, or vice versa. This standardization of Kubernetes is a key advantage of open source software, and allows you to avoid vendor lock-in.

Individual Kubernetes distributions may offer additional features beyond conformance testing, but if you change distributions, these features can't be expected to be provided.

**NOTE:** Kubernetes documentation also describes the concept of [node conformance tests][3]. Although they are useful, these tests are more component-focused than system-wide. They validate only the behavior of a specific node, not cluster behavior as a whole.

See the [official documentation][4] for Kubernetes's existing conformance tests.

## Integration with Sonobuoy

Sonobuoy's [plugin architecture][5] enables you to integrate conformance test results into your reporting.  The e2e tests can be configured with the plugin. The default configuration runs the basic set of conformance tests against a local provider.

To customize the set of tests that are run as part of the report, the following environmental variables can be set in the [plugin-specific YAML config][7]:

| Variable | Default Value | Description |
|---|---|---|
| `E2E_FOCUS` | "Conformance" | The test suite to run.<br><br>*NOTE*: Because the real conformance suite can take up to an hour to run, the quickstart example's [e2e config][8] specifies just a single test, "Pods should be submitted and removed". |
| `E2E_SKIP` | "Alpha&#124;Disruptive&#124;Feature&#124;Flaky&#124;Kubectl" | Which subset of tests to skip |
| `E2E_PROVIDER` | "local" | The platform that the cluster is running on |

*NOTE: The length of time it takes to run conformance can vary based on the size of your cluster---the timeout can be adjusted in the [Server.timeoutseconds][9] field of the Sonobuoy `config.json`.*

[0]: #overview
[1]: #integration-with-sonobuoy
[2]: https://docs.google.com/spreadsheets/d/1LxSqBzjOxfGx3cmtZ4EbB_BGCxT_wlxW_xgHVVa23es/edit#gid=0
[3]: https://kubernetes.io/docs/admin/node-conformance/
[4]: https://github.com/kubernetes/community/blob/master/contributors/devel/e2e-tests.md#conformance-tests
[5]: plugins.md
[6]: https://github.com/kubernetes/kubernetes/issues/49313
[7]: ../plugins.d/e2e.tmpl
[8]: ../examples/quickstart.yaml#L133
[9]: ../examples/quickstart.yaml#L102
