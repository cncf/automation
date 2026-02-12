# OCI GitHub Actions Runner Cluster (ARM64)

Infrastructure-as-code for the CNCF GitHub Actions self-hosted runner cluster running on Oracle Cloud Infrastructure (OCI) with ARM64 architecture.

## Overview

This directory contains the complete configuration for a Kubernetes-based GitHub Actions runner infrastructure deployed on OKE (Oracle Kubernetes Engine) targeting ARM64 (Ampere A1) processors. All resources are managed via ArgoCD using a GitOps workflow — changes merged to `main` are automatically synced to the cluster.

This cluster is the ARM64 counterpart to the amd64 cluster defined in [`../oci/`](../oci/). It provides **container-based runners** only (no VM-based runners or Karpenter node autoscaling).

## Architecture

```
ArgoCD (argo-automation.yaml)
    │
    ├─ sync-wave -1 ── external-secrets/    → OCI Vault secret sync
    ├─ sync-wave  2 ── arc/                 → ARC controller (v0.11.0)
    ├─ sync-wave  3 ── runners/             → Container-based ARM64 runners (5 sizes)
    ├─ sync-wave  4 ── monitoring/          → Prometheus + Grafana stack
    └─ sync-wave  5 ── hacks/               → cgroups v2 enabler
```

## Directory Structure

| Directory | Description |
|-----------|-------------|
| [`arc/`](arc/) | Actions Runner Controller deployment (Helm chart via ArgoCD) |
| [`autoscaler/`](autoscaler/) | OCI Cluster Autoscaler configuration |
| [`external-secrets/`](external-secrets/) | External Secrets Operator + OCI Vault integration |
| [`hacks/`](hacks/) | cgroups v2 enabler DaemonSet |
| [`monitoring/`](monitoring/) | kube-prometheus-stack with ARC dashboard and Slack alerting |
| [`runners/`](runners/) | Container-based AutoscalingRunnerSets (5 sizes: 2–24 CPUs) |

## Key Files

| File | Description |
|------|-------------|
| `argo-automation.yaml` | Root ArgoCD Application-of-Apps defining all sync-wave ordered applications |

## Runner Inventory

### Container Runners (Kubernetes Pods)

| Label | CPUs | Memory | Max |
|-------|------|--------|-----|
| `oracle-2cpu-8gb-arm64` | 2 | 8Gi | 100 |
| `oracle-4cpu-16gb-arm64` | 4 | 16Gi | 100 |
| `oracle-8cpu-32gb-arm64` | 8 | 32Gi | 100 |
| `oracle-16cpu-64gb-arm64` | 16 | 64Gi | 100 |
| `oracle-24cpu-384gb-arm64` | 24 | 384Gi | 100 |

## Differences from the amd64 Cluster (`oci/`)

| Feature | `oci/` (amd64) | `oci-arm64/` (ARM64) |
|---------|----------------|----------------------|
| Architecture | x86-64 | ARM64 (Ampere A1) |
| Container Runners | 5 sizes | 5 sizes |
| VM Runners | 14 configs (x86, ARM, GPU) | None |
| Karpenter | Yes (node autoscaling) | No |
| Cluster Autoscaler | Legacy config | Active |
| EphemeralRunner Cleanup | Yes | No |
| VM Cleaner | Yes | No |
| Cluster Label | `oci-gha-amd64-runners` | `oci-gha-arm64-runners` |

## Prerequisites

1. **OKE Cluster** with ARM64 node pools and cluster autoscaler support
2. **ArgoCD** installed and configured to watch this repository
3. **Kubernetes Secrets** (pre-provisioned or via External Secrets):
   - `github-arc-secret` in `arc-systems` — GitHub App credentials for ARC
   - `oracle-secret` in `external-secrets` — OCI API key for vault access
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

### Using a Runner in GitHub Actions

```yaml
jobs:
  build:
    runs-on: oracle-4cpu-16gb-arm64  # or any runner label from the inventory
    steps:
      - uses: actions/checkout@v4
      # ...
```

## Monitoring

- **Grafana dashboards** are auto-provisioned for ARC metrics
- **Alertmanager** sends alerts to Slack
- **Prometheus** scrapes ARC controller and listener pods on port 8080
