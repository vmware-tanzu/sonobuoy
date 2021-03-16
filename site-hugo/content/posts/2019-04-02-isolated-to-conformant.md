---
title: Isolated to Conformant - Testing Air-Gapped Kubernetes with Sonobuoy 0.14
image: /img/sonobuoy.svg
excerpt: With support for running Kubernetes end-to-end tests in air-gapped environments, it is now possible to run the end-to-end suite and validate your cluster’s state without Internet connectivity.
author_name: John Schnake
author_url: https://github.com/johnschnake
author_avatar: /img/contributors/john-schnake.png
categories: [kubernetes]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team']
date: 2019-04-02
---
_Previously posted on the [VMware Cloud Native Apps blog](https://blogs.vmware.com/cloudnative/)_

Today we are releasing Sonobuoy 0.14.0, which delivers on one of our top roadmap goals: support for running Kubernetes end-to-end tests in air-gapped environments. It is now possible to run the end-to-end suite and validate your cluster’s state without Internet connectivity or investment in a custom, ad hoc work around.

## Air-Gapped Installations and Testing

Running critical systems in air-gapped environments, where the system can’t reach out to the Internet, has long been a common practice to limit the attack surface. Although installing Kubernetes in air-gapped environments has been possible since before Kubernetes 1.6, testing those clusters for conformance was difficult.

The end-to-end suite uses numerous test images, which it tries to pull from public Docker registries; without access to those registries, the cluster is unable to run many of the tests required. Even without an air-gapped environment, restrictive networking policies can make it impossible to effectively run the Kubernetes end-to-end test suite.

Since 2017, multiple people in the community have been working together to make the Kubernetes changes necessary to centralize and then customize the registries used during testing. Since Kubernetes 1.13, testing in air-gapped installations using private registries has technically been possible but was still a pain point for most users. The images were localized in the code, but you still had to manually look them up, get them into your private registry, and inform the tests to use your registry.

Sonobuoy solves all three of these problems for you. By tracking the images required to run the end-to-end test suite and automating all the scripting to move the images, the new version of Sonobuoy makes it easier than ever to test your air-gapped installations.

[Read the full article here.](https://blogs.vmware.com/cloudnative/2019/04/02/testing-air-gapped-kubernetes-sonobuoy-0-14/)
