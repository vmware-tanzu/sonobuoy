---
version: v0.52.0
cascade:
  layout: docs
  gh: https://github.com/vmware-tanzu/sonobuoy/tree/v0.52.0
---
# ![Sonobuoy Logo](img/sonobuoy-logo.png)

[![Test](https://github.com/vmware-tanzu/sonobuoy/actions/workflows/ci-test.yaml/badge.svg)](https://github.com/vmware-tanzu/sonobuoy/actions/workflows/ci-test.yaml/badge.svg)
[![Lint](https://github.com/vmware-tanzu/sonobuoy/actions/workflows/ci-lint.yaml/badge.svg)](https://github.com/vmware-tanzu/sonobuoy/actions/workflows/ci-lint.yaml/badge.svg)

## [Overview][oview]

Sonobuoy is a diagnostic tool that makes it easier to understand the
state of a Kubernetes cluster by running a set of plugins (including [Kubernetes][k8s] conformance
tests) in an accessible and non-destructive manner. It is a customizable,
extendable, and cluster-agnostic way to generate clear, informative reports
about your cluster.

Its selective data dumps of Kubernetes resource objects and cluster nodes allow
for the following use cases:

* Integrated end-to-end (e2e) [conformance-testing][e2ePlugin]
* Workload debugging
* Custom data collection via extensible plugins

Starting v0.20, Sonobuoy supports Kubernetes v1.17 or later.
Sonobuoy releases will be independent of Kubernetes release, while ensuring that new releases continue to work functionally across different versions of Kubernetes.
Read more about the new release cycles in [our blog][decoupling-sonobuoy-k8s].

> Note: You can skip this version enforcement by running Sonobuoy with the `--skip-preflight` flag.

## Prerequisites

* Access to an up-and-running Kubernetes cluster. If you do not have a cluster,
  we recommend following the [AWS Quickstart for Kubernetes][quickstart] instructions.

* An admin `kubeconfig` file, and the KUBECONFIG environment variable set.

* For some advanced workflows it may be required to have `kubectl` installed. See [installing via Homebrew (MacOS)][brew] or [building
  the binary (Linux)][linux].

* The `sonobuoy images` subcommand requires [Docker](https://www.docker.com) to be installed. See [installing Docker][docker].

## Installation

1. Download the [latest release][releases] for your client platform.
2. Extract the tarball:

   ```
   tar -xvf <RELEASE_TARBALL_NAME>.tar.gz
   ```

   Move the extracted `sonobuoy` executable to somewhere on your `PATH`.

## Getting Started

To launch conformance tests (ensuring [CNCF][cncf] conformance) and wait until they are finished run:

```bash
sonobuoy run --wait
```

> Note: Using `--mode quick` will significantly shorten the runtime of Sonobuoy. It runs just a single test, helping to quickly validate your Sonobuoy and Kubernetes configuration.

Get the results from the plugins (e.g. e2e test results):

```bash
results=$(sonobuoy retrieve)
```

Inspect results for test failures.  This will list the number of tests failed and their names:

```bash
sonobuoy results $results
```

> Note: The `results` command has lots of useful options for various situations. See the [results page][results] for more details.

You can also extract the entire contents of the file to get much more [detailed data][snapshot] about your cluster.

Sonobuoy creates a few resources in order to run and expects to run within its
own namespace.

Deleting Sonobuoy entails removing its namespace as well as a few cluster
scoped resources.

```bash
sonobuoy delete --wait
```

> Note: The --wait option ensures the Kubernetes namespace is deleted, avoiding conflicts if another Sonobuoy run is started quickly.

### Other Tests

By default, `sonobuoy run` runs the Kubernetes conformance tests but this can easily be configured. The same plugin that has the conformance tests has all the Kubernetes end-to-end tests which include other tests such as:

* tests for specific storage features
* performance tests
* scaling tests
* provider specific tests
* and many more

To modify which tests you want to run, checkout our page on the [e2e plugin][e2ePlugin].

If you want to run other tests or tools which are not a part of the Kubernetes end-to-end suite, refer to our documentation on [custom plugins][customPlugins].

### Monitoring Sonobuoy during a run

You can check on the status of each of the plugins running with:

```bash
sonobuoy status
```

You can also inspect the logs of all Sonobuoy containers:

```bash
sonobuoy logs
```

## Troubleshooting

If you encounter any problems that the documentation does not address, [file an
issue][issue].

## Docker Hub rate limit

This year, Docker has started rate limiting image pulls from Docker Hub. We're planning a future release with a better user interface to work around this. Until then, this is the recommended approach.

### Sonobuoy Pod

Sonobuoy by default pulls from Docker Hub for [`sonobuoy/sonobuoy` image](https://hub.docker.com/r/sonobuoy/sonobuoy). If you're encountering rate limit on this, you can use VMware-provided mirror with:

```bash
sonobuoy run --sonobuoy-image projects.registry.vmware.com/sonobuoy/sonobuoy:<VERSION>
```

### Conformance

Kubernetes end-to-end conformance test pulls several images from Docker Hub as part of testing. To override this, you will need to create a registry manifest file locally (e.g. `conformance-image-config.yaml`) containing the following:

```yaml
dockerLibraryRegistry: mirror.gcr.io/library
```

Then on running conformance:

```bash
sonobuoy run --sonobuoy-image projects.registry.vmware.com/sonobuoy/sonobuoy:<VERSION> --e2e-repo-config conformance-image-config.yaml
```

Technically `dockerGluster` is also a registry pulling from Docker Hub, but it's not part of Conformance test suite at the moment, so overriding `dockerLibraryRegistry` should be enough.

## Known Issues

### Leaked End-to-end namespaces

There are some Kubernetes e2e tests that may leak resources. Sonobuoy can
help clean those up as well by deleting all namespaces prefixed with `e2e`:

```bash
sonobuoy delete --all
```

### Run on Google Cloud Platform (GCP)

Sonobuoy requires admin permissions which won't be automatic if you are running via Google Kubernetes Engine (GKE) cluster. You must first create an admin role for the user under which you run Sonobuoy:

```bash
kubectl create clusterrolebinding <your-user-cluster-admin-binding> --clusterrole=cluster-admin --user=<your.google.cloud.email@example.org>
```

## Contributing

Thanks for taking the time to join our community and start contributing! We
welcome pull requests. Feel free to dig through the [issues][issue] and jump in.

### Before you start

* Please familiarize yourself with the [Code of Conduct][coc] before
  contributing.
* See [CONTRIBUTING.md][contrib] for instructions on the developer certificate
  of origin that we require.
* There is a [Slack channel][slack] if you want to
  interact with other members of the community

## Changelog

See [the list of releases][releases] to find out about feature changes.

[decoupling-sonobuoy-k8s]: https://sonobuoy.io/decoupling-sonobuoy-and-kubernetes
[airgap]: airgap
[brew]: https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-with-homebrew-on-macos
[cncf]: https://github.com/cncf/k8s-conformance#certified-kubernetes
[coc]: https://github.com/vmware-tanzu/sonobuoy/blob/master/CODE_OF_CONDUCT.md
[contrib]: https://github.com/vmware-tanzu/sonobuoy/blob/master/CONTRIBUTING.md
[docker]: https://docs.docker.com/get-docker/
[docs]: https://sonobuoy.io/docs/v0.52.0
[e2ePlugin]: e2eplugin
[customPlugins]: plugins
[gen]: gen
[issue]: https://github.com/vmware-tanzu/sonobuoy/issues
[k8s]: https://github.com/kubernetes/kubernetes
[linux]: https://kubernetes.io/docs/tasks/tools/install-kubectl/#tabset-1
[oview]: https://youtu.be/8QK-Hg2yUd4
[plugins]: plugins
[quickstart]: https://aws.amazon.com/quickstart/architecture/vmware-kubernetes/
[releases]: https://github.com/vmware-tanzu/sonobuoy/releases
[results]: results
[slack]: https://kubernetes.slack.com/messages/sonobuoy
[snapshot]:snapshot
[sonobuoyconfig]: sonobuoy-config
