# Release

## Steps to cut a release

1. Bump the version defined in the code. As of the time of writing it is in
   `pkg/buildinfo/version.go`.
1. Commit and open a pull request.
1. Create a tag after that pull request gets squashed onto master. `git tag -a v0.13.0`.
1. Push the tag with `git push --tags` (note this will push all tags). To push
   just one tag do something like: `git push <remote> refs/tags/v0.13.0` where
   `<remote>` refers to github.com/heptio/sonobuoy (this might be something like
   `upstream` or `origin`). If you are unsure, use the first option.
1. Open a browser tab and go to: https://travis-ci.org/heptio/sonobuoy/builds 
    and verify go releaser for tag v0.13.0 completes successfully

