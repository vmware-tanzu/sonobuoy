# Viewing Plugin Results

The `sonobuoy results` command can be used to print the results of a plugin without first having to extract the files from the tarball.

## Canonical Data Format

Plugin results undergo post-processing on the server to produce a tree-like file which contains information about the tests run (or files generated) by the plugin. This is the file which enables `sonobuoy results` to present reports to the user and navigate the tarball effectively.

Currently, plugins are specified as either producing `junit` results (like the `e2e` plugin) or `raw` results (like the `systemd-logs` plugin).

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

## Summary

 - `sonobuoy results` can show you results of a plugin without extracting the tarball
   - Plugins are either `junit` or `raw` type currently
   - When viewing `junit` results, json data is dumped for each test
   - When viewing `raw` results, file contents are dumped directly
 - Use the `--mode` flag to see either report, detail, or dump level data
 - Use the `--node` flag to view results rooted at a different location
 - Use the `--skip-prefix` flag to print only file output