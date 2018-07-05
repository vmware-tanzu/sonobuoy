# Error Logging Improvement

## Table of Contents

* [Summary](#summary)
* [Objectives](#objectives)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [User Stories](#user-stories)
* [Unresolved Questions](#unresolved-questions)

# Summary

The log produced by Sonobuoy is used to be able to diagnose clusters and
troubleshoot users. This makes Sonobuoy's error logging a requirement, not a
nice-to-have. A requirement must have automated testing or it's going to break.
Error logging in Sonobuoy, at the time of writing this document, is difficult if
not impossible to unit test.

# Objectives

The main objective is to unify how we write to the error log and treat the error
log as a proper dependency.

## Goals

* Write unit tests for functions that call the error log.
* Treat the error log as a dependency, not a global.

## Non-Goals

* Treat info/debug logging as a proper dependency. This would be nice but is not
  the focus.
* Get 100% unit test coverage on error log call-sites. Again, nice but probably
  not worth the effort.

# Proposal

## Features of the error log

1. The error log should always (some exceptions are ok, like power disruption) be
   complete or end with a line indicating why it's not complete.
1. The error log should have uniform formatting.

## User Stories

### Consumer

As a consumer of sonobuoy I always want to know what happened during a run. If
the run failed, I want to know why it failed so I can diagnose and fix.

### Scanner user

As a scanner user, my run never completed or I got the "it broke" message. I asked
for help in Slack and was asked to provide the sonobuoy error log. It must be complete
so that the troubleshooters can accurately assess the run.

### Sonobuoy maintainer

As a Sonobuoy maintainer, the logs from the sonobuoy pod are the first thing we ask for.
They must be complete for us to understand the run.
