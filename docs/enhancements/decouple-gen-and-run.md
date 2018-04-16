# Customize Sonobuoy Manifest

## Summary

In order to cleanly integrate the sonobuoy client with sonobuoy scanner we need
a way to customize the objects produced by sonobuoy gen. There are a few
approaches outlined below ordered by difficulty and benefit.

## Objectives

### Goals

* Allow clients of Sonobuoy to customize the generated manifest and then run it.

### Non Goals

* Make the customizations built in to sonobuoy via config or command line flags.

## Proposal

Preferred solution is having Gen return a slice `kubernetes.Objects` and Run
accept a slice of `kubernetes.Object`. This solutino gives the cleanest
interface to clients wishing to customize objects before submitting them to the
apiserver. This also provides a nice escape hatch for things like
image-pull-secrets that wasn't implemented in v0.11. Advanced users could
generate the objects, modify, and run.

### Do Nothing

Technically it is possible today to customize the objects that sonobuouy Gen
returns. However it is a lengthy process and not a particularly nice one to
implement. The basic solution goes.

```go
// generate the manifest bytes
// split manifest bytes on `---`
// For each yaml object found convert it into a kubernetes.Object
// for each Object figure out what Kind it is
//     customize it if you like
// convert back to YAML/json bytes
// use a dynamic client to run the bytes
```

#### Things of note

* In this scenario Run is unusable as it calls Gen as a dependency.

### Decouple Gen & Run

```
Gen(config) => []bytes
Run([]bytes)
```

If Run were to take YAML bytes and submit to the cluster based (basically
reimplement kubectl apply -f) then we could call run instead of having to have
the client configure some dynamic client.

### Change the Output of Gen

```
Gen(config) => []kubernetes.Object
Run([]bytes)
```

In addition to the decoupling of Gen and Run: if the Gen command were to return
[]kubernetes.Object then a client would have a much easier time customizing the
objects. However the client would still be responsible for the serialization of
the list of objects it is trying to create.

With this change it would make sense to convert the YAML template to go structs
representing our manifest. It is more verbose but also more precise.

### Forego bytes and use kubernetes objects

```
Gen(config) => []kubernetes.Object
Run([]kubernetes.Object)
```

Working with kubernetes objects instead of bytes would give a client the easiest
way to customize the sonobuoy manifest. Gen returns a list of Objects and Run
accepts a list of Objects.

### User Stories

#### Sonobuoy Scanner developer (general interface developer)

As a third party developer I have some post-processing tasks I'd like to
accomplish in the aggregator server's pod. For example, it would be cool if I
could add the [namespace deleter][1] pod to run after the aggregation server has
run. I might also add in another container to ship the results tarball
somewhere.

## Unresolved Questions

None at this time

[1]: https://github.com/heptiolabs/namespace-deleter
