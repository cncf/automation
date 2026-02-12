# External Secrets

Configuration for the External Secrets Operator (ESO) on the OCI amd64 runner cluster, used to synchronize secrets from OCI Vault into Kubernetes.

## Overview

The External Secrets Operator is deployed via an ArgoCD Application using the official Helm chart. It connects to an OCI Vault and syncs secrets into various namespaces for use by Alertmanager, ArgoCD, and Grafana.

## Files

| File | Description |
|------|-------------|
| `external-secrets-operator.yaml` | ArgoCD Application deploying the ESO Helm chart (v0.16.0) into the `external-secrets` namespace |
| `secrets/cluster-secret-store.yaml` | `ClusterSecretStore` connecting to OCI Vault using `UserPrincipal` auth |
| `secrets/alertmanager-secrets.yaml` | `ExternalSecret` syncing Alertmanager Slack API URL into the `monitoring` namespace |
| `secrets/argocd-notifications-secret.yaml` | `ExternalSecret` syncing ArgoCD Slack token into the `argocd` namespace |
| `secrets/grafana-credentials.yaml` | `ExternalSecret` syncing Grafana admin credentials into the `monitoring` namespace |

## Architecture

```
OCI Vault
    │
    ▼
ClusterSecretStore (oci-secret-store)
    │
    ├──► ExternalSecret: alertmanager-secrets    → Secret in monitoring namespace
    ├──► ExternalSecret: argocd-notifications    → Secret in argocd namespace
    └──► ExternalSecret: grafana-credentials     → Secret in monitoring namespace
```

## Configuration Details

- **Helm Chart**: `external-secrets` v0.16.0 from `charts.external-secrets.io`
- **Namespace**: `external-secrets` (operator), secrets are created in their target namespaces
- **Vault Region**: Configured in the `ClusterSecretStore` manifest
- **Auth Method**: `UserPrincipal` — requires `oracle-secret` Kubernetes Secret containing `privateKey` and `fingerprint` in the `external-secrets` namespace
- **Refresh Interval**: 1 hour for all ExternalSecrets

## Prerequisites

The following Kubernetes Secret must exist in the `external-secrets` namespace before the ClusterSecretStore can authenticate:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oracle-secret
  namespace: external-secrets
data:
  privateKey: <base64-encoded-OCI-API-private-key>
  fingerprint: <base64-encoded-fingerprint>
```

## Relationship to Other Components

- This Application is deployed at sync-wave `-1` via [`../argo-automation.yaml`](../argo-automation.yaml) (before other components)
- Alertmanager secrets are consumed by the Prometheus stack in [`../monitoring/`](../monitoring/)
- Grafana credentials are consumed by the Grafana instance deployed via [`../monitoring/`](../monitoring/)
- ArgoCD notification secret enables Slack notifications for ArgoCD sync events
