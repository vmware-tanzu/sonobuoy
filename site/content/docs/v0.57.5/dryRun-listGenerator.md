# Plugins used to help create test lists

This process has been dramatically simplified. All that is necessary is that you create a new Git branch/PR
and run `./scripts/gather_e2e_data.sh` and commit the changes.

You can look through the script and the supporting files for a more in depth explanation of the process but
it just boils down to finding the proper versions to check, running them in DRY_RUN mode, and extracting
the necessary data.

**DEBUG**
If running this against a local kind cluster the server could run out of space for more conformance images (e.g. "no space left on device")
 - Clear docker cache with `docker image prune -a`
 - Add more disc space to docker (60GB to 300GB)