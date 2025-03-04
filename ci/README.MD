# External GitHub Action Runners

Maintained by the CNCF Projects Team, cncf/automation/ci orchestrates external
GitHub Action Runners to run on resources donated to the CNCF.

CNCF Projects can make use of the External GitHub Action Runners that are
defined here for CI jobs.

## Custom runners

We have created two custom runners that allow us to run any containerizable CI
job on both x86_64/amd64 and arm63 Architechtures.

In a Github Action workflow, you can define multiple jobs and for each of those
jobs you choose a runner. 

We have defined two 16CPU/64GB runners that can used by adding either of the
following statements to your job definitions:

**For an amd64/x86_64 runner use:**

`runs-on: oracle-16cpu-64gb-x86-64`

**For an arm64 runner**

`runs-on: oracle-16cpu-64gb-arm64`

## Runner definition

The above runners are defined in this repo <./gha-runner-image/Dockerfile>

You can review the Dockerfile to see what tools have been added to the runner image.
