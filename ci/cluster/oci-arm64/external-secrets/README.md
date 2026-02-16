# External Secrets

Syncs secrets from OCI Vault into Kubernetes via the External Secrets Operator (ESO). Deployed at **sync-wave `-1`** so secrets are available before other components start.

## Files

| File | Description |
|------|-------------|
| `external-secrets-operator.yaml` | ArgoCD Application — ESO Helm chart (v0.16.0) |
| `secrets/cluster-secret-store.yaml` | `ClusterSecretStore` connecting to OCI Vault (`UserPrincipal` auth) |
| `secrets/alertmanager-secrets.yaml` | Slack API URL → `monitoring` namespace |
| `secrets/argocd-notifications-secret.yaml` | ArgoCD Slack token → `argocd` namespace |
| `secrets/grafana-credentials.yaml` | Grafana admin credentials → `monitoring` namespace |

## How It Works

```
OCI Vault
    │
    ▼
ClusterSecretStore (oci-secret-store)
    │
    ├──► alertmanager-secrets    → monitoring namespace
    ├──► argocd-notifications    → argocd namespace
    └──► grafana-credentials     → monitoring namespace
```

All ExternalSecrets refresh every 1 hour.

## Prerequisite

The `oracle-secret` Kubernetes Secret must exist in the `external-secrets` namespace with the OCI API `privateKey` and `fingerprint` before the ClusterSecretStore can authenticate.

Consumed by [`../monitoring/`](../monitoring/) (Grafana + Alertmanager) and ArgoCD notifications.
