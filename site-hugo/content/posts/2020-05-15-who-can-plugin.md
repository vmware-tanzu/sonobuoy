---
title: Who has access to your Kubernetes cluster?
image: /img/sonobuoy.svg
excerpt: Check which subjects have RBAC permissions to perform actions in your cluster with the new who-can plugin.
author_name: Bridget McErlean
author_url: https://github.com/zubron
author_avatar: /img/contributors/bridget-mcerlean.png
categories: [kubernetes, sonobuoy]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team', 'Bridget McErlean']
date: 2020-05-15
---

The Sonobuoy team is continuing to expand its range of custom plugins and we would like to introduce the latest plugin in our collection: [who-can](https://github.com/vmware-tanzu/sonobuoy-plugins/tree/master/who-can).

This plugin utilizes a project from [Aqua Security](https://www.aquasec.com/): [kubectl-who-can](https://github.com/aquasecurity/kubectl-who-can).
kubectl-who-can shows which subjects have RBAC permissions to perform actions (verbs) against different resources in all namespaces in your Kubernetes clusters.
It shows which subjects can perform those actions, but also the role bindings and cluster role bindings that enable them to do so.

There are existing tools that allow you to visualize the RBAC rules, or find the roles and cluster roles bound to a particular subject.
While these tools are useful, kubectl-who-can goes a step further and provides a view of what subjects can actually do in your cluster.
We thought this was such a powerful concept that we decided to leverage it as a Sonobuoy plugin to provide an overall view of the RBAC permissions granted within a cluster for all subjects.

Having a thorough understanding of the RBAC permissions in your cluster is important as it helps you apply the principle of least privilege, where subjects are only granted permissions to the resources that they need to access.
As the number of users and workloads grow and change on your clusters, and different permissions are granted, it can become difficult to track changes in RBAC configuration.
There may be users or subjects in your cluster that have access to resources that they no longer require, or users or subjects that have permissions that are too open for their use cases.

By using the who-can plugin, you can obtain a more comprehensive view of all the permissions granted within your clusters and enforce the desired configuration or eliminate configuration drift.
More importantly, you can execute this plugin on a periodic schedule and compile audit-level logs on access permissions.
You can even go a step further and create alerts any time there is a difference in access permissions, utilizing the difference in results as reported by Sonobuoy, logging the diff for additional security analysis.

## How does the plugin work?

The plugin includes a small runner which uses kubectl-who-can as a library to perform the queries on your cluster.
The configuration of the plugin also includes a list of namespaces to check.
This allows you to focus your queries on those namespaces that are of most importance.
By default, the plugin will check which actions can be performed in the `kube-system` namespace, and in all namespaces `*`.

To perform a query using the kubectl-who-can library, we need to create an `Action` that will be checked.
An `Action` comprises a resource, a verb, and a namespace, and the result of the query will be all the RBAC subjects that can perform that `Action`.
Some example actions are:

* who can `delete` `pods` in the `kube-system` namespace
* who can `get` `secrets` in the `production` namespace
* who can `create` `bindings` in `*` (all) namespaces

The plugin runner produces a list of all the built-in and custom resource types available in your cluster and, for each resource, determines which verbs are supported.
Using the list of namespaces provided in the plugin configuration, the runner creates actions from all possible combinations of resources, supported verbs, and namespaces.
It then iterates over all of these actions and uses the kubectl-who-can library to find the list of subjects that have RBAC permissions to perform that action.

> Currently, the plugin only supports Kubernetes resources and subresources and custom resources.
> It does not provide information about non-resource URLs (e.g. /healthz) or specific named resources (e.g. pods/mynamedpod).
> We plan to add support for this in a future version of the plugin as kubectl-who-can already supports querying these resources.
> If you wish to check the permissions for non-resource URLs or named resources, we recommend using kubectl-who-can directly.

Once all the queries have been performed, the runner processes the data to provide the different views of the results.
The default configuration for the plugin produces reports in all of the available formats which will be explained in detail in a later section.

## Running the plugin

To run the plugin, use the following command:

```bash
sonobuoy run --plugin https://raw.githubusercontent.com/vmware-tanzu/sonobuoy-plugins/master/who-can/who-can.yaml
```

This fetches the plugin from GitHub and uses the default configuration.
If you wish to configure the plugin (for example, by adding other namespaces to scan), save this plugin definition locally then pass it when calling the `sonobuoy run` command instead of the plugin URL.

Once the plugin completes its run, retrieve the results using the `sonobuoy retrieve` command. This will download the results tarball for you to inspect.

## Inspecting the results

You can view a summary of the results by using the `sonobuoy results` command with the results tarball.

```
$ resultsFile=$(sonobuoy retrieve)
$ sonobuoy results $resultsFile
Plugin: who-can
Status: complete
Total: 5197
Passed: 0
Failed: 0
Skipped: 0
complete: 5197
```

The summary result shows that 5197 different checks were performed.

The results tarball contains three different reports:

* Subjects report (`plugins/who-can/results/global/subjects-report.json`)
* Resources report (`plugins/who-can/results/global/resources-report.json`)
* Sonobuoy results (`plugins/who-can/results/global/sonobuoy_results.yaml`)

These report formats are detailed below:

### Subjects report

This report is a view of the RBAC data in the cluster grouped by each subject.
For each subject, it details which actions they can perform in each of the queried namespaces.
For each action that the subject can perform, the role bindings and cluster role bindings that enable that subject to perform that action are also listed.

This report is a JSON file with the following format:

```
[
  {
    "kind": "ServiceAccount",
    "name": "expand-controller",
    "namespace": "kube-system",
    "permissions": [
      {
        "namespace": "*",
        "actions": [
          {
            "resource": "endpoints",
            "verb": "get",
            "cluster-role-bindings": [
              "system:controller:expand-controller"
            ]
          },
          ...
        ]
      }
    ]
  },
  {
    "kind": "User",
    "apiGroup": "rbac.authorization.k8s.io",
    "name": "pod-lister",
    "permissions": [
      {
        "namespace": "secret",
        "actions": [
          {
            "resource": "pods",
            "verb": "list",
            "role-bindings": [
              "list-secret-pods"
            ]
          }
        ]
      }
    ]
  },
  ...
}
```


As we can see in the above example, the ServiceAccount `expand-controller`, which is in the namespace `kube-system`, has permissions to `get` `endpoints` in all namespaces (`*`) due to the `system:controller:expand-contoller` cluster role binding.
We can also see that the User, `pod-lister`, only has permissions to `list` `pods` in the `secret` namespace due to the `list-secret-pods` role binding.

### Resources report

This report is a view of the RBAC data in the cluster, detailing which subjects can perform actions against a resource in a particular namespace.
Along with each subject are the role bindings and cluster role bindings that allow them to perform that action.

This report is a JSON file with the following format:

```
[
  {
    "resource": "pods",
    "verb": "list",
    "namespace": "secret",
    "subjects": [
      {
        "kind": "ServiceAccount",
        "name": "pvc-protection-controller",
        "namespace": "kube-system",
        "cluster-role-bindings": [
          "system:controller:pvc-protection-controller"
        ]
      },
      {
        "kind": "User",
        "apiGroup": "rbac.authorization.k8s.io",
        "name": "pod-lister",
        "role-bindings": [
          "list-secret-pods"
        ]
      },
      ...
    ]
  },
  ...
]
```


In the above example, we can see two subjects that have the ability to `list` `pods` in the `secret` namespace.
The ServiceAccount `pvc-protection-controller`, which is in the namespace `kube-system`, can perform this action due to the `system:controller:pvc-protection-controller` cluster role binding.
Like in the Subjects report example above, we can see again that the User `pod-lister` has permission to perform this action due to the `list-secret-pods` role binding.

### Sonobuoy results

This report is a variant of the Resources report using the [Sonobuoy Results](https://sonobuoy.io/docs/results/) format.
This enables the plugin results to be processed and presented through Sonobuoy using the `results` command.

This report is a YAML file with the following format:

```
name: who-can
status: complete
items:
- name: system:masters can create bindings in default
  status: complete
  details:
    cluster-role-bindings: cluster-admin
    namespace: default
    resource: bindings
    subject-kind: Group
    subject-name: system:masters
    verb: create
- name: sonobuoy-serviceaccount can create bindings in default
  status: complete
  details:
    cluster-role-bindings: sonobuoy-serviceaccount-sonobuoy
    namespace: default
    resource: bindings
    subject-kind: ServiceAccount
    subject-name: sonobuoy-serviceaccount
    subject-namespace: sonobuoy
    verb: create
  ...
```

The Sonobuoy results format is used to describe the results for a plugin.
In this report, we can see that the who-can plugin has the status `complete`.
The `items` entry is an array where each entry represents a check that was performed.
The first item describes that the `system:masters` subject can `create` `bindings` in the `default` namespace.

Within the `details` map, we can see the details for the check and the results.
The subject details are prefixed with `subject-`.
We can see that the `subject-name` is `system:masters` and its `subject-kind` is `Group`.
If the subject-kind is `ServiceAccount`, the `subject-namespace` will also be included.
The details for the check can be found in the `verb`, `resource` and `namespace` fields.
These describe what the subject can do, for example `create` `bindings` in the `default` namespace.

Finally, the `role-bindings` or `cluster-role-bindings` that allow that subject to perform that action are provided as a comma separated list.

## Querying the results
To explore the results and extract information, you can use a tool like [jq](https://stedolan.github.io/jq/) to filter and extract the data you are interested in.
The following are some examples using the Subjects and Resources reports:

### Show all unique subjects that have RBAC permissions in the cluster

The following example uses `jq` to select all the subject entries from the Subjects report and display only their details, such as their name, kind, and, in the case where the subject is a ServiceAccount, the namespace where it exists.

```
$ cat plugins/who-can/results/global/subjects-report.json | jq '.[] | {name, kind, namespace}'
{
  "name": "kube-dns",
  "kind": "ServiceAccount",
  "namespace": "kube-system"
}
{
  "name": "system:kube-proxy",
  "kind": "User",
  "namespace": null
}
...
```

### Show which actions a particular subject can perform in a namespace

The following example performs a `jq` query on the Subjects report to select all the actions that the `system:kube-controller-manager` can perform in the `kube-system` namespace:

```
$ cat results/plugins/who-can/results/global/subjects-report.json | jq '.[] | select(.name == "system:kube-controller-manager") | .permissions[] | select(.namespace == "kube-system")' 
{
  "namespace": "kube-system",
  "actions": [
    {
      "resource": "componentstatuses",
      "verb": "list",
      "cluster-role-bindings": [
        "system:kube-controller-manager"
      ]
    },
    {
      "resource": "configmaps",
      "verb": "get",
      "cluster-role-bindings": [
        "system:kube-controller-manager"
      ]
    },
...
```

### Show which subjects can interact with a particular resource in a particular namespace

The following example performs a `jq` query on the Resources report to select all the subjects that can perform actions against `secrets` in all namespaces (`*`):

```
$ cat results/plugins/who-can/results/global/resources-report.json | jq '.[] | select(.resource == "secrets" and .namespace == "*")'
{
  "resource": "secrets",
  "verb": "create",
  "namespace": "*",
  "subjects": [
    {
      "kind": "Group",
      "apiGroup": "rbac.authorization.k8s.io",
      "name": "system:masters",
      "cluster-role-bindings": [
        "cluster-admin"
      ]
    },
    {
      "kind": "User",
      "apiGroup": "rbac.authorization.k8s.io",
      "name": "system:kube-controller-manager",
      "cluster-role-bindings": [
        "system:kube-controller-manager"
      ]
    }
  ]
}
...
```

In the output, we can see the list of subjects that can `create` `secrets` in all namespaces (`*`) as well as the cluster role bindings that allow them to do that.

## Summary

We hope that by providing a comprehensive view of the permissions granted within your cluster, you will be able to understand and audit the permissions more easily and observe the impact of RBAC changes over time. 

We would like to thank [Aqua Security](https://www.aquasec.com/) for creating [kubectl-who-can](https://github.com/aquasecurity/kubectl-who-can), and we would like to give special thanks to [Daniel Pacak](https://github.com/danielpacak), the maintainer of the project, who supported us in our use of kubectl-who-can and adapted the codebase so that we could easily use the project for this plugin.

## Join the Sonobuoy Community

- Get updates on Twitter ([@projectsonobuoy](https://twitter.com/projectsonobuoy))
- Chat with us on the Kubernetes Slack ([#sonobuoy](https://kubernetes.slack.com/messages/sonobuoy))
- Join the [Kubernetes Software Conformance Working Group](https://github.com/cncf/k8s-conformance)
