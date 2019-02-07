# Sonobuoy

<img src="../assets/img/vmw-os-lgo-heptio-sonobuoy.png" width="100px" >

[![Build Status][status]][travis]

[heptio]: https://github.com/heptio
[status]: https://travis-ci.org/heptio/sonobuoy.svg?branch=master
[travis]: https://travis-ci.org/heptio/sonobuoy/#

## Overview

Sonobuoy is a diagnostic tool that makes it easier to understand the
state of a Kubernetes cluster by running a set of [Kubernetes][k8s] conformance
tests in an accessible and non-destructive manner. It is a customizable,
extendable, and cluster-agnostic way to generate clear, informative reports
about your cluster.

Its selective data dumps of Kubernetes resource objects and cluster nodes allow
for the following use cases:

* Integrated end-to-end (e2e) [conformance-testing][e2e]
* Workload debugging
* Custom data collection via extensible plugins

Sonobuoy supports Kubernetes versions 1.8 and later.

[k8s]: https://github.com/kubernetes/kubernetes
[e2e]: conformance-testing.md

## More information

[The documentation][docs] provides detailed information about the conformance
tests, plugins, and snapshots.

[docs]: https://heptio.github.io/sonobuoy/

## Prerequisites

* Access to an up-and-running Kubernetes cluster. If you do not have a cluster,
  follow the [AWS Quickstart Kubernetes Tutorial][quickstart] to set one up with
  a single command.

[quickstart]: http://docs.heptio.com/content/tutorials/aws-cloudformation-k8s.html

* `kubectl` installed. See [installing via Homebrew (MacOS)][brew] or [building
  the binary (Linux)][linux].

* An admin `kubeconfig` file, and the KUBECONFIG environment variable set.

[brew]: https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-with-homebrew-on-macos
[linux]: https://kubernetes.io/docs/tasks/tools/install-kubectl/#tabset-1

## Using the CLI

Sonobuoy also provides a CLI that lets you run Sonobuoy on your cluster. The CLI
records the following results:

* Information about your cluster's hosts, Kubernetes resources, and versions.
* [`systemd`][systemd] logs from each host. Requires a plugin.
* The results of a single e2e conformance test ("Pods should be submitted and
  removed"). See the [conformance guide][conformance] for configuration details.
  Requires a plugin.

[systemd]: https://github.com/systemd/systemd

### CLI Prerequisites

* Golang installed. We recommend [gimme][gimme], with golang version 1.9.4.

* Your $PATH configured:

```
$ export PATH=$GOROOT/bin:$GOPATH/bin:$PATH 
```  

[gimme]: https://github.com/travis-ci/gimme

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

To view the output, copy the output directory from the main Sonobuoy pod to
somewhere local:

```
$ sonobuoy retrieve .
```

This copies a single `.tar.gz` snapshot from the Sonobuoy pod into your local
`.` directory. Extract the contents into `./results` with:

```
mkdir ./results; tar xzf *.tar.gz -C ./results
```

For information on the contents of the snapshot, see the [snapshot
documentation][snapshot].

[snapshot]: snapshot.md

### Cleanup

To clean up Kubernetes objects created by Sonobuoy, run:

```
sonobuoy delete
```

### Run on Google Cloud Platform (GCP)

Note that if you run Sonobuoy on a Google Kubernetes Engine (GKE) cluster, you
must first create an admin role for the user under which you run Sonobuoy:

```
kubectl create clusterrolebinding <your-user-cluster-admin-binding> --clusterrole=cluster-admin --user=<your.google.cloud.email@example.org>
```

## Troubleshooting

If you encounter any problems that the documentation does not address, [file an
issue][issue].

[issue]: https://github.com/heptio/sonobuoy/issues

## Contributing

Thanks for taking the time to join our community and start contributing! We
welcome pull requests. Feel free to dig through the [issues][issue] and jump in.

#### Before you start

* Please familiarize yourself with the [Code of Conduct][coc] before
  contributing.
* See [CONTRIBUTING.md][contrib] for instructions on the developer certificate
  of origin that we require.
* There is a [mailing list][list] and [Slack channel][slack] if you want to
  interact with other members of the community

[coc]: https://github.com/heptio/sonobuoy/blob/master/CONTRIBUTING.md
[contrib]: https://github.com/heptio/sonobuoy/blob/master/CODE_OF_CONDUCT.md
[list]: https://groups.google.com/forum/#!forum/heptio-sonobuoy
[slack]: https://kubernetes.slack.com/messages/sonobuoy

## Changelog

See [the list of releases][releases] to find out about feature changes.

[releases]: https://github.com/heptio/sonobuoy/releases
