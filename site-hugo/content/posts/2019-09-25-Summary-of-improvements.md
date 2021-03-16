---
title: Recent Improvements in Sonobuoy
image: /img/sonobuoy.svg
excerpt: Now with support for Kubernetes 1.16.0, let's review all the recent improvements to Sonobuoy.
author_name: John Schnake
author_url: https://github.com/johnschnake
author_avatar: /img/contributors/john-schnake.png
categories: [kubernetes, sonobuoy, conformance]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team']
date: 2019-09-25
---

With the recent release of [Sonobuoy][github] 0.16.0, we thought it would be a good time to recap all the improvements made to Sonobuoy in our recent releases.

First off, Sonobuoy now supports Kubernetes 1.16.0 clusters.  Secondly, in the past three months we’ve had seven releases and made major improvements in almost every area of Sonobuoy. There are too many changes to list them all, so check out our [release notes][releases] for more information. Here are the largest improvements you should be aware of.

## Improvements Running Sonobuoy
### Keeping Your Workloads Safe
In Kubernetes 1.16.0, there are a few end-to-end tests that are disruptive to non-test workloads. We understand that many users run Sonobuoy and the Kubernetes E2E tests regularly on live clusters to monitor and document cluster health. As a result, we’ve changed some of the options for `sonobuoy run --mode`.

By default, `sonobuoy run` will run the non-disruptive tests so that no one accidentally impacts their production workloads. This default mode is the same as explicitly specifying:

```
sonobuoy run --mode=non-disruptive-conformance
```

If you are trying to generate results to qualify for the [Kubernetes Software
Conformance Certification][cncf] program run by the CNCF, you must run all the conformance tests. To do so, run the following command: 

```
sonobuoy run --mode=certified-conformance
```

### Plug-in Progress Updates
Sonobuoy now enables plug-ins to report their progress back to you. The reporting means that you have an extra tool at your disposal to understand how a plug-in is progressing or if it has stalled. This early and continuous feedback from the plug-in has the opportunity to drastically improve many workflows. It is our hope that this functionality will be leveraged by the Kubernetes E2E plug-in as part of Kubernetes 1.17.0. To see an example use of this feature, check out the readme file for the [progress reporter][progressExample].

### Checking Logs
The `sonobuoy logs` command has had a few improvements to make it more useful. If there is a single plug-in of interest, you can specify that with the `--plugin` flag so that you only see the logs that are relevant to you. In addition, when you tail logs with the `sonobuoy logs -f` command, Sonobuoy will now retry the command on containers that have not yet started running (previously, the command would fail if the container wasn’t already running). This approach facilitates useful snippets like:

```
sonobuoy run && sonobuoy logs -f
```

## Improvements Understanding Results
### More Visible, Clearer Results, Tarball Optional
A common frustration of Sonobuoy users was that the results were opaque. After a test run, you would have to download a tarball, untar it somewhere, and then find the results you were interested in. These steps were frustrating and, in some edge cases, made scripting difficult.

We listened to your feedback and simplified the process, making it much easier to see the data that is most important to you.

First, you can use `sonobuoy status` to see not only whether plug-ins have completed but also whether the plug-in itself passed or failed without ever needing to download the tarball:

```
$ sonobuoy status
   PLUGIN     STATUS   RESULT   COUNT
   e2e            complete   passed     1
```

In addition, you can provide the `--json` flag to get even more detailed results about how many tests passed, failed, or were skipped, and details about the results tarball before you download it:

```
$ sonobuoy status --json
{
  "plugins": [
    {
      "plugin": "e2e",
      "node": "global",
      "status": "complete",
      "result-status": "passed",
      "result-counts": {
        "passed": 1,
        "skipped": 4896
      }
    }
  ],
  "status": "complete",
  "tar-info": {
    "name": "201909201458_sonobuoy_e8d39f5e-68c2-455e-8c1c-2ac2201da6ed.tar.gz",
    "created": "2019-09-20T14:58:49.5100259Z",
    "sha256": "c24fba627a845bdac76f64bc655a861e73fb75f21d69c7840cda80b46775a836",
    "size": 211681
  }
}
```

