# Sonobuoy Config

The commands "run" and "gen" both accept a parameter for a Sonobuoy config file which allows you to customize multiple aspects of the run.

We've provided a command to generate the JSON file necessary so that it is easier to edit for your runs. Run the command:

```
sonobuoy gen config
```

and you will see the default configuration. Below is a description of each of the values.

## General options

`Description`: A string which provides consumers a way to add extra context to a configuration that may be in memory or saved to disk. Unused by Sonobuoy itself.

`UUID`: A unique identifier used to identify the run of this configuration. Used in a few places including the name of the results file.

`Namespace`: The namespace in which to run Sonobuoy.

`WorkerImage`: The image for the Sonobuoy worker container which runs as a sidecar along the plugins. Responsible for reporting results back to the Sonobuoy aggregator.

`ImagePullPolicy`: The image pull policy to set on the Sonobuoy worker sidecars as well as each of the plugins.

`ResultsDir`: The location on the Sonobuoy aggregator where the results are placed.

`Version`: The version of Sonobuoy which created the configuration file.


## Plugin options

`Plugins`: An array of plugin selection objects of the plugins you want to run. When running custom plugins (or avoiding running a particular plugin) this value needs modified.

`PluginSearchPath`: The aggregator pod looks for plugin configurations in these locations. You shouldn't need to edit this unless you are doing development work on the aggregator itself.

## Query options

`Resources`: A list of resources which Sonobuoy will query for in every namespace in which it runs queries. In the namespace in which Sonobuoy is running, `PodLogs`, `Events`, and `HorizontalPodAutoscalers` are also added.

`Filters`: Options for filtering which resources queries should be run against.

 * `Namespace`: A regexp which specifies which namespaces to run queries against.
 * `LabelSelector`: A Kubernetes [label selector][labelselector] which will be added to every query run.

`Limits`: Options for limiting the scope of response.

 * `PodLogs`: limits the scope when getting logs from pods. The supported parameters are:

    * `Namespaces`: string

        * A regular expression for the targeted namespaces.
        * Default is empty string
        * To get logs from all namespaces use ".*"
    * `SonobuoyNamespace`: bool

        * If set to true, get pod logs from the namespace Sonobuoy is running in. Can be set along with a `Namespaces` field or on its own.
        * Default value is true
    * `FieldSelectors`: []string

        * A list of field selectors, with OR logic.
          For example, to get logs from two specified namespaces `FieldSelectors = ["metadata.namespace=default","metadata.namespace=heptio-sonobuoy"]`
        * Each field selector contains one or more chained operators, with AND logic
          For example, to get logs from a specified pod `FieldSelectors = ["metadata.namespace=default,metadata.name=pod1"]`
        * Each field selector follows the same format as a [Kubernetes Field Selector][fieldselector].
        * Can be set along with the `Namespaces` or `SonobuoyNamespace` field or on its own.
    * `LabelSelector`: string

        * Filters candidate pods by their labels, using the same format as a [Kubernetes Label Selector][labelselector].
          For example: `LabelSelector = "app=nginx,layer in (frontend, backend)"`
        * When set together with other fields, the scope of pods is defined by:
            ```
            (Namespaces OR SonobuoyNamespace OR FieldSelectors) AND LabelSelector
            ```
    
    * For each candidate pod, the format and size of logs are defined by other fields. These will be passed onto Kubernetes [PodLogOptions][podlogopts]:
        * `Previous`: bool
        * `SinceSeconds`: int
        * `SinceTime`: string. RFC3339 format.
        * `Timestamps`: bool
        * `TailLines`: int
        * `LimitBytes`: int


[fieldselector]: https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
[labelselector]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
[podlogopts]: https://godoc.org/k8s.io/api/core/v1#PodLogOptions
