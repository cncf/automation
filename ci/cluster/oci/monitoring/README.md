# Monitoring

Prometheus and Grafana monitoring stack for the OCI amd64 runner cluster.

## Overview

The monitoring stack is deployed via the `kube-prometheus-stack` Helm chart through ArgoCD. It provides metrics collection, alerting, and dashboarding for both the Kubernetes cluster and GitHub Actions Runner Controller (ARC) workloads.

## Files

| File | Description |
|------|-------------|
| `kube-prometheus-stack.yaml` | ArgoCD Application deploying the Helm chart (v70.4.2) into the `monitoring` namespace |
| `values.yaml` | Helm values configuring Prometheus rules, Alertmanager, Grafana, and ARC-specific PodMonitors |
| `dashboards/arc-autoscaling-runner-set.yaml` | Grafana dashboard ConfigMap for ARC autoscaling runner set metrics |
| `dashboards/github-arc-monitoring.yaml` | Grafana dashboard ConfigMap for comprehensive ARC monitoring |

## Helm Chart Configuration

- **Chart**: `kube-prometheus-stack` v70.4.2 from `prometheus-community.github.io`
- **Namespace**: `monitoring`
- **Sync Policy**: Automated with pruning, self-healing, and server-side apply

### Key Settings

| Component | Status |
|-----------|--------|
| Prometheus | Enabled |
| Grafana | Enabled (credentials from External Secrets) |
| Alertmanager | Enabled (Slack integration) |
| kube-state-metrics | Enabled |
| kube-proxy metrics | Enabled |
| CoreDNS metrics | Enabled |
| kube-controller-manager | Disabled |
| kube-scheduler | Disabled |
| etcd | Disabled |

## ARC Metrics Integration

Two `PodMonitors` are configured to scrape ARC-specific metrics:

1. **`gha-rs-controller`** — scrapes the ARC controller pods on port 8080
2. **`gha-rs-listener`** — scrapes the runner scale set listener pods on port 8080

## Alerting

Alerts are sent to a configured Slack channel. Configured alerts include:

| Alert | Condition | Severity |
|-------|-----------|----------|
| GHA Listener at High Capacity | ≥80% runner capacity for 5 minutes | Warning |
| GHA Listener at Max Capacity | ≥99% runner capacity for 1 minute | Major |
| GHA Pending EphemeralRunners | >10 pending runners for 1 minute | Warning |
| GHA Failed EphemeralRunners | >5 failed runners for 1 minute | Warning |

## Grafana Dashboards

Two Grafana dashboards are deployed as ConfigMaps with the `grafana_dashboard: "1"` label:

- **ARC Autoscaling Runner Set** — monitors runner scaling behavior, job counts, and resource utilization
- **GitHub ARC Monitoring** — comprehensive view including controller performance, reconciliation metrics, API calls, and workqueue depth

## Prerequisites

- Grafana credentials must be synced from OCI Vault via External Secrets (see [`../external-secrets/`](../external-secrets/))
- Alertmanager Slack API URL must be synced from OCI Vault

## Relationship to Other Components

- Deployed at sync-wave `4` via [`../argo-automation.yaml`](../argo-automation.yaml)
- Consumes secrets from [`../external-secrets/`](../external-secrets/)
- Monitors the ARC controller deployed in [`../arc/`](../arc/)
- Monitors runners from [`../runners/`](../runners/) and [`../vm-runners/`](../vm-runners/)
