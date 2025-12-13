 Oracle Cloud Infrastructure (OCI) GitHub Actions Runner Setup

This directory contains the complete GitOps configuration for deploying and managing GitHub Actions runners on Oracle Kubernetes Engine (OKE) using ArgoCD.

 📋 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Components](#components)
- [Prerequisites](#prerequisites)
- [Deployment Guide](#deployment-guide)
- [Runner Configuration](#runner-configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

 🎯 Overview

This setup provides self-hosted GitHub Actions runners for CNCF projects on Oracle Cloud Infrastructure. The infrastructure is managed using GitOps principles with ArgoCD, ensuring declarative and version-controlled deployments.

Key Features:
- Multiple runner sizes (2-32 CPU configurations)
- Both AMD64 and ARM64 architecture support
- Auto-scaling based on workload demand
- Comprehensive monitoring with Prometheus and Grafana
- Secure secrets management with External Secrets Operator
- Automated cleanup and maintenance tasks

 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    GitHub (CNCF Projects)                    │
│                  Workflow Jobs Triggered                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│              Oracle Kubernetes Engine (OKE)                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                      ArgoCD                           │  │
│  │         (Continuous Deployment & Sync)               │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                    │
│         ┌───────────────┼───────────────┐                   │
│         ↓               ↓               ↓                    │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐               │
│  │   ARC    │   │ Runners  │   │ Karpenter│               │
│  │Controller│   │(Multiple │   │(Node Auto│               │
│  │          │   │  Sizes)  │   │ Scaling) │               │
│  └──────────┘   └──────────┘   └──────────┘               │
│         │               │               │                    │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐               │
│  │External  │   │Monitoring│   │  Hacks   │               │
│  │ Secrets  │   │(Prom/    │   │(Cleanup  │               │
│  │Operator  │   │ Grafana) │   │  Jobs)   │               │
│  └──────────┘   └──────────┘   └──────────┘               │
└─────────────────────────────────────────────────────────────┘
```

📦 Components
 1. Actions Runner Controller (ARC)
Path: `ci/cluster/oci/arc/`

The ARC manages the lifecycle of GitHub Actions runners in Kubernetes.

- Chart: `gha-runner-scale-set-controller` v0.11.0
- Namespace: `arc-systems`
- Sync Wave: 2 (deployed early in the sequence)

Key Features:
- Automatic runner provisioning
- Scale-to-zero capability
- Runner lifecycle management
- GitHub webhook integration

### 2. GitHub Runners
Paths:
- `ci/cluster/oci/runners/` (Container runners)
- `ci/cluster/oci/vm-runners/` (VM-based runners)

Multiple runner configurations for different workload requirements:

| Runner Type | CPU | Memory | Architecture | Label |
|-------------|-----|--------|--------------|-------|
| Small | 2 | 8GB | AMD64/ARM64 | `oracle-2cpu-8gb-*` |
| Medium | 4 | 16GB | AMD64/ARM64 | `oracle-4cpu-16gb-*` |
| Standard | 8 | 32GB | AMD64/ARM64 | `oracle-8cpu-32gb-*` |
| Large | 16 | 64GB | AMD64/ARM64 | `oracle-16cpu-64gb-*` |
| X-Large | 24 | 384GB | AMD64 | `oracle-24cpu-384gb-x86-64` |
| XX-Large | 32 | 128GB | ARM64 | `oracle-32cpu-128gb-arm64` |

Sync Wave: 3 (deployed after ARC)

### 3. Karpenter
**Path:** `ci/cluster/oci/karpenter/`

Kubernetes node autoscaler optimized for Oracle Cloud.

- **Chart:** `karpenter` v1.4.2 (from zoom.github.io/karpenter-oci)
- **Namespace:** `karpenter`
- **Sync Wave:** 10 (deployed last)

**Configuration:**
- Cluster: `gha-amd64-runners`
- Region: `us-sanjose-1`
- Flexible CPU/Memory ratios: 4, 8, 16

### 4. External Secrets Operator
**Path:** `ci/cluster/oci/external-secrets/`

Manages secrets synchronization from external secret stores (e.g., Oracle Vault, AWS Secrets Manager).

- **Namespace:** `default`
- **Sync Wave:** -1 (deployed first, before everything else)

**Purpose:**
- Secure GitHub PAT/token management
- Centralized secrets management
- Automatic secret rotation support

### 5. Monitoring Stack
**Path:** `ci/cluster/oci/monitoring/`

Prometheus and Grafana stack for observability.

- **Chart:** `kube-prometheus-stack`
- **Namespace:** `default`
- **Sync Wave:** 4

**Includes:**
- Prometheus for metricollection
- Grafana dashboards for visualization
- AlertManager for notifications
- Custom dashboards for runner metrics

### 6. Cluster Autoscaler
**Path:** `ci/cluster/oci/autoscaler/`

OKE-native cluster autoscaler (alternative/complement to Karpenter).

**Features:**
- OCI Workload Identity integration
- Dynamic node pool scaling
- Cost optimization

### 7. Hacks & Utilities
**Path:** `ci/cluster/oci/hacks/`

Maintenance and cleanup utilities:

- **cgroups-v2-enabler-ds.yaml:** Enables cgroups v2 support for containers
- **ephemeralrunner-cleanup-cj.yaml:** CronJob to clean up stale ephemeral runners
- **vm-cleaner.yaml:** Cleanup job for VM-based runners

**Sync Wave:** 5

## 🔧 Prerequisites

Before deploying this setup, ensure you have:

1. **Oracle Cloud Infrastructure Access:**
   - OKE cluster provisioned (see `ci/services/cluster/`)
   - Appropriate IAM policies configured
   - Workload Identity or Instance Principal setup

2. **Tools Installed:**
   - `kubectl` (configured for your OKE cluster)
   - `argocd` CLI (optional, for management)
   - `helm` (for manual operations)

3. **GitHub Configuration:**
   - GitHub App or Personal Access Token (PAT) with appropriate permissions
   - Token stored in External Secrets backend

4. **ArgoCD Deployed:**
   - ArgoCD installed in the cluster (see `ci/argocd/README.md`)
   - Slack notifications configured (optional)

## 🚀 Deployment Guide

### Step 1: Verify Cluster Access

```bash
# Check cluster connectivity
kubectl get nodes

# Verify ArgoCD is running
kubectl get pods -n argocd
```

### Step 2: Configure Secrets

Create the GitHub token secret (if not using External Secrets Operator):

```bash
kubectl create namespace arc-systems

kubectl create secret generic github-arc-secret \
  --from-literal=github_token=YOUR_GITHUB_TOKEN \
  --namespace=arc-systems
```

**Note:** In production, use External Secrets Operator to manage this secret.

### Step 3: Deploy ArgoCD Applications

Apply the main ArgoCD application manifest:

```bash
kubectl apply -f ci/cluster/oci/argo-automation.yaml
```

This will deploy all components in the correct order based on sync waves:

1. **Wave -1:** External Secrets Operator
2. **Wave 2:** Actions Runner Controller
3. **Wave 3:** GitHub Runners (container & VM)
4. **Wave 4:** Monitoring Stack
5. **Wave 5:** Hacks & Utilities
6. **Wave 10:** Karpenter

### Step 4: Verify Deployment

Check ArgoCD application status:

```bash
# Using kubectl
kubectl get applications -n argocd

# Using ArgoCD CLI
argocd app list
```

Check runner pods:

```bash
kubectl get pods -n arc-systems
```

### Step 5: Monitor Sync Progress

Watch ArgoCD sync the applications:

```bash
# Watch all applications
kubectl get applications -n argocd -w

# Check specific application
argocd app get github-runners
```

## 🎮 Runner Configuration

### Using Runners in GitHub Workflows

In your `.github/workflows/*.yml` files:

```yaml
name: CI Pipeline
on: [push, pull_request]

jobs:
  build-amd64:
    runs-on: oracle-16cpu-64gb-x86-64
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: make build

  build-arm64:
    runs-on: oracle-16cpu-64gb-arm64
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: make build
```

### Available Runner Labels

See the [Components](#components) section for the complete list of runner labels.

### Customizing Runner Configurations

To add or modify runner configurations:

1. Navigate to `ci/cluster/oci/runners/` or `ci/cluster/oci/vm-runners/`
2. Copy an existing runner directory
3. Modify the resource specifications
4. Update the runner label
5. Commit and push - ArgoCD will automatically sync

## 📊 Monitoring

### Accessing Grafana

```bash
# Port-forward to Grafana
kubectl port-forward -n default svc/kube-prometheus-stack-grafana 3000:80
```

Access at: http://localhost:3000

**Default credentials:** Check the monitoring values.yaml or secret

### Key Metrics to Monitor

- **Runner Queue Length:** Number of pending workflow jobs
- **Runner Utilization:** Active vs idle runners
- **Job Duration:** Time taken for workflow jobs
- **Node Scaling:** Karpenter/autoscaler activity
- **Resource Usage:** CPU, memory, disk usage per runner

### Prometheus Queries

```promql
# Active runners
sum(kube_pod_status_phase{namespace="arc-systems", phase="Running"})

# Runner job duration
histogram_quantile(0.95, rate(github_runner_job_duration_seconds_bucket[5m]))

# Failed jobs
rate(github_runner_job_status{status="failed"}[5m])
```

## 🔍 Troubleshooting

### Runners Not Starting

**Check ARC controller logs:**
```bash
kubectl logs -n arc-systems -l app.kubernetes.io/name=gha-runner-scale-set-controller
```

**Common issues:**
- Invalid GitHub token
- Network connectivity to GitHub
- Insufficient cluster resources

### Runners Stuck in Pending

**Check node availability:**
```bash
kubectl get nodes
kubectl describe pod <runner-pod-name> -n arc-systems
```

**Possible causes:**
- Karpenter/autoscaler not scaling
- Resource constraints
- Node affinity/taints issues

### ArgoCD Sync Failures

**Check application status:**
```bash
argocd app get <app-name>
kubectl describe application <app-name> -n argocd
```

**Common fixes:**
- Verify source repository access
- Check YAML syntax
- Review sync waves and dependencies

### Secrets Not Available

**Verify External Secrets Operator:**
```bash
kubectl get externalsecrets -A
kubectl logs -n default -l app.kubernetes.io/name=external-secrets
```

**Check secret synchronization:**
```bash
kubectl get secret github-arc-secret -n arc-systems
```

### High Resource Usage

**Check runner resource limits:**
```bash
kubectl top pods -n arc-systems
kubectl describe pod <runner-pod-name> -n arc-systems
```

**Solutions:**
- Adjust runner resource requests/limits
- Scale down unused runners
- Review Karpenter node provisioning

## 📚 Additional Resources

- [Actions Runner Controller Documentation](https://github.com/actions/actions-runner-controller)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Karpenter for OCI](https://github.com/zoom/karpenter-oci)
- [OKE Documentation](https://docs.oracle.com/en-us/iaas/Content/ContEng/home.htm)
- [External Secrets Operator](https://external-secrets.io/)

## 🤝 Contributing

When making changes to this configuration:

1. Test changes locally using a KIND cluster (see `documentation.txt` in repo root)
2. Create a feature branch
3. Submit a PR with detailed description
4. Ensure all ArgoCD applications sync successfully
5. Monitor the deployment in the OCI cluster

## 📝 Notes

- **Sync Waves:** Control deployment order. Lower numbers deploy first.
- **Auto-Sync:** Most applications have automated sync enabled for continuous deployment.
- **Pruning:** Enabled for most apps to remove resources deleted from Git.
- **Notifications:** Slack notifications configured for deployment events.

---

**Maintained by:** CNCF Projects Team  
**Last Updated:** December 2025
