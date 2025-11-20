# Issue Regarding Certified-Conformance Mode

### Versions of Sonobuoy Impacted
 - v0.53.0
 - v0.53.1

### Description

When running `sonobuoy run --mode=certified-conformance` the `E2E_SKIP` value is not properly cleared, leading to `disruptive` tests being skipped. In certified-conformance mode, all tests must be run to be valid for submission to the CNCF so this bug would invalidate your results.

### Work-around #1

You can manually work-around this issue by adding an extra flag at the end of your command:
```
sonobuoy run --mode=certified-conformance --plugin-env e2e.E2E_SKIP
```
This will set the focus value to conformance as expected and then remove the E2E_SKIP value.

### Work-around #2

Use a patched version of Sonobuoy. After this bug was reported and patched, we released v0.53.2.

### Original issue

Thanks to [BobyMCbobs](https://github.com/BobyMCbobs) for reporting the original issue: https://github.com/vmware-tanzu/sonobuoy/issues/1388