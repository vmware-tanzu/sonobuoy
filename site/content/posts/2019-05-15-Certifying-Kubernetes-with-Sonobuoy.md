---
title: Certifying Kubernetes with Sonobuoy
image: /img/image1.png
excerpt: There are many ways to create Kubernetes clusters and many environments that can host them. As a result, platform operators find it difficult to determine whether a cluster is properly configured and whether it is working as it should.
author_name: Sonobuoy Team
# author_avatar: https://placehold.it/64x64
categories: [kubernetes]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team']
date: 2019-05-15
slug: certifying-kubernetes-with-sonobuoy
---
There are many ways to create Kubernetes clusters and many environments that can host them. As a result, platform operators find it difficult to determine whether a cluster is properly configured and whether it is working as it should.

**Sonobuoy is an open-source diagnostic tool that makes it easier to understand the state of a Kubernetes cluster by running a set of upstream Kubernetes tests in an accessible and non-destructive manner.** It is a customizable, extensible, and cluster-agnostic way to generate clear, informative reports about your cluster — regardless of your deployment details. Sonobuoy is the underlying technology powering the [Certified Kubernetes Conformance Program](https://www.cncf.io/certification/software-conformance), which was created by the Cloud Native Computing Foundation (CNCF) and is used by every [Certified Kubernetes Service Provider](https://www.cncf.io/certification/kcsp/).

![Certified Kubernetes](/img/image3.png "Certified Kubernetes")

Sonobuoy has three components:
* A command-line utility that you use to trigger conformance tests, check status, view activity logs, and retrieve and analyze test results
* An aggregator that runs in a Kubernetes pod to start plugins and aggregate their test results
* Plugins that execute in ephemeral namespaces with a Sonobuoy sidecar to run specific tests or conformance frameworks

![Sonobuoy Run](/img/image1.png "Sonobuoy Run")

With a single Sonobuoy command, you can run the same tests that are used to qualify an upstream Kubernetes release. This ability provides strong levels of assurance that your cluster is configured correctly, and you can use the tool to debug configuration problems.

## Native Extensibility Through Plugins
Sonobuoy provides several plugins out of the box, including a systemd log collector and the upstream end-to-end Kubernetes conformance test suite. Sonobuoy is the community standard tool for executing conformance tests on a Kubernetes cluster; however, its architecture is designed to accomplish much more.

The open plugin architecture equips you, as a platform operator, with the means to develop custom conformance and validation tests for environments before they go into production. A [custom plugin](https://github.com/vmware-tanzu/sonobuoy/blob/main/docs/plugins.md) can be developed by creating a plugin definition file that describes how the plugin is structured and what parameters the plugin requires. The plugin then needs to follow a documented API that provides a communication mechanism for Sonobuoy to inform it of the plugin’s status including whether it is pending, running, or complete.

![Sonobuoy Plugin](/img/image4.png "Sonobuoy Plugin")

Other plugins from the community exist, such as [Bulkhead](https://github.com/bgeesaman/sonobuoy-plugin-bulkhead). Bulkhead assesses the compliance of a cluster’s control plane and worker nodes with the security guidelines for Kubernetes established in the [CIS Benchmarks](https://www.cisecurity.org/benchmark/kubernetes/). These benchmarks are executed using kube-bench, a tool that implements the CIS Benchmarks based upon the version of Kubernetes that is deployed.

The Sonobuoy team is community driven and would love to hear of any plugins that you have created or ideas on what you would like to see developed. Please open an [issue](https://github.com/vmware-tanzu/sonobuoy/issues/new/choose) or find us on the Kubernetes Slack [#sonobuoy](https://kubernetes.slack.com/messages/C6L3G051C)!

## Future Plans
One of the recurring requests from the community has been the need to [run Sonobuoy in an air-gapped environment](https://github.com/vmware-tanzu/sonobuoy/issues/160), meaning that the cluster is physically isolated from the Internet. This is important for medical and financial services industries with stringent security requirements. For months, the upstream Kubernetes community has been working to make this possible and we are committed to ensuring that capability is implemented. This will empower anyone to verify and debug their clusters, regardless of Internet connectivity.

Another high-priority item is improving the developer experience and user documentation for creating and running custom plugins. The existing API that plugins have to meet is small but we know there are still some pain points in developing and using your own plugins. Expect documentation improvements, more examples, and improved integration with the CLI for custom plugins. By streamlining the plugin process, we hope to empower the community to create their own plugins and solve even more problems.

Check out the planned features for the next release (version 0.14) by looking at our [Github milestone](https://github.com/vmware-tanzu/sonobuoy/issues?utf8=%E2%9C%93&q=is%3Aissue+milestone%3Av0.14+).  Sonobuoy is built by the community so make your voice heard! By creating or commenting on Github issues and communicating in Slack you can help influence future priorities. Issues labeled with [help wanted](https://github.com/vmware-tanzu/sonobuoy/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22+label%3A%22help+wanted%22) or [good first issue](https://github.com/vmware-tanzu/sonobuoy/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) are a great place to start engaging with the project.

## Join the Sonobuoy Community!
* Get updates on Twitter ([@projectsonobuoy](https://twitter.com/projectsonobuoy))
* Chat with us on Slack ([#sonobuoy​](https://kubernetes.slack.com/messages/sonobuoy) on Kubernetes)
* Join the K8s-conformance working group: <https://github.com/cncf/k8s-conformance>

_Previously posted at: <https://blogs.vmware.com/cloudnative/2019/02/21/certifying-kubernetes-with-sonobuoy/>_
