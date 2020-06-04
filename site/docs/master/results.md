# Viewing Plugin Results

The `sonobuoy results` command can be used to print the results of a plugin without first having to extract the files from the tarball.

## Canonical Data Format

Plugin results undergo post-processing on the server to produce a tree-like file which contains information about the tests run (or files generated) by the plugin. This is the file which enables `sonobuoy results` to present reports to the user and navigate the tarball effectively.

Currently, plugins are specified as either producing `junit` results (like the `e2e` plugin), `raw` results (like the `systemd-logs` plugin), or you can specify your own results file in the format used by Sonobuoy by specifying the option `manual`.

To see this file directly you can either open the tarball and look for `plugins/<name>/sonobuoy_results.yaml` or run:

```
sonobuoy results $tarball --mode=dump
```

## Working with any Plugin

By default, the command produces a human-readable report corresponding to the `e2e` plugin. However, you can specify other plugins by name. For example:

```
$ sonobuoy results $tarball --plugin systemd-logs
Plugin: systemd-logs
Status: passed
Total: 1
Passed: 1
Failed: 0
Skipped: 0
```

> In the above output, notice that even though the `systemd-logs` plugin doesn't run "tests" per se, each file produced by the plugin is reported on: a readable file is reported as a success.

## Detailed Results

If you would like to view or script around the individual tests/files, use the `--mode detailed` flag. In the case of junit tests, it will write a list of json objects which can be piped to other commands or saved to another file.

To see the passed tests, one approach would be:

```
$ sonobuoy results $tarball --mode=detailed | jq 'select(.status=="passed")'
```

To list the conformance tests, one approach would be:

```
$ sonobuoy results $tarball --mode=detailed|jq 'select(.name | contains("[Conformance]"))'
```

When dealing with non-junit plugins, the `--mode detailed` results will print the file output with a prefix that reports on the nature/location of the file:

```
$ sonobuoy results $tarball --mode=detailed --plugin systemd-logs|head -n1
systemd-logs|kind-control-plane|systemd_logs {"_HOSTNAME":"kind-control-plane",...}
```

The prefix is telling you that this result came from the "systemd-logs" plugin, was from the "kind-control-plane" node, and the filename was "systemd_logs".

If you had multiple nodes, you could look at just one by adding the `--node` flag. It walks the result tree and will return only results rooted from the given node:

```
$ sonobuoy results $tarball --mode=detailed --plugin systemd-logs --node=kind-control-plane|head -n1
kind-control-plane|systemd_logs {"_HOSTNAME":"kind-control-plane",...}
```

Now if you wanted to script around the actual file output (in this case it is json), you wouldn't want to keep that prefix around. Just add the `--skip-prefix` flag to get only the raw file output so that you can manipulate it easily:

```
$ sonobuoy results $tarball --mode=detailed --plugin systemd-logs --node=kind-control-plane --skip-prefix|head -n1|jq .MESSAGE
{"_HOSTNAME":"kind-control-plane",...}
```

## Providing results manually

When creating a plugin, you can choose to have your plugin write its results in the same format as the Sonobuoy results metadata.
This allows you to take advantage of the `sonobuoy results` workflow even if your plugin doesn't produce output in one of the other supported formats.

When using this option, Sonobuoy will process files in the Sonobuoy result format and perform any necessary aggregation to produce a single report for your plugin.
How these results are aggregated depends on how many result files your plugin produces and whether or not the plugin is a `Job` or `DaemonSet` plugin.

To use this feature, you must set the `result-type` to `manual` in the plugin definition.
When gathering the results files to aggregate, Sonobuoy will look for files listed in the `result-files` array entry in the plugin definition, or if no files are provided, it will look for a `sonobuoy_results.yaml` file in the results directory.
When using this mode, any files written to the results directory will still be available in the results tarball however only the plugin `result-files` or the `sonobuoy_results.yaml` file will be used when generating the results metadata.

The following is an example of a plugin definition using manual results:

```yaml
sonobuoy-config:
  driver: Job
  plugin-name: manual-results-plugin
  result-format: manual
  result-files:
    - manual-results-1.yaml
    - manual-results-2.yaml
spec:
  command:
  - ./run.sh
  image: custom-image:latest
  name: plugin
  resources: {}
  volumeMounts:
  - mountPath: /tmp/results
    name: results
```

### Manual results format

The format for manual results is the same as the format used by Sonobuoy when writing its results metadata.
It is a tree-like recursive data structure, where child nodes are the same type as the parent node, allowing nesting of items.
The definition for this format can be found [here](https://github.com/vmware-tanzu/sonobuoy/blob/v0.18.0/pkg/client/results/processing.go#L91-L100).

Each result `item` comprises:

 * `name`: string
 * `status`: string
 * `meta`: map of string to string
 * `details`: map of string to interface{}
 * `items`: array of `item`

An example of this format is given below:

```yaml
name: manual-results-plugin
status: custom-status
meta:
  type: summary
items:
- name: Manual test suite
  status: complete
  items:
  - name: Manual test
    status: custom-status-1
    details:
      stdout: "stdout from the test"
      messages:
        - message from the test
        - another message
  - name: Another manual test
    status: custom-status-2
    details:
      stderr: "stderr from the test"
```

The format is flexible, with no restrictions on the values used for each of the fields.

### Manual results aggregation

Sonobuoy will aggregate the results from any manual results files that it processes.
Like other plugins, it will aggregate all the results that it processes into a single results metadata file.

Each manual result file processed by Sonobuoy will be collected to form the `items` entry in the aggregated results file.
In the case of a `DaemonSet` plugin, any manual result files will be grouped by the node from which they were retrieved.

The aggregated `status` for a plugin will be based on the `status` reported within each manual result file.
In the case where the same status is found across all result files, that will be the reported status for the plugin.
Where a plugin produces multiple results files and multiple different statuses are reported, the aggregate `status` for the plugin will be the `status` from each file grouped by count into a single human readable format.
It will take the form of `status: count, status: count, ...`.
For `DaemonSet` plugins, where results files will be generated for each node, the status will be aggregated for each node in addition to the overall summary level.

## Summary

 - `sonobuoy results` can show you results of a plugin without extracting the tarball
   - Plugins are either `junit`, `raw` or `manual` type currently
   - When viewing `junit` results, json data is dumped for each test
   - When viewing `raw` results, file contents are dumped directly
   - When viewing `manual` results, results are included as provided by the plugin
 - Use the `--mode` flag to see either report, detail, or dump level data
 - Use the `--node` flag to view results rooted at a different location
 - Use the `--skip-prefix` flag to print only file output
