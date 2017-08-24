# Examples

This directory contains two sub-directories:
* [`/ksonnet`][4] - You can autogenerate Sonobuoy's YAML manifests with the [ksonnet][0] files in this directory. To do so, execute the `make generate-examples` command in the root of the Sonobuoy repo.

* [`/quickstart`][5] - You can use the YAML config files in this directory to quickly deploy a containerized Sonobuoy pod.

## /ksonnet

Sonobuoy's `make generate-examples` command compiles `*.jsonnet` files into YAML. This compilation process uses the [`kubecfg`][2] executable in the [official ksonnet Docker image][1].

This sub-directory is itself broken down into:
* `/components` - Each of the files in this directory covers a fairly distinct part of Sonobuoy's setup. The YAML that they generate is a more human-friendly way of viewing how Sonobuoy's component resources are set up.

  * `00-rbac.jsonnet` - This sets up basic scaffolding (e.g. a dedicated namespace). If RBAC is enabled on the cluster, it also configures the necessary permissions for Sonobuoy to read data and dispatch pods into the cluster.

  * `10-configmaps.jsonnet` - This sets up configurations for Sonobuoy as well as its plugins. The [configuration guide][3] explains in depth how to change these values.

  * `20-pod.jsonnet` - This sets up the pod that actually runs the `sonobuoy` executable.

* `aggregate.jsonnet` - This combines all the `components` together to generate a standalone YAML file. As demonstrated in the README, the YAML file can be used to immediately take a Sonobuoy snapshot with `kubectl apply -f`.

## /quickstart

This directory structure maps 1:1 to that of the `/ksonnet` subdirectory (`*.yaml` instead of `*.jsonnet`), so the descriptions above should suffice.

[0]: http://ksonnet.heptio.com
[1]: https://hub.docker.com/r/ksonnet/ksonnet-lib/
[2]: https://github.com/ksonnet/kubecfg
[3]: /docs/configuration.md
[4]: #ksonnet
[5]: #quickstart
