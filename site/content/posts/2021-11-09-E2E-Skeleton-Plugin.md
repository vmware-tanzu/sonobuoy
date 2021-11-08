---
title: "Skip the Boilerplate: A Plugin Ready For Your Business Logic"
image: /img/sonobuoy.svg
excerpt: A new plugin helper library along with the new, upstream e2e-framework helps you jump straight into your testing your business logic. 
author_name: John Schnake
author_url: https://github.com/johnschnake
categories: [kubernetes, sonobuoy, plugins]
tags: ['Sonobuoy Team', 'John Schnake']
date: 2021-11-09
slug: plugin-starter
---

## Introduction

When you decide to test your Kubernetes native code it can be a big chore. When I've been in that position, I do one of the following:
 - start a new project
   - Simple in concept but you're left with a ton of boilerplate code so that you can do simple things like
 check if your app is running. You've got to deal with setting up client-go, you've got to figure out all the wiring
 to set up builds, images, API clients, avoid test collision in the cluster, etc.
 - vendor upstream k8s testing code
   - This has the benefit of giving you lots of helper functions, but it is a huge dependency and bloats your tests. In addition,
 even with all the new helpers, there is SO much code that it can be confusing about what to use and how to use it.

Instead, we want to suggest you try our new [e2e-skeleton plugin][skeleton]. It is obviously just a starting point (not a plugin you would
run without modification), but it handles much of the basic boilerplate code for you, including:
 - Configures an API client for k8s CRUD operations
 - Reports results in a format that Sonobuoy understands (e.g. `sonobuoy results` would be able to tell you pass/fail/etc)
 - Reports results to the Sonobuoy aggregator once done
 - Creates/destroys namespaces for each test to avoid collisions
 - Sends progress updates to the Sonobuoy aggregator so that `sonobuoy status` can report on test progress while running

In this blog post I'll show off the plugin, its features, and show you how to skip the boilerplate so you can get to testing your business logic.

## How to use the e2e-skeleton plugin

This is a "plugin" which is _meant_ to be modified prior to use. Currently, it is
just a skeleton for you to put your tests into. So in order to get set up and write your own tests,
you have to:

 - Clone the repo (or copy the example code) `git clone https://github.com/vmware-tanzu/sonobuoy-plugins`
 - Modify `build.sh` to contain your own registry/image name
 - Modify `plugin.yaml` to reference the registry/image for your tests
 - Add more tests!

## Integrations with Sonobuoy

#### Preconfigured Project Files
 
Since this code is meant to be your starting point for your own plugin, we
include the basic files you'll need during your plugin's lifecycle:

- Dockerfile
  - All Sonobuoy plugins end up launching containers in your Kubernetes cluster. So to make a plugin you'll need to containerize your code. We've done that for you already and used a base image that was easy to work with and debug. Feel free to modify this as desired.
- Instructions/script to build your code
  - You've got the code and Dockerfile, but you still have to actually build/push the container. We provide a simple `build.sh` script that will build/push the image to your specified registry. (This is currently a very minimal starting point and would love help making it more robust.)
- The plugin.yaml definition
   - This is the file that Sonobuoy reads to understand your plugin and what to execute. As you create your plugin, you'll just need to tweak this file to reference your container registry/image and any other small changes you may want to add such as providing unique flags to the test invocation.
 
Once you copy this plugin code locally and write your tests, all you'll need to do to build and run your plugin is:
```bash
$ ./build.sh
$ sonobuoy run -p ./plugin.yaml --wait
``` 

#### Reporting Format

The test output is JSON produced via `go test` which Sonobuoy
parses to provide useful status/results reports via `sonobuoy status` and `sonobuoy results`.
 
When you use this plugin you won't have to worry about the
test output format at all or think about which testing library to import. Just write your tests.

#### Progress updates

This plugin skeleton will provide progress updates back to the Sonobuoy aggregator
so that users waiting on results can see not only that tests are still running, but
how many have passed/failed/completed (and what those failures are).

To get basic progress information, just run:
```bash
$ sonobuoy status
```

To get the complete status information, add the `--json` flag.

## Benefits of E2E Framework

#### Namespaces for Test Isolation

By taking advantage of the e2e-framework's testing hooks, we create test Kubernetes namespaces
for each test so that its resources can be isolated from the others. We also
handle deleting those namespaces once the test is completed.

#### API Client

The e2e-framework provides a Kubernetes API that has already been configured/instantiated so
that you don't have to deal with client-go or loading kubeconfig files.

To access the client within a test, just access it via:
```
    cfg.Client()
```

And use it to make API calls such as listing pods:
```
    err := cfg.Client().Resources("kube-system").List(context.TODO(), &pods)
```

## Conclusion & Roadmap

We hope that this plugin skeleton provides a useful jumping off point for starting
to code your own Kubernetes integration tests.

If you use the plugin and find yourself adding other code that would be useful for others
to start out with, let us know so we can work together to integrate more, useful
features into this base.

A huge thank you to [Vladimir Vivien][vlad] and the other contributors on for the [e2e-framework][e2e-framework].
Their whole goal is to help make testing in Kubernetes easier to take a look and see what else their framework can help
you with.

Happy testing!

## Join the Sonobuoy community:

- Star/watch us on Github: [sonobuoy][sonobuoy] and [sonobuoy-plugins][sonobuoy-plugins]
- Get updates on Twitter (@projectsonobuoy)
- Chat with us on Slack ([#sonobuoy](https://kubernetes.slack.com/archives/C6L3G051C) on Kubernetes)
- Join the K8s-conformance [working group](https://github.com/cncf/k8s-conformance)

[sonobuoy]: https://github.com/vmware-tanzu/sonobuoy
[sonobuoy-plugins]: https://github.com/vmware-tanzu/sonobuoy-plugins
[skeleton]: https://github.com/vmware-tanzu/sonobuoy-plugins/tree/main/examples/e2e-skeleton
[vlad]: https://github.com/vladimirvivien
[e2e-framework]: https://github.com/kubernetes-sigs/e2e-framework