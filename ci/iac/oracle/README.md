# CNCF Infrastructure as Code

This repository contains the necessary codes for the infrastructure setup on Oracle Cloud Infrastructure (OCI), including OKE clusters and Object Storage buckets with IAM access controls.

## Prerequisites

> [!IMPORTANT]
> You need to make sure that below environment variables are set before running terraform/opentofu commands:
>
> ```bash
> export TF_VAR_compartment_ocid=ocid1.compartment.oc1..xxx
> export TF_VAR_tenancy_ocid=ocid1.tenancy.oc1..xxx
> export TF_VAR_user_ocid=ocid1.user.oc1..xxx
> export TF_VAR_fingerprint=xx:xx
> export TF_VAR_private_key_path=/local/path/to/private/key
> ```

## Managing the OKE cluster

> [!IMPORTANT]
> You must create tfbackend and tfvars files under respective directories with the cluster name.
> Then this value must be used as the `OKE_CLUSTER` varible. e.g.
> 
> ```bash
> touch cluster/tfbackends/my-cluster.tfbackend
> touch cluster/tfvars/my-cluster.tfvars
> OKE_CLUSTER=my-cluster make ...
> ```

```bash
# inits terraform
OKE_CLUSTER=<cluster-name> make cluster-init

# Creates plan.out file
OKE_CLUSTER=<cluster-name> make cluster-plan

# Applies the plan.out file
make cluster-apply

# Destroy all resources
OKE_CLUSTER=<cluster-name> make cluster-destroy

# Get outputs
make cluster-output | jq -r '.cluster.value.kubeconfig'

# Clean terraform directories to work with another cluster
make clean-cluster
```

## Managing Object Storage Buckets

Manages OCI Object Storage buckets along with a dedicated IAM service user for write access (with S3-compatible credentials).

> [!IMPORTANT]
> You must create tfbackend and tfvars files under respective directories with the bucket group name.
> Then this value must be used as the `BUCKETS` variable. e.g.
>
> ```bash
> touch buckets/tfbackends/my-buckets.tfbackend
> touch buckets/tfvars/my-buckets.tfvars
> BUCKETS=my-buckets make ...
> ```

```bash
# inits terraform
BUCKETS=<bucket-group> make buckets-init

# Creates plan.out file
BUCKETS=<bucket-group> make buckets-plan

# Applies the plan.out file
make buckets-apply

# Destroy all resources
BUCKETS=<bucket-group> make buckets-destroy

# Get outputs
make buckets-output

# Retrieve S3-compatible secret key (sensitive)
terraform -chdir=./buckets output -raw s3_compatible_secret_key

# Clean terraform directories
make clean-buckets
```

## Clean up

```bash
# Clean all terraform directories
make clean

# Or clean individually
make clean-cluster
make clean-buckets
```
