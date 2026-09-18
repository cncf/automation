# External GitHub Action Runners

Maintained by the CNCF Projects Team, cncf/automation/ci orchestrates external
GitHub Action Runners to run on resources donated to the CNCF.

CNCF Projects can make use of the External GitHub Action Runners that are
defined here for CI jobs.

## Custom runners

We provide runners for both x86_64/amd64 and arm64 architectures.

In a Github Action workflow, you can define multiple jobs and for each of those
jobs you choose a runner.

Every label below matches a runner scale set defined in this repository. Please
check your label against these lists. A `runs-on` label with no matching scale
set does not fail: the job stays queued until GitHub cancels it 24 hours later.

### Container runners

These run your job in a container. They are defined in
[`ci/cluster/oci/runners`](./cluster/oci/runners) and
[`ci/cluster/lke-gha-iad2/manifests/container-runners`](./cluster/lke-gha-iad2/manifests/container-runners).

For an amd64/x86_64 runner you can choose from:

- `runs-on: oracle-2cpu-8gb-x86-64`
- `runs-on: oracle-8cpu-32gb-x86-64`
- `runs-on: oracle-16cpu-64gb-x86-64`
- `runs-on: cncf-c-ubuntu-2-8-x86`

### VM runners

These give your job a whole virtual machine, so it can use nested
virtualisation and privileged workloads. They are defined in
[`ci/cluster/oci/vm-runners`](./cluster/oci/vm-runners) and in the
`manifests/vm-runners` directory of each cluster.

For an amd64/x86_64 runner you can choose from:

- `runs-on: cncf-ubuntu-2-8-x86`
- `runs-on: cncf-ubuntu-4-16-x86`
- `runs-on: cncf-ubuntu-8-32-x86`
- `runs-on: cncf-ubuntu-16-64-x86`
- `runs-on: cncf-ubuntu-24-96-x86`
- `runs-on: cncf-ubuntu-32-128-x86`

For an arm64 runner choose one of:

- `runs-on: cncf-ubuntu-2-8-arm`
- `runs-on: cncf-ubuntu-4-16-arm`
- `runs-on: cncf-ubuntu-8-32-arm`
- `runs-on: cncf-ubuntu-16-64-arm`
- `runs-on: cncf-ubuntu-24-96-arm`
- `runs-on: cncf-ubuntu-32-128-arm`

For a bare-metal arm64 runner:

- `runs-on: cncf-ubuntu-bm-arm`

Please reach out to the CNCF team on the `#cncf-ci-infra` channel for usage.

For a GPU runner choose one of:

- `runs-on: oracle-vm-gpu-a10-1`
- `runs-on: oracle-vm-gpu-a10-2`

Please reach out to the CNCF team on the `#cncf-ci-infra` channel for usage.

## Runner definition

The container runners are built from this [Dockerfile](./gha-runner-image/Dockerfile)

You can review the Dockerfile to see what tools have been added to the runner image.
