# VM Runners

GitHub Actions self-hosted VM-based runners for the OCI amd64 cluster, provisioned as OCI compute instances through Actions Runner Controller (ARC).

## Overview

Unlike the container-based runners in [`../runners/`](../runners/), VM runners launch dedicated OCI compute instances for each job using the `gha-cloudrunner` image. This provides full VM isolation, making them suitable for workloads that require bare-metal-like environments, nested virtualization, or GPU access.

## Available Runner Sizes

### Standard x86-64 (AMD) Runners

| Directory | Runner Label | Shape | OCPUs | Memory | Max Runners |
|-----------|-------------|-------|-------|--------|-------------|
| `2cpu-8gb/` | `oracle-vm-2cpu-8gb-x86-64` | `VM.Standard.E6.Flex` | 2 | 8 GB | 100 |
| `4cpu-16gb/` | `oracle-vm-4cpu-16gb-x86-64` | `VM.Standard.E6.Flex` | 4 | 16 GB | 100 |
| `8cpu-32gb/` | `oracle-vm-8cpu-32gb-x86-64` | `VM.Standard.E6.Flex` | 8 | 32 GB | 100 |
| `16cpu-64gb/` | `oracle-vm-16cpu-64gb-x86-64` | `VM.Standard.E6.Flex` | 16 | 64 GB | 100 |
| `24cpu-96gb/` | `oracle-vm-24cpu-96gb-x86-64` | `VM.Standard.E6.Flex` | 24 | 96 GB | 100 |
| `32cpu-128gb/` | `oracle-vm-32cpu-128gb-x86-64` | `VM.Standard.E6.Flex` | 32 | 128 GB | 100 |

### ARM64 Runners

| Directory | Runner Label | Shape | OCPUs | Memory | Max Runners |
|-----------|-------------|-------|-------|--------|-------------|
| `2cpu-8gb-arm64/` | `oracle-vm-2cpu-8gb-arm64` | `VM.Standard.A1.Flex` | 2 | 8 GB | 100 |
| `4cpu-16gb-arm64/` | `oracle-vm-4cpu-16gb-arm64` | `VM.Standard.A1.Flex` | 4 | 16 GB | 100 |
| `8cpu-32gb-arm64/` | `oracle-vm-8cpu-32gb-arm64` | `VM.Standard.A1.Flex` | 8 | 32 GB | 100 |
| `16cpu-64gb-arm64/` | `oracle-vm-16cpu-64gb-arm64` | `VM.Standard.A1.Flex` | 16 | 64 GB | 100 |
| `24cpu-96gb-arm64/` | `oracle-vm-24cpu-96gb-arm64` | `VM.Standard.A1.Flex` | 24 | 96 GB | 100 |
| `32cpu-128gb-arm64/` | `oracle-vm-32cpu-128gb-arm64` | `VM.Standard.A1.Flex` | 32 | 128 GB | 100 |

### GPU Runners

| Directory | Runner Label | Shape | GPUs | Max Runners | Runner Group |
|-----------|-------------|-------|------|-------------|--------------|
| `gpu-a10-1/` | `oracle-vm-gpu-a10-1` | `VM.GPU.A10.1` | 1x NVIDIA A10 | 10 | `GPUs` |
| `gpu-a10-2/` | `oracle-vm-gpu-a10-2` | `VM.GPU.A10.2` | 2x NVIDIA A10 | 10 | `GPUs` |

> **Note**: GPU runners use fixed shapes (not flex) and are assigned to the restricted `GPUs` runner group with a lower `maxRunners` limit of 10.

## Files Per Runner Size

Each runner size directory contains a single file:

| File | Description |
|------|-------------|
| `install.yaml` | Full Kubernetes manifests: ServiceAccount, RBAC Role/RoleBinding, and `AutoscalingRunnerSet` |

## How VM Runners Work

Instead of running jobs directly in Kubernetes pods, VM runners use a lightweight controller pod that provisions an OCI compute instance for each job:

```
┌──────────────────────────────────────────────────────────┐
│ Kubernetes Pod (controller)                              │
│                                                          │
│  ┌──────────────────────────────────────────────────┐    │
│  │ gha-cloudrunner                                   │    │
│  │  --arch=<amd64|arm64>                             │    │
│  │  --shape=<VM.Standard.E6.Flex|VM.Standard.A1.Flex>│    │
│  │  --shape-ocpus=<2.0-32.0>                         │    │
│  │  --shape-memory-in-gbs=<8.0-128.0>                │    │
│  │  --availability-domain=<availability-domain>      │    │
│  │  --compartment-id=<compartment-ocid>               │    │
│  │  --subnet-id=<subnet-ocid>                        │    │
│  └──────────────────────────────────────────────────┘    │
│           │                                              │
│           ▼                                              │
│     OCI Compute Instance (actual runner)                 │
└──────────────────────────────────────────────────────────┘
```

- **Controller Image**: `ghcr.io/cncf/gha-cloudrunner` (tag is pinned to a specific build in the manifests)
- **Region / Availability Domain**: Configured per deployment in each `install.yaml`
- **Requires**: `oci-config` and `oci-api-key` Secrets for OCI API authentication

## Scaling Configuration

- **GitHub Config**: `https://github.com/enterprises/cncf`
- **Secret**: `github-arc-secret`
- **Min Runners**: 0
- **Max Runners**: 100 (standard) / 10 (GPU)

## Node Scheduling

All VM runner controller pods are scheduled on Karpenter-managed nodes:

```yaml
tolerations:
- key: "cncf.io/gha-runner"
  operator: "Exists"
  effect: "NoSchedule"
nodeSelector:
  nodepool: karpenter
```

## Metrics

Each `AutoscalingRunnerSet` exports the same comprehensive listener metrics as container runners — see [`../runners/README.md`](../runners/README.md#metrics).

## Relationship to Other Components

- Deployed at sync-wave `3` via [`../argo-automation.yaml`](../argo-automation.yaml)
- Managed by the ARC controller in [`../arc/`](../arc/)
- Controller pods scheduled on nodes provisioned by [`../karpenter/`](../karpenter/)
- Stale VMs cleaned up by [`../hacks/vm-cleaner.yaml`](../hacks/vm-cleaner.yaml)
- Metrics scraped by [`../monitoring/`](../monitoring/)
