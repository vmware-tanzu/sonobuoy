# Using a Private Sonobuoy Image with ImagePullSecrets

This document describes how to use the ImagePullSecrets option in order to run Sonobuoy using a private Sonobuoy image.

## Setting ImagePullSecrets

The name of the secret to use when pulling the image can be set easily in the configuration file passed to `sonobuoy run` or `sonobuoy gen`:

```
echo '{"ImagePullSecrets":"mysecret"}' > secretconfig.json
sonobuoy gen --config secretconfig.json
```

Doing this properly passes the value and places it into the YAML for the Sonobuoy aggregator pod and all the pods for each plugin.

## Creating the Secret

The main complication for this flow is that secrets can only be referenced from within their own namespace. As a result we need to create the secret at the same time we create the initial resources.

Sonobuoy does not have built in support for this, but it can be manually achieved via the following process:
 - Manually create the YAML for the secret
 - Insert the YAML into the output from `sonobuoy gen --config secretconfig.json`
 - Run with `kubectl apply -f ...`

As an example of how to create the secret you can follow the instructions [here][dockersecret] in order to create a secret in the default namespace.

Then use copy most of its YAML via:

```
kubectl get secret <secret name> -o yaml > secret.json
```

Manually edit the file and remove/adjust the metadata as appropriate. The namespace should be adjusted to your desired Sonobuoy namespace (default: heptio-sonobuoy) and the following fields can be removed:
 - annotations
 - creationTimestamp
 - resourceVersion
 - selfLink
 - uid

Then just insert that YAML into the output from `sonobuoy gen` and run with `kubectl apply -f ...`

[dockersecret]: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
