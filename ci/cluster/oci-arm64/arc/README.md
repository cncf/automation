# Actions Runner Controller (ARC)

Deploys the `gha-runner-scale-set-controller` Helm chart (v0.11.0) via ArgoCD into the `arc-systems` namespace. Manages the lifecycle and autoscaling of GitHub Actions self-hosted runners.

## Files

| File | Description |
|------|-------------|
| `arc.yaml` | ArgoCD Application deploying the ARC Helm chart |
| `values.yaml` | Helm values — metrics endpoints and Prometheus scrape annotations |

## Key Details

- **Chart**: `gha-runner-scale-set-controller` v0.11.0 from `ghcr.io/actions/actions-runner-controller-charts`
- **Sync Policy**: Automated with pruning and self-healing
- **Metrics**: Port `8080` at `/metrics`
- **Sync-wave**: `2` (via [`../argo-automation.yaml`](../argo-automation.yaml))

Manages the `AutoscalingRunnerSet` resources in [`../runners/`](../runners/). Metrics are scraped by [`../monitoring/`](../monitoring/).
