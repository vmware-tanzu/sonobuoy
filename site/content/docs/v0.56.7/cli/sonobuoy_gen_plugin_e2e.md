## sonobuoy gen plugin e2e

Generates the e2e plugin definition based on the given options

### Synopsis

Generates the e2e plugin definition based on the given options

```
sonobuoy gen plugin e2e [flags]
```

### Options

```
      --aggregator-node-selector nodeSelectors   Node selectors to add to the aggregator. Values can be given multiple times and are in the form key:value (default map[])
      --aggregator-permissions string            Type of aggregator permission to use in the cluster. Allowable values are [namespaceAdmin, clusterRead, clusterAdmin] (default "clusterAdmin")
      --config Sonobuoy config                   Path to a sonobuoy configuration JSON file.
      --configmap stringArray                    Specifies files to read and add as configMaps. Will be mounted to the plugin at /tmp/sonobuoy/configs/<filename>.
      --context string                           Context in the kubeconfig to use.
      --dns-namespace string                     The namespace to check for DNS pods during preflight checks. (default "kube-system")
      --dns-pod-labels strings                   The label selectors to use for locating DNS pods during preflight checks. Can be specified multiple times or as a comma-separated list. (default [k8s-app=kube-dns,k8s-app=coredns])
      --e2e-focus envModifier                    Specify the E2E_FOCUS value for the e2e plugin, specifying which tests to run. Shorthand for --plugin-env=e2e.E2E_FOCUS=<string> (default \[Conformance\])
      --e2e-repo envModifier                     Specify a registry to use as the default for pulling Kubernetes test images. Same as providing --e2e-repo-config but specifying the same repo repeatedly.
      --e2e-repo-config yaml-filepath            Specify a yaml file acting as KUBE_TEST_REPO_LIST, overriding registries for test images.
      --e2e-skip envModifier                     Specify the E2E_SKIP value for the e2e plugin, specifying which tests to skip. Shorthand for --plugin-env=e2e.E2E_SKIP=<string> (default \[Disruptive\]|NoExecuteTaintManager)
  -f, --file -                                   If set, loads the file as if it were the output from sonobuoy gen. Set to - to read from stdin.
      --force-image-pull-policy                  Force plugins' imagePullPolicy to match the value for the Sonobuoy pod
  -h, --help                                     help for e2e
      --image-pull-policy string                 Set the ImagePullPolicy for the Sonobuoy image and all plugins (if --force-image-pull-policy is set). Valid options are Always, IfNotPresent, Never. (default "IfNotPresent")
      --kube-conformance-image image             Container image override for the e2e plugin. Shorthand for --plugin-image=e2e:<string> (default map[])
      --kubeconfig Kubeconfig                    Path to explicit kubeconfig file.
      --kubernetes-version string                Use default E2E image, but override the version. Default is 'auto', which will be set to your cluster's version if detected, erroring otherwise. 'ignore' will try version resolution but ignore errors. 'latest' will find the latest dev image/version upstream.
  -m, --mode Mode                                What mode to run the e2e plugin in. Valid modes are [certified-conformance conformance-lite non-disruptive-conformance quick]. (default non-disruptive-conformance)
  -n, --namespace string                         The namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously. (default "sonobuoy")
  -p, --plugin pluginList                        Which plugins to run. Can either point to a URL, local file/directory, or be one of the known plugins (e2e or systemd-logs). Can be specified multiple times to run multiple plugins.
      --plugin-env pluginenvvar                  Set env vars on plugins. Values can be given multiple times and are in the form plugin.env=value (default map[])
      --plugin-image plugin:image                Override a plugins image from what is in its definition (e.g. myPlugin:testimage) (default map[])
      --rbac RBACMode                            Whether to enable RBAC on Sonobuoy. Valid modes are Enable, Disable, and Detect (query the server to see whether to enable RBAC). (default Enable)
      --rerun-failed tar.gz file                 Read the given tarball and set the E2E_FOCUS to target all the failed tests
      --security-context-mode string             Type of security context to use for the aggregator pod. Allowable values are [none, nonroot] (default "nonroot")
      --show-default-podspec                     If true, include the default pod spec used for plugins in the output.
      --skip-preflight                           If true, skip all checks before starting the sonobuoy run.
      --sonobuoy-image string                    Container image override for the sonobuoy worker and aggregator. (default "sonobuoy/sonobuoy:v0.56.6-7-gc1568607")
      --ssh-key yamlFile                         Path to the private key enabling SSH to cluster nodes. May be required by some tests from the e2e plugin.
      --ssh-user envModifier                     SSH user for ssh-key. Required if running e2e plugin with certain tests that require SSH access to nodes.
      --systemd-logs-image image                 Container image override for the systemd-logs plugin. Shorthand for --plugin-image=systemd-logs:<string> (default map[])
      --timeout int                              How long (in seconds) Sonobuoy aggregator will wait for plugins to complete before exiting. 0 indicates no timeout. (default 21600)
      --wait int[=1440]                          How long (in minutes) for the CLI to wait for sonobuoy run to be completed or fail, where 0 indicates do not wait. If specified, the default wait time is 1 day.
      --wait-output string                       Specify the type of output Sonobuoy should produce when --wait is used. Valid modes are silent, spinner, or progress (default "Progress")
```

### Options inherited from parent commands

```
      --level level   Log level. One of {panic, fatal, error, warn, info, debug, trace} (default info)
```

### SEE ALSO

* [sonobuoy gen plugin](sonobuoy_gen_plugin.md)	 - Generates the manifest Sonobuoy uses to define a plugin

###### Auto generated by spf13/cobra on 10-Jun-2022
