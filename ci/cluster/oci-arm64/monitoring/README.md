# Monitoring

Prometheus and Grafana monitoring stack for the OCI ARM64 runner cluster.

## Overview

The monitoring stack is deployed via the `kube-prometheus-stack` Helm chart through ArgoCD. It provides metrics collection, alerting, and dashboarding for both the Kubernetes cluster and GitHub Actions Runner Controller (ARC) workloads.

## Files

| File | Description |
|------|-------------|
| `kube-prometheus-stack.yaml` | ArgoCD Application deploying the Helm chart into the `monitoring` namespace |
| `values.yaml` | Helm values configuring Prometheus rules, Alertmanager, Grafana, and ARC-specific PodMonitors |
| `dashboards/arc-autoscaling-runner-set.yaml` | Grafana dashboard ConfigMap for ARC autoscaling runner set metrics |

## Helm Chart Configuration

- **Chart**: `kube-prometheus-stack` from `prometheus-community.github.io`
- **Namespace**: `monitoring`
- **Cluster Label**: `oci-gha-arm64-runners`
- **Sync Policy**: Automated with pruning, self-healing, and server-side apply

### Key Settings

| Component | Status |
|-----------|--------|
| Prometheus | Enabled |
| Grafana | Enabled (credentials from External Secrets) |
| Alertmanager | Enabled (Slack integration) |
| kube-state-metrics | Enabled |
| CoreDNS metrics | Enabled |

## ARC Metrics Integration

PodMonitors are configured to scrape ARC-specific metrics from the controller and listener pods on port 8080.

## Alerting

Alerts are sent to Slack. Configured alerts include rules for:
- GHA Listener at high capacity (≥80%)
- GHA Listener at max capacity (≥99%)
- Pending EphemeralRunners
- Failed EphemeralRunners

## Prerequisites

- Grafana credentials must be synced from OCI Vault via External Secrets (see [`../external-secrets/`](../external-secrets/))
- Alertmanager Slack API URL must be synced from OCI Vault

## Relationship to Other Components

- Deployed at sync-wave `4` via [`../argo-automation.yaml`](../argo-automation.yaml)
- Consumes secrets from [`../external-secrets/`](../external-secrets/)
- Monitors the ARC controller deployed in [`../arc/`](../arc/)
- Monitors runners from [`../runners/`](../runners/)
