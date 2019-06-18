# Streamlined Templates

## How it works now

Currently, Sonobuoy plugins are large, sprawling YAML files. This works, but it
makes the boundary between Sonobuoy and the plugin very messy.

By way of illustration, here is the current `heptio-e2e` plugin file. Every part
of user configuration is in green, all boilerplate required by Sonobuoy is in
red.

```diff
- ---
- apiVersion: v1
- kind: Pod
- metadata:
-   annotations:
+     sonobuoy-driver: Job
+     sonobuoy-plugin: heptio-e2e
+     sonobuoy-result-type: heptio-e2e
-   labels:
-     component: sonobuoy
-     sonobuoy-run: '{{.SessionID}}'
-     tier: analysis
-   name: sonobuoy-heptio-e2e-job-{{.SessionID}}
-   namespace: '{{.Namespace}}'
- spec:
-   containers:
+   - image: gcr.io/heptio-images/heptio-e2e:master
+     imagePullPolicy: Always
+     name: heptio-e2e
-     volumeMounts:
-     - mountPath: /tmp/results
-       name: results
-       readOnly: false
-   - command:
-     - sh
-     - -c
-     - /sonobuoy worker global -v 5 --logtostderr
-     env:
-     - name: NODE_NAME
-       valueFrom:
-         fieldRef:
-           fieldPath: spec.nodeName
-     - name: RESULTS_DIR
-       value: /tmp/results
-     - name: MASTER_URL
-       value: '{{.MasterAddress}}'
-     - name: RESULT_TYPE
-       value: heptio-e2e
-     image: gcr.io/heptio-images/sonobuoy:master
-     imagePullPolicy: Always
-     name: sonobuoy-worker
-     volumeMounts:
-     - mountPath: /tmp/results
-       name: results
-       readOnly: false
-   restartPolicy: Never
-   serviceAccountName: sonobuoy-serviceaccount
-   tolerations:
-   - effect: NoSchedule
-     key: node-role.kubernetes.io/master
-     operator: Exists
-   - key: CriticalAddonsOnly
-     operator: Exists
-   volumes:
-   - emptyDir: {}
-     name: results
```

## Summary

Removing this boiler plate will not just greatly simplify the process of making
a plugin. It will also allow us to make changes behind the scenes to the
Sonobuoy execution model without requiring updates to template files.

All this boiler plate isn't just annoying. It paralyses changes to the Sonobuoy
execution model. Adding any new environment variables or metadata requires
changing all existing plugin definitions.

## Objectives

Replace the existing templates with a greatly streamlined one.

### Goals
* Conceal the execution mechanics from users
* Allow seamless upgrading of Sonobuoy without user intervention

### Non-goals
* Do not modify the done contract betweens producer and consumer container

## Proposals

### Proposal 1

An entirely new YAML-based file will be used for plugins. This will consist of
two sections: Metadata and container spec.

```yaml
sonobuoy-config:
  driver: Job
  plugin-name: heptio-e2e
  result-type: heptio-e2e
spec:
  image: gcr.io/heptio-images/heptio-e2e:master
    imagePullPolicy: Always
    name: heptio-e2e
    volumeMounts:
    - mountPath: /tmp/results
      name: results
      readOnly: false
```

This will be merged into a larger Pod document provided by the Sonobuoy YAML
driver

```yaml
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    component: sonobuoy
    sonobuoy-run: '{{.SessionID}}'
    tier: analysis
  name: sonobuoy-heptio-e2e-job-{{.SessionID}}
  namespace: '{{.Namespace}}'
spec:
  containers:
  - {{.PluginContainerSpec}}
    volumeMounts:
    - mountPath: /tmp/results
      name: results
      readOnly: false
  - command:
    - sh
    - -c
    - /sonobuoy worker global -v 5 --logtostderr
    env:
    - name: NODE_NAME
      valueFrom:
        fieldRef:
          fieldPath: spec.nodeName
    - name: RESULTS_DIR
      value: /tmp/results
    - name: MASTER_URL
      value: '{{.MasterAddress}}'
    - name: RESULT_TYPE
      value: heptio-e2e
    image: gcr.io/heptio-images/sonobuoy:master
    imagePullPolicy: Always
    name: sonobuoy-worker
    volumeMounts:
    - mountPath: /tmp/results
      name: results
      readOnly: false
  restartPolicy: Never
  serviceAccountName: sonobuoy-serviceaccount
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  - key: CriticalAddonsOnly
    operator: Exists
  volumes:
  - emptyDir: {}
   name: results
```

#### Advantages:
* Simple substitution
* Can add additional fields later

#### Disadvantages
* Not a Kubernetes API object
* Not as clear how it'll be executed

### Proposal 2

In this scenario, the user provides a (templated) valid Kubernetes API Document.

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    sonobuoy-driver: Job
    sonobuoy-plugin: heptio-e2e
    sonobuoy-result-type: heptio-e2e
  labels: {{.SonobuoyLabels}}
  name: sonobuoy-heptio-e2e-job-{{.SessionID}}
  namespace: '{{.Namespace}}'
spec:
  containers:
  - image: gcr.io/heptio-images/heptio-e2e:master
    imagePullPolicy: Always
    name: heptio-e2e
    {{.SonobuoyMount}}
  - {{.SonobuoyConsumer}}
  {{.SonobuySpecExtras}}
```

Then, at runtime, the template would be filled with values by the plugin driver.
These could be based on several templates themselves, or created at runtime out
of Go structs.

#### Advantages
* Easier to see how the completed template will be finished
* Native (therefore familiar) Kubernetes object

#### Disadvantages
* More boilerplate for plugin authors
* Template for plugin drivers is harder write/understand
* Extension points are fixed: No new Pod attributes could be added outsite of
  `spec` or Labels.

# Results

Proposal 1 accepted.

## User stories:

1. As a power user, I would like to see the generated YAML for a plugin.yml file.
2. As a plugin author, I can write a simplified plugin.yml file
3. As an end user, I should see no changes as a result of this plugin work
