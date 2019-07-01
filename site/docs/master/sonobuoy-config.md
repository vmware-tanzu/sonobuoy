# Sonobuoy Config

The commands "run" and "gen" both accept a parameter for a Sonobuoy config file which allows you to customize multiple aspects of the run.

We've provided a command to generate the JSON file necessary so that it is easier to edit for your runs. Run the command:

```
sonobuoy gen config
```

and you will see the default configuration. Below is a description of each of the values.

## General options

Description
 - A string which provides consumers a way to add extra context to a configuration that may be in memory or saved to disk. Unused by Sonobuoy itself.

UUID
 - A unique identifier used to identify the run of this configuration. Used in a few places including the name of the results file.

Namespace
 - The namespace in which to run Sonobuoy.

WorkerImage
 - The image for the Sonobuoy worker container which runs as a sidecar along the plugins. Responsible for reporting results back to the Sonobuoy aggregator.

ImagePullPolicy
 - The image pull policy to set on the Sonobuoy worker sidecars as well as each of the plugins.

ResultsDir
 - The location on the Sonobuoy aggregator where the results are placed.

Version
 - The version of Sonobuoy which created the configuration file.


## Plugin options

Plugins
 - An array of plugin selection objects of the plugins you want to run. When running custom plugins (or avoiding running a particular plugin) this value needs modified.

PluginSearchPath
 - The aggregator pod looks for plugin configurations in these locations. You shouldn't need to edit this unless you are doing development work on the aggregator itself.

## Query options

Resources
 - A list of resources which Sonobuoy will query for in every namespace in which it runs queries. In the namespace in which Sonobuoy is running, PodLogs, Events, and HorizontalPodAutoscalers are also added.

Filters
 - Namespace
   - A regexp which specifies which namespaces to run queries against.
 - LabelSelector
   - A Kubernetes [label selector][labelselector] which will be added to every query run.

Limits
 - Options for limiting limits the scope of response when getting logs from pods. These will be passed onto Kubernetes [PodLogOptions][podlogopts]
 - The supported parameters are:
    - Previous
    - SinceSeconds
    - SinceTime
    - Timestamps
    - TailLines
    - LimitBytes
    
[labelselector]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
[podlogopts]: https://godoc.org/k8s.io/api/core/v1#PodLogOptions