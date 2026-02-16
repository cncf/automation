# Container Runners

GitHub Actions self-hosted runners for the OCI ARM64 cluster. Each subdirectory defines an `AutoscalingRunnerSet` running jobs as Kubernetes pods with Docker-in-Docker (DinD). Deployed at **sync-wave `3`**.

## Runner Sizes

| Runner Label | CPUs (req/lim) | Memory (req/lim) | Min | Max |
|-------------|----------------|------------------|-----|-----|
| `oracle-2cpu-8gb-arm64` | 2 / 4 | 8Gi / 12Gi | 1 | 100 |
| `oracle-4cpu-16gb-arm64` | 4 / 6 | 16Gi / 20Gi | 1 | 100 |
| `oracle-8cpu-32gb-arm64` | 8 / 10 | 32Gi / 36Gi | 1 | 100 |
| `oracle-16cpu-64gb-arm64` | 4 / 6 | 16Gi / 20Gi | 1 | 100 |
| `oracle-24cpu-384gb-arm64` | 24 / 26 | 384Gi / 392Gi | 0 | 100 |

> **Note:** `oracle-16cpu-64gb-arm64` currently has resources identical to `oracle-4cpu-16gb-arm64` — likely a config issue to review.

Each directory contains an `argo.yaml` (ArgoCD Application) and `install.yaml` (ServiceAccount, RBAC, AutoscalingRunnerSet).

## Pod Structure

Each runner pod runs two containers:

```
┌──────────────────────────────────────┐
│  runner (GHA runner)                 │
│  ◄──── /var/run/docker.sock ────►    │
│  dind (Docker-in-Docker, MTU 1400)   │
└──────────────────────────────────────┘
```

- **Runner image**: `ghcr.io/cncf/external-gha-runner:noble`
- **DinD image**: `docker.io/library/docker:dind`
- **Storage**: 50Gi ephemeral (`oci-bv`) + 10Gi overlay
- **GitHub Enterprise**: `https://github.com/enterprises/cncf`

Managed by the ARC controller in [`../arc/`](../arc/). Metrics scraped by [`../monitoring/`](../monitoring/).
