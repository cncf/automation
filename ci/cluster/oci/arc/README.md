# Actions Runner Controller (ARC)

Configuration for the GitHub Actions Runner Controller (ARC) on the OCI amd64 runner cluster.

## Overview

ARC is deployed as an ArgoCD `Application` that installs the `gha-runner-scale-set-controller` Helm chart from `ghcr.io/actions/actions-runner-controller-charts`. It manages the lifecycle of GitHub Actions self-hosted runners using Kubernetes-native autoscaling.

## Files

| File | Description |
|------|-------------|
| `arc.yaml` | ArgoCD Application manifest that deploys the ARC controller Helm chart (v0.11.0) into the `arc-systems` namespace |
| `values.yaml` | Helm values for the ARC controller — configures metrics endpoints and Prometheus scrape annotations |

## Configuration Details

- **Chart**: `gha-runner-scale-set-controller` v0.11.0
- **Namespace**: `arc-systems`
- **Cluster Label**: `oci-gha-amd64-runners`
- **Sync Policy**: Automated with pruning and self-healing
- **Metrics**: Exposed on port `8080` at `/metrics`, with Prometheus scrape annotations enabled

## Relationship to Other Components

- The ARC controller manages the `AutoscalingRunnerSet` resources defined in [`../runners/`](../runners/) and [`../vm-runners/`](../vm-runners/)
- This Application is referenced by [`../argo-automation.yaml`](../argo-automation.yaml) (sync-wave `2`)
- Metrics are scraped by the Prometheus stack configured in [`../monitoring/`](../monitoring/)
