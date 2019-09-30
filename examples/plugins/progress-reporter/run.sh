#!/bin/bash

set -x

# This is the entrypoint for the image and meant to wrap the
# logic of gathering/reporting results to the Sonobuoy worker.

results_dir="${RESULTS_DIR:-/tmp/results}"
progress_port="${SONOBUOY_PROGRESS_PORT:-8099}"
total_updates=60
sleep_seconds=1

# saveResults prepares the results for handoff to the Sonobuoy worker.
# See: https://github.com/vmware-tanzu/sonobuoy/blob/master/docs/plugins.md
saveResults() {
    # Signal to the worker that we are done and where to find the results.
    echo "hello world" > ${results_dir}/myresults
    printf ${results_dir}/myresults > ${results_dir}/done
}

# This is the main point of this example. All you have to do to update the plugin progress
# is to post to this URL with a valid ProgressUpdate object.
updateProgress () {
    curl -X POST http://localhost:${progress_port}/progress -d "{\"msg\":\"$1\",\"completed\":$2,\"total\":$3}"
}

# Ensure that we tell the Sonobuoy worker we are done regardless of results.
trap saveResults EXIT

# Iterate through each argument. Grab each by location and shift
# to preserve spacing (using "$@" will lose that information).
# Iterate through each argument. Grab each by location and shift
# to preserve spacing (using "$@" will lose that information).
for (( i=1; i<=$total_updates; i++ ))
do
    updateProgress "This is a message for update $i" $i $total_updates
    sleep $sleep_seconds
done
