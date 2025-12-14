# Oracle Cloud GitHub Actions Controller (GHAC) & ArgoCD Setup Guide

## Overview

This document provides a comprehensive guide for understanding and managing the Oracle Cloud Infrastructure (OCI) based GitHub Actions runner infrastructure for CNCF projects.

## What This Setup Does

This infrastructure enables CNCF projects to run GitHub Actions workflows on self-hosted runners deployed in Oracle Kubernetes Engine (OKE), providing:

- Cost Efficiency: Utilizing donated Oracle Cloud resources
- Performance: Dedicated runners with various CPU/memory configurations
- Flexibility: Support for both AMD64 and ARM64 architectures
- Scalability: Auto-scaling based on workflow demand
- Reliability: GitOps-based deployment with ArgoCD

## High-Level Architecture

### Component Flow

```
GitHub Repository (CNCF Project)
    │
    │ 1. Workflow triggered
    ↓
GitHub Actions API
    │
    │ 2. Job queued
    ↓
Actions Runner Controller (ARC)
    │
    │ 3. Detects pending job
    ↓
Kubernetes (OKE Cluster)
    │
    │ 4. Creates runner pod
    ↓
Karpenter/Autoscaler
    │
    │ 5. Provisions nodes if needed
    ↓
Runner Pod Executes Job
    │
    │ 6. Runs workflow steps
    ↓
Job Completes & Pod Terminates
    │
    │ 7. Results sent to GitHub
    ↓
GitHub Actions UI (Results visible)
```

### GitOps Deployment Flow

```
Developer
    │
    │ 1. Commits config changes
    ↓
GitHub Repository (cncf/automation)
    │
    │ 2. ArgoCD monitors repo
    ↓
ArgoCD
    │
    │ 3. Detects changes
    │ 4. Syncs to cluster
    ↓
Kubernetes Cluster (OKE)
    │
    │ 5. Resources updated
    ↓
Runners Updated Automatically
```

## Repository Structure

```
ci/
├── argocd/                          # ArgoCD installation configs
│   ├── argocd-cm.yaml              # ArgoCD ConfigMap
│   ├── argocd-notification-cm.yaml # Notification settings
│   └── kustomization.yaml          # Kustomize config
│
├── cluster/
│   └── oci/                        # Oracle Cloud specific configs
│       ├── argo-automation.yaml    # Main ArgoCD Application (App-of-Apps)
│       ├── README.md               # Detailed OCI setup guide
│       │
│       ├── arc/                    # Actions Runner Controller
│       │   ├── arc.yaml           # ARC Helm chart config
│       │   └── values.yaml        # ARC customization
│       │
│       ├── runners/                # Container-based runners
│       │   ├── 2cpu-8gb/
│       │   ├── 4cpu-16gb/
│       │   ├── 8cpu-32gb/
│       │   ├── 16cpu-64gb/
│       │   └── 24cpu-384gb/
│       │
│       ├── vm-runners/             # VM-based runners
│       │   ├── 2cpu-8gb/
│       │   ├── 4cpu-16gb/
│       │   ├── 8cpu-32gb/
│       │   ├── 16cpu-64gb/
│       │   ├── 24cpu-96gb/
│       │   └── 32cpu-128gb/
│       │
│       ├── karpenter/              # Node autoscaling
│       │   ├── nodepool.yaml
│       │   └── ocinodeclasss.yaml
│       │
│       ├── external-secrets/       # Secrets management
│       │   ├── external-secrets-operator.yaml
│       │   └── secrets/
│       │
│       ├── monitoring/             # Prometheus & Grafana
│       │   ├── kube-prometheus-stack.yaml
│       │   ├── values.yaml
│       │   └── dashboards/
│       │
│       ├── autoscaler/             # OKE cluster autoscaler
│       │   ├── cluster-autoscaler.yaml
│       │   ├── deployment.yaml
│       │   └── README.MD
│       │
│       └── hacks/                  # Maintenance utilities
│           ├── cgroups-v2-enabler-ds.yaml
│           ├── ephemeralrunner-cleanup-cj.yaml
│           └── vm-cleaner.yaml
│
├── gha-runner-image/               # Custom runner Docker image
│   ├── Dockerfile
│   └── Makefile
│
├── gha-runner-vm/                  # VM runner orchestration
│   ├── main.go
│   └── cloud-init/
│
└── services/
    └── cluster/                    # OKE infrastructure (Terraform)
        ├── cluster.tf
        ├── network.tf
        └── README.md
```

## Deployment Workflow

### Initial Setup (One-Time)

1. Provision OKE Cluster
   ```bash
   cd ci/services/cluster
   make init
   make plan
   make apply
   ```

2. Install ArgoCD
   ```bash
   kubectl apply -k ci/argocd
   ```

3. Configure GitHub Secrets
   - Create GitHub App or PAT
   - Store in External Secrets backend (Oracle Vault)

4. Deploy Applications
   ```bash
   kubectl apply -f ci/cluster/oci/argo-automation.yaml
   ```

### Ongoing Changes (GitOps)

1. Make Configuration Changes
   - Edit files in `ci/cluster/oci/`
   - Commit to Git

2. ArgoCD Auto-Sync
   - ArgoCD detects changes
   - Automatically applies to cluster
   - Slack notification sent

3. Verify Deployment
   - Check ArgoCD UI
   - Monitor runner pods
   - Review metrics in Grafana

## Using the Runners

### In GitHub Workflows

