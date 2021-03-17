# Custom registries and air-gapped testing

In air-gapped deployments where there is no access to the public Docker registries
Sonobuoy supports running end-to-end tests with custom registries. This enables
you to test your air-gapped deployment once you've loaded the necessary images
into a registry that is reachable by your cluster.

## Test images

Just provide the `--e2e-repo-config` parameter and pass it the path to a local
YAML file pointing to the registries you'd like to use. This will instruct the
Kubernetes end-to-end suite to use your registries instead of the default ones.

```
sonobuoy run --e2e-repo-config custom-repos.yaml
```

The registry list is a YAML document specifying a few different registry
categories and their values:

```
dockerLibraryRegistry: docker.io/library
e2eRegistry: gcr.io/kubernetes-e2e-test-images
gcRegistry: k8s.gcr.io
etcdRegistry: quay.io/coreos
privateRegistry: gcr.io/k8s-authenticated-test
sampleRegistry: gcr.io/google-samples
```

The keys in that file are specified in the Kubernetes test framework itself. You
may provide a subset of those and the defaults will be used for the others.

## Other required images

The list of custom registries is consumed by the Kubernetes end-to-end tests, but outside of that there are 2 images you will need to be able to access:
 - the sonobuoy image
 - the conformance image

To get those images into your cluster you need to pull/tag/push those images yourself:

```
PRIVATE_REG=<private registry>
SONO_VERSION=<version of Sonobuoy you are targeting; e.g. v0.14.0>
CLUSTER_VERSION=<version of k8s you are targeting; e.g. v1.14.0>

docker pull gcr.io/google-containers/conformance:$CLUSTER_VERSION
docker pull gcr.io/heptio-images/sonobuoy:$SONO_VERSION

docker tag gcr.io/google-containers/conformance:$CLUSTER_VERSION $PRIVATE_REG/conformance:$CLUSTER_VERSION
docker tag gcr.io/heptio-images/sonobuoy:$SONO_VERSION $PRIVATE_REG/sonobuoy:$SONO_VERSION

docker push $PRIVATE_REG/conformance:$CLUSTER_VERSION
docker push $PRIVATE_REG/sonobuoy:$SONO_VERSION
```

If you want to run the systemd_logs plugin you'll need to pull/tag/push it as well. In addition, you'll have to manually specify the image you want to use via `sonobuoy gen` -> `kubectl apply` since that image is not overridable on the CLI. The default value is: `gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest`

If you do not wish to run it in your air-gapped cluster, just remove it from the list of [plugins][plugins] to be run (again, using `sonobuoy gen` -> `kubectl apply`).

[plugins]: plugins.md#choosing-which-plugins-to-run