# The Kubernetes End-To-End Testing Plugin

The Kubernetes end-to-end testing plugin (the e2e plugin) is used to run tests which are maintained by the upstream Kubernetes community in the [kubernetes/kubernetes][kubernetesRepo] repo.

There are numerous ways to run this plugin in order to meet your testing needs.

## Choosing Which Tests To Run

The most common point of customization is changing the set of tests to run. This is controlled by two environment variables the test image recognizes:

* E2E_FOCUS
* E2E_SKIP

Each of these is a regular expression describing which tests to run or skip. The "E2E_FOCUS" value is applied first and the "E2E_SKIP" value then further restricts that list. These can be set using Sonobuoy flags:

```
sonobuoy run \
  --e2e-focus=<run tests matching this regexp> \
  --e2e-skip=<skip tests matching this regexp>
```

> Note: These flags are just special cases of the more general flag `--plugin-env`. For instance, you could set the env vars by using the flag `--plugin-env e2e.E2E_SKIP=<value>`

# Built-In Configurations

There are a few commonly run configurations which Sonobuoy hard-codes for convenience:

* non-disruptive-conformance

This is the default mode and will run all the tests in the `e2e` plugin which are marked `Conformance` which are known to not be disruptive to other workloads in your cluster. This mode is ideal for checking that an existing cluster continues to behave is conformant manner.

> NOTE: The length of time it takes to run conformance can vary based on the size of your cluster---the timeout can be adjusted in the Server.timeoutseconds field of the Sonobuoy `config.json` or on the CLI via the `--timeout` flag.

* quick

This mode will run a single test from the `e2e` test suite which is known to be simple and fast. Use this mode as a quick check that the cluster is responding and reachable.

* certified-conformance

This mode runs all of the `Conformance` tests and is the mode used when applying for the [Certified Kubernetes Conformance Program](https://www.cncf.io/certification/software-conformance). Some of these tests may be disruptive to other workloads so it is not recommended that you run this mode on production clusters. In those situations, use the default "non-disruptive-conformance" mode.

> NOTE: The length of time it takes to run conformance can vary based on the size of your cluster---the timeout can be adjusted in the Server.timeoutseconds field of the Sonobuoy `config.json` or on the CLI via the `--timeout` flag.

## Dry Run

When specifying your own focus/skip values, it may be useful to set the run to operate in dry run mode:

```
sonobuoy run \
  --plugin-env e2e.E2E_FOCUS=pods \
  --plugin-env e2e.E2E_DRYRUN=true
```

By setting `E2E_DRYRUN`, the run will execute and produce results like normal except that the actual test code won't execute, just the test selection. Each test that _would have been run_ will be reported as passing. This can help you fine-tune your focus/skip values to target just the tests you want without wasting hours on test runs which target unnecessary tests.

## Why Conformance Matters

With such a [wide array][configs] of Kubernetes distributions available, *conformance tests* help ensure that a Kubernetes cluster meets the minimal set of features. They are a subset of end-to-end (e2e) tests that should pass on any Kubernetes cluster.

A conformance-passing cluster provides the following guarantees:

* **Best practices**: Your Kubernetes is properly configured. This is useful to know whether you are running a distribution out of the box or handling your own custom setup.

* **Predictability**: All your cluster behavior is well-documented. Available features in the official Kubernetes documentation can be taken as a given. Unexpected bugs should be rare, because distribution-specific issues are weeded out during the conformance tests.

* **Interoperability**: Workloads from other conforming clusters can be ported into your cluster, or vice versa. This standardization of Kubernetes is a key advantage of open source software, and allows you to avoid vendor lock-in.

Individual Kubernetes distributions may offer additional features beyond conformance testing, but if you change distributions, these features can't be expected to be provided.

See the [official documentation][conformanceDocs] for Kubernetes's existing conformance tests.

[configs]: https://docs.google.com/spreadsheets/d/1LxSqBzjOxfGx3cmtZ4EbB_BGCxT_wlxW_xgHVVa23es/edit#gid=0
[conformanceDocs]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/e2e-tests.md#conformance-tests
[kubernetesRepo]: https://github.com/kubernetes/kubernetes/tree/master/cluster/images/conformance