```yaml
name: Build and Test
on: [push, pull_request]

jobs:
  # AMD64 build
  build-x86:
    runs-on: oracle-16cpu-64gb-x86-64
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: make build
      - name: Test
        run: make test

  # ARM64 build
  build-arm:
    runs-on: oracle-16cpu-64gb-arm64
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: make build
      - name: Test
        run: make test

  # Small runner for quick tasks
  lint:
    runs-on: oracle-2cpu-8gb-x86-64
    steps:
      - uses: actions/checkout@v4
      - name: Lint
        run: make lint
```

### Runner Selection Guide

| Use Case | Recommended Runner | Reason |
|----------|-------------------|--------|
| Linting, formatting | `oracle-2cpu-8gb-*` | Fast, lightweight tasks |
| Unit tests | `oracle-4cpu-16gb-*` | Moderate resource needs |
| Integration tests | `oracle-8cpu-32gb-*` | More memory for test databases |
| Full builds | `oracle-16cpu-64gb-*` | Parallel compilation |
| Large builds (e.g., Kubernetes) | `oracle-24cpu-384gb-x86-64` | Memory-intensive builds |
| Multi-arch builds | Both `*-x86-64` and `*-arm64` | Cross-platform testing |

## Security Considerations

### Secrets Management

- Never commit secrets to Git
- Use External Secrets Operator for all sensitive data
- Rotate GitHub tokens regularly
- Use GitHub Apps instead of PATs when possible

### Network Security

- Runners operate in isolated namespaces
- Network policies restrict inter-pod communication
- Egress filtering for external connections

### Runner Isolation

- Ephemeral runners (destroyed after each job)
- No persistent state between jobs
- Clean environment for each workflow

## Monitoring & Observability

### Key Metrics

1. Runner Availability
   - Number of idle runners
   - Queue depth
   - Average wait time

2. Resource Utilization
   - CPU usage per runner
   - Memory consumption
   - Disk I/O

3. Job Performance
   - Job duration
   - Success/failure rates
   - Retry counts

### Accessing Dashboards

```bash
# Grafana
kubectl port-forward -n default svc/kube-prometheus-stack-grafana 3000:80

# ArgoCD
kubectl port-forward -n argocd svc/argocd-server 8080:443
```

### Alerts

Configured alerts for:
- Runner pod failures
- High queue depth
- Resource exhaustion
- Sync failures in ArgoCD

## Maintenance Tasks

### Regular Maintenance

1. Update Runner Images
   - Modify `ci/gha-runner-image/Dockerfile`
   - Build and push new image
   - Update runner manifests

2. Update ARC Version
   - Edit `ci/cluster/oci/arc/arc.yaml`
   - Change `targetRevision`
   - ArgoCD auto-syncs

3. Review Resource Limits
   - Monitor actual usage
   - Adjust requests/limits
   - Optimize costs

### Cleanup Jobs

Automated cleanup via CronJobs:
- Ephemeral Runner Cleanup: Runs every 6 hours
- VM Cleaner: Runs daily
- Old Pod Cleanup: Kubernetes garbage collection

## Common Issues & Solutions

### Issue: Runners Not Picking Up Jobs

Symptoms:
- Jobs queued in GitHub
- No runner pods created

Solutions:
1. Check ARC controller logs
2. Verify GitHub token validity
3. Ensure webhook connectivity

### Issue: Pods Stuck in Pending

Symptoms:
- Runner pods in Pending state
- Jobs timing out

Solutions:
1. Check node availability
2. Verify Karpenter is running
3. Review resource requests

### Issue: ArgoCD Out of Sync

Symptoms:
- Applications show "OutOfSync" status
- Changes not applied

Solutions:
1. Manual sync: `argocd app sync <app-name>`
2. Check for YAML errors
3. Review sync waves

## Learning Resources

### Essential Reading

1. Actions Runner Controller
   - [Official Documentation](https://github.com/actions/actions-runner-controller)
   - [Scaling Strategies](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners/autoscaling-with-self-hosted-runners)

2. ArgoCD
   - [Getting Started](https://argo-cd.readthedocs.io/en/stable/getting_started/)
   - [Best Practices](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)

3. Karpenter
   - [OCI Provider](https://github.com/zoom/karpenter-oci)
   - [Node Provisioning](https://karpenter.sh/docs/concepts/nodepools/)

### Video Tutorials

- [GitOps with ArgoCD](https://www.youtube.com/results?search_query=argocd+tutorial)
- [Self-Hosted GitHub Runners](https://www.youtube.com/results?search_query=github+actions+self+hosted+runners)

## Contributing

### Making Changes

1. Fork the repository
2. Create a feature branch
   ```bash
   git checkout -b feature/improve-runner-config
   ```
3. Make changes
4. Test locally (see `documentation.txt` for KIND testing)
5. Submit PR with detailed description

### PR Guidelines

- Clear description of changes
- Reference related issues
- Include testing steps
- Update documentation if needed

## Support

### Getting Help

- GitHub Issues: [cncf/automation/issues](https://github.com/cncf/automation/issues)
- CNCF Slack: #cncf-ci channel
- Documentation: This repository's README files

### Reporting Issues

Include:
- Description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs/screenshots

---

## Quick Start Checklist

For new contributors or operators:

- [ ] Read this document completely
- [ ] Review `ci/cluster/oci/README.md`
- [ ] Understand ArgoCD sync waves
- [ ] Access OKE cluster with kubectl
- [ ] Verify ArgoCD is running
- [ ] Check runner pods in `arc-systems` namespace
- [ ] Access Grafana dashboards
- [ ] Test a workflow with Oracle runners
- [ ] Review monitoring alerts
- [ ] Understand secrets management

---

Document Version: 1.0  
Last Updated: December 2025  
Maintained By: CNCF Projects Team
