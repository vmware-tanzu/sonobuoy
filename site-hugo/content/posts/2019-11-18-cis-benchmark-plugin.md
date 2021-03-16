---
title: A Sonobuoy Plugin to Check Cluster Security with the CIS Kubernetes Benchmark
excerpt: Check your cluster's security configuration with the new CIS Kubernetes Benchmark plugin.
author_name: John Schnake
author_url: https://github.com/johnschnake
author_avatar: /img/contributors/john-schnake.png
categories: [kubernetes, sonobuoy, conformance]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team']
---

Sonobuoy was always designed to facilitate third-party plugins in order to accommodate custom testing requirements but, until recently, the design of Sonobuoy made some advanced plugins impossible to create.

One of the most requested plugins is for the [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes/) from the Center for Internet Security (CIS). These benchmarks are prescriptive tests for establishing a secure configuration posture for Kubernetes. However, we had difficulty implementing them as a Sonobuoy plugin until recently due to their numerous customization requirements, which were not supported. Over numerous releases, we've chipped away at these problems and we now have a CIS Kubernetes Benchmark plugin which utilizes the implementation provided by [kube-bench](https://github.com/aquasecurity/kube-bench). kube-bench is a Go application that runs the tests documented in the CIS Kubernetes Benchmark.

Now that we’ve implemented these benchmarks as a Sonobuoy plugin, you can easily spot security concerns in your own clusters. In this article, we will:

- Demonstrate how to run the new CIS benchmarks plugin
- Review the new Sonobuoy features that make the plugin possible
- Describe how the plugin was created so that you can understand how it works and how to implement your own custom plugin

## Running the Plugin

