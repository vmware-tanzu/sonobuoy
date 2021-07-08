---
title: Understanding Kubernetes E2E Tests
image: /img/sonobuoy.svg
excerpt: Kubernetes testing can be confusing. Let us demystify the test suite for you.
author_name: John Schnake
author_url: https://github.com/johnschnake
categories: [kubernetes, sonobuoy, windows]
tags: ['Sonobuoy Team', 'John Schnake']
date: 2021-07-06
slug: understanding-e2e-tests
---

One of the most common questions that we get from Sonobuoy users is: “Why are so many tests skipped?” Understandably, users want to know that they are running the right set of tests for their Kubernetes clusters but the complexity of the test suite, misnomers, and inaccurate language can make that difficult.

We wanted to explain the nomenclature of the Kubernetes test suite and describe when and how to run those tests.

## End-to-End Tests

The entirety of the Kubernetes test suite lives in the main [Kubernetes][kubernetesRepo] repository on GitHub. When taken collectively, these are usually referred to as the Kubernetes **end-to-end tests.**

There are over 4,000 of these tests and they cover all features of the system: scheduling, blue-green deployments, every type of storage provider, disaster recovery and so on. You will never run all the end-to-end tests. Not only are there so many that it would be impractical to, but some tests are fundamentally incompatible (e.g. some only apply to Windows nodes, some only to Linux nodes).

## Tags

Each test can have 0 or more **tags**. These are just special parts of the test name put into brackets so that they are easy to search/filter with. For example:

```
Job [Feature:SuspendJob] should not create pods when created in suspend state
```

Has one tag: `[Feature:SuspendJob]`, while:

```
[sig-api-machinery] CustomResourceDefinition resources [Privileged:ClusterAdmin] should include custom resource definition resources in discovery documents [Conformance]
```

has 3 tags: `[sig-api-machinery]`, `[Privileged:ClusterAdmin]`, and `[Conformance]`

By themselves, tags do nothing. However, the test framework uses regular expressions to choose which tests to run so the tags are useful since they allow us to select certain subsets of the end-to-end test suite easily.

#### How to run tests marked with a tag

Target tests with a tag in the name using:

```
$ sonobuoy run --e2e-focus RUN_THESE
```

Avoid tests with a tag in the name using:

```
$ sonobuoy run --e2e-skip DONT_RUN_THESE
```

Combined these to narrowly select your tests:

```
$ sonobuoy run --e2e-focus RUN_THESE --e2e-skip DONT_RUN_THESE
```

#### How many tests are there

Varies by the tag provided

#### When to run tests using “--e2e-focus”

- If you are focused on a single aspect or feature of the cluster
- You are trying to obtain more information about a failure in a particular feature of the cluster
- You are developing against Kubernetes or tooling related to a feature

Be careful when running tests with a particular tag: you may get all the tests you want and some that you do not. Since a test may have multiple tags, it may have some overlap with other features that your configuration does not support. 

You can add the `E2E_DRYRUN=true` option to execute the plugin in a fast way to just see which tests it _would_ run based on your regular expressions:

```
$ sonobuoy run --focus TAG --plugin-env=e2e.E2E_DRYRUN=true
```

This will appear like a normal Sonobuoy run, except it will finish very quickly and all the tests that _would_ have run will be reported as having passed. If you check the logs you can see it was simply running in dry-run mode though.

## Conformance Tests

The **Conformance tests** refer to all the end-to-end tests that have the `[Conformance]` tag. These test the core functionality of Kubernetes that any user (regardless of configuration or cloud provider) should be able to rely on.

