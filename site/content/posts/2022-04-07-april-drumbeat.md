---
title: "April Drumbeat: Better Output, Faster Debugging, and More Plugins"
image: /img/sonobuoy.svg
excerpt: An update about the recent improvements to Sonobuoy, which include more support for sidecars, more debugging information surfaced, and unique new plugins. 
author_name: John Schnake
author_url: https://github.com/johnschnake
categories: [kubernetes, sonobuoy, plugins]
tags: ['Sonobuoy Team', 'John Schnake']
date: 2022-04-07
slug: april-drumbeat
---

# Sonobuoy Drumbeat

We’ve recently switched to releasing Sonobuoy in more rapid, smaller releases and so we wanted to take an opportunity to discuss recent changes more in depth.

## Release Cadence

We’ve moved to releasing Sonobuoy on a weekly basis as long as we have actual code changes. There had been a few instances in the past where features were on the main code branch and being used by developers but not readily available to the wider community and we want to limit that.

## Improved Plugins with Sidecar Containers

Plugins have been able to have sidecar containers ever since the podSpec was made configurable. However, they weren’t able to share in some of the features that Sonobuoy provided to the main plugin container; namely automatic environment variables and config-maps.

Recent changes have made it so that if you have a sidecar container, it will have access to the same environment variables that Sonobuoy adds to the main container (e.g. `SONOBUOY_K8S_VERSION`, `SONOBUOY_RESULTS_DIR`, etc).

In addition, the config-maps specified in the plugin definition will also be mounted in the sidecar containers at the same location (`SONOBUOY_CONFIG_DIR`).
This allows you to have complex, configurable behavior in both the main container and the sidecar.
For instance, the plugin and sidecar may run different on specific versions of Kubernetes, take complex configuration files via configmaps (as opposed to just flags/arguments), and the sidecar can easily access the plugin's results as they are created. 

## Better Messaging When Sonobuoy Fails to Start

Despite adding more output to Sonobuoy when using the `--wait` flag, if the main aggregator pod doesn’t start up there was no useful output provided on the command line.

We’ve added additional output so that if the aggregator pod is not ready we begin printing information on the command line. This will help quickly and easily identify when there are problems pulling the image or other scheduling issues with the pod that prevent Sonobuoy from running at all.

## More Debug Information Surfaced from Results

The Sonobuoy results tarball has lots of useful information in it to help debug problems. However, it's a pain to have to dig through the large number of files to find simple pieces of information.

As a result, we’ve surfaced this information automatically to make your debugging process more efficient. In particular, when running `sonobuoy results`, data regarding cluster version, node health, pod health, and errors detected in logs will be printed alongside the plugin results.

If you only want this information you can also refer to it via the psuedo-plugin name, `sonobuoy`:

```
> sonobuoy results -p sonobuoy results.tar.gz
Run Details:
API Server version: v1.20.0
Node health: 3/3 (100%)
Pods health: 19/19 (100%)
Errors detected in files:
Errors:
25024 podlogs/kube-system/kube-controller-manager-kind-control-plane/logs/kube-controller-manager.txt
 9744 podlogs/kube-system/kube-apiserver-kind-control-plane/logs/kube-apiserver.txt
  126 podlogs/kube-system/kube-scheduler-kind-control-plane/logs/kube-scheduler.txt
    2 podlogs/kube-system/etcd-kind-control-plane/logs/etcd.txt
Warnings:
1627 podlogs/kube-system/kube-controller-manager-kind-control-plane/logs/kube-controller-manager.txt
 330 podlogs/kube-system/kube-apiserver-kind-control-plane/logs/kube-apiserver.txt
  35 podlogs/kube-system/kube-scheduler-kind-control-plane/logs/kube-scheduler.txt
…

```

> Note: All of this data comes from the automatic queries that Sonobuoy does in addition to your configured plugins and not from the plugins themselves. So errors/warnings are only detected in the configured namespaces (defaults to kube-system and the namespace Sonobuoy is running in).

There are clearly lots of errors and warnings in the control-plane logs, but if you are trying to trace down errors it can be helpful to see if there are an increased number in a particular file. 

The error/warning detection currently checks for basic string patterns only, but this functionality may be extended or made more configurable in the future.

## Ongoing Plugin Efforts

As part of [strategy][strategy] we are continuing to develop additional plugins that can be utilized by the community. We wanted to mention two that are currently in development which we hope will find broad support.

### Sonolark

[Sonolark][sonolark] is a plugin that allows you to script your own business logic without any of the image management that normally goes with maintaining a plugin. It uses the [Starlark][starlark] language as the scripting language, which allows you to take advantage of our base image's library of features and assertions. It provides access to the Kubernetes API, integration with Sonobuoy, and an expanding library of assertion/translation functions.

This makes a really lightweight solution which can help in numerous situations where you might otherwise simply provide end-users with instructions to follow.
In addition, since there is no image management involved, development can be extremely fast since you just tweak the script and have your new plugin ready to use or share.
You can even run Sonolark locally with just the binary, so you can tweak your checks and error messaging in the script, run `sonolark -f <your script>`, and get feedback right away without building any new binaries or images.

Sonolark has just had its initial commit so we hope that you will give it a try and help us expand its functionality and make it even better.

### Post-Processor

The [post-processor][postprocessor] is unconventional in that it isn’t a plugin on its own. Instead, it is intended to be used as a sidecar container for any of your other plugins.

The idea is that internally Sonobuoy uses a YAML format to report results and there are occasions where you may want to make modifications to the data after the fact. This is particularly true when you can’t modify the plugin code directly (e.g. for the upstream Kubernetes E2E plugin).

By utilizing [ytt][ytt], the user can provide a data transform via a config-map and do things like:

 - Add additional context to errors, such as links to relevant KB articles
 - Remove unnecessary tests to make output more readable (e.g. remove those 5,000 skipped tests from the e2e output)
 - Automatically tag or remove tests known to be flaky while they get fixed
 - Inject debugging steps directly into failure messages

Since this is a generic tool, we are excited to see how others might decide to utilize it. The post-processor “plugin” just had its initial commit so we hope you’ll experiment with it and provide comments or code as it continues to develop.

## Join the Sonobuoy community

 - Star/watch us on Github [sonobuoy](https://github.com/vmware-tanzu/sonobuoy) and [sonobuoy-plugins](https://github.com/vmware-tanzu/sonobuoy-plugins)
 - Get updates on Twitter ([@projectsonobuoy][twitterLink])
 - Chat with us on Slack (#sonobuoy on Kubernetes)

[postprocessor]: https://github.com/vmware-tanzu/sonobuoy-plugins/tree/main/post-processor
[ytt]: https://carvel.dev/ytt/
[sonolark]: https://github.com/vmware-tanzu/sonobuoy-plugins/tree/main/sonolark
[starlark]: https://github.com/bazelbuild/starlark
[twitterLink]: https://twitter.com/projectsonobuoy