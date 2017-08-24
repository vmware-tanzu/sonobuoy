# Build From Scratch

Sonobuoy can run as (1) **a standalone program** on a node in your cluster, or (2) **containerized** in the context of a Kubernetes pod. You can build it yourself with the following steps.

* [0. Prerequisites][0]
* [1. Download][1]
* [2. Build][2]
* [3. Run][3]

## 0. Prerequisites

### General
In addition to the handling the prerequisites mentioned in the [Quickstart][4], you should have [Go][5] installed (minimum version 1.8).

### Standalone
> **Make sure that you are SSHed into one of the nodes on your cluster.** The subsequent instructions assume that this is the case.

While it is possible to run the standalone locally on a non-node machine, you may encounter networking issues if you run any plugins. This is due to HTTP requests that Sonobuoy agents make during [data aggregation][6]. Sonobuoy will only work if (1) you do not use plugins or (2) your non-node machine is on the same network as your cluster, and you set `$SONOBUOY_ADVERTISE_IP` accordingly.

## 1. Download

Install with go:
```
go get github.com/heptio/sonobuoy
```
The files are installed in `$GOPATH/src/github.com/heptio/sonobuoy`.

## 2. Build

### Standalone
Run the following in the Sonobuoy root directory:
```
make local
```
This will create a `sonobuoy` executable.
### Containerized
Depending on your use case, you may want to set environmental variables that are used in the `Makefile`:
* Set `$REGISTRY` if you want to push the Sonobuoy images to your own registry. This allows nodes in your cluster to pull the image that you built on your local machine.

* Set `$VERSION` if you want to keep track of your own versioning (used to tag the generated container image).

Run the following in the Sonobuoy root directory:
```
sudo make all
```

To push your local image to a registry, use `make push`.

## 3. Run

### Standalone
Check that your [Sonobuoy config][7] (`config.json`) points to the *absolute path* of a valid `Kubeconfig`. You will need to replace the template's usage of `~` because Go cannot properly resolve it.

Then run:
```
./sonobuoy master -v 5 --logtostderr
```
The results will be placed in the `ResultsDir` specified by `config.json`, where they can be uncompressed and inspected.

### Containerized

>Configuring Sonobuoy's data collection itself is straightforward; however, read about [pod manifest best practices](configuration.md#considerations) to ensure that your Sonobuoy pod will run properly on your cluster.
>
> Before running the following commands, make sure that your manifest addresses:
> * Which container image is being used
> * Where your Sonobuoy output will be written

You should create a directory for your YAML manifest files. Feel free to use the [provided manifests][8] as a template to get started, or the [ksonnet files][9] to autogenerate the YAML more quickly.

*Standup*
```
kubectl apply -f <YAML_CONFIG_DIR>
```

By default, Sonobuoy pods and other resources are created with the label `component=sonobuoy` and namespace `heptio-sonobuoy`, so specify these when querying them with `kubectl`.

*Teardown*
```
kubectl delete -f <YAML_CONFIG_DIR>
```

[0]: #0-prerequisites
[1]: #1-download
[2]: #2-build
[3]: #3-run
[4]: ../README.md#0-prerequisites
[5]: https://golang.org/doc/install
[6]: /docs/plugins.md#overview
[7]: configuration.md#sonobuoy-config
[8]: /examples/quickstart
[9]: /examples/ksonnet
