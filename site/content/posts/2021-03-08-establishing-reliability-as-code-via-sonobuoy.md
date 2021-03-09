# Establishing Reliability-As-Code via Sonobuoy

We are happy to introduce the [Reliability Scanner for Kubernetes as a Sonobuoy Plugin](https://github.com/vmware-tanzu/sonobuoy-plugins/tree/master/reliability-scanner)! It is a light-weight Go program that includes an extensible set of reliability assessments, or checks, performed against various components of a cluster, such as Pods, Namespaces, Services, etc. The Reliability Scanner runs as a container that serves as a Sonobuoy plugin and uses a configuration file to define customized checks. Kubernetes cluster operators can then configure appropriate constraints to run reliability scanner checks against their clusters. This project is based upon efforts and practices of [VMware’s Customer Reliability Engineering (CRE) team](https://tanzu.vmware.com/content/blog/hello-world-meet-vmware-cre).

This article provides a walk-through of the components usedby the initial set of checks recommended by VMware CRE.  You can create your own customized checks or extend the existing checks. 

## Probes
Probes are periodic checks against containers that run our services and notify the kubelet when the container is alive and ready to accept traffic. Probes help Kubernetes make more informed decisions about the current status of one or many particular Pods behind a Service. 

There are three kinds of probes: 

- `startupProbe`: confirms the application within the container is available
- `livelinessProbe`: confirms the container is in a running state
- `readinessProbe`: confirms the container is ready to respond to requests 

The reliability scanner checks allow Kubernetes cluster operators to report Pods that are missing the liveliness and readiness  probes, as part of the Sonobuoy report.  Currently only liveliness and readiness checks are recommended by the VMware CRE team, however this check could extend to include startupProbe, if necessary, for your clusters.  Additionally, a potential improvement in future releases would be to add a labeling capability for operators to specify which Pods to skip.

### Owner annotations
Owner annotations provide the ability to replicate patterns of deployment across multiple physical hosts. In a multi-tenant Kubernetes deployment, owner annotations support incident management by making it easy to figure out parties who should be notified.  The scanner will check any user-provided annotations and allowed domains, against the Services running in the cluster, to reports back if the desired values are not set appropriately.

### Minimum desired quality of service
Quality of service (QoS) in Kubernetes is used by the scheduler to make decisions around scheduling and evicting pods on a node. There are three classes of QoS in Kubernetes: **BestEffort**, **Burstable**, and **Guaranteed**.

The upfront reservation on scheduled Nodes depend on how our application requirement was defined.  One of the reliability practices that allows applications to operate more efficiently is to ensure that the Node has or reserves enough resources when a container is scheduled.  The reliability scanner will report any Pod that does not meet the minimum desired QoS class defined in the constraint.

### Getting Started
With the Reliability Scanner, you can add new reliability checks easily.  Please refer to the [README](https://github.com/vmware-tanzu/sonobuoy-plugins/blob/master/reliability-scanner/README.md) for more details.

First, make sure that [Sonobuoy CLI](https://github.com/vmware-tanzu/sonobuoy), Docker, make, ytt, and kubectl are installed on your local machine.  There’s a helpful [Makefile](https://github.com/vmware-tanzu/sonobuoy-plugins/blob/master/reliability-scanner/Makefile) that handles the necessary setup and templating needed to run the customized reliability check YAML.  


### Adding a new QOS reliability check
The reliability scanner is ready to be used to scan your cluster. However, a cluster operator who is interested in creating a customized reliability check, will need to make modifications in two places in order to create a new customized reliability check.  First, the new check must be defined by name, description, kind, and spec, in a [YAML configuration file](https://github.com/vmware-tanzu/sonobuoy-plugins/blob/master/reliability-scanner/plugin/reliability-scanner-custom-values.lib.yml) as shown below:

```
name: ”Pod QOS check”
   description: Checks each pod to see if the minimum desired QOS is defined
   kind: v1alpha1/pod/qos # location of the check logic
   spec: # configurable by cluster operators
     minimum_desired_qos_class: Guaranteed # defines minimum expected QoS
     include_detail: true # includes the actual Pod QoS in the report
```

Next, some Golang code must be written which actually does the check we need.  As a starting point, the check will have to satisfy the [Querier](https://github.com/vmware-tanzu/sonobuoy-plugins/blob/master/reliability-scanner/api/v1alpha1/pod/qos/qos.go) Go interface ( the project comes with a default [QoS implementation](https://github.com/vmware-tanzu/sonobuoy-plugins/blob/master/reliability-scanner/api/v1alpha1/pod/qos/qos.go) of the Querier interface), and a mapping must be made in the [scanner.go](https://github.com/vmware-tanzu/sonobuoy-plugins/blob/master/reliability-scanner/cmd/reliability-scanner/scanner.go) file.

### Testing the new check
The QoS check, defined above, allows the scanner to look across the cluster to report back on the current state of a Pod within the cluster to understand workloads that do not define our minimum [QoS class](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/).

Let us create a Pod with a guaranteed QoS class and run our scan, we should see it reflected in the report:

```
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - name: test
    image: nginx
    resources:
      limits:
        memory: "200Mi"
        cpu: "100Mi"
EOF
pod/test created
```



To run the reliability scanner, use command `make run` (run `make clean` to clean up any previous run). Once it’s complete, you can use `make results` to view the output.  The scanner will send its results to the Sonobuoy aggregator to be collected using the command `sonobuoy result`. 

Based on our defined check, we have two configuration options: minimum_desired_qos_class and include_detail. Both options are telling us that for our report, we will fail any check that does not meet the minimum QoS class defined here. The included detail configuration option allows for the report runner to return the current QoS class of the Pod being assessed.

Let’s review an excerpt from our Sonobuoy generated report to see how our scan went.

```
$ make results
# ...
- name: qos
   status: failed # overall test status
   meta:
     file: “”
     type: “”
   items:
   - name: default/test
     status: passed # subtest status
     details:
       qos_class: Guaranteed
# …
```




We can see that, although our report is showing a failed status (as none of the  Pods in the cluster meet the minimum desired QoS class), our guaranteed Pod, which we created earlier  (`default/test`), has passed the check.

Using this Reliability Scanner within a cluster is an easy way for cluster operators to identify any workloads or configurations that do not meet requirements and report them.

For the probes and owner annotations checks, please refer to the [Reliability Scanner repo](https://github.com/vmware-tanzu/sonobuoy-plugins/tree/master/reliability-scanner).

We hope that in time we are able to build sets of checks for multiple concerns and would love any feedback about the Reliability Scanner. Additionally, the VMware CRE team would love to hear from the broader community about good practices for operating workloads on Kubernetes.

Community Shoutout
This plugin contribution is thanks to site reliability engineers Peter Grant (Twitter @peterjamesgrant) and Kalai Wei of the  [VMware CRE](https://tanzu.vmware.com/content/blog/hello-world-meet-vmware-cre) team who work together with customers and partner teams to learn and apply reliability engineering practices using the Tanzu portfolio. 
Collaborate with the Sonobuoy Community!
Get updates on Twitter at
[@projectsonobuoy](https://twitter.com/projectsonobuoy)
Chat with us on Slack at
[#sonobuoy](https://kubernetes.slack.com/messages/sonobuoy) on the Kubernetes Slack
Collaborate with us on GitHub:
[github.com/vmware-tanzu/sonobuoy](https://github.com/vmware-tanzu/sonobuoy)
 
