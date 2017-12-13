# v2 Worker-Master communications

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [v2 Worker-Master communications](#v2-worker-master-communications)
    - [How it works now](#how-it-works-now)
    - [Summary](#summary)
    - [Objectives](#objectives)
        - [Goals](#goals)
        - [Non-goals](#non-goals)
    - [Proposal](#proposal)
        - [User Stories](#user-stories)
    - [Unresolved Questions](#unresolved-questions)

<!-- markdown-toc end -->

## How it works now 

Sonobuoy uses a worker-master model, where a master delegates tasks to worker
pods. When those pods have finished, they need to report the results of their
work back to the master. Presently this is done over an ill-defined, ad hoc
REST-ish client/server model embedded in the server.

The data passed back to the master is pretty much completely opaque. PUT
receives a path like /results/node1/systemd, and a blob of data as the body.
That data is written out, unexamined, to a file like node1/systemd in the
finished tarball.

The worker is given JSON config blob which tells it where and how to report its
results. This contains, amongst other things, the URL of the master where those
results will be sent.

The plugins themselves use a two-container model. One container, that created by
the plugin author, outputs its results in a shared directory. It touches a
`done` file with the name of the file it created. The other container is
responsible for submitting these results back to the master. It watches for the
`done` file, then uses the config JSON to `PUT` those results back to the
master.

## Summary

The interaction between the master and worker is ad hoc and poorly documented.
All data is also transmitted plain text over HTTP with no authentication or
verification.

## Objectives

Replace the existing, organically grown architecture with a mature, secure
solution that doesn't compromise the existing project's extensibility or
existing users.

### Goals

* Secure and authenticate the channel between the consumer and the producer
* Formalise and document the communication channel between the worker container
  and the master
* Extract and clean up the worker client so other projects can vendor it
  directly
* (Ancillary) Prevent simultaneous or poorly-cleaned up Sonobuoy runs from
  interfering with each other
* Transition should be seamless for existing plugin authors

### Non-goals

* The `done` interface between the Producer and Consumer containers is not part
of this proposal. This contract is already used by public consumers outside the
control of Heptio.

## Proposal

My original proposal was to use GRPC, but given that we are submitting wholly
unstructured data this is likely overkill.

Instead, the current URL (`/api/v1/results/by_node/:node_name/:plugin_name`,
`/api/v1/resutlts/global/:plugin_name`) will be kept. Instead of HTTP, though,
the results will communicate over mutual authentication TLS. The client will
authenticate the server by a vendored CA certificate, and the server will
authenticate the client through an individually assigned client certificates.

The CA certificate and client certificate will be added to the JSON config blob
all nodes are defined, not baked into the images themselves. The node private
key will be transmitted as a Kubernetes secret. Every master run should create a
brand-new certificate authority and credentials. This ensures duplicate or
overlapping runs cannot interfere with each other.

The code for communicating this information back will be isolated to a specific
package that other projects can use to write their own workers that don't use
our containers.

### User Stories

* As a Sonobuoy user, I want my tests to continue to work as they did before
  this change was made
* As an existing Sonobuoy plugin author, it should take little to no work to
  update my plugin
* As a new plugin author, it should be clear how I author a new plugin
* As an author of a new Go-based plugin, I should be able to directly use the
  client library if I need to

## Unresolved Questions

* No Metadata other than node name and plug-in name is currently collected. If
  at some point we wish to do this, there will not be any out-of-band way to
  collect this other than URL parameters or headers.
