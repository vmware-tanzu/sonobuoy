# Release

## Preparing a new release

1. Update the version defined in the code to the new version number.
   As of the time of writing, the version is defined in `pkg/buildinfo/version.go`.
1. If the new release corresponds to a new Kubernetes release, the following steps must be performed:
   * Add the new list of E2E test images.
     For an example of the outcome of this process, see the [change corresponding to the Kubernetes v1.14 release](https://github.com/heptio/sonobuoy/commit/68f15a260e60a288f91bc40347c817b382a3d45c).
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
   * Update the minimum and maximum Kubernetes API versions that Sonobuoy supports.
     Edit `pkg/buildinfo/version.go` and update the `MinimumKubeVersion` to be 2 minor version below the new Kubernetes release version and update the `MaximumKubeVersion` to support future point releases.
     For example, for the Kubernetes 1.15.0 release, the `MinimumKubeVersion` would become `1.13.0` and the `MaximumKubeVersion` would become `1.15.99`.
1. Commit and open/merge a pull request with these changes.
1. Create an annotated tag for the commit once the changes are merged:
    ```
    git tag -a v0.x.y -m "Release v0.x.y"
    ```
1. Push the tag to the [`github.com/heptio/sonobuoy`](https://github.com/heptio/sonobuoy/) repository.
   * To ensure that the tag is pushed to the correct repository, check which remote corresponds to that repository using the following command:
     ```
     git remote -v
     ```
     The output of this command should include at least two configured remotes, typically `origin`, which refers to your personal fork, and `upstream` which refers to the upstream Sonobuoy repository.
     For example:
     ```
     origin	git@github.com:<username>/sonobuoy.git (fetch)
     origin	git@github.com:<username>/sonobuoy.git (push)
     upstream	https://github.com/heptio/sonobuoy (fetch)
     upstream	https://github.com/heptio/sonobuoy (push)
     ```
     For the following steps, use the remote configured for the `heptio/sonobuoy` repository.
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

1. Open a browser tab and go to: https://travis-ci.org/heptio/sonobuoy/builds and verify go releaser for tag v0.x.y completes successfully.
1. Upon successful completion of build job above, check the [releases tab of Sonobuoy](https://github.com/heptio/sonobuoy/releases) and verify the artifacts and changelog were published correctly.
1. Run the [Jenkins job](https://jenkins.hepti.center/job/build-image-heptio-sonobuoy-release/build?delay=0sec) for pushing release images, specifying the release tag `v0.x.y` and confirm that the images get pushed correctly.
2. Update the release notes if desired on GitHub by editing the newly created release.

## Validation
1. Run the following command to make sure the image was pushed correctly:
   ```
   docker run -it gcr.io/heptio-images/sonobuoy:v0.x.y /sonobuoy version
   ```
   The `Sonobuoy Version` in the output should match the release tag above.
1. Go to the [GitHub release page](https://github.com/heptio/sonobuoy/releases) and download the release binaries and make sure the version matches the expected values.
2. Run a [Kind](https://github.com/kubernetes-sigs/kind) cluster locally and ensure that you can run `sonobuoy run --mode quick`.
   If this release corresponds to a new Kubernetes release as well, ensure:
    - you're testing with the new Kind images by checking the output from:
      ```
      export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
      kubectl version --short
      ```
      and verify that the server version matches the intended Kubernetes version.
    - you can run `sonobuoy images` and get a list of test images as expected

### Follow up
Following the release when the new tag is made, the documentation will need to be updated to include the new version.

### Notes
1. Before releasing, ensure all parties are available to resolve any issues that come up. If not, just bump the release.