Conformance tests are treated slightly differently than other end-to-end tests in that there are extra processes and tooling to ensure that tests do not get marked as conformance tests without being properly approved. The reason for this is that conformance tests are the foundation of the CNCF’s [Certified Kubernetes program](https://github.com/cncf/k8s-conformance#certified-kubernetes).

#### How to run the whole set of conformance tests

```
$ sonobuoy run --mode certified-conformance
```

#### How many tests are there

~500 (varies by release)

#### When to run certified-conformance mode

- To obtain results that you submit to obtain Certified Kubernetes status
- As a relatively thorough and standard check of cluster creation or configuration, typically in CI.

#### Risks:

- Be aware that the time to run the full set of conformance tests can run into hours, especially depending on the size of the cluster.
- Some tests may be disruptive to existing workloads (more details below)

## Disruptive tests

**Disruptive tests** are all the tests that have the `[Disruptive]` tag. Most tests typically try to sandbox themselves: they create their own namespaces to run in and only use resources scoped to that namespace.  However, some Kubernetes features can’t be tested in this way, for instance, cluster level permissions or fault tolerance and disaster recovery. By testing these features, there may be side-effects that impact other workloads in the cluster.

As a result, it is not advisable to run these tests on clusters running production workloads.

There isn’t any particular reason to run this set of tests on their own, but if you did want to run them, you would provide the tag to Sonobuoy’s e2e-focus flag just like any other tagged set of tests (described above).

#### How many tests are there

~50

#### When to run them

- You would generally not want to just run the disruptive tests.
- Potentially if developing against Kubernetes itself or tooling related to failure recovery of clusters and workloads. Though there may be a more targeted set of tests relevant to your need (i.e. target disruptive tests that target a single feature).

#### Risks

As the name says, the tests are disruptive. Nodes may be taken down, API server made unresponsive, cluster permissions altered, or resources hogged.

## Non-Disruptive Conformance Tests

Now that we understand how tests are tagged and the meaning of conformance and disruptive tests, it should be clear that these non-disruptive conformance tests are just tests that have the `[Conformance]` tag and NOT the `[Disruptive]` tag.

This group of tests is significant because the value of the conformance tests extends beyond the Certified Kubernetes Program. Since it tests the core of Kubernetes functionality, most users consider the conformance tests to be a good baseline to test the health of existing clusters so they will routinely run them to validate their cluster creation or to help catch and diagnose issues quickly. If they are running these tests on existing clusters, they should avoid running the disruptive tests or else they compromise the health of the cluster in question.

Since this is the most common use case for Sonobuoy, **it is the default choice for Sonobuoy**.

#### How to run them

```
$ sonobuoy run
```

This is the same thing as providing the tags directly:

```
$ sonobuoy run --e2e-focus=’[Conformance]’ --e2e-skip=’[Disruptive]’
```

#### How many tests are there

~500 (only a few tests are both conformance and disruptive)

#### When to run them

- After you’ve set up a new cluster to check its health
- Periodically on running clusters to ensure their continued normal operation
- After making large changes to the cluster
- As a relatively thorough and standard check of cluster creation or configuration, in CI or production environments

## Summary

|  | Non-disruptive Conformance | Run by tag | Certified Conformance | Disruptive |
|:-:|:-:|:-:|:-:|:-:|
| How | sonobuoy run | sonobuoy run –e2e-focus=THESE –e2e-skip=NOT_THESE | sonobuoy run –mode=certified-conformance | sonobuoy run –e2e-focus=Disruptive |
| # of tests | ~500 | Varies | ~500 | ~50 |
| Notes | - Default for Sonobuoy - Safe way to test general cluster health/conformance | - Can target nearly any set of tests if you want to test a unique set of features | - Used when applying for CNCF Certified Kubernetes Program - May impact existing workloads | - Only used in combination with other tags to target disaster recovery features |

We hope this primer on the Kubernetes test suite cleared up some of the confusion you may have had regarding the Kubernetes tests. If you have more questions, don't hesitate to reach out. We love making your testing easier!

## Update: Sonobuoy's Quick mode

After posting this blog, it was pointed out that we weren't as clear as possible regarding the `--mode` flag and its options.

To clarify, the `--mode` flag is just a convenience wrapper for setting the `--e2e-focus` and `--e2e-skip` flags in the most common situations.

Discussed above, were non-destructive-conformance mode (Sonobuoy's default) and certified-conformance mode. However, we failed to mention **quick mode**. Quick mode runs a single, quick test on your cluster as a smoke test:

```
    sonobuoy run --mode=quick
```

This is ideal for when:

 - You've just created a cluster and want to make sure that it is functioning at all and can run pods as expected.
 - You have a long-lived cluster up and want to run periodic smoke tests to ensure the cluster is still operating as expected without all the time/load of full non-destructive-conformance run.

## Join the Sonobuoy community:

- Star/watch us on Github: [sonobuoy][sonobuoy] and [sonobuoy-plugins][sonobuoy-plugins]
- Get updates on Twitter (@projectsonobuoy)
- Chat with us on Slack ([#sonobuoy](https://kubernetes.slack.com/archives/C6L3G051C) on Kubernetes)
- Join the K8s-conformance [working group](https://github.com/cncf/k8s-conformance)

[kubernetesRepo]: https://github.com/kubernetes/kubernetes
[sonobuoy]: https://github.com/vmware-tanzu/sonobuoy
[sonobuoy-plugins]: https://github.com/vmware-tanzu/sonobuoy-plugins
