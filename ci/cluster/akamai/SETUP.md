# Setting Up Linode Kubernetes Engine with OpenTofu

This guide walks you through deploying a Linode Kubernetes Engine (LKE) cluster using OpenTofu.

## Prerequisites

- [OpenTofu](https://opentofu.org/docs/intro/install/) installed
- Linode account with API token
- GitHub Personal Access Token with appropriate permissions

## Installation Steps

### 1. Install OpenTofu

If you haven't already installed OpenTofu, follow these instructions:

```bash
# For Linux/macOS
brew install opentofu/tap/opentofu

# Alternatively, you can download directly from the releases
# https://github.com/opentofu/opentofu/releases
```

### 2. Configure Environment Variables

Create a `.env` file (which is ignored by git) to store your sensitive credentials:

```bash
# Create and edit .env file
touch .env
```

Add the following content to the `.env` file:

```
export TF_VAR_linode_api_token="your-linode-api-token"
export TF_VAR_github_token="your-github-pat"
```

Source the environment variables:

```bash
source .env
```

### 3. Initialize OpenTofu

```bash
cd ci/cluster/akamai
tofu init
```

This will download the necessary providers defined in the configuration.

### 4. Review the Execution Plan

```bash
tofu plan
```

This will show you what resources will be created without actually creating them.

### 5. Apply the Configuration

When you're ready to create the cluster:

```bash
tofu apply
```

Review the planned changes and type `yes` to confirm.

### 6. Access Your Kubernetes Cluster

After successful deployment, OpenTofu will generate a `kubeconfig.yaml` file in the current directory:

```bash
export KUBECONFIG=$(pwd)/kubeconfig.yaml
kubectl get nodes
```

### 7. Verify Actions Runner Controller Installation

Check that ARC is running in the cluster:

```bash
kubectl -n arc-system get pods
```

### 8. Create Runner Scale Sets

After ARC is installed, you can create runner scale sets for your GitHub organizations or repositories:

```bash
kubectl apply -f - <<EOF
apiVersion: actions.summerwind.dev/v1alpha1
kind: RunnerDeployment
metadata:
  name: cncf-runner-deployment
  namespace: arc-system
spec:
  replicas: 1
  template:
    spec:
      organization: your-github-org
      labels:
        - cncf-runner
EOF
```

## Cleaning Up

When you're done with the resources, you can destroy them:

```bash
tofu destroy
```

This will remove all resources created by OpenTofu.

## Troubleshooting

- **Authentication Issues**: Make sure your Linode API token has read/write permissions
- **Connection Issues**: Ensure network connectivity to Linode's API
- **Helm Failures**: Check if Helm is installed and properly configured

For detailed error logs, run commands with increased verbosity:

```bash
tofu apply -v=1
```

For additional support, refer to the [OpenTofu documentation](https://opentofu.org/docs/) or [Linode API documentation](https://www.linode.com/docs/api/).
