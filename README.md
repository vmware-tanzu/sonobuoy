# Sonobuoy

**Maintainers:** [Heptio][0]

[![Build Status][1]][2]

**NOTE:**
Sonobuoys master branch is in high flux atm, and if you are looking for a more stable version please checkout the v0.10.0 tag. 

## Overview

Heptio Sonobuoy is a diagnostic tool that makes it easier to understand the state of a Kubernetes cluster by running a set of [Kubernetes][3] conformance tests in an accessible and non-destructive manner.  It is a customizable, extendable, and cluster-agnostic way to generate clear, informative reports about your cluster.

Its selective data dumps of Kubernetes resource objects and cluster nodes allow for the following use cases:

* Integrated end-to-end (e2e) [conformance-testing][4]
* Workload debugging
* Custom data collection via extensible plugins

Sonobuoy supports all upstream supported versions of Kubernetes (1.7.x-v1.9.y).

## Prerequisites

* **You should have access to an up-and-running Kubernetes cluster.** If you do not have a cluster, follow the [AWS Quickstart Kubernetes Tutorial][5] to set one up with a single command.

* **You should have `kubectl` installed.** If not, follow the instructions for [installing via Homebrew (MacOS)][6] or [building the binary (Linux)][7].

* **You should have an admin `kubeconfig file` installed, and KUBECONFIG environment variable set.**

## Getting Started

To easily get a Sonobuoy scan started on your cluster, use the browser-based [Sonobuoy Scanner tool][18]. Sonobuoy Scanner also provides a more user-friendly way of viewing your scan results.

*Note that Sonobuoy Scanner runs conformance tests only.*

![tarball overview screenshot][20]

## Using the CLI

This guide executes a Sonobuoy run on your cluster, and records the following results:
* Basic info about your cluster's hosts, Kubernetes resources, and versions.
* *(Via plugin)* [`systemd`][14] logs from each host
* *(Via plugin)* The results of a single e2e conformance test ("Pods should be submitted and removed"). See the [conformance guide][4] for configuration details.

### 1. Download

Sonobuoy is written in golang and can easily be obtained by running:
```
$ go get github.com/heptio/sonobuoy
```

### 2. Run

Now you're ready to deploy a Sonobuoy pod to your cluster! Run the following command:
```
$ sonobuoy run
```

You can view actively running pods with the following command:
```
$ sonobuoy status 
```

To inspect the logs:
```
$ sonobuoy logs
```

To view the output, copy the output directory from the main Sonobuoy pod to somewhere local:
```
$ sonobuoy cp .
```

This should copy a single `.tar.gz` snapshot from the Sonobuoy pod into your local `.` directory. You can extract its contents into `./results` with:
```
mkdir ./results; tar xzf *.tar.gz -C ./results
```

For information on the contents of the snapshot, see the [documentation](docs/snapshot.md).

### 3. Cleanup

To clean up Kubernetes objects created by Sonobuoy, run the following command:
```
sonobuoy delete
```

## Further documentation

 To learn how to configure your Sonobuoy runs and integrate plugins, see the [`/docs` directory][9].

## Troubleshooting

If you encounter any problems that the documentation does not address, [file an issue][10].

## Contributing

Thanks for taking the time to join our community and start contributing!  We welcome pull requests. Feel free to dig through the [issues][10] and jump in.

#### Before you start

* Please familiarize yourself with the [Code of
Conduct][12] before contributing.
* See [CONTRIBUTING.md][11] for instructions on the
developer certificate of origin that we require.
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
[9]: /docs
[10]: https://github.com/heptio/sonobuoy/issues
[11]: /CONTRIBUTING.md
[12]: /CODE_OF_CONDUCT.md
[14]: https://github.com/systemd/systemd
[15]: #3-tear-down
[16]: https://groups.google.com/forum/#!forum/heptio-sonobuoy
[17]: https://kubernetes.slack.com/messages/sonobuoy
[18]: https://scanner.heptio.com/
[19]: #quickstart
[20]: docs/img/scanner.png
