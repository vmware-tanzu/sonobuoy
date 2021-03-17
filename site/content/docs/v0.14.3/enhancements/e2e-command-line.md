# Sub Command for E2E Results Viewing

## Table of Contents

* [Summary](#summary)
* [Objectives](#objectives)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [User Stories](#user-stories)
* [Unresolved Questions](#unresolved-questions)

## Summary

End to end (e2e) tests in the Kubernetes ecosystem are built on top of ginkgo
and output junit XML. XML is not designed for human readability and thus it is
very hard to grok. Users can use scanner but if they have results there is no
way for a user to upload those results to scanner to get the nice output.

## Objectives

Make the sonobuoy e2e output be more useful to users.

### Goals

- Output a list of e2e tests (filterable by passed/failed).
- Generate YAML that can rerun failed tests.

### Non-Goals

- Replace scanner.heptio.com
- Updating scanner features
- Re-implement gen
- Make a CLI per plugin. This is a special case that generates a lot of
  questions and we think this will improve the user experience by a lot.

## Proposal

Add a subcommand to `sonobuoy` that has arguments/subcommands that satisfies the
goals above:

`sonobuoy e2e <path/to/results_archive>` will show a list of failed tests from
the sonobuoy archive file.

`sonobuoy e2e --show=passed` will only display passed tests.

`sonobuoy e2e --rerun-failed <path/to/archive>` will figure out the failed
tests, build a dumb regex and pass that into Gen and pass that into run.

## Prerequisite work

* Gen will have to be able to generate the specific type of YAML we'd need for a
  rerun.
* Run will have to be able to take a generic manifest and execute it. The
  generation step will move into the cmd/run.go file.

### User Stories

#### Operator

- An operator is playing with their cluster. Sonobuoy is telling them they have
  failed some tests so they tweak some settings and want to try to run the
  failed tests to see if they pass.

Currently they would have to build their own YAML with a custom FOCUS variable.
This command line tool aims to simplify this workflow by producing the YAML they
need to rerun failed tests.

#### Firewalled user

- A user wants to have a nice experience looking at e2e test results but their
  cluster cannot talk to an external network (scanner). Today they would have to
  run sonobuoy and collect the xml and use a tool like xunit-viewer.

This proposal aims to simplify their workflow by allowing them to use the
sonobuoy CLI to explore their results.

## Unresolved Questions



