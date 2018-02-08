# Sonobuoy Snapshot Layout

- [Filename](#filename)
- [Contents](#contents)
	- [/resources](#resources)
	- [/hosts](#hosts)
	- [/podlogs](#podlogs)
	- [/plugins](#plugins)
	- [/meta](#meta)
	- [/serverversion.json](#serverversionjson)
- [File formats](#file-formats)

This document describes the standard layout of a Sonobuoy results tarball, how it is formatted, as well as how data is named and laid out.

## Filename

A Sonobuoy snapshot is a gzipped tarball, named `YYYYmmDDHHMM_sonobuoy_<uuid>.tar.gz`.

Where YYYYmmDDHHMM is a timestamp containing the year, month, day, hour, and minute of the run.  The `<uuid>` string in the Sonobuoy must be an RFC4122 UUID, consisting of lowercase hexadecimal characters and dashes (e.g. "dfe30ebc-f635-42f4-9608-8abcd6311916").  This UUID should match the UUID from the snapshot's [meta/config.json][1], stored at the root of the tarball.

## Contents

The top-level directories in the results tarball look like the following:

![tarball overview screenshot][3]

### /resources

The `/resources` directory contains JSON-serialized Kubernetes objects, taken from querying the Kubernetes REST API. The directory has the following structure:

- `/resources/ns/<namespace>/<type>.json` - For all resources that belong to a namespace, where `<namespace>` is the namespace of that resource (eg. `kube-system`), and `<type>` is the type of resource, pluralized (eg. `Pods`).
- `/resources/cluster/<type>.json` - For all resources that don't belong to a namespace, where `<type>` is the type of resource, pluralized (eg. `Nodes`).

This looks like the following:

![tarball resources screenshot][4]

### /hosts

The `/hosts` directory contains information gathered about each host in the system by directly querying their HTTP endpoints.  *This is distinct from information you'd find in `/resources/cluster/Nodes.json`*, because it contains things like health and configuration for each host, which aren't part of the Kubernetes API objects.

- `/hosts/<hostname>/configz.json` - Contains the output of querying the `/configz` endpoint for this host, that is the hosts's component configuration.
- `/hosts/<hostname>/healthz.json` - Contains a json-formatted representation of the result of querying `/healthz` for this host, example: `{"status":200}`

This looks like the following:

![tarball hosts screenshot][5]

### /podlogs

The `/podlogs` directory contains logs for each pod found during the Sonobuoy run, similarly to what you would get with `kubectl logs -n <namespace> <pod> <container>`.

- `/podlogs/<namespace>/<podname>/<containername>.log` - Contains the logs for the each container, for each pod in each namespace.

This looks like the following:

![tarball podlogs screenshot][6]

### /plugins

The `/plugins` directory contains output for each plugin selected for this Sonobuoy run.  The structure is listed below.

- `/plugins/<plugin>/results.<format>` - For plugins that run on an any arbitrary node to collect cluster-wide data (ones that use the Job driver, for instance) and don't run one-per-node, this will contain the results for this plugin, using the format that the plugin expects.  See [file formats][2] for details.

- `/plugins/<plugin>/results/<hostname>.<format>` - For plugins that run once on every node to collect node-specific data (ones that use the DaemonSet driver, for instance), this will contain the results for this plugin, for each node, using the format that the plugin expects.  See [file formats][2] for details.

Some plugins can include several files as part of their results.  To do this, a plugin will submit a `.tar.gz` file, the contents of which are extracted into the following:

- `/plugins/<plugin>/results/<extracted files>` - For plugins that collect cluster-wide data into a `.tar.gz` file

- `/plugins/<plugin>/<node>/<extracted files>` - For plugins that collect per-node data into a `.tar.gz` file

This looks like the following:

![tarball plugins screenshot][7]

### /meta

The `/meta` directory contains metadata about this Sonobuoy run, including configuration and query runtime.

- `/meta/query-time.json` - Contains metadata about how long each query took, example: `{"queryobj":"Pods","time":12.345ms"}`
- `/meta/config.json` - A copy of the Sonobuoy configuration that was set up when this run was created, but with unspecified values filled in with explicit defaults, and with a `UUID` field in the root JSON, set to a randomly generated UUID created for that Sonobuoy run.

This looks like the following:

![tarball meta screenshot][8]

### /serverversion.json

`/serverversion.json` contains the output from querying the server's version, including the major and minor version, git commit, etc.

## File formats

For each type of result in a Sonobuoy tarball, the file extension should denote how the file should be parsed.  This allows any automated system to ingest the contents of the Sonobuoy tarball, even without precise knowledge of its directory layout.

- `.json` - This file must be a valid JSON file, with either an array at the root (`[...]`), or a json object (`{...}`)
- `.txt` - This file should be treated as opaque text, and there is no reliable way for its contents to be parsed.  Generally this is used for data intended to be read by humans.
- `.log` - This file should be parsed as a series of log entries, one per line.
- `.xml` - This file must be valid XML

[1]: #meta
[2]: #file-formats
[3]: img/snapshot-00-overview.png
[4]: img/snapshot-10-resources.png
[5]: img/snapshot-20-hosts.png
[6]: img/snapshot-30-podlogs.png
[7]: img/snapshot-40-plugins.png
[8]: img/snapshot-50-meta.png
