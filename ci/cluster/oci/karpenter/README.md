# Karpenter

Node autoscaling configuration using [Karpenter for OCI](https://zoom.github.io/karpenter-oci) on the OCI amd64 runner cluster.

## Overview

Karpenter is deployed via ArgoCD to provide just-in-time node provisioning for GitHub Actions runner workloads. It replaces or supplements the OCI Cluster Autoscaler with faster, more flexible node scaling using `NodePool` and `OciNodeClass` resources.

## Files

| File | Description |
|------|-------------|
| `nodepool.yaml` | `NodePool` resource defining node provisioning constraints and limits |
| `ocinodeclass.yaml` | `OciNodeClass` resource defining OCI-specific node configuration (image, boot volume, networking) |

## NodePool Configuration

The `karpenter-np` NodePool defines how and when Karpenter provisions new nodes:

- **Max Nodes**: 30
- **Instance Shape**: `VM.Standard.E6.Flex` (flexible AMD shapes)
- **CPU Sizes**: 8, 16, 24, or 32 OCPUs
- **Capacity Type**: On-demand only
- **OS**: Linux
- **Disruption Policy**: Consolidate empty nodes after 1 minute, with a budget allowing 20% of nodes to be disrupted at once
- **Taint**: `cncf.io/gha-runner:NoSchedule` — ensures only runner pods with matching tolerations are scheduled
- **Label**: `nodepool: karpenter`
- **Termination Grace Period**: 10 minutes

## OciNodeClass Configuration

The `karpenter-onc` OciNodeClass defines OCI-specific settings for provisioned nodes:

- **Boot Volume**: 100 GB with 120 VPUs/GB performance
- **Image**: Oracle Linux OKE-optimized image (pinned version specified in the manifest)
- **Image Family**: `OracleOKELinux`
- **Networking**: OKE native pod networking enabled
- **Kubelet Settings**:
  - Eviction hard limits for disk, inode, and memory
  - 100Mi system-reserved memory
- **VCN / Subnets / Security Groups**: Configured per deployment (see manifest for IDs)

## Helm Chart Details

Karpenter is deployed via two Helm charts (defined in [`../argo-automation.yaml`](../argo-automation.yaml)):

| Chart | Version | Purpose |
|-------|---------|---------|
| `karpenter-crd` | 1.4.2 | Custom Resource Definitions |
| `karpenter` | 1.4.2 | Controller and webhook |

Key Helm values:
- **Cluster Name**: `gha-amd64-runners`
- **Cluster Endpoint**: Private API server endpoint (configured in Helm values)
- **Cluster DNS**: Cluster DNS service IP (configured in Helm values)
- **Region**: Configured per deployment
- **Flex CPU:Memory Ratios**: `4,8,16`

## Relationship to Other Components

- Deployed at sync-wave `10` via [`../argo-automation.yaml`](../argo-automation.yaml) (after runners are configured)
- Both container runners ([`../runners/`](../runners/)) and VM runners ([`../vm-runners/`](../vm-runners/)) use `nodeSelector: { nodepool: karpenter }` and tolerate the `cncf.io/gha-runner` taint to be scheduled on Karpenter-managed nodes
