---
title: Simple Approaches to Customizing the Kubernetes E2E Tests
image: /img/sonobuoy.svg
excerpt: Two approaches to running the E2E tests with custom options to support your workflow.
author_name: John Schnake
author_url: https://github.com/johnschnake
author_avatar: /img/contributors/john-schnake.png
categories: [kubernetes, sonobuoy, conformance]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team', 'John Schnake']
date: 2019-10-09
slug: custom-e2e-image
---

# Simple Approaches to Customizing the Kubernetes E2E Tests

Sonobuoy can be used to run any set of custom tests and data-gathering logic, but the most common use case continues to be running the Kubernetes E2E tests.

The [E2E test image](https://github.com/kubernetes/kubernetes/tree/master/test/conformance/image) is maintained by the Kubernetes community and is automatically generated for each release of Kubernetes. The test image bundles:

- The binary, which contains the test logic
- Ginkgo, a tool for running tests
- An entrypoint script, which starts the tests and reports the results to Sonobuoy

Historically, the entrypoint for the image was a bash script, which started the tests. In this script, a handful of environment variables were handled and turned into command-line arguments in order to customize the run.

Due to heavy use of this image specifically for conformance testing, numerous edge cases aren’t fully supported by this test image. As many users know, several configuration options are available for a Kubernetes test run. The `E2E_PROVIDER` environment variable serves as an example. If you attempt to run provider-specific tests with just this environment variable set, tests may not execute because other provider-specific values are also expected. For instance,  if you specify `E2E_PROVIDER=gce`, you must also specify a region and zone for the cluster and may also need to set `gce-api-endpoint`, `gce-project`, `gce-multizone`, `gce-multimaster`, and `gke-cluster`.

If you need to set those values, there are two approaches worth discussing:

- Using the new, Golang-based entrypoint for the test image, which supports passing arbitrary flags to the tests
- Building a custom test image

# Using the Golang-based Entrypoint

This new feature was added in Kubernetes 1.16.0 and was designed to make the conformance image more robust. It is not the default entrypoint, so if you want to use it, you need to set `E2E_USE_GO_RUNNER=true` when running the image. For Sonobuoy 0.16.1, this setting is the default mode.

Its most significant feature is the new environment variables, which can be used to set arbitrary flags when  the tests are invoked. For example, to set the GCE regions mentioned above, you run Sonobuoy with the following command:

```
$ sonobuoy run --plugin-env=e2e.E2E_USE_GO_RUNNER=true \
  --plugin-env=e2e.E2E_PROVIDER=gce \
  --plugin-env=e2e.E2E_EXTRA_ARGS=”--gce-zone=foo --gce-region=bar”
```

By default, the extra arguments are split by spaces, so if you have values that require spaces, you can also set a [custom separator](https://github.com/kubernetes/kubernetes/blob/master/cluster/images/conformance/go-runner/cmd_test.go#L101-L110) with `E2E_EXTRA_ARGS_SEP`.

The only downside to this approach is that it is only available for clusters using Kubernetes 1.16.0. If you have an older cluster and want to utilize this feature, you will have to build your own test image (otherwise, the image will run the wrong set of tests for your cluster).

# Building Your Own Test Image

I’ve often referred people to the Kubernetes repo (and its [instructions](https://github.com/kubernetes/kubernetes/tree/master/test/conformance/image#how-to-release-by-hand)) when they need help building a custom test image.  That approach would surely work, but it involves actually building the Kubernetes tests yourself, which may tax your local machine or be difficult to set up the first time.

For most users, I’d recommend taking a much simpler approach: Just built a new image based on the existing one and swap out the script as needed.

By building the existing test image, the Kubernetes build process has already done the hard work: Built the tests, downloaded ginkgo, loaded everything into the right places, and included the script to execute the tests. All you have to do is tweak a single script.

So, for instance, if you want to support the provider-specific values, all you have to do is:

- Create your custom script (usually just a slight modification of the existing one)
- Create a new image and make it available (a two-line Dockerfile)
- Create a custom plug-in definition that will use your image instead of the upstream Kubernetes conformance image (a single command)
- Run Sonobuoy with the custom plug-in

Almost all of this can be entirely scripted:

```
# Be sure to push images to the correct registry and use the right version of tests.
$ export REGISTRY=schnake
$ export k8sVersion=$(kubectl version -o json|jq .serverVersion.gitVersion -r) 

# Download the existing run script as a starting place and change its mode to be executable.
$ curl https://raw.githubusercontent.com/kubernetes/kubernetes/master/cluster/images/conformance/run_e2e.sh -o run.sh && chmod +x run.sh

# The Dockerfile adds your custom script to the existing image.
$ cat << EOF > Dockerfile
FROM gcr.io/google-containers/conformance:${k8sVersion}
COPY run.sh /run.sh
EOF

# Use `sonobuoy gen plugin` to generate the plug-in YAML file.
$ sonobuoy gen plugin --name=my-e2e \
  --image=$REGISTRY/custom-conformance:v0 \
  --cmd=/run.sh > custom-conformance.yaml
```

After you’ve edited the `run.sh` script to include your custom logic, build your Docker image and run it:

```
$ docker build . -t $REGISTRY/custom-conformance:v0 && docker push $REGISTRY/custom-conformance:v0

$ sonobuoy run --plugin custom-conformance.yaml \
  --plugin-env=my-e2e.E2E_FOCUS=Conformance \
  --plugin-env=my-e2e.E2E_PROVIDER=gce \
  --plugin-env=my-e2e.E2E_GCE_ZONE=zone \
  --plugin-env=my-e2e.E2E_GCE_REGION=region
```

You can even modify this approach to load the new Golang-based runner into a test image with the older tests.

# Summary

These are just two of the simplest approaches to running highly customized Kubernetes tests. I hope that it helps you run the tests exactly how you need to. If you repeatedly encounter problems like these and think that yours is a common use case, please consider creating an [issue](https://github.com/kubernetes/kubernetes/issues/new/choose) or making a pull request to improve the upstream logic. I also wanted to say thank you to [@eddytruyen](https://github.com/eddytruyen), whose questions encouraged me to write up this workflow for others to benefit from.

Happy testing.

Join the Sonobuoy community:

- Get updates on Twitter ([@projectsonobuoy](https://twitter.com/projectsonobuoy))
- Chat with us on Slack ([#sonobuoy](https://kubernetes.slack.com/messages/sonobuoy) on Kubernetes)
