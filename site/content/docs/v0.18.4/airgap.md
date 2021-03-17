# Custom registries and air-gapped testing

In air-gapped deployments where there is no access to the public Docker registries Sonobuoy supports running the end-to-end tests with custom registries.
This enables you to test your air-gapped deployment once you've loaded the necessary images into a registry that is reachable by your cluster.

You will need to make the Sonobuoy image available as well as the images for any plugins you wish to run.
Below, you will find the details of how to use the Sonobuoy image, as well as the images for the `e2e` and `systemd-logs` plugins in this kind of deployment.

## Sonobuoy Image
To run any Sonobuoy plugin in an air-gapped deployment, you must ensure that the Sonobuoy image is available in a registry that is reachable by your cluster.
You will need to pull, tag, and then push the image as follows:

```
PRIVATE_REG=<your private registry>
SONOBUOY_VERSION=<version of Sonobuoy you are targeting; e.g. v0.16.0>

docker pull sonobuoy/sonobuoy:$SONOBUOY_VERSION
docker tag sonobuoy/sonobuoy:$SONOBUOY_VERSION $PRIVATE_REG/sonobuoy:$SONOBUOY_VERSION
docker push $PRIVATE_REG/sonobuoy:$SONOBUOY_VERSION
```

By default, Sonobuoy will attempt to use the image available in the public registry.
To use the image in your own registry, you will need to override it when using the `gen` or `run` command with the `--sonobuoy-image` flag as follows:

```
sonobuoy run --sonobuoy-image $PRIVATE_REG/sonobuoy:$SONOBUOY_VERSION
```

## E2E Plugin

To use the `e2e` plugin, the conformance test image and the images the tests use must be available in your registry.

### Conformance Image
The process for making the conformance image available in your registry is the same as the Sonobuoy image.
You need to pull, tag, and then push the image.
To ensure you use the correct version of the conformance image, check your server version using `kubectl version`.


```
PRIVATE_REG=<private registry>
CLUSTER_VERSION=<version of k8s you are targeting; e.g. v1.16.0>

docker pull k8s.gcr.io/conformance:$CLUSTER_VERSION
docker tag k8s.gcr.io/conformance:$CLUSTER_VERSION $PRIVATE_REG/conformance:$CLUSTER_VERSION
docker push $PRIVATE_REG/conformance:$CLUSTER_VERSION
```

To use the conformance image in your registry, you will need to override the default when using the `gen` or `run` commands with the `--kube-conformance-image` flag as follows:

```
sonobuoy run --kube-conformance-image $PRIVATE_REG/conformance:$CLUSTER_VERSION
```

### Test Images

The end-to-end tests use a number of different images across multiple registries.
When running the `e2e` plugin, you must provide a mapping that details which custom registries should be used instead of the public registries.

This mapping is a YAML file which maps the registry category to the corresponding registry URL.
The keys in this file are specified in the Kubernetes test framework.
The tests for each minor version of Kubernetes use a different set of registries so the mapping you create will depend on which Kubernetes version you are testing against.

To create this mapping, you can use the `gen default-image-config` command to provide the mapping with the default registry values for your cluster version.
The following is an example of using this command with a v1.16 cluster:

```
$ sonobuoy gen default-image-config
dockerLibraryRegistry: docker.io/library
e2eRegistry: gcr.io/kubernetes-e2e-test-images
gcRegistry: k8s.gcr.io
googleContainerRegistry: gcr.io/google-containers
sampleRegistry: gcr.io/google-samples
```

You can save this output to a file and modify it to specify your own registries instead.
You can modify all of the registry values or just a subset.
If you specify only a subset, the defaults will be used instead.

Sonobuoy provides the command `images` to help you easily pull the test images and push them to your own custom registries.
First, you must pull the images to your local machine using the following command:

```
sonobuoy images pull
```

> **NOTE:** Some versions of Kubernetes reference images that do not exist or cannot be pulled without authentication.
> You may see these errors when running the above command. This is expected behaviour.
> These images are referenced by some end-to-end tests, but **not** by the conformance tests.

To push the images, you must provide the mapping using the `--e2e-repo-config` flag as follows: 

```
sonobuoy images push --e2e-repo-config <path/to/custom-repo-config.yaml>
```

Sonobuoy will read the mapping config and will push the images to the repositories defined in that mapping.

When running the `e2e` plugin, you will need to provide this file using the same flag as follows:

```
sonobuoy run --e2e-repo-config <path/to/custom-repo-config.yaml>
```

## systemd-logs plugin

If you want to run the `systemd-logs` plugin you will again need to pull, tag, and push the image.


```
PRIVATE_REG=<private registry>

docker pull gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest
docker tag gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest $PRIVATE_REG/sonobuoy-plugin-systemd-logs:latest
docker push $PRIVATE_REG/sonobuoy-plugin-systemd-logs:latest
```

To use the image in your own registry, you will need to override the default when using the `gen` or `run` commands with the `--systemd-logs-image` flag as follows:

```
sonobuoy run --systemd-logs-image $PRIVATE_REG/sonobuoy-plugin-systemd-logs:latest
```

If you do not wish to run this plugin, you can remove it from the list of [plugins][plugins] to be run within the manifest, or you can explicitly specify which plugin you with to run with the `--plugin` flag.

[plugins]: plugins.md#choosing-which-plugins-to-run
