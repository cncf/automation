# Cluster Autoscaler

Adjusts the number of OKE nodes based on pending pod demand. **Not managed by ArgoCD** — applied directly to the cluster.

## Files

| File | Description |
|------|-------------|
| `cluster-autoscaler.yaml` | ServiceAccount, RBAC, and Deployment manifest |

## Key Details

- **Image**: `iad.ocir.io/oracle/oci-cluster-autoscaler:1.33.0-3`
- **Namespace**: `kube-system` (3 replicas)
- **Node Pool Range**: 1–10 nodes
- **Auth**: OCI Instance Principal
- **Metrics**: Prometheus scrape on port `8085`

## Prerequisites

Requires OCI IAM setup:
1. **Dynamic Group** containing the worker node compute instances
2. **IAM Policy** allowing the group to manage node pools

Reference: [OCI Cluster Autoscaler Docs](https://docs.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengusingclusterautoscaler_topic-Working_with_the_Cluster_Autoscaler.htm) · [OKE Workload Identity](https://blogs.oracle.com/cloud-infrastructure/post/oke-workload-identity-greater-control-access)

## Deployment

```bash
kubectl apply -f cluster-autoscaler.yaml
```

Scales the node pool hosting the runner pods in [`../runners/`](../runners/).

