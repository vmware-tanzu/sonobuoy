# Example Windows Plugin

This directory contains the code to build a Docker image which can be run as a Sonobuoy plugin on a Windows node.

## Prerequisites

The preferred method of building the image (when on a Mac or Linux machine) is to utilize `docker buildx` to build the Windows image. Alternatively, you can use a Windows machine (or VM). One example of how to set up a Windows VM and utilize it as your Docker context in [StefanScherer/windows-docker-machine](https://github.com/StefanScherer/windows-docker-machine).

To run the plugin you need to have a Kubernetes cluster with Windows nodes. The Windows nodes should have the label `kubernetes.io/os=windows`. There are multiple ways to set up such a cluster; one example is to add a Windows node group to an EKS cluster as described on this [page](https://docs.aws.amazon.com/eks/latest/userguide/windows-support.html).

## Running the plugin

You run the plugin just like any other; there are no special requirements with it being a Windows plugin:

```
sonobuoy run --plugin plugin.yaml --wait
```

That will schedule the Sonobuoy aggregator on a Linux node and the plugin on the Windows node. If you want the Sonobuoy aggregator to also run on a Windows node, you'll need to add the `--aggregator-node-selector=kubernetes.io/os:windows` flag:

```
sonobuoy run --plugin plugin.yaml --aggregator-node-selector=kubernetes.io/os:windows --wait
```

Optionally, if you have multiple types of Windows nodes, you may need to add the node selector for the architecture as well:

```
sonobuoy run --plugin plugin.yaml --aggregator-node-selector=kubernetes.io/os:windows --aggregator-node-selector=kubernetes.io/arch:amd64 --wait
```

## Known issues

If the results tarball is opened on a non-Windows machine, the resulting files/folders will be flattened awkwardly. This is because the Windows path separator is just being interpreted as part of the individual file names. As a result, you end up with folders/files like:

```
...
should\be\a\series\of\nested\folders
should\be\a\series\of\nested\folders\file-in-folder
...
```

We are open to solutions for this problem; for the time being it is easy to tolerate and surely has various client-side solutions.
