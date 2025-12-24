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

## Virtual Machine Runners

The external GitHub Action runners use VM images built on Oracle Cloud Infrastructure (OCI). These images are created using the `gha-runner-vm` tool located in `ci/gha-runner-vm/`.

### Available VM Runner Labels

CNCF Project Maintainers can use the following runner labels in their GitHub Actions workflows:

| Runner Label | Architecture | CPU | Memory | Nested Virtualization | Launch Mode |
|--------------|--------------|-----|--------|----------------------|-------------|
| `oracle-4cpu-16gb-x86-64` | x86_64/amd64 | 4 CPU | 16 GB | Not supported* | PARAVIRTUALIZED |
| `oracle-8cpu-32gb-x86-64` | x86_64/amd64 | 8 CPU | 32 GB | Not supported* | PARAVIRTUALIZED |
| `oracle-16cpu-64gb-x86-64` | x86_64/amd64 | 16 CPU | 64 GB | Not supported* | PARAVIRTUALIZED |
| `oracle-24cpu-384gb-x86-64` | x86_64/amd64 | 24 CPU | 384 GB | Not supported* | PARAVIRTUALIZED |
| `oracle-2cpu-8gb-arm64` | arm64/aarch64 | 2 CPU | 8 GB | Not supported | PARAVIRTUALIZED |
| `oracle-16cpu-64gb-arm64` | arm64/aarch64 | 16 CPU | 64 GB | Not supported | PARAVIRTUALIZED |
| `oracle-32cpu-128gb-arm64` | arm64/aarch64 | 32 CPU | 128 GB | Not supported | PARAVIRTUALIZED |

*Note: While AMD x86_64 instances may support nested virtualization on bare metal configurations, the standard paravirtualized VM runners do not support nested virtualization. ARM instances do not support nested virtualization.

### VM Capabilities

All VM runners are configured with the following capabilities:

**Compute:**
- Firmware: UEFI_64 (default), BIOS supported
- Launch Mode: PARAVIRTUALIZED
- Secure Boot: Disabled
- AMD SEV: Disabled

**Network:**
- Attachment Type: PARAVIRTUALIZED
- IPv6-Only: Not supported

**Storage:**
- Boot/Data Volume Type: PARAVIRTUALIZED (default), ISCSI supported
- Consistent Volume Naming: Enabled
- Paravirtualization Encryption in Transit: Enabled
- Storage Attachment Version: 2

To view detailed VM capabilities, use:
```bash
cd ci/gha-runner-vm
go run main.go list-capabilities
```
