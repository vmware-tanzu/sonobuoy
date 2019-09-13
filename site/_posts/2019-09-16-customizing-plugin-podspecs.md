---
title: Customizing Plug-in PodSpecs
# image: https://placehold.it/200x200
excerpt: With custom PodSpecs, you can configure precisely how your plug-ins run on your Kubernetes cluster
author_name: Bridget McErlean
author_url: https://github.com/zubron
# author_avatar: https://placehold.it/64x64
categories: [kubernetes, sonobuoy, conformance]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team']
---

The [Sonobuoy][github] team is continuing to work on one of our main goals: improving the experience for users who want to create their own custom plug-ins.

With the release of Sonobuoy 0.15.2, we’ve introduced a new option in the plug-in definition, `podSpec`.
This option allows you to customize the [Kubernetes PodSpec][kubernetes-podspec] used by Sonobuoy when you create Pods or DaemonSets, giving you even more flexibility and control over how your plug-ins run on your Kubernetes cluster.
Whether you want to add additional containers or volumes; use the host’s PID, network, or IPC namespaces; or configure a security context, you now have the ability to tailor plug-ins precisely to your needs.


Before this option was available, Sonobuoy used a fixed PodSpec for each of the resource types it could create.
These fixed PodSpecs were designed to work best for the two most commonly used plug-ins provided by Sonobuoy: `e2e` and `systemd-logs`.
As our users started to develop more custom plug-ins, we began to see demand for the ability to provide and control specific pod options.
Rather than allowing just a subset of the many possible options to be modified, we decided to give you the ability to specify any option you need.

The use of this feature is entirely optional; if you have existing plug-ins that already fit your needs, they will continue to work as before.

## Getting Started

To make use of this feature, we recommend starting with the default PodSpec used by Sonobuoy and building on top of that.
To view this default PodSpec, you can make use of the new flag `--show-default-podspec` with both the `gen` and `gen plugin` commands.
When this flag is used, the default options will be included in the YAML output for your plug-ins.

Let’s start by creating a new plug-in, based on one of the [Sonobuoy plug-in examples][sonobuoy-examples], by using the `--show-default-podspec` flag with the `gen plugin` command:

```
sonobuoy gen plugin \
  --name customized-pod-spec \
  --image zubron/easy-sonobuoy-cmds:v0.1 \
  --arg="echo hello world" \
  --show-default-podspec
```

This command produces the following plug-in definition:

```
podSpec:
  containers: []
  restartPolicy: Never
  serviceAccountName: sonobuoy-serviceaccount
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  - key: CriticalAddonsOnly
    operator: Exists
sonobuoy-config:
  driver: Job
  plugin-name: customized-pod-spec
spec:
  args:
  - echo hello world
  command:
  - ./run.sh
  image: zubron/easy-sonobuoy-cmds:v0.1
  name: plugin
  resources: {}
  volumeMounts:
  - mountPath: /tmp/results
    name: results
```

As you can see, there is a section in the output named `podSpec`.
This object maps directly to a [Kubernetes PodSpec][kubernetes-podspec], and so any of the options that can be specified there can be added to this definition.
It’s important to note that Sonobuoy will take a given PodSpec and add any resources that it needs on top of it (such as a container to run the Sonobuoy worker), but it will not remove or change anything that was added.

Here, we add another container and enable `hostPID`.
This additional container will run the `ps` command to list all the processes.
Since we have enabled `hostPID`, we will be using the host’s PID namespace.
As a result, the output of this command will list all the processes running on the host, not just within this container.

```
podSpec:
  containers:
    - name: show-host-processes
      image: ubuntu:18.04
      command: ["ps", "-e"]
  restartPolicy: Never
  serviceAccountName: sonobuoy-serviceaccount
  hostPID: true
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  - key: CriticalAddonsOnly
    operator: Exists
sonobuoy-config:
  driver: Job
  plugin-name: customized-pod-spec
spec:
  args:
  - echo hello world
  command:
  - ./run.sh
  image: zubron/easy-sonobuoy-cmds:v0.1
  name: plugin
  resources: {}
  volumeMounts:
  - mountPath: /tmp/results
    name: results
```

We can verify that the custom PodSpec was used by inspecting the logs from the `show-host-processes` container, which are available within the Sonobuoy output. Here’s an example:

```
# Retrieve the results and extract into a directory
outfile=$(sonobuoy retrieve) \
  && mkdir sonobuoy-output \
  && tar -C sonobuoy-output -xf $outfile

# Show the output from the additional container
cat sonobuoy-output/podlogs/heptio-sonobuoy/sonobuoy-customized-pod-spec-job-ee31580678d54e92/logs/show-host-processes.txt
  PID TTY          TIME CMD
    1 ?        00:00:03 systemd
   32 ?        00:00:04 systemd-journal
   46 ?        00:02:52 containerd
  225 ?        00:07:52 kubelet
32708 ?        00:00:00 ps
```


## Adapting Existing Plug-ins

If you already have an existing plug-in, you can take the default PodSpec for your plug-in type and add it to your definition, or you can generate a Sonobuoy manifest, specifying your plug-in and the flag `--show-default-podspec`, and then edit the resulting YAML code:

```
sonobuoy gen -p my-plugin.yaml --show-default-podspec
```

In the output, you will find the following section, where the plug-in definition is loaded into a ConfigMap to provide it to the Sonobuoy aggregator:

```
---
apiVersion: v1
data:
  plugin-0.yaml: |
    podSpec:
      containers: []
      dnsPolicy: ClusterFirstWithHostNet
      hostIPC: true
      hostNetwork: true
      hostPID: true
      serviceAccountName: sonobuoy-serviceaccount
      tolerations:
      - operator: Exists
      volumes:
      - hostPath:
          path: /
        name: root
    sonobuoy-config:
      driver: DaemonSet
      plugin-name: my-plugin
      result-type: my-plugin
    spec:
      command:
      - ./run.sh
      image: my-plugin:v1
      name: plugin
      resources: {}
      volumeMounts:
      - mountPath: /tmp/results
        name: results
kind: ConfigMap
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-plugins-cm
  namespace: sonobuoy
```

As you can see in the `plugin-0.yaml` object, the default PodSpec for a DaemonSet plug-in has been added to our plug-in definition.
We can now edit the `podSpec` as needed and run Sonobuoy using this manifest.

## See It in Action

The ability to customize plug-ins lets you develop more interesting plug-ins.
We look forward to seeing what you create.
The Sonobuoy team is also taking advantage of these features to develop new plug-ins for the community.
To see how we’re making use of custom PodSpecs and other new features, check out our upcoming blog post on creating a Sonobuoy plug-in to run the [CIS Kubernetes Benchmark][cis-benchmark].


Join the Sonobuoy community:

- Get updates on Twitter ([@projectsonobuoy][twitter])
- Chat with us on Slack ([#sonobuoy][slack] on Kubernetes)
- Join the Kubernetes Software Conformance Working Group: [github.com/cncf/k8s-conformance][conformance-wg]

[kubernetes-podspec]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#podspec-v1-core
[sonobuoy-examples]: https://github.com/heptio/sonobuoy/tree/master/examples/plugins/cmd-runner
[cis-benchmark]: https://www.cisecurity.org/benchmark/kubernetes/
[twitter]: https://twitter.com/projectsonobuoy
[slack]: https://kubernetes.slack.com/messages/C6L3G051C
[conformance-wg]: https://github.com/cncf/k8s-conformance
[github]: https://github.com/heptio/sonobuoy
[cncf]: https://www.cncf.io/certification/software-conformance/
