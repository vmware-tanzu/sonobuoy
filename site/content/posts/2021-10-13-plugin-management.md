---
title: Plugin Management with Sonobuoy
image: /img/sonobuoy.svg
excerpt: Sonobuoy now makes it easier to work with various plugins and configurations with built-in plugin management.
author_name: John Schnake
author_url: https://github.com/johnschnake
categories: [kubernetes, sonobuoy, plugins]
tags: ['Sonobuoy Team', 'John Schnake']
date: 2021-10-13
slug: plugin-management
---

## Introduction

Our new plugin management features are meant to solve 3 of the most common issues when routinely developing custom
[Sonobuoy][sonobuoy] plugins (and plugin configurations).

First, almost as a matter of law, if you are quickly iterating against a plugin you tend to accumulate more and more
plugin definition files (*.yaml). As the number of plugin definitions increase (e.g. quicktest.yaml, version2.yaml,
debug1.yaml, etc) it becomes cluttered and difficult to recall the different changes.

Secondly, Sonobuoy requires full filepaths (or URLs) to run custom plugins. This means that even on your own machine you
either have to keep translating short, relative paths to absolute paths or you can't copy/paste the same Sonobuoy
command because the relative paths will no longer reference the plugin correctly.

Lastly, how do you find new plugins to run? Currently, there is no intentional system around this other than a GitHub
repo
([sonobuoy-plugins][sonobuoy-plugins]) which contains published plugins. We recommend users explore this repo and
sometimes publish blog posts on new plugins, but that is a very unreliable method of gaining traction.

So to reduce clutter, simplify command reuse, and improve plugin discoverability, Sonobuoy has added plugin mangagement
commands (`install`, `list`, `show`, and `uninstall`).

Now, as you develop (or configure) your plugins, you can "install" them so that they are stored in a known
location (`~/.sonobuoy`) and Sonobuoy will automatically be able to inspect or run them.

## Installing a Plugin

Installing a plugin is extremely simple, just like running an arbitrary plugin.

To install the plugin you just have to specify a local filename (not a full path) and the source. For instance:

```bash
$ sonobuoy plugin install myPerfectPlugin ./tmp/pluginCode.yaml
Installed plugin customPlugin into file /Users/jschnake/.sonobuoy/myPerfectPlugin.yaml from source tmp/pluginCode.yaml 
```

Just like when running plugins, you can specify the source in terms of a file or as a URL. Notice that the filename
locally does not need to match either the plugin name (within the defintion file) or the source filename/URL. This is so
that the plugin is guaranteed to be locally unique.

Since the filename does not need to be identical to the plugin name, you can install _multiple configurations for the
same plugin_. This is incredibly useful for any plugin, but as an example consider the default e2e plugin. You could,
locally, save copies of your most common configurations:

- default e2e run
- a configuration which runs tests for a feature you're developing against
- a configuration to run on Windows nodes
- etc

## Listing Installed Plugins

You can list the plugins you've got installed locally via the command:

```bash
$ sonobuoy plugins list
filename: /Users/jschnake/.sonobuoy/myPerfectPlugin.yaml
plugin name: myPlugin
source URL: www.example.com/some-plugin-source.yaml
description: Details about what the plugin does.
```

The output includes:

- the filename, (which determines how to invoke the plugin)
- the plugin name (what shows up during `sonobuoy status`)
- source URL (optional; location which may post updates to this plugin)
- description (optional; description of the plugin to clarify the contents or configuration of the plugin)

## Showing an Installed Plugin

If you want to see the full definition of the plugin, simply use the `show` command:

```bash
$ sonobuoy plugin show myPerfectPlugin
```

The full YAML for the plugin will be printed. Similar to how `sonobuoy gen plugin e2e` prints the whole, default `e2e`
plugin.

## Running an Installed Plugin

Once a plugin is installed, you run it by specifying the plugin by its filename (not the full path):

```bash
$ sonobuoy run -p myPerfectPlugin
```

You can even run multiple installed plugins or continue to use absolute filepaths or URLs:

```bash
$ sonobuoy run -p myPerfectPlugin -p e2e -p example.com/raw/yourPlugin.yaml
```

## Uninstalling a Plugin

Uninstalling a plugin is exactlty as you'd expect; just specify the plugin by filename in the command:

```bash
$ sonobuoy plugin uninstall myPerfectPlugin
```

## Conclusion & Roadmap

We hope you find these commands intuitive and helpful. By installing a plugin you can more easily save various
configurations rather than having a pile of YAML files throughout multiple directories.

We'd like to continue to expand this feature to be even more helpful including:

- Integration with [sonobuoy-plugins][sonobuoy-plugins] repo in order to easily list/show/install plugins that are
  available there.
- Improved output. This is definitely a first pass and as we get some feedback about how the feature is used we can
  improve the UI to be more streamlined and helpful.
- Semi-automatic updates. Since a plugin can define its own source URL, we could check for changes to the plugin and
  update the plugin (e.g. `sonobuoy plugin update yourPerfectPlugin`)

Try it out and let us know your successes or frustrations with this new feature.

Happy testing!

## Join the Sonobuoy community:

- Star/watch us on Github: [sonobuoy][sonobuoy] and [sonobuoy-plugins][sonobuoy-plugins]
- Get updates on Twitter (@projectsonobuoy)
- Chat with us on Slack ([#sonobuoy](https://kubernetes.slack.com/archives/C6L3G051C) on Kubernetes)
- Join the K8s-conformance [working group](https://github.com/cncf/k8s-conformance)

[sonobuoy]: https://github.com/vmware-tanzu/sonobuoy

[sonobuoy-plugins]: https://github.com/vmware-tanzu/sonobuoy-plugins
