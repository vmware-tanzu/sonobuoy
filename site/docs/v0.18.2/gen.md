# Customization

Sonobuoy provides many flags to customize your run but sometimes you have a special use case that isn't supported yet.  For these cases, Sonobuoy provides `sonobuoy gen`.

The command `sonobuoy gen` will print the YAML for your run to stdout instead of actually creating it. It accepts all of the relevant flags for customizing the run just like `sonobuoy run` would. You can then edit it yourself and apply it as if Sonobuoy had run it.

Output the YAML Sonobuoy would create to a file:

```
sonobuoy gen --e2e-focus="sig-networking" --e2e-skip="Alpha" > sonobuoy.yaml
```

Then manually modify it as necessary. Maybe you need special options for plugins or want your own sidecar to be running with the images.

Finally, create the resources yourself via kubectl.

```
kubectl apply -f sonobuoy.yaml
```

> Note: If you find that you need this flow to accomplish your work, talk to us about it in our [Slack][slack] channel or file an [issue][issue] in Github. Others may have the same need and we'd love to help support you.

[slack]: https://kubernetes.slack.com/messages/sonobuoy
[issue]: https://github.com/vmware-tanzu/sonobuoy/issues