The CIS Benchmark plugin is actually two separate plugins: one for master nodes and one for worker nodes. The default YAML files for the plugins have been published in our new [Sonobuoy plugins repo](https://github.com/vmware-tanzu/sonobuoy-plugins). You can run the new plugin by using the following command:

```
$ sonobuoy run \
  --plugin https://raw.githubusercontent.com/vmware-tanzu/sonobuoy-plugins/cis-benchmarks/cis-benchmarks/kube-bench-plugin.yaml \
  --plugin https://raw.githubusercontent.com/vmware-tanzu/sonobuoy-plugins/cis-benchmarks/cis-benchmarks/kube-bench-master-plugin.yaml \
  --wait
```

Once the process has completed, you can see the results by running the following command:

```
$ outfile=$(sonobuoy retrieve) && sonobuoy results $outfile
```

You can also list each test by using the `--mode detailed` option and pipe the results through other tools like `jq` for further analysis. Kube-bench even serializes the entire test object into JSON and places it into the “system-out” field for your inspection. The command below prints the JSON serialization for each of the tests that failed:

```
$ sonobuoy results $outfile --plugin kube-bench-master --mode detailed | jq 'select(.status=="failed")|.details|.["system-out"]' -r | jq
{
  "test_number": "1.1.6",
  "test_desc": "Ensure that the --insecure-port argument is set to 0 (Scored)",
  "audit": "/bin/ps -ef | grep kube-apiserver | grep -v grep",
  "AuditConfig": "",
  "type": "",
  "remediation": "Edit the API server pod specification file /etc/kubernetes/manifests/kube-apiserver.yaml\napiserver.yaml on the master node and set the below parameter.\n--insecure-port=0\n",
  "test_info": [
    "Edit the API server pod specification file /etc/kubernetes/manifests/kube-apiserver.yaml\napiserver.yaml on the master node and set the below parameter.\n--insecure-port=0\n"
  ],
  "status": "FAIL",
  "actual_value": "",
  "scored": true,
  "expected_result": ""
}
...
```

And that's it! If you want to learn more about how we created the plugin, keep reading below. Otherwise, enjoy the plugin and let us know how it works for you.

## Creating the CIS Benchmark Plugin

There are two steps to make a plugin:

- Creating or choosing the image to run in the container
- Creating the plugin definition file for Sonobuoy

For a refresher on how Sonobuoy plugins work, check out our earlier blog post, [Fast and Easy Sonobuoy Plugins for Custom Testing of Almost Anything](https://blogs.vmware.com/cloudnative/2019/04/30/sonobuoy-plugins-custom-testing/).

### Choosing the Image

For our starting line, we are targeting the CIS benchmarks implemented in the [kube-bench](https://github.com/aquasecurity/kube-bench) repo. These benchmarks implement version 1.4.0 of the benchmarks and were written for Kubernetes 1.13.

Luckily for us, the Aqua Security provides an image ready to run the benchmarks. We are able to leverage this image and use our custom plugin specification to redirect the results to Sonobuoy.

_Note: This post was originally published while we were using a custom image temporarily made from the master branch the kube-bench repo. That was meant to be temporary until a release supporting the `--junit` flag was created. The plugin (and this post) have now been updated to use `aquasec/kube-bench:0.2.1`_

### Creating the Plugin Definition

Now we need to tell Sonobuoy how to run the image. The plugin for the CIS benchmarks provides a bit of a challenge compared with simpler plugins, because the benchmarks require privileged access to the host file system, including `hostPID: true` and numerous volume mounts. Up until recently, there was no way for a user to change this information but a recent [pull request for Sonobuoy](https://github.com/vmware-tanzu/sonobuoy/pull/837) enables you to set any pod spec options.

Our steps will be:

- Utilize the `sonobuoy gen plugin` command with the new `--show-default-podspec` option to generate a plugin definition
- Modify the plugin definition with our custom `podSpec` options

First, we need to decide on the command that the plugin should run. The `kube-bench` image defaults to printing results to stdout. Instead, we want to redirect them to a file (kube-bench has a flag for this) and then notify Sonobuoy when complete. 

We can chain these desired commands together as a single bash command and then save the plugin definition to a file:

```
$ sonobuoy gen plugin \
    --name kube-bench-worker \
    --image=aquasec/kube-bench:0.2.1 \
    --type=DaemonSet \
    --format=junit \
    --cmd=/bin/sh \
    --arg=-c \
    --arg="kube-bench --version 1.13 --outputfile /tmp/results/output.xml --junit ; echo -n /tmp/results/output.xml > /tmp/results/done" \
    --show-default-podspec > kube-bench-worker.yaml
```

Now open up `kube-bench-worker.yaml` and modify the `podSpec` to include the extra required volumes and `hostPID: true`:

```
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
  - name: var-lib-etcd
    hostPath:
      path: "/var/lib/etcd"
  - name: var-lib-kubelet
    hostPath:
      path: "/var/lib/kubelet"
  - name: etc-systemd
    hostPath:
      path: "/etc/systemd"
  - name: etc-kubernetes
    hostPath:
      path: "/etc/kubernetes"
  - name: usr-bin
    hostPath:
      path: "/usr/bin"
```

_Note: All these values came from reading the [sample code](https://github.com/aquasecurity/kube-bench/blob/master/job.yaml) in the kube-bench repo._

Lastly, we need to understand that the CIS benchmarks come in two different flavors: one for worker nodes and one for master nodes. Therefore, we need to run different versions of the plugin on the different nodes. To accomplish this, we will:

- Add the appropriate [node affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) to the worker node plugin so it runs on the right nodes.
- Copy the plugin definition to a new file and transform the necessary values for the master plugin name, command, and node affinity.

The affinity section for the worker node plugin should look like this:

```
affinity:
    nodeAffinity: 
      requiredDuringSchedulingIgnoredDuringExecution: 
        nodeSelectorTerms:
        - matchExpressions:
          - key: node-role.kubernetes.io/master
            operator: DoesNotExist
```

For the master nodes, we need to do three things:

- Change `DoesNotExist` to `Exists` to target the correct nodes
- Change the plugin name so the plugins have unique names, which is a Sonobuoy requirement
- Tweak the containers command so that the correct set of checks is run

Each of these things can be done manually, or with the following command:

```
$ cat kube-bench-worker.yaml | \
  sed 's/kube-bench-worker/kube-bench-master/g' | \
  sed 's/- kube-bench/- kube-bench master/g' | \
  sed 's/DoesNotExist/Exists/g' > kube-bench-master.yaml
```

Now you can run the plugins with the following command:

```
$ sonobuoy run \
  --plugin kube-bench-worker.yaml \
  --plugin kube-bench-master.yaml \
  --wait
```

## Summary

Now that a Sonobuoy plugin exists for the CIS Kubernetes Benchmark, you can easily integrate security tests into your workflows and feel more confident in your Kubernetes deployment configuration. Sonobuoy has made great improvements to unblock advanced use cases like this, and we hope to continue providing more valuable feedback about your clusters in an increasingly simple format.



Join the Sonobuoy community:

- Get updates on Twitter ([@projectsonobuoy](https://twitter.com/projectsonobuoy))
- Chat with us on the Kubernetes Slack ([#sonobuoy](https://kubernetes.slack.com/messages/sonobuoy))
- Join the [Kubernetes Software Conformance Working Group](https://github.com/cncf/k8s-conformance)
