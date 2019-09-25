# Results files

This directory contains the static files used for results (instead of actually running tests which may be flaky/slow).

Different invocations of the test image will copy these files into the appropriate places and (optionally) tar them up for reporting to Sonobuoy.

These files (currently in the `resources` directory) are expected to be copied into the test image at the absolute directory `/resources`.