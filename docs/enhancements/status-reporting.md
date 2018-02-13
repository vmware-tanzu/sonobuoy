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
        - [For Scanner](#for-scanner)
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
* Create an API on Aggregator so Scanner can report status instead of the
  existing spinner
* (optional) Workers can report their status to the aggregator

### Non-goals

* Actually updating Scanner is outside of our scope, though we should make sure
  the API is useful to that team
* Complex metrics and monitoring. The first pass should be simple.

## Proposal

### For Sonobuoy Aggregator

In addition to the HTTPS server running for the plugin workers, a second HTTPS
server and accompanying k8s `service` will be run for the purpose of
communicating status back to the client. This service will be `NodePort` or
`LoadBalanacer` rather than the `ClusterIP` of the aggregation service.

To secure this, the client will provide a server TLS cert/key to serve HTTPS
with, and a CACertificate to validate the client's peer certificates with.

This secondary HTTPS server will report status initially based on the percentage
of plugins that have reported results. Later, the workers will be able to report
heartbeats or more granular status.

The API for this status will look like:

`GET /status`

```json
{
  "status": "running",
  "percent_complete": 0.5
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

`sonobuoy run` is responsible for creating the certs for the aggregator. The CA
certificate, along with a client cert and key, will need to be stored so they
can be used later by `sonobuoy status`. To this end, they will be PEM-encoded
and stored in a JSON file along with the source address of the cluster

`cat /tmp/sonobuoy.json`

```json
{
  "host_address": "https://172.217.6.206:8002/",
  "created_at": "2008-09-15T15:53:00+05:00",
  "ca_cert": "-----BEGIN CERTIFICATE-----\nMIIDADCC<snip>rdPiFCXw==\n-----END CERTIFICATE-----",
  "client_cert": "-----BEGIN CERTIFICATE-----\nAz4cADRC<snip>qu32zv00==\n-----END CERTIFICATE-----",
  "client_key": "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIE<snip>h5bg==\n-----END EC PRIVATE KEY-----"
}
```

`sonobuoy status` will look for `/tmp/sonobuoy.json`, and verify that the
`created_at` was within some reasonable duration. If so, the PEMs are loaded,
and an HTTPS request is made to `<host_address>/status`.

### For Scanner

The certificate is created as for `sonobuoy status`, but instead of being
written out to a file it's stored in an ephemeral session tied to the session
ID. The URL can be queried periodically and the UI updated with the status.

### Heartbeats (optional/future)

Having workers report heartbeats helps detect common issues like DNS and
networking failures.

On a set interval, workers will GET a URL like
`/api/v1/health/by-node/node1/systemd-logs` or `/api/v1/health/global/e2e`.

The aggregator will keep track of this information and use it when reporting
status to clients.


```json
{
  "status": "failed",
  "percent_complete": 0.5
  "plugins": [
    {
      "name":"systemd",
      "node": "node1",
      "status": "running",
      "last-heartbeat" 17,
    },
    {
      "name":"e2e",
      "node": "",
      "status": "timed_out",
      "last-heartbeat" 200,
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
