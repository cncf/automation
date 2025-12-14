# External GitHub Action Runners

Maintained by the CNCF Projects Team, cncf/automation/ci orchestrates external
GitHub Action Runners to run on resources donated to the CNCF.

CNCF Projects can make use of the External GitHub Action Runners that are
defined here for CI jobs.

## Custom runners

We have created two custom runners that allow us to run any containerizable CI
job on both x86_64/amd64 and arm64 Architectures.

In a Github Action workflow, you can define multiple jobs and for each of those
jobs you choose a runner.

We have defined two 16CPU/64GB runners that CNCF Projects can use by adding
either of the following statements to your job definitions:

**For an amd64/x86_64 runner you can choose from:**

`runs-on: oracle-4cpu-16gb-x86-64`
`runs-on: oracle-8cpu-32gb-x86-64`
`runs-on: oracle-16cpu-64gb-x86-64`
`runs-on: oracle-24cpu-384gb-x86-64`

**For an arm64 runner choose one of:**

`runs-on: oracle-2cpu-8gb-arm64`
`runs-on: oracle-16cpu-64gb-arm64`
`runs-on: oracle-32cpu-128gb-arm64`

## Runner definition

The above runners are defined in this [Dockerfile](./gha-runner-image/Dockerfile)

You can review the Dockerfile to see what tools have been added to the runner image.

## E2E Testing

All runners are automatically tested through comprehensive E2E tests that validate:
- Basic functionality and system information
- Docker container capabilities
- Development tool availability
- Kubernetes functionality (for VM runners)
- Resource allocation and limits

Tests are triggered automatically when:
- Runner image changes are made
- Upstream dependencies are updated
- Weekly scheduled validation runs
- Manual testing requests

For detailed information about the E2E testing system, see [README-E2E-TESTING.md](./README-E2E-TESTING.md).
