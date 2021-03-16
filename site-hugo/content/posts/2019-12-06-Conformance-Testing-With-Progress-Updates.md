---
title: Does Testing Kubernetes Conformance Leave You in the Dark? Get Progress Updates as Tests Run
image: /img/sonobuoy.svg
excerpt: Get real-time progress updates from the long-running conformance tests.
author_name: John Schnake
author_url: https://github.com/johnschnake
author_avatar: /img/contributors/john-schnake.png
categories: [kubernetes, sonobuoy, conformance]
# Tag should match author to drive author pages
tags: ['Sonobuoy Team', 'John Schnake']
date: 2019-12-06
---

# Does Testing Kubernetes Conformance Leave You in the Dark? Get Progress Updates as Tests Run

In Sonobuoy 0.15.4, we introduced the ability for plugins to report their plugin's progress to Sonobuoy by using a customizable webhook. Reporting status is incredibly important for long-running, opaque plugins like the `e2e` plugin, which runs the Kubernetes conformance tests.

We're happy to announce that as of Kubernetes 1.17.0, the Kubernetes end-to-end (E2E) test framework will utilize this webhook to provide feedback about how many tests will be run, have been run, and which tests have failed.

This feedback helps you see if tests are failing (and which ones) before waiting for the entire run to finish. It also helps you identify whether tests are hanging or progressing.

## How to Use It

There are two requirements to using this feature for the `e2e` plugin:

 - The conformance image used must correspond to Kubernetes 1.17 or later
 - Sonobuoy 0.16.5 or later must be used; we added this support prior to 0.17.0 to support Kubernetes prereleases.
 
First, start a run of the `e2e` plugin by running the following command, which kicks off a long-running set of tests:

```
$ sonobuoy run
```

Now, you can poll the status by using this command:

```
$ sonobuoy status --json | jq
```

After the tests start running, you will start to see output that includes a section like this:

```
{
    "plugin": "e2e",
    "node": "global",
    "status": "running",
    "result-status": "",
    "result-counts": null,
    "progress": {
        "name": "e2e",
        "node": "global",
        "timestamp": "2019-11-25T17:21:32.5456932Z",
        "msg": "PASSED [sig-storage] HostPath should give a volume the correct mode [LinuxOnly] [NodeConformance] [Conformance]",
        "total": 278,
        "completed": 2
    }
}
```

Voila! Anytime during a run, you can now check in and be more informed about how the run is going. As tests fail, the output will also return an array of strings with the test names in the `failures` field (and the "msg" field just reports the last test finished and its result). For example:

```
    {
      ...
      "progress": {
        ...
        "msg": "FAILED [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should create service with ipv4 cluster ip [Feature:IPv6DualStackAlphaFeature:Phase2]",
        ...
        "failures": [
          "[sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should create service with ipv4 cluster ip [Feature:IPv6DualStackAlphaFeature:Phase2]"
        ]
      }
    },
```

## Q and A

**Q:** I'm using a new version of Kubernetes but am using an advanced test configuration I store as a YAML file. Can I still get progress updates?

**A:** Yes, there are just two environment variables for the `e2e` plugin that need to be set in order for this to work:

```
- name: E2E_USE_GO_RUNNER
  value: "true"
- name: E2E_EXTRA_ARGS
  value: --progress-report-url=http://localhost:8099/progress
```

The `E2E_USE_GO_RUNNER` value ensures that the conformance test image uses the Golang-based runner, which enables passing extra arguments when the tests are invoked. The `E2E_EXTRA_ARGS` value sets the flag to inform the framework about where to send the progress updates.

The status updates are just sent to `localhost` because the test container and the Sonobuoy sidecar are co-located in the same pod.

**Q:** I want to try out this feature but don't have a Kubernetes 1.17.0 cluster available; how can I test it?

**A:** The important thing is that the conformance test image is 1.17 or later so you can manually specify the image version if you just want to tinker. Since the test image version and the API server version do not match, the results might not be reliable (it might, for instance, test features your cluster doesn't support) and would not be valid for the [Certified Kubernetes Conformance Program](https://www.cncf.io/certification/software-conformance).

You can specify the version that you want to use when you run Sonobuoy; here’s an example:

```
sonobuoy run --kube-conformance-image-version=v1.17.0-beta.2
```

**Q:** I'd like to implement progress updates in my own custom plugin. How do I do that?

**A:** To see an example use of this feature, check out the readme file for the [progress reporter](https://github.com/vmware-tanzu/sonobuoy/tree/master/examples/plugins/progress-reporter). The Sonobuoy sidecar will always be listening for progress updates if your plugin wants to send them, so it is just a matter of posting some JSON data to the expected endpoint.

## Join the Sonobuoy Community

- Get updates on Twitter ([@projectsonobuoy](https://twitter.com/projectsonobuoy))
- Chat with us on the Kubernetes Slack ([#sonobuoy](https://kubernetes.slack.com/messages/sonobuoy))
- Join the [Kubernetes Software Conformance Working Group](https://github.com/cncf/k8s-conformance)
