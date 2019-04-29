# Command Runner

This example creates an Sonobuoy plugin which runs each of its arguments (via /bin/sh) and saves the output in a results file (the output for the Nth command is output to a file called "outN").

Kubectl is added to the image so that it can grab information from/about the cluster.

All of the results files are added into a tar file and then the 'done' file is written, telling the Sonobuoy worker where to find the output.

## Example

```
$ docker build . -t user/easy-sonobuoy-cmds:v0.1
$ docker push user/easy-sonobuoy-cmds:v0.1

$ sonobuoy gen plugin \
--name=hello-world \
--image user/easy-sonobuoy-cmds:v0.1 \
--arg="echo hello world" \
--arg="kubectl cluster-info" > hello-world.yaml

$ sonobuoy run --plugin hello-world.yaml
```

You can obtain all the Sonobuoy results locally by running:
```
outfile=$(sonobuoy retrieve) && \
mkdir results && tar -xf $outfile -C results
```

Print our specific results with:
```
cat results/plugins/hello-world/results/out*
```


