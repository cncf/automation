# GitHub Actions Self-Hosted Runners on Akamai/Linode

OpenTofu configuration for deploying GitHub Actions self-hosted runners on Akamai infrastructure using Linode Kubernetes Engine (LKE).

## Overview

This infrastructure-as-code provisions a Linode Kubernetes Engine cluster and deploys Actions Runner Controller (ARC) to manage GitHub Actions self-hosted runners. OpenTofu is used as the open source, Linux Foundation-maintained infrastructure-as-code tool.

## Prerequisites

**Required Tools:**
- [OpenTofu](https://opentofu.org/docs/intro/install/) v1.6.0+
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

**Required Credentials:**
- Linode API Token with read/write permissions
- GitHub Personal Access Token with `admin:org` scope (required for managing organization runners)

## Configuration

### Using Environment Variables

The recommended approach is to use environment variables:

```bash
export TF_VAR_linode_api_token="your-linode-token"
export TF_VAR_github_token="your-github-pat"
export TF_VAR_github_organization="your-org-name"
```

### Using terraform.tfvars File

Alternatively, copy `terraform.tfvars.example` to `terraform.tfvars` and fill in your values:

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values
```

**Required Variables:**
- `linode_api_token` - Linode API token
- `github_token` - GitHub PAT for runner registration
- `github_organization` - GitHub organization name

**Optional Variables with Defaults:**
- `cluster_name` - Cluster name (default: "github-runners")
- `kubernetes_version` - Kubernetes version (default: "1.34")
- `region` - Linode region (default: "us-east")
- `node_type` - Instance type (default: "g6-standard-1")
- `node_count` - Initial node count (default: 1)
- `autoscaler_min` - Minimum nodes (default: 1)
- `autoscaler_max` - Maximum nodes (default: 3)
- `environment` - Environment name (default: "dev")
- `arc_version` - ARC Helm chart version (default: "0.23.7")

See [variables.tf](variables.tf) for complete variable documentation.

## Deployment

### Phase 1: Initialize and Deploy Cluster

```bash
# Initialize OpenTofu
tofu init

# Validate configuration
tofu validate

# Deploy cluster and kubeconfig
tofu apply -target=linode_lke_cluster.github_runners -target=local_file.kubeconfig
```

### Phase 2: Deploy Kubernetes Resources

```bash
# Deploy namespace, secrets, and ARC controller
tofu apply -target=kubernetes_namespace.arc_system \
           -target=kubernetes_secret.github_token \
           -target=helm_release.arc
```

### Phase 3: Deploy Runner Scale Set

```bash
# Deploy runner deployment (requires ARC CRDs to be installed)
tofu apply
```

### Access the Cluster

```bash
export KUBECONFIG=$(pwd)/kubeconfig.yaml
kubectl get nodes
kubectl get pods -n arc-system
```

## Multi-Phase Deployment Requirement

The deployment must be executed in multiple phases due to OpenTofu provider dependencies:

1. **Phase 1**: Create the LKE cluster and generate kubeconfig file
   - The Kubernetes and Helm providers require the cluster to exist
   - The kubeconfig is needed for provider configuration

2. **Phase 2**: Deploy Kubernetes namespace, secrets, and ARC Helm chart
   - Requires the cluster from Phase 1
   - Installs ARC Custom Resource Definitions (CRDs)

3. **Phase 3**: Deploy runner scale set manifest
   - Requires ARC CRDs to be installed from Phase 2
   - Creates the actual runner deployment

This phased approach ensures proper dependency resolution and successful deployment.

## What Gets Deployed

- **LKE Cluster**: Managed Kubernetes cluster with auto-scaling nodes
- **Actions Runner Controller**: Helm chart deployed in `arc-system` namespace
- **Runner Scale Set**: GitHub Actions runners managed as Kubernetes pods
- **Kubeconfig**: Local file for cluster access (automatically generated)

## Viewing Outputs

After successful deployment, view cluster information:

```bash
tofu output
```

Available outputs:
- `cluster_id` - Unique identifier of the LKE cluster
- `api_endpoints` - Kubernetes API endpoints
- `cluster_status` - Operational status
- `kubeconfig_path` - Path to generated kubeconfig file
- `region` - Deployment region
- `k8s_version` - Kubernetes version

## Cleanup

To destroy all infrastructure:

```bash
tofu destroy
```

This will remove all resources including the cluster, runner controller, and associated resources.

## Cost Optimization

This configuration uses minimal resources for cost-effectiveness:
- Instance type: `g6-standard-1` (2 vCPU, 2GB RAM)
- Node count: 1 (minimum), auto-scaling up to 3
- Single availability zone deployment

For production workloads, consider adjustments for:
- Performance requirements (larger instance types)
- High availability (multiple nodes, multi-zone)
- Security requirements (private networking, ACLs)

Estimated costs vary by region and usage. Check current [Linode pricing](https://www.linode.com/pricing/) for accurate estimates.

## Troubleshooting

### Runners Not Registering

If runners show `0 CURRENT` in the deployment:

```bash
# Check ARC controller logs
kubectl logs -n arc-system -l app.kubernetes.io/name=actions-runner-controller

# Common issue: GitHub token lacks admin:org scope
# Solution: Create a new token with admin:org scope and redeploy
export TF_VAR_github_token="new-token-with-admin-org-scope"
tofu apply
```

### Provider Configuration Errors

If you encounter provider initialization errors:
- Ensure Phase 1 completes successfully before Phase 2
- Verify the kubeconfig file was created: `ls -l kubeconfig.yaml`
- Check file permissions on kubeconfig: `stat kubeconfig.yaml`

### Helm Chart Installation Failures

If ARC Helm chart fails to install:
- Verify Kubernetes cluster is running: `kubectl get nodes`
- Check namespace exists: `kubectl get namespaces`
- Review Helm release status: `kubectl get all -n arc-system`

## Security Best Practices

**State File Management:**
- Terraform state files contain sensitive credentials including API tokens and cluster credentials
- Never commit state files to version control (already in `.gitignore`)
- For production environments, use remote state backends with encryption
- Consider using S3 with encryption, Terraform Cloud, or HashiCorp Consul

**Credential Management:**
- Use environment variables for credentials (do not commit in terraform.tfvars)
- Rotate API tokens and GitHub PATs regularly
- Use GitHub fine-grained tokens with minimal required permissions
- Store credentials in secure secret management systems
- Avoid using long-lived credentials in production

**Access Control:**
- Apply principle of least privilege to all API tokens
- Use separate credentials for dev/staging/prod environments
- Enable audit logging on cloud provider accounts
- Review access logs and runner activity regularly

**Network Security:**
- Configure firewall rules to restrict cluster access
- Use private networking where possible
- Enable LKE control plane ACLs for production clusters
- Review and implement Linode security best practices

**Production Deployment Considerations:**
- Enable high availability for the control plane
- Implement Pod Security Standards and network policies
- Configure resource quotas and limits
- Set up comprehensive monitoring and alerting
- Implement backup and disaster recovery procedures
- Use separate node pools for different workload types

## Architecture

**Components:**
- **Linode LKE Cluster**: Managed Kubernetes with auto-scaling (1-3 nodes by default)
- **Actions Runner Controller**: Manages GitHub Actions runners as Kubernetes pods
- **Runner Deployment**: Scales runners based on GitHub Actions workload
- **Kubeconfig**: Automatically generated for cluster access with 0600 permissions

**Resource Dependencies:**
```
linode_lke_cluster.github_runners
    ├── local_file.kubeconfig
    ├── kubernetes_namespace.arc_system
    │   ├── kubernetes_secret.github_token
    │   └── helm_release.arc
    │       └── kubernetes_manifest.runner_scale_set
```

## Provider Versions

This configuration uses the following provider versions:
- Linode provider: `~> 2.41.0`
- Helm provider: `~> 2.17.0`
- Kubernetes provider: `~> 2.38.0`

These are pinned to minor versions for stability while allowing patch updates.
