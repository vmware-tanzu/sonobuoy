# Status Reporting

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Status Reporting](#status-reporting)
    - [Summary](#summary)
    - [Objective](#objective)
        - [Goals](#goals)
        - [Non-goals](#non-goals)
    - [Proposal](#proposal)
        - [For Sonobuoy Aggregator](#for-sonobuoy-aggregator)
        - [For `sonobuoy status`](#for-sonobuoy-status)
        - [Heartbeats (optional/future)](#heartbeats-optionalfuture)
        - [User Stories](#user-stories)
    - [Unresolved Questions](#unresolved-questions)

<!-- markdown-toc end -->

## Summary

Sonobuoy -- both `sonobuoy run` and through Scanner -- has really lackluster UX.
At present the tests will be kicked off and no information will be reported
until the run either succeeds or times out. This is not ideal. We would like to
have a concept of test status that can be reported, in near-real time, back to
the user running the service. Both the CLI and Scanner should be able to display
this progress in a way that makes sense for their individual UX.

## Objective

`sonobuoy run` and Scanner should report the status of an ongoing Sonobuoy test suite.
This status should be secure so only the intended audience can see it.

### Goals

* Create a command `sonobuoy status` that shows the results of an ongoing run.
* (optional) Workers can report their status to the aggregator

### Non-goals

* Actually updating Scanner is outside of our scope, though we should make sure
  the API is useful to that team
* Complex metrics and monitoring. The first pass should be simple.
* Scanner support

## Proposal

### For Sonobuoy Aggregator

An annotation will be attached to the sonobuoy pod, periodically updated by the
aggregation server.

The annotation will be `sonobuoy.heptio.com/status`.

```json
{
  "status": "running",
  "plugins": [
    {
      "name":"systemd",
      "node": "node1",
      "status": "complete"
    },
    {
      "name":"e2e",
      "node": "",
      "status": "running"
    }
  ]
}

```

### For `sonobuoy status`

`sonobuoy status` will use the kubeconfig acquisition logic from `sonobuoy run`,
then lookup the `sonobuoy` pod in the provided namespace. The status annotation can
then be decoded and displayed to the user.

### Heartbeats (optional/future)

The workers will set their own status annotations similar to the aggregators.
They will use the key `sonobuoy.heptio.com/status/worker`

The aggregator will keep regularly collect and collate this information and use
it when reporting status to clients.


```json
{
  "status": "failed",
  "plugins": [
    {
      "name":"systemd",
      "node": "node1",
      "status": "running",
      "last-heartbeat" "2008-09-15T15:53:00+05:00",
    },
    {
      "name":"e2e",
      "node": "",
      "status": "timed_out",
      "last-heartbeat" "2008-09-15T15:53:00+05:00",
    }
  ]
}
```

### User Stories

1. As a user running Sonobuoy, I want feedback on my ongoing runs
2. As a cluster admin, I want my cluster to fail early in the case of DNS or
   Network misconfiguration

## Unresolved Questions

How useful will this status information actually be? Will it make it easier to diagnose failures?
