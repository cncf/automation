# Container Runners

GitHub Actions self-hosted container runners for the OCI ARM64 cluster, managed by Actions Runner Controller (ARC).

## Overview

Each subdirectory defines an `AutoscalingRunnerSet` that runs GitHub Actions jobs in Kubernetes pods with Docker-in-Docker (DinD) support. Runners are registered to the `cncf` GitHub Enterprise and automatically scale between 0 and 100 instances based on demand.

## Available Runner Sizes

| Directory | Runner Label | CPUs | Memory | Storage |
|-----------|-------------|------|--------|---------|
| `2cpu-8gb/` | `oracle-2cpu-8gb-arm64` | 2 (request) / 4 (limit) | 8Gi (request) / 12Gi (limit) | 50Gi ephemeral |
| `4cpu-16gb/` | `oracle-4cpu-16gb-arm64` | 4 (request) / 8 (limit) | 16Gi (request) / 24Gi (limit) | 50Gi ephemeral |
| `8cpu-32gb/` | `oracle-8cpu-32gb-arm64` | 8 (request) / 16 (limit) | 32Gi (request) / 48Gi (limit) | 50Gi ephemeral |
| `16cpu-64gb/` | `oracle-16cpu-64gb-arm64` | 16 (request) / 32 (limit) | 64Gi (request) / 96Gi (limit) | 50Gi ephemeral |
| `24cpu-384gb/` | `oracle-24cpu-384gb-arm64` | 24 (request) / 48 (limit) | 384Gi (request) / 576Gi (limit) | 50Gi ephemeral |

## Files Per Runner Size

Each runner size directory contains:

| File | Description |
|------|-------------|
| `argo.yaml` | ArgoCD Application that syncs the runner configuration from this repository |
| `install.yaml` | Full Kubernetes manifests including ServiceAccount, RBAC, and `AutoscalingRunnerSet` |
| `values.yaml` | *(optional)* Additional Helm values override |

## Runner Pod Architecture

Each runner pod runs two containers plus init containers:

```
┌─────────────────────────────────────────────┐
│ Pod: oracle-{size}-arm64                    │
│                                             │
│  ┌──────────────┐   ┌───────────────────┐   │
│  │   runner      │   │   dind            │   │
│  │ (GHA runner)  │◄─►│ (Docker-in-Docker)│   │
│  │               │   │                   │   │
│  └──────────────┘   └───────────────────┘   │
│        ▲                                    │
│        │ shared: /var/run/docker.sock        │
│                                             │
│  Init: chowner (set /tmp permissions)       │
│  Init: init-dind-externals (copy externals) │
└─────────────────────────────────────────────┘
```

- **Runner image**: `ghcr.io/cncf/external-gha-runner:noble`
- **DinD image**: `docker.io/library/docker:dind` (privileged, MTU 1400)
- **Storage**: 50Gi ephemeral volume (`oci-bv` StorageClass) + 10Gi overlay volume
- **Shared mounts**: `_work`, `.cache`, `.gradle`, `go`, `.m2`, `tmp`, `docker`

## Scaling Configuration

- **GitHub Config**: `https://github.com/enterprises/cncf`
- **Secret**: `github-arc-secret`
- **Min Runners**: 0
- **Max Runners**: 100

## Metrics

Each `AutoscalingRunnerSet` exposes comprehensive listener metrics:

- **Counters**: `gha_started_jobs_total`, `gha_completed_jobs_total`
- **Gauges**: `gha_assigned_jobs`, `gha_running_jobs`, `gha_registered_runners`, `gha_busy_runners`, `gha_min_runners`, `gha_max_runners`, `gha_desired_runners`, `gha_idle_runners`
- **Histograms**: `gha_job_startup_duration_seconds`, `gha_job_execution_duration_seconds`

## Relationship to Other Components

- Deployed at sync-wave `3` via [`../argo-automation.yaml`](../argo-automation.yaml)
- Managed by the ARC controller in [`../arc/`](../arc/)
- Metrics scraped by [`../monitoring/`](../monitoring/)
