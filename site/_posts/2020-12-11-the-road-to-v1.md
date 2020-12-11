---
title: The Road to Sonobuoy version 1.0
excerpt: With the release of version 0.20, Sunobuoy starts its journey toward 1.0 -- let's see what it will take to get there.
author_name: Vladimir Vivien
author_url: https://github.com/vladimirvivien
author_avatar: /img/contributors/vladimir-vivien.png
categories: [kubernetes, sonobuoy, roadmap]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team']
---

After numerous releases, the Sonobuoy project has planted itself as a permanent and useful fixture in the cloud native community.  The release of version 0.20 marks the start of a vision to take the project toward a 1.0 release.  There are many enhancements, however, that need to be put in place before the project can get to this coveted milestone.  This blog is part roadmap and part vision statement to the community of where the team would like to take the project to get to 1.0.

## Decoupling Sonobuoy releases from that of Kubernetes
One of the first improvements (introduced in version 0.20) is the ability to release Sonobuoy without relying directly on code from the main Kubernetes project.  Prior to release v0.20, Sonobuoy used code copied and pasted from upstream Kubernetes to support features that rely on test image dependencies baked in Kubernetes packages. This forced the Sonobuoy team to do a release after each Kubernetes release to ensure compatibility.  

With the 0.20 release, the code has been enhanced to dynamically generate image dependency information without relying directly on Kubernetes code.  This allows Sonobuoy to have its own release cadence while it is still able to run conformance tests for current and future versions of Kubernetes. [Read more about this feature here](/decoupling-sonobuoy-and-kubernetes).

## Sonobuoy Plugin Framework Enhancements
Sonobuoy plugins are a way to expand the capabilities of the base project.  The plugin model uses Kubernetes pods as abstraction for a pluggable architecture.  There are several Sonobuoy plugins already published upstream here (with more to come).  As a step toward 1.0, it is crucial to consider re-implementing the Sonobuoy plugin control mechanism using the Controller pattern with CRDs.  This may introduce new capabilities not currently supported such as:

 * Represent running plugin as a manageable API resource
 * Ability to query current state and progress of a running plugin
 * Better lifecycle control management of plugin components
 * Direct management of Sonobuoy resources with kubectl directives

Other enhancements being considered for plugins include:
 * Simpler manifest to describe the plugin
 * Support for value injection (using tool like ytt)
 * Improved and uniform command-line UX experience with plugin management
 * Automatically save and caching plugin manifest locally

## Externalizing the E2E Plugin
 
When Sonobuoy started, its main function was to run E2E conformance tests.  Then the project introduced its pluggable model and has since released many plugins.  However, the E2E plugin remained embedded directly in the Sonobuoy source code.  While this makes it convenient for users, having E2E plugin inside the main source code has the following downsides:

 * Bloated codebase
 * Breaks the plugin model
 * Abstraction and usability leakage throughout the project
 * Inconsistent command-line interface to accommodate e2e plugin

Part of the projected work, as Sonobuoy marches toward 1.0, is the externalization of the  E2E plugin with the following benefits:

 * Follows a uniform Sonobuoy plugin model for all plugins
 * Remove convenient, but complex, embedded plugin code from Sonobuoy source
 * Shift all E2E concerns to the plugin rather than the Sonobuoy binary
 * Reduces the concerns of the sonobuoy binary to that of just a plugin management and deployment

## Plugins, plugins, plugins
Today, the Sonobuoy ecosystem hosts a handful of plugins to help assess the readiness of Kubernetes clusters.  As the project marches to version 1.0, the Sonobuoy team will be considering the development of plugins with a wide range of functionalities to help cluster operators with readiness strategies including:

 * Networking diagnostics and performance
 * Cluster health and readiness
 * Storage conformance
 * Diagnostics of Cluster-API managed clusters
 * Visual UX to present Sonobuoy data
 * Support for plugins created with [Crashd scripts](https://github.com/vmware-tanzu/crash-diagnostics)
 * And more

## Windows support
Support for Sonobuoy plugin on Windows nodes stalled in 2020.  One of the main goals for 2021 will be to revive this effort so that Sonobuoy can easily launch and manage plugins running on Windows nodes.
