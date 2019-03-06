# Custom registries and air-gapped testing

In air-gapped deployments where there is no access to the public Docker registries
Sonobuoy supports running end-to-end tests with custom registries. This enables
you to test your air-gapped deployment once you've loaded the necessary images
into a registry that is reachable by your cluster.

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
privateRegistry: gcr.io/k8s-authenticated-test
sampleRegistry: gcr.io/google-samples
```

The keys in that file are specified in the Kubernetes test framework itself. You
may provide a subset of those and the defaults will be used for the others.