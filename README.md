# Sonobuoy

**Maintainers:** [Heptio][0]

[![Build Status][1]][2]

## Overview

Heptio Sonobuoy is a diagnostic tool that makes it easier to understand the state of a Kubernetes cluster by running a set of [Kubernetes][3] conformance tests in an accessible and non-destructive manner.  It is a customizable, extendable, and cluster-agnostic way to generate clear, informative reports about your cluster.

Its selective data dumps of Kubernetes resource objects and cluster nodes allow for the following use cases:

* Integrated end-to-end (e2e) [conformance-testing][4]
* Workload debugging
* Custom data collection via extensible plugins

Sonobuoy supports Kubernetes versions 1.8 and later.

## More information

[The documentation][23] provides detailed information about the conformance tests, plugins, and snapshots.

## Prerequisites

* Access to an up-and-running Kubernetes cluster. If you do not have a cluster, follow the [AWS Quickstart Kubernetes Tutorial][5] to set one up with a single command.

* *`kubectl` installed. See [installing via Homebrew (MacOS)][6] or [building the binary (Linux)][7].

* An admin `kubeconfig` file, and the KUBECONFIG environment variable set.

## Getting Started

The browser-based [Sonobuoy Scanner tool][18] is the quickest way to get started with Sonobuoy. Sonobuoy Scanner also provides a user-friendly way of viewing your scan results.

**NOTE:** Sonobuoy Scanner runs conformance tests only.

![tarball overview screenshot][20]

## Using the CLI

Sonobuoy also provides a CLI that lets you run Sonobuoy on your cluster. The CLI records the following results:

* Information about your cluster's hosts, Kubernetes resources, and versions.
* [`systemd`][14] logs from each host. Requires a plugin.
* The results of a single e2e conformance test ("Pods should be submitted and removed"). See the [conformance guide][4] for configuration details. Requires a plugin.

### CLI Prerequisites

* Golang installed. We recommend [gimme][21], with golang version 1.9.4. 

* Your $PATH configured:

```
$ export PATH=$GOROOT/bin:$GOPATH/bin:$PATH 
```  

### Download and run

Download the CLI by running:

```
$ go get -u -v github.com/heptio/sonobuoy
```

Deploy a Sonobuoy pod to your cluster with:

```
$ sonobuoy run
```

View actively running pods:

```
$ sonobuoy status 
```

To inspect the logs:

```
$ sonobuoy logs
```

To view the output, copy the output directory from the main Sonobuoy pod to somewhere local:

```
$ sonobuoy retrieve .
```

This copies a single `.tar.gz` snapshot from the Sonobuoy pod into your local `.` directory. Extract the contents into `./results` with:

```
mkdir ./results; tar xzf *.tar.gz -C ./results
```

For information on the contents of the snapshot, see the [snapshot documentation][22].

### Cleanup

To clean up Kubernetes objects created by Sonobuoy, run:

```
sonobuoy delete
```

### Run on Google Cloud Platform (GCP)

Note that if you run Sonobuoy on a Google Kubernetes Engine (GKE) cluster, you must first create an admin role for the
user under which you run Sonobuoy:

```
kubectl create clusterrolebinding <your-user-cluster-admin-binding> --clusterrole=cluster-admin --user=<your.google.cloud.email@example.org>
```

## Troubleshooting

If you encounter any problems that the documentation does not address, [file an issue][10].

## Contributing

Thanks for taking the time to join our community and start contributing!  We welcome pull requests. Feel free to dig through the [issues][10] and jump in.

#### Before you start

* Please familiarize yourself with the [Code of Conduct][12] before contributing.
* See [CONTRIBUTING.md][11] for instructions on the developer certificate of origin that we require.
* There is a [mailing list][16] and [Slack channel][17] if you want to interact with
other members of the community

## Changelog

See [the list of releases](https://github.com/heptio/sonobuoy/releases) to find out about feature changes.

[0]: https://github.com/heptio
[1]: https://jenkins.i.heptio.com/buildStatus/icon?job=sonobuoy-deployer
[2]: https://jenkins.i.heptio.com/job/sonobuoy-deployer/
[3]: https://github.com/kubernetes/kubernetes
[4]: /docs/conformance-testing.md
[5]: http://docs.heptio.com/content/tutorials/aws-cloudformation-k8s.html
[6]: https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-with-homebrew-on-macos
[7]: https://kubernetes.io/docs/tasks/tools/install-kubectl/#tabset-1
[8]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-persistent-volume-storage/
[10]: https://github.com/heptio/sonobuoy/issues
[11]: https://github.com/heptio/sonobuoy/blob/master/CONTRIBUTING.md
[12]: https://github.com/heptio/sonobuoy/blob/master/CODE_OF_CONDUCT.md
[14]: https://github.com/systemd/systemd
[16]: https://groups.google.com/forum/#!forum/heptio-sonobuoy
[17]: https://kubernetes.slack.com/messages/sonobuoy
[18]: https://scanner.heptio.com/
[20]: docs/img/scanner.png
[21]: https://github.com/travis-ci/gimme
[22]: docs/snapshot.md
[23]: https://heptio.github.io/sonobuoy/
