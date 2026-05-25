# CNCF Infrastructure as Code - Akamai/Linode

This directory contains the necessary codes for the infrastructure setup on Akamai Cloud (Linode), including LKE clusters with VPC networking and cloud firewall.

## Prerequisites

> [!IMPORTANT]
> You need to make sure that below environment variables are set before running terraform/opentofu commands:
>
> ```bash
> export TF_VAR_linode_api_token=<your-linode-api-token>
> ```
>
> For the S3 backend, set the backend credentials:
>
> ```bash
> export AWS_ACCESS_KEY_ID=<your-access-key>
> export AWS_SECRET_ACCESS_KEY=<your-secret-key>
> ```

## Managing the LKE cluster

> [!IMPORTANT]
> You must create tfbackend and tfvars files under respective directories with the cluster name.
> Then this value must be used as the `LKE_CLUSTER` variable. e.g.
>
> ```bash
> touch cluster/tfbackends/my-cluster.tfbackend
> touch cluster/tfvars/my-cluster.tfvars
> LKE_CLUSTER=my-cluster make ...
> ```

```bash
# inits terraform
LKE_CLUSTER=<cluster-name> make cluster-init

# Creates plan.out file
LKE_CLUSTER=<cluster-name> make cluster-plan

# Applies the plan.out file
make cluster-apply

# Destroy all resources
LKE_CLUSTER=<cluster-name> make cluster-destroy

# Get outputs
make cluster-output | jq -r '.kubeconfig.value'

# Clean terraform directories to work with another cluster
make clean-cluster
```

## Clean up

```bash
# Clean all terraform directories
make clean

# Or clean individually
make clean-cluster
```
