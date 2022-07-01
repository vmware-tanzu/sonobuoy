---
title: "July Drumbeat"
image: /img/sonobuoy.svg
excerpt: An update about the recent improvements to Sonobuoy. 
author_name: John Schnake
author_url: https://github.com/johnschnake
categories: [kubernetes, sonobuoy, plugins]
tags: ['Sonobuoy Team', 'John Schnake']
date: 2022-07-01
slug: july-drumbeat
---

# Sonobuoy Drumbeat

Now that we release on a fairly frequent basis (every 1-3 weeks) we like to present a summary of relevant changes periodically for users updating Sonobuoy.

## Dealing with encoded newlines and tabs in results

When running `sonobuoy results <tarball> --mode dump` you get the full results of the plugins in a YAML format.
The problem is that often, newlines and tabs are encoded as "\n" and "\t" making it very difficult to read large blocks of text.

We've now added a different mode `--mode=readable` which will dump the same YAML output but making it more human-friendly.
This is for human-readability and not for machine parsing, which should still utilize modes `detailed` for JSON data or `dump` for the exact YAML source data.

## Improving Contributor Experience and Documentation

We've added the CLI documentation (e.g. `sonobuoy run -h`) to the documentation [site](https://sonobuoy.io/docs/v0.56.7/cli/sonobuoy/).
This can make it easier to explore the CLI and share information about options with others.

We've also re-added a `Makefile` to the repository in order to streamline the most common build functions.
The old Makefile we had as too bloated and had multiple bugs throughout the years.
The new Makefile will avoid this by simply being a wrapper around the build scripts for the most common use cases.
All build logic will continue to be put into the Bash build scripts, but by putting basic build/test calls into the Makefile should simplify things for new developers and users.

## New Options and Commands

#### Sonobuoy get pods

Pods for plugins get assigned names with generated values in them, meaning that in order to query them with `kubectl` you first have to
query the Sonobuoy namespace then copy/paste the name into your command.
To make this even easier to script with, we've added a new command `sonobuoy get pods` that will grab the pod names for the plugins you've launched.
You can also add the `-p <plugin name>` option to just get one of them.
We hope this will make Sonobuoy even easier to combine with other tools.

#### Customizable Service Account

Sonobuoy now supports `--service-account-name` and `--existing-service-account` flags which will allow you to customize the SA
used for the aggregator/plugins.
This gives you finer control of permission when necessary.
 
## Published Roadmap

We've published a roadmap for the project on the GitHub [Wiki](https://github.com/vmware-tanzu/sonobuoy/wiki).
We'll update it periodically but our hope is that it will give you a basic insight into what we're focusing on for the project.

## Join the Sonobuoy community

 - Star/watch us on GitHub [sonobuoy](https://github.com/vmware-tanzu/sonobuoy) and [sonobuoy-plugins](https://github.com/vmware-tanzu/sonobuoy-plugins)
 - Get updates on Twitter ([@projectsonobuoy][twitterLink])
 - Chat with us on Slack (#sonobuoy on Kubernetes)

[postprocessor]: https://github.com/vmware-tanzu/sonobuoy-plugins/tree/main/post-processor
[ytt]: https://carvel.dev/ytt/
[sonolark]: https://github.com/vmware-tanzu/sonobuoy-plugins/tree/main/sonolark
[starlark]: https://github.com/bazelbuild/starlark
[twitterLink]: https://twitter.com/projectsonobuoy