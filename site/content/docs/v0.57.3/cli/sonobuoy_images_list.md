## sonobuoy images list

List images

```
sonobuoy images list [flags]
```

### Options

```
      --context string              Context in the kubeconfig to use.
      --dry-run                     If true, only print the image operations that would be performed.
  -h, --help                        help for list
      --kubeconfig Kubeconfig       Path to explicit kubeconfig file.
      --kubernetes-version string   Use default E2E image, but override the version. Default is 'auto', which will be set to your cluster's version if detected, erroring otherwise. 'ignore' will try version resolution but ignore errors. 'latest' will find the latest dev image/version upstream.
  -p, --plugin strings              Describe which plugin's images to interact with (Valid plugins are 'e2e', 'systemd-logs'). (default [e2e,systemd-logs])
```

### Options inherited from parent commands

```
      --level level   Log level. One of {panic, fatal, error, warn, info, debug, trace} (default info)
```

### SEE ALSO

* [sonobuoy images](sonobuoy_images.md)	 - Manage images used in a plugin to facilitate running them in airgapped (or similar) environments. Supported plugins are: 'e2e'

###### Auto generated by spf13/cobra on 18-Feb-2025
