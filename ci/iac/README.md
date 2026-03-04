# CNCF Infrastructure as Code

This repository contains the necessary codes for the infrastructure setup of the OKE (Oracle Kubernetes Engine).

## Managing the OKE cluster

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

> [!IMPORTANT]
> You must create tfbackend and tfvars files under respective directories with the cluster name.
> Then this value must be used as the `OKE_CLUSTER` varible. e.g.
> 
> ```bash
> touch tfbackends/my-cluster.tfbackend
> touch tfvars/my-cluster.tfvars
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
make clean
```
