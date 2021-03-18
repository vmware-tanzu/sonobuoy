# Release

## Preparing a new release

1. Update the version defined in the code to the new version number.
   As of the time of writing, the version is defined in `pkg/buildinfo/version.go`.
1. Generate a new set of [versioned docs][gendocs] for this release.
1. If there an a Kubernetes release coming soon, do the following to ensure the upstream conformance script is
working appropriately:
  * Build the kind images for this new version.
    1. Checkout K8s locally at the tag in question
    1. Run `make check-kind-env` to ensure the repo/tag are correct
    1. Run `make kind_images`
    1. Run `make push_kind_images`
  * Update our CI build our kind cluster with the new image.
1. If the new release corresponds to a new Kubernetes release, the following steps must be performed:
  * Add the new list of E2E test images.
    For an example of the outcome of this process, see the [change corresponding to the Kubernetes v1.14 release](https://github.com/vmware-tanzu/sonobuoy/commit/68f15a260e60a288f91bc40347c817b382a3d45c).
      1. Within `pkg/image/`, copy the latest `v1.x.go` file to a file which corresponds to the new Kubernetes release number.
         For example, if the new Sonobuoy release corresponds to Kubernetes `v1.15`, copy the `v1.14.go` file to `v1.15.go`.

         ```
         cp pkg/image/v1.{14,15}.go
         ```
         This file will contain a function to return the list of test images for this new release.
      1. Update the name of the function in the newly created file.
        For example, if the file is for the v1.15 release, ensure the function name is `v1_15`.
      1. Replace the map of images within the previously mentioned function with the map of images for the new release.
        To do this, copy the equivalent map entries for the release from the Kubernetes repository.
        For an example, see the entries [from the v1.15.0 release](https://github.com/kubernetes/kubernetes/blob/v1.15.0/test/utils/image/manifest.go#L202-L252).
        Within the new function, remove any entries in the `config` map and replace with those copied from the Kubernetes repository.
        The entries from the Kubernetes repository use an `int` as the key in the map however in the Sonobuoy repository the keys are strings.
        Convert the new key names to strings.
      1. To make use of these new images, update the `GetImageConfigs` function within `pkg/image/manifest.go`.
        Add a new case to the minor version check which will be the minor version of the new Kubernetes release.
        In this new case, call the newly created function (e.g. `r.v1_15()`).
  * Add the new default image registry configuration.
    Once the images for the release have been added, update the function `GetDefaultImageRegistries` within `pkg/image/manifest.go` to return the default image registries for the new version.
    To do this, add a new case to the minor version check which will be the minor version of the new Kubernetes release.
    Within this case, return a new `RegistryList` object which includes only the registry keys used within the registry config for that version.
    Some registries are not applicable to include in this object as they are there to test specific image pull behavior such as pulling from a private or non-existent registry.
    This object should only include registries that can be successfully pulled from.
    The other registries are not used within the end-to-end tests.
    For an example, see the addition [from the v1.17.0 release](https://github.com/vmware-tanzu/sonobuoy/commit/93f63ef51e135dccf22407a0cdbf22f6c4a2cd26#diff-655c3323e53de3dff85eadd7592ca218R173-R188).
  * Update the minimum and maximum Kubernetes API versions that Sonobuoy supports.
    Edit `pkg/buildinfo/version.go` and update the `MinimumKubeVersion` to be 2 minor version below the new Kubernetes release version and update the `MaximumKubeVersion` to support future point releases.
    For example, for the Kubernetes 1.15.0 release, the `MinimumKubeVersion` would become `1.13.0` and the `MaximumKubeVersion` would become `1.15.99`.
1. Commit and open/merge a pull request with these changes.
1. Create an annotated tag for the commit once the changes are merged:

    ```
    git tag -a v0.x.y -m "Release v0.x.y"
    ```

    > NOTE: Tag the new tip of master, not the branch you just merged.

1. Push the tag to the [`github.com/vmware-tanzu/sonobuoy`](https://github.com/vmware-tanzu/sonobuoy/) repository.
   * To ensure that the tag is pushed to the correct repository, check which remote corresponds to that repository using the following command:

     ```
     git remote -v
     ```
     The output of this command should include at least two configured remotes, typically `origin`, which refers to your personal fork, and `upstream` which refers to the upstream Sonobuoy repository.
     For example:

     ```
     origin	git@github.com:<username>/sonobuoy.git (fetch)
     origin	git@github.com:<username>/sonobuoy.git (push)
     upstream	https://github.com/vmware-tanzu/sonobuoy (fetch)
     upstream	https://github.com/vmware-tanzu/sonobuoy (push)
     ```
     For the following steps, use the remote configured for the `vmware-tanzu/sonobuoy` repository.
     The following instructions will use `upstream`.
   * Push the tag with the following command.
     > NOTE: This will push all tags.

     ```
     git push upstream --tags
     ```
     To push just one tag, use the following command format (replacing `v0.x.y` with the tag created in the previous step):

     ```
     git push upstream refs/tags/v0.x.y
     ```
     If there is a problem and you need to remove the tag, run the following commands:

     ```
     git tag -d v0.x.y
     git push upstream :refs/tags/v0.x.y
     ```
     > NOTE: The `:` preceding the tag ref is necessary to delete the tag from the remote repository.
     > Git refspecs have the format `<+><src>:<dst>`.
     > By pushing an empty `src` to the remote `dst`, it makes the destination ref empty, effectively deleting it.
     > For more details, see the [`git push` documentation](https://git-scm.com/docs/git-push) or [this concise explanation on Stack Overflow](https://stackoverflow.com/a/7303710).


## Validation
1. Open a browser tab and go to: https://circleci.com/gh/vmware-tanzu/sonobuoy and verify go releaser for tag v0.x.y completes successfully.
1. Upon successful completion of build job above, check the [releases tab of Sonobuoy](https://github.com/vmware-tanzu/sonobuoy/releases) and verify the artifacts and changelog were published correctly.
1. Run the following command to make sure the image was pushed correctly to [Docker Hub][dockerhub]:

   ```
   docker run -it sonobuoy/sonobuoy:v0.x.y /sonobuoy version
   ```
   The `Sonobuoy Version` in the output should match the release tag above.
1. Go to the [GitHub release page](https://github.com/vmware-tanzu/sonobuoy/releases) and download the release binaries and make sure the version matches the expected values.
2. Run a [Kind](https://github.com/kubernetes-sigs/kind) cluster locally and ensure that you can run `sonobuoy run --mode quick`.
   If this release corresponds to a new Kubernetes release as well, ensure:

    * you're using the correct Kubernetes context by checking the output from:

      ```
      kubectl config current-context
      ```

      and verifying that it is set to the context for the Kind cluster just created (`kind-kind` or `kind-<custom_cluster_name>`)
    * you're testing with the new Kind images by checking the output from:

      ```
      kubectl version --short
      ```

      and verifying that the server version matches the intended Kubernetes version.
    * you can run `sonobuoy images` and get a list of test images as expected
2. Update the release notes if desired on GitHub by editing the newly created release.

### Generating a new set of versioned docs
The changes for this can almost all be completed by running the command:

```
./scripts/update_docs.sh v0.x.y
```

This will copy the current master docs into the version given and update
a few of the links in the README to be correct. It will also update
the website config to add the new version and consider it the newest
version of the docs.

### Notes
1. Before releasing, ensure all parties are available to resolve any issues that come up. If not, just bump the release.

[gendocs]: #generating-a-new-set-of-versioned-docs
[dockerhub]: https://cloud.docker.com/u/sonobuoy/repository/docker/sonobuoy/sonobuoy/tags

2. If you are building a Windows release you must currently build/push the Windows image outside of CI and push the manifest to also include it. To do this you must:

 - Have built the Windows binaries (can be done on a Linux box and should be the default now)
 - Have a Windows machine available for the build. The steps below will assume a `docker context` which is a Windows machine.
 - (Recommended) Build the sample Windows plugin (in our examples directory) to test the image
 - (Recommended) Have a cluster with Windows available for testing

```
docker context use default
make build/windows/amd64/sonobuoy.exe
docker context use 2019-box
make windows_containers
PUSH_WINDOWS=true make push

```