The `--json` flag can be really useful since you can download and save your tarball along with this data. As a result, you have:

 - all the source data if you need it
 - a summary of the results for easier viewing
 - a way to correlate the source data and the results

### More Data, Untar Unnecessary
If you do decide to download the tarball, we also provide a new command, `sonobuoy results`, to keep you from having to open the tarball to understand if your plug-ins passed or failed:

```
$ sonobuoy results results.tar.gz
Plugin: e2e
Status: passed
Total: 4897
Passed: 1
Failed: 0
Skipped: 4896
```

The `sonobuoy results` command works for any plug-in (just provide the `--plugin` flag) and can provide summary results or detailed results about every test failure or generated file. Here’s an example:

```
sonobuoy results results.tar.gz --mode=detailed
{"name":"[sig-storage] In-tree Volumes [Driver: cinder] [Testpattern: Dynamic PV (filesystem volmode)] multiVolume [Slow] should access to two volumes with different volume mode and retain data across pod recreation on the same node","status":"skipped","meta":{"path":"e2e|junit_01.xml|Kubernetes e2e suite"}}
…
```

Since Sonobuoy outputs that information in JSON, it can be incredibly powerful since you can pipe the data to other tools for more advanced filtering and analysis. For more details and examples, check out [@johnSchnake][schnakeGithub]'s [blog post][resultsBlogPost].

## Improvements for Custom Plug-ins
### It’s Your Plug-in, Run It How You Want
We’ve continued to improve support for custom plug-ins and want to empower you to make the best plug-ins and workflows for your unique situation.

To support custom plug-ins, the plug-in definition (the YAML you provide to Sonobuoy to run your plug-in) now allows you to specify all the `PodSpec` fields. This usage is optional, so any existing plug-ins will continue to function and many plug-ins can simply accept the default value.

However, by allowing each plug-in to specify the PodSpec for itself, we believe we unlock a lot of use cases that were previously unsupported. For instance, you can run multi-container plug-ins, plug-ins with more access to the host machine, and plug-ins with unique networking requirements. Check out [@zubrons][zubronGithub] [blog post][podSpecBlogPost] about this feature for more details and examples.

### Keep Your Plug-ins Alive
Plug-ins can specify `SkipCleanup` in their plug-in definition so that Sonobuoy won’t delete their pods after running. This setting can be really useful when you’re debugging cluster issues and  want continued access to the pods after a plug-in has finished running.

### Environment Variables for Everyone
Environment variables can be specified for any plug-in from the command-line by using the `--plugin-env` flag. This flag makes it easier to reuse custom plug-ins without having to constantly modify their YAML definitions.

## Summary

We are incredibly proud of the improvements we’ve made recently to the project and hope that these features make your workflows more simple and more powerful. If you have feedback on any of these features or have an idea for a new one, we’d love to hear from you.

Join the Sonobuoy community:

- Get updates on Twitter ([@projectsonobuoy][twitter])
- Chat with us on Slack ([#sonobuoy][slack] on Kubernetes)
- Join the Kubernetes Software Conformance Working Group: [github.com/cncf/k8s-conformance][conformance-wg]

[twitter]: https://twitter.com/projectsonobuoy
[slack]: https://kubernetes.slack.com/messages/C6L3G051C
[conformance-wg]: https://github.com/cncf/k8s-conformance
[github]: https://github.com/vmware-tanzu/sonobuoy
[cncf]: https://www.cncf.io/certification/software-conformance/
[schnakeGithub]: https://github.com/johnschnake
[zubronGithub]: https://github.com/zubron
[releases]: https://github.com/vmware-tanzu/sonobuoy/releases
[progressExample]: https://github.com/vmware-tanzu/sonobuoy/tree/master/examples/plugins/progress-reporter
[resultsBlogpost]: https://sonobuoy.io/simplified-results-reporting-with-sonobuoy/
[podSpecBlogPost]: https://sonobuoy.io/customizing-plugin-podspecs/