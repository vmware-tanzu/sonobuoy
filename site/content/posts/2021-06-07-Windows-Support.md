---
title: Introducing Support for Windows Clusters
image: /img/sonobuoy.svg
excerpt: Sonobuoy can now run on Windows nodes meaning faster, consistent testing on all of your clusters.
author_name: John Schnake
author_url: https://github.com/johnschnake
categories: [kubernetes, sonobuoy, windows]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team', 'John Schnake']
date: 2021-06-07
slug: sonobuoy-adds-windows-support
---

Sonobuoy’s latest release, v0.51.0 introduces beta support for Windows, closing one of our longest standing [roadmap items](https://github.com/vmware-tanzu/sonobuoy/issues/732). This means that the Sonobuoy image can run on Windows nodes (including Windows versions "1809", "1903", "1909", "2004", and "20H2") in Kubernetes clusters as well as having a Windows compatible Sonobuoy client. This will simplify testing of Kubernetes clusters with Windows nodes and speed development of full Kubernetes support and certification of Windows clusters.

In this blog post, we want to show you how to use Sonobuoy to run either the Kubernetes end-to-end tests or your own custom plugins on mixed node clusters. In a follow-up post, we will discuss specifically running the “e2e” plugin for the Kubernetes tests.

## Why Does Sonobuoy Need to Run on Windows Nodes?

A reasonable question would be: why does Sonobuoy needs to run on Windows nodes at all? If the cluster has some Linux nodes available, can the Sonobuoy pod just be placed there? Yes and no. First take a look at the general Sonobuoy architecture:

![Architecture](/img/image1.png "Sonobuoy architecture")

The aggregator pod can be placed anywhere in the cluster, so yes it can be placed on a Linux node if one is available. However, on any node where you want a plugin to run, Sonobuoy runs as a sidecar so that it can report progress and results. For any information gathering on Windows nodes such as: gathering logs, executing custom tests, or checking OS settings Sonobuoy needs to be compatible with Windows.

By supporting Windows, Sonobuoy enables you to use the same testing and debugging flow that you use for your existing Linux clusters. That means it is easier and faster to experiment with Windows cluster adoption.

## Getting A Windows Cluster

If you would like to run Sonobuoy from a Windows node, you need to set up a Kubernetes cluster with Windows nodes. One of the easiest ways to get started is to use an Azure account and create an AKS cluster. It is easy but does require an account and costs based on resources used. Find more instructions [here](https://docs.microsoft.com/en-us/azure/aks/windows-container-cli).

For the purposes of this blog post, you can start with a 2 node cluster:
- 1 control plane, Linux node 
- 1 Windows node for workloads

We will discuss different cluster configurations and their implications in the following post.

## Your Choice of CLI Client 

Along with adding Windows support for the Sonobuoy image, we also officially support a Windows client. We've fixed numerous issues so that the experience on Windows matches that of the other platforms. So regardless of whether or not you're running from a Mac, Linux, or Windows machine, Sonobuoy has a compatible client.

## Running A Windows Plugin

Now that you have a Windows cluster and client, let's show how easy it is to run a plugin. In fact, it is exactly the same as running any other Sonobuoy plugin. Let’s run the “windows-events” [example plugin](https://github.com/vmware-tanzu/sonobuoy-plugins/tree/master/examples/windows-plugin) from the [sonobuoy-plugins](https://github.com/vmware-tanzu/sonobuoy-plugins) repository.

Since the Windows support is new, you need to either have the latest version of Sonobuoy or specify the image via a flag. For explicitness, we'll show the flag but realize that it is optional if you’ve updated your CLI client.

```
url=https://raw.githubusercontent.com/vmware-tanzu/sonobuoy-plugins/master/examples/windows-plugin/plugin.yaml
img=sonobuoy/sonobuoy:v0.51.0

sonobuoy run --plugin $url --sonobuoy-image=$img --wait
```

Once that command completes you can check the status and/or download results:

```
sonobuoy status
tarball=$(sonobuoy retrieve)
```

Once you have the tarball of results you can either extract the contents yourself or just inspect it with `sonobuoy results`. Check out our previous [post](https://sonobuoy.io/simplified-results-reporting-with-sonobuoy/) about different ways to inspect the results. The plugin reported the actual events output from the Windows machine:

```
Sonobuoy results $tarball --mode=detailed



   ProviderName: Microsoft-Windows-DHCPv6-Client

TimeCreated                     Id LevelDisplayName Message                    
-----------                     -- ---------------- -------                    
5/25/2021 1:05:39 AM          1008 Error            An error occurred in ini...
5/25/2021 1:05:39 AM          1008 Error            An error occurred in ini...


   ProviderName: Microsoft-Windows-Security-Auditing

TimeCreated                     Id LevelDisplayName Message                    
-----------                     -- ---------------- -------                    
5/25/2021 1:05:39 AM          4688 Information      A new process has been c...


   ProviderName: Microsoft-Windows-PowerShell
…
…
…
```

That’s all it takes to run a Windows plugin: that is to say, nothing extra at all! Now that Sonobuoy publishes the Windows images as part of a multi-arch image, each node will automatically pull the correct version (Linux/Windows). The only thing you need is a plugin that targets a Windows compatible image and a cluster to run it on.

## How To Help

One of the easiest ways you can help contribute is to simply tell us what you want, for instance:
- What test frameworks or output formats are common in your Windows workflows?
- What kind of logs or system information do you routinely find yourself needing from the Windows machines?
- What part of Kubernetes do you routinely/specifically want to test? All of Conformance to see where Windows is not compatible? Just Windows features? Some other subset?

If telling us these things is good, helping us actually build or test them is even better. You can create an issue or pull request to [sonobuoy][sonobuoy] or [sonobuoy-plugins][sonobuoy-plugins] repos with complete code or even just ideas and we can help build tools to help the community together.

## Join the Sonobuoy community:

- Star/watch us on Github: [sonobuoy][sonobuoy] and [sonobuoy-plugins][sonobuoy-plugins]
- Get updates on Twitter (@projectsonobuoy)
- Chat with us on Slack (#sonobuoy on Kubernetes)
- Join the K8s-conformance [working group](https://github.com/cncf/k8s-conformance)

[sonobuoy]: https://github.com/vmware-tanzu/sonobuoy
[sonobuoy-plugins]: https://github.com/vmware-tanzu/sonobuoy-plugins
