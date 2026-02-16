# Monitoring

Prometheus + Grafana stack for the OCI ARM64 runner cluster, deployed via `kube-prometheus-stack` Helm chart (v70.4.2) at **sync-wave `4`**.

## Files

| File | Description |
|------|-------------|
| `kube-prometheus-stack.yaml` | ArgoCD Application deploying the chart into `monitoring` namespace |
| `values.yaml` | Helm values — Prometheus rules, Alertmanager, Grafana, ARC PodMonitors |
| `dashboards/arc-autoscaling-runner-set.yaml` | Grafana dashboard ConfigMap for ARC metrics |

## ARC Metrics

PodMonitors scrape the ARC controller and listener pods on port 8080 every 15s.

## Alerting

Alerts go to `#internal-gha-prmths-alrtmngr` on Slack:
- GHA Listener at high capacity (>=80%) or max capacity (>=99%)
- Pending or failed EphemeralRunners

## Prerequisites

- Grafana credentials and Alertmanager Slack URL must be synced from OCI Vault (see [`../external-secrets/`](../external-secrets/))

Monitors the ARC controller in [`../arc/`](../arc/) and runners in [`../runners/`](../runners/).
