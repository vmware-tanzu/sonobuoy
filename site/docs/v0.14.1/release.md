# Release

## Steps to cut a release

1. Bump the version defined in the code. As of the time of writing it is in
   `pkg/buildinfo/version.go`.
1. Commit and open/merge a pull request.
1. Create an annotated tag: `git tag -a v0.x.y -m "Release tag"`.
1. Push the tag with `git push --tags` (note this will push all tags). To push
   just one tag do something like: `git push <remote> refs/tags/v0.13.0` where
   `<remote>` refers to github.com/vmware-tanzu/sonobuoy (this might be something like
   `upstream` or `origin`). If you are unsure, use the first option.
    - if there is a problem and you need to remove the tag use `git tag -d v0.x.y` and `git push origin refs/tags/v0.x.y` (assuming origin refers to github.com/vmware-tanzu/sonobuoy)
1. Open a browser tab and go to: https://travis-ci.org/vmware-tanzu/sonobuoy/builds 
    and verify go releaser for tag v0.x.y completes successfully
1. Upon successful completion of build job above, check the releases tab of
   https://github.com/vmware-tanzu/sonobuoy and verify the artifacts and changelog were published correctly.
1. Run the Jenkins job for pushing release images and manually run for the tag `v0.x.y` and confirm images get pushed correctly.
2. Update the release notes as desired on github.

## Validation
1. Run the following command to make sure the image was pushed
   correctly: `docker run -it gcr.io/heptio-images/sonobuoy:v0.x.y /sonobuoy version`. The output should
   match the release tag above.  
1. Go the the github release page (https://github.com/vmware-tanzu/sonobuoy/releases) and download the release binaries and make sure the version matches the expected values.
2. Run a Kind cluster locally and ensure that you can run `sonobuoy run --mode quick`. If this release corresponds to a new Kubernetes release as well, ensure:
    - you're testing with the new Kind images
    - you can run `sonobuoy images` and get a list of test images as expected

### Notes
1. Before releasing, ensure all parties are available to resolve any issues that come up. If not, just bump the release.