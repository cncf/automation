# OCI GitHub Actions Runner Cluster (amd64)

Infrastructure-as-code for the CNCF GitHub Actions self-hosted runner cluster running on Oracle Cloud Infrastructure (OCI) with amd64 architecture.

## Overview

This directory contains the complete configuration for a Kubernetes-based GitHub Actions runner infrastructure deployed on OKE (Oracle Kubernetes Engine). All resources are managed via ArgoCD using a GitOps workflow — changes merged to `main` are automatically synced to the cluster.

The cluster provides both **container-based runners** (Kubernetes pods with Docker-in-Docker) and **VM-based runners** (dedicated OCI compute instances) registered to the `cncf` GitHub Enterprise.

## Architecture

```
ArgoCD (argo-automation.yaml)
    │
    ├─ sync-wave -1 ── external-secrets/    → OCI Vault secret sync
    ├─ sync-wave  2 ── arc/                 → ARC controller (v0.11.0)
    ├─ sync-wave  3 ── runners/             → Container-based runners (5 sizes)
    ├─ sync-wave  3 ── vm-runners/          → VM-based runners (14 configs)
    ├─ sync-wave  4 ── monitoring/          → Prometheus + Grafana stack
    ├─ sync-wave  5 ── hacks/               → Operational workarounds
    └─ sync-wave 10 ── karpenter/           → Node autoscaling
```

## Directory Structure

| Directory | Description |
|-----------|-------------|
| [`arc/`](arc/) | Actions Runner Controller deployment (Helm chart via ArgoCD) |
| [`autoscaler/`](autoscaler/) | OCI Cluster Autoscaler configuration (legacy, see also Karpenter) |
| [`external-secrets/`](external-secrets/) | External Secrets Operator + OCI Vault integration |
| [`hacks/`](hacks/) | cgroups v2 enabler, ephemeral runner cleanup, VM cleanup CronJobs |
| [`karpenter/`](karpenter/) | Karpenter node autoscaler with NodePool and OciNodeClass |
| [`monitoring/`](monitoring/) | kube-prometheus-stack with ARC dashboards and Slack alerting |
| [`runners/`](runners/) | Container-based AutoscalingRunnerSets (5 sizes: 2–24 CPUs) |
| [`vm-runners/`](vm-runners/) | VM-based runners (x86-64, ARM64, GPU — 14 configurations) |

## Key Files

| File | Description |
|------|-------------|
| `argo-automation.yaml` | Root ArgoCD Application-of-Apps defining all sync-wave ordered applications |

## Runner Inventory

### Container Runners (Kubernetes Pods)

| Label | CPUs | Memory | Max |
|-------|------|--------|-----|
| `oracle-2cpu-8gb-x86-64` | 2 | 8Gi | 100 |
| `oracle-4cpu-16gb-x86-64` | 4 | 16Gi | 100 |
| `oracle-8cpu-32gb-x86-64` | 8 | 32Gi | 100 |
| `oracle-16cpu-64gb-x86-64` | 16 | 64Gi | 100 |
| `oracle-24cpu-384gb-x86-64` | 24 | 384Gi | 100 |

### VM Runners (OCI Compute Instances)

| Label | Shape | OCPUs | Memory | Max |
|-------|-------|-------|--------|-----|
| `oracle-vm-{2,4,8,16,24,32}cpu-*-x86-64` | `VM.Standard.E6.Flex` | 2–32 | 8–128 GB | 100 |
| `oracle-vm-{2,4,8,16,24,32}cpu-*-arm64` | `VM.Standard.A1.Flex` | 2–32 | 8–128 GB | 100 |
| `oracle-vm-gpu-a10-1` | `VM.GPU.A10.1` | Fixed | Fixed | 10 |
| `oracle-vm-gpu-a10-2` | `VM.GPU.A10.2` | Fixed | Fixed | 10 |

## Prerequisites

1. **OKE Cluster** with Karpenter and cluster autoscaler support
2. **ArgoCD** installed and configured to watch this repository
3. **Kubernetes Secrets** (pre-provisioned or via External Secrets):
   - `github-arc-secret` in `arc-systems` — GitHub App credentials for ARC
   - `oracle-secret` in `external-secrets` — OCI API key for vault access
   - `oci-config` + `oci-api-key` in `arc-systems` — OCI CLI credentials for VM runners
4. **OCI Vault** containing:
   - `alertmanager-secrets` — Slack API URL for alerts
   - `argocd-slack-token` — Slack token for ArgoCD notifications
   - `grafana-credentials` — Grafana admin username/password

## Usage

### Deploying the Full Stack

Apply the root automation file to ArgoCD:

```bash
kubectl apply -f argo-automation.yaml
```

ArgoCD will deploy all components in sync-wave order.

### Adding a New Runner Size

1. Create a new directory:
   - Under `runners/`, use the naming convention `{cpus}cpu-{memory}gb/` (for example, `4cpu-16gb/`)
   - Under `vm-runners/`, follow the existing patterns, such as `{cpus}cpu-{memory}gb-x86-64/` for amd64, `*-arm64/` for ARM64 VMs, or `gpu-*` (for example, `gpu-a10-1/`) for GPU runners
2. Copy an existing `install.yaml` and modify the resource requests/limits and runner name
3. For container runners, also create an `argo.yaml` ArgoCD Application
4. Commit and push — ArgoCD will auto-sync the new runner

### Using a Runner in GitHub Actions

```yaml
jobs:
  build:
    runs-on: oracle-4cpu-16gb-x86-64  # or any runner label from the inventory
    steps:
      - uses: actions/checkout@v4
      # ...
```

## Monitoring

- **Grafana dashboards** are auto-provisioned for ARC metrics
- **Alertmanager** sends alerts to a configured Slack channel
- **Prometheus** scrapes ARC controller and listener pods on port 8080
