#!/bin/sh

# Running this file executes `sonobuoy gen plugin` numerous times
# to reproduce all the YAML for the plugin definitions. It assumes
# you have the yq tool and sonobuoy in your PATH.

# Should generate/test more cases; this is just a starting point.

sonobuoy gen plugin \
--name=job-junit-passing-singlefile \
--image=sonobuoy/testimage:v0.1 \
--cmd="/testImage" \
--arg="single-file" \
--arg="/resources/junit-passing-tests.xml" \
--format="junit" > job-junit-passing-singlefile.yaml

sonobuoy gen plugin \
--name=job-raw-passing-singlefile \
--image=sonobuoy/testimage:v0.1 \
--cmd="/testImage" \
--arg="single-file" \
--arg="/resources/hello-world.txt" \
--format="raw" > job-raw-singlefile.yaml

sonobuoy gen plugin \
--name=ds-junit-passing-tar \
--image=sonobuoy/testimage:v0.1 \
--type=daemonset \
--cmd="/testImage" \
--arg="tar-file" \
--arg="/resources/hello-world.txt" \
--arg="/resources/junit-multi-suite-single-failure.xml" \
--format="junit" > ds-junit-passing-tar.yaml

sonobuoy gen plugin \
--name=ds-raw-passing-tar \
--image=sonobuoy/testimage:v0.1 \
--type=daemonset \
--cmd="/testImage" \
--arg="tar-file" \
--arg="/resources/hello-world.txt" \
--arg="/resources/junit-multi-suite-single-failure.xml" \
--format="raw" > ds-raw-passing-tar.yaml

sonobuoy gen plugin \
--name=job-junit-singlefile-configmap \
--image=sonobuoy/testimage:v0.1 \
--cmd="/testImage" \
--arg="single-file" \
--arg="/tmp/sonobuoy/config/junit-via-configmap.xml" \
--configmap="../resources/junit-via-configmap.xml" \
--format="junit" > job-junit-singlefile-configmap.yaml
