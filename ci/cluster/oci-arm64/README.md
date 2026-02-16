# OCI GitHub Actions Runner Cluster (ARM64)

Configuration for the CNCF GitHub Actions self-hosted runner cluster on OKE (Oracle Kubernetes Engine) with ARM64 (Ampere A1) processors. All resources are managed via ArgoCD — changes merged to `main` are automatically synced.

ARM64 counterpart to the amd64 cluster in [`../oci/`](../oci/). This cluster provides **container-based runners only** (no VM runners or Karpenter).

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

| Directory | Description |
|-----------|-------------|
| [`arc/`](arc/) | Actions Runner Controller (Helm chart via ArgoCD) |
| [`autoscaler/`](autoscaler/) | OCI Cluster Autoscaler (deployed separately, not via ArgoCD) |
| [`external-secrets/`](external-secrets/) | External Secrets Operator + OCI Vault integration |
| [`hacks/`](hacks/) | cgroups v2 enabler DaemonSet |
| [`monitoring/`](monitoring/) | kube-prometheus-stack with ARC dashboards and Slack alerting |
| [`runners/`](runners/) | Container-based AutoscalingRunnerSets (5 sizes) |

## Runner Inventory

| Label | CPUs (req/lim) | Memory (req/lim) | Min | Max |
|-------|----------------|------------------|-----|-----|
| `oracle-2cpu-8gb-arm64` | 2 / 4 | 8Gi / 12Gi | 1 | 100 |
| `oracle-4cpu-16gb-arm64` | 4 / 6 | 16Gi / 20Gi | 1 | 100 |
| `oracle-8cpu-32gb-arm64` | 8 / 10 | 32Gi / 36Gi | 1 | 100 |
| `oracle-16cpu-64gb-arm64` | 4 / 6 | 16Gi / 20Gi | 1 | 100 |
| `oracle-24cpu-384gb-arm64` | 24 / 26 | 384Gi / 392Gi | 0 | 100 |

> **Note:** `oracle-16cpu-64gb-arm64` currently has resources identical to `oracle-4cpu-16gb-arm64` in its manifest — likely a config issue to review.

## Usage

Deploy everything via ArgoCD:

```bash
kubectl apply -f argo-automation.yaml
```

Use a runner in a workflow:

```yaml
jobs:
  build:
    runs-on: oracle-4cpu-16gb-arm64
    steps:
      - uses: actions/checkout@v4
```

## Prerequisites

- **OKE Cluster** with ARM64 node pools
- **ArgoCD** watching this repository
- **Secrets**: `github-arc-secret` (GitHub App creds) and `oracle-secret` (OCI API key)
- **OCI Vault**: `alertmanager-secrets`, `argocd-slack-token`, `grafana-credentials`

## Differences from the amd64 Cluster

| Feature | `oci/` (amd64) | `oci-arm64/` (ARM64) |
|---------|----------------|----------------------|
| VM Runners | 14 configs | None |
| Karpenter | Yes | No |
| Cluster Autoscaler | Legacy | Active |
| Cluster Label | `oci-gha-amd64-runners` | `oci-gha-arm64-runners` |
