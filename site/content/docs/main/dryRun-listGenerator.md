# Plugins used to help create test lists

To get the lists of tests for a version, we need to first gather the list of tests for each of those versions.

There are too many releases to get _all_ k8s releases so I used the following as a guide:

```bash
git tag -l --sort=-creatordate|grep -v "alpha\|beta\|rc" |head -n75
```

This gets the latest 75 releases that aren't alpha/beta/rc releases. We will use this list
to create a Sonobuoy plugin for each release.

First I generate a list of the versions:
```bash
# From my kubernetes/kubernetes repo directory
rm ./tmpversions.txt
git fetch --all --tags
git tag -l --sort=-creatordate |
  grep -v "alpha\|beta\|rc" |
  head -n75|sort|xargs -t -I % sh -c \
  'echo % >> ~/go/src/github.com/vmware-tanzu/sonobuoy/tmpversions.txt'
```

After trial and error I realized we need to trim that list a bit since
older versions will not have E2E_DRYRUN at all. Manually removing values from the versions list
before v1.14.0 (if there are any).

Since we already have some versions data, we only need to find the new ones. To see the new versions:
```
ls cmd/sonobuoy/app/e2e/testLists|cut -f 1-3 -d '.' > existingversions.txt
comm -13 existingversions.txt tmpversions.txt 
```

You should expect to see the v0.0.0 as a difference (a test value) but then modify the tmpVersions.txt to only include the new versions.

Then, using xargs and sonobuoy I generate the plugin for the releases of k8s. I need to modify the default e2e plugin in two ways:
 - make the name unique
 - remove E2E_EXTRA_ARGS since some of the older versions dont have the progress URL flag.

```bash
# From the sonobuoy directory
rm ./tmpplugins/p*
cat tmpversions.txt|xargs -t -I % sh -c \
  'sonobuoy gen plugin e2e --plugin-env=e2e.E2E_EXTRA_ARGS= --plugin-env=e2e.E2E_DRYRUN=true --kubernetes-version=% | sed "s/plugin-name: e2e/plugin-name: e2e%/" > ./tmpplugins/p%.yaml'
```

Now, when I run sonobuoy I can run with each of those plugins, get the results, and gzip them to save space:

```bash
# From the root of this project
sonobuoy run -p ./tmpplugins --wait
sonobuoy retrieve -f output.tar.gz
cat tmpversions.txt | xargs -t -I % sh -c \
  "sonobuoy results output.tar.gz -p e2e% --mode=detailed | jq .name -r | sort > ./cmd/sonobuoy/app/e2e/testLists/%"  
cd ./cmd/sonobuoy/app/e2e/testLists
gzip *
# Any older ones archives will just need you to say not to overwrite.
```

Ensure this works by building locally and running with one of the new versions:

```bash
go install && sonobuoy e2e --version <new version> -i offline
```

**DEBUG**
Server could run out of space for more conformance images (e.g. "no space left on device")
 - Clear docker cache with `docker image prune -a`
 - Add more disc space to docker (60GB to 300GB)