# Aggregator Permissions

By default, the Sonobuoy aggregator is given very elevated permissions in order to successfully run the Kubernetes end-to-end tests. In some situations you may want to (or need to) limit the permissions of the aggregator so that the aggregator and the pods that it creates do not have such wide-reaching permissions. You can always customize the exact permissions of the ServiceAccount via editing `sonobuoy gen` output manually, but Sonobuoy also provides useful presets via the CLI flag, `--aggregator-permissions`.

## Type of Aggregator Permissions

Allowable values are `[namespaced, clusterAdmin, clusterRead]`, `clusterAdmin` is default value.

### clusterAdmin

- `clusterAdmin` is the default value. With this value Sonobuoy can do pretty much everything in the run, it does not implement any restrictions. Most of these are required for the e2e conformance tests to work since they create/destroy namespaces, pods etc.

### namespaceAdmin

namespaceAdmin is the most restrictive preset permissions Sonobuoy provides and ensures that Sonobuoy and its plugins do not impact other namespaces at all.

Due to these limitations there are a number of things to note:
 - Sonobuoy does not create the namespace so it needs to already exist
 - You must provide `--skip-preflight` to avoid Sonobuoy from complaining about the preexisting namespace
 - The `e2e` plugin (conformance tests) will not work in this mode and won't even start up due to severely limited permissions
 - Daemonset plugins will not work in this mode because Sonobuoy monitors them on a per-node basis. Since Sonobuoy can't query the list of nodes in the cluster, it can't properly monitor or gather results from them. At this time, Daemonset plugins will simply be ignored.

### clusterRead

`clusterRead` is a compromise between `namespaceAdmin` and `clusterAdmin`. It adds ability to GET any resource from the API so that the Sonobuoy queries work OK, it is able to get nodes so daemonsets run fine, and e2e tests can technically start. Sonobuoy can't create namespaces so e2e tests can't run in this mode in any useful manner either. However, this may be a more reasonable mode to run less intrusive, custom plugins in. In this mode Sonobuoy don't create the namespace either so it has to be created first and sonobuoy run with the `--skip-preflight` flag.
