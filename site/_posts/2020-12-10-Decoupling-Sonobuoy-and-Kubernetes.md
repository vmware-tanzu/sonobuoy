---
title: Decoupling Sonobuoy and Kubernetes
excerpt: Sonobuoy v0.20.0 release and the future of Sonobuoy project
author_name: Wilson Husin
author_url: https://github.com/wilsonehusin
author_avatar: https://avatars3.githubusercontent.com/u/14004487
categories: [kubernetes, sonobuoy]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team']
---

Historically, we have been following Kubernetes releases to ensure the usability of Sonobuoy with the most recent Kubernetes version. We are excited to announce that beginning Sonobuoy 0.20, Sonobuoy releases will no longer be attached to Kubernetes releases.

## Under the hood

Sonobuoy provides a one-stop solution to run Kubernetes conformance testing. It also handles airgapped testing nicely for users to prepare container images used by the test suite. This feature, however, is also the reason why Sonobuoy releases are tied to Kubernetes releases -- Sonobuoy maintains a copy of list of images internally for each Kubernetes minor. With this release, Sonobuoy no longer maintains that list and will determine the images at runtime.

## Scope of impact

This change only applies to `sonobuoy images` workflow, where one can pull and push container images from official registries to custom registries. Previously, not all `sonobuoy images` subcommand required Docker client to be present. With this release, Docker client is required for all `sonobuoy images` subcommands.
Specifically, `sonobuoy images list` now runs `docker exec` under the hood. This allowed us to always dynamically determine required container images based on the matching Kubernetes conformance image. We’re working on having this API exposed for all plugins, [let us know if you have thoughts](https://github.com/vmware-tanzu/sonobuoy/issues/1199) on this!

## Future of Sonobuoy

We have future plans for Sonobuoy beyond conformance testing which we're excited to announce. Stay tuned!

