---
title: Setting Environment Variables for Plugins on the Fly with Sonobuoy 0.15.0
image: /img/sonobuoy.svg
excerpt: It is now possible to easily modify the environment variables of any plugin without editing a YAML file.
author_name: John Schnake
author_url: https://github.com/johnschnake
author_avatar: /img/contributors/john-schnake.png
categories: [kubernetes]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team', 'John Schnake']
date: 2019-06-26
slug: env-vars-on-the-fly
---
With the release of [Sonobuoy][first-blog] 0.15.0, we continue to support one of our top roadmap goals: enhanced support for custom plugins. It is now possible to easily modify the environment variables of any plugin without editing a YAML file.

[Sonobuoy][github], an open source diagnostic tool, runs upstream Kubernetes tests to generate reports that help you understand the state of your cluster. Sonobuoy is the underlying technology powering the [Certified Kubernetes Conformance Program][cncf], which was created by the Cloud Native Computing Foundation (CNCF) and is used by every Certified Kubernetes Service Provider.

The new flag (`--plugin-env`) ensures that plugins will no longer require Sonobuoy to add new flags in order to support setting a simple environment variable. So as plugins (like the Kubernetes end-to-end tests) expose more customizability through environment variables, that functionality is immediately available via the Sonobuoy command-line interface. In addition, custom plugins have the same flexibility and can have their environment variables tweaked from the command-line, too.

## Use Case: Running E2E Tests in Dry-Run Mode

When running Sonobuoy, you’re often faced with two questions:

 1. What tests actually get run?
 2. How can you be sure that the custom focus or skip parameters that you provide will lead to the tests you want being run?

The answer to both of those questions is to run the tests in “dry-run” mode. The underlying test framework exposes this mode in the test image through the environment variable `E2E_DRYRUN`. When that variable is set, the test run skips all the execution logic of the tests and just reports the tests it would have run as “passed” and the others as “skipped.”

Until now, using the dry-run variable was not very user-friendly — it was not particularly well known and required you to save the `sonobuoy gen` output and manually add the environment variable to the YAML file.

Now you can do all that with a simple one-liner:

```
sonobuoy run --plugin-env e2e.E2E_DRYRUN=true
```

## Use Case: Customizing Your Own Plugin

You can edit not only the two built-in plugins (`e2e` and `systemd-logs`) this way, but also custom plugins of your own.  The `--plugin-env` flag takes values of the form:

```
[plugin name].[env var]=[value]
```

For instance, here’s how you would instruct Sonobuoy to load your custom plugin and then customize it:

```
sonobuoy run --plugin myPlugin --plugin-env myPlugin.ENV=VAL
```

These changes make it easier than ever to modify the runs of your plugins, and I’m excited to see what you will do with it.

A big thank you goes out to the Sonobuoy community for your continuous feedback and contributions for this release — and a special thanks to [padlar][padlar] for his contribution.

Join the Sonobuoy community:

 - Get updates on Twitter ([@projectsonobuoy][twitter])
 - Chat with us on Slack ([#sonobuoy][slack] on Kubernetes)
 - Join the Kubernetes Software Conformance Working Group: [github.com/cncf/k8s-conformance][conformance-wg]

[padlar]: https://github.com/padlar
[twitter]: https://twitter.com/projectsonobuoy
[slack]: https://kubernetes.slack.com/messages/sonobuoy
[conformance-wg]: https://github.com/cncf/k8s-conformance
[first-blog]: https://blogs.vmware.com/cloudnative/2019/02/21/certifying-kubernetes-with-sonobuoy/
[github]: https://github.com/vmware-tanzu/sonobuoy
[cncf]: https://www.cncf.io/certification/software-conformance/