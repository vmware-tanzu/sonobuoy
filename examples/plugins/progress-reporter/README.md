# Progress Reporter

This example creates an Sonobuoy plugin which reports its progress to the Sonobuoy worker (which runs as a sidecar to the plugin). It produces a tiny, trivial result which can be ignored (a "hello world" file).

To communicate with the Worker, it uses localhost and a port which it gets from the environment: `SONOBUOY_PROGRESS_PORT`.

Check the documentation for details on the progress update object itself.

## Example

```
$ export USER=<your public registry>
$ docker build . -t $USER/progress-reporter:v0.1
$ docker push $USER/progress-reporter:v0.1

$ sonobuoy gen plugin \
--name=progress \
--image $USER/progress-reporter:v0.1 > progress.yaml

$ sonobuoy run --plugin progress.yaml
```

While the plugin is running (1 minute) you can use Sonobuoy to check its progress:

```
sonobuoy status --json
```

Or with `jq` you can try something like:

```
sonobuoy status --json|jq '.plugins|.[].Progress'
```


