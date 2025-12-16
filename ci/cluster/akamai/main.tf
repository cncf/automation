# OpenTofu configuration
# This configuration is compatible with OpenTofu 1.6.0+ and Terraform 1.6.0+
terraform {
  required_version = ">= 1.6.0"

  required_providers {
    # Linode provider - pinned to minor version for stability
    linode = {
      source  = "linode/linode"
      version = "~> 2.41.0"
    }
    # Helm provider - pinned to minor version for stability
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17.0"
    }
    # Kubernetes provider - pinned to minor version for stability
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.38.0"
    }
  }
}

# Local values for kubeconfig parsing and common resource tagging
locals {
  # Decode the base64-encoded kubeconfig from Linode LKE
  kubeconfig_decoded = yamldecode(base64decode(linode_lke_cluster.github_runners.kubeconfig))

  # Extract Kubernetes cluster credentials for provider configuration
  k8s_host    = local.kubeconfig_decoded.clusters[0].cluster.server
  k8s_ca_cert = base64decode(local.kubeconfig_decoded.clusters[0].cluster.certificate-authority-data)
  k8s_token   = local.kubeconfig_decoded.users[0].user.token

  # Common tags applied to all Linode resources
  common_tags = [
    "github-actions-runners",
    "environment:${var.environment}"
  ]
}

# Linode Kubernetes Engine cluster for GitHub Actions runners
resource "linode_lke_cluster" "github_runners" {
  label       = var.cluster_name
  k8s_version = var.kubernetes_version
  region      = var.region
  tags        = local.common_tags

  pool {
    type  = var.node_type
    count = var.node_count

    autoscaler {
      min = var.autoscaler_min
      max = var.autoscaler_max
    }
  }
}

# Write kubeconfig to local file for kubectl access
# WARNING: This file contains sensitive credentials and is ignored by .gitignore
resource "local_file" "kubeconfig" {
  content         = base64decode(linode_lke_cluster.github_runners.kubeconfig)
  filename        = "${path.module}/kubeconfig.yaml"
  file_permission = "0600" # Restrict to owner read/write only
}

# Create namespace for Actions Runner Controller (ARC)
resource "kubernetes_namespace" "arc_system" {
  metadata {
    name = "arc-system"
  }

  depends_on = [linode_lke_cluster.github_runners]
}

# Create Kubernetes secret containing GitHub PAT for runner authentication
resource "kubernetes_secret" "github_token" {
  metadata {
    name      = "github-token"
    namespace = kubernetes_namespace.arc_system.metadata[0].name
  }

  data = {
    github_token = var.github_token
  }

  type = "Opaque"

  depends_on = [kubernetes_namespace.arc_system]
}

# Install Actions Runner Controller (ARC) via Helm chart
# ARC manages the lifecycle of GitHub Actions self-hosted runners as Kubernetes pods
resource "helm_release" "arc" {
  name       = "arc"
  repository = "https://actions-runner-controller.github.io/actions-runner-controller"
  chart      = "actions-runner-controller"
  namespace  = kubernetes_namespace.arc_system.metadata[0].name
  version    = var.arc_version

  # Disable cert-manager dependency for simplified deployment
  set {
    name  = "certManagerEnabled"
    value = "false"
  }

  # Use the separately created GitHub token secret
  set {
    name  = "authSecret.create"
    value = "false"
  }

  set {
    name  = "authSecret.name"
    value = kubernetes_secret.github_token.metadata[0].name
  }

  depends_on = [
    kubernetes_secret.github_token,
    linode_lke_cluster.github_runners
  ]
}

# Deploy a runner scale set for the GitHub organization
# This creates actual runner pods that will register with GitHub Actions
resource "kubernetes_manifest" "runner_scale_set" {
  manifest = {
    apiVersion = "actions.summerwind.dev/v1alpha1"
    kind       = "RunnerDeployment"
    metadata = {
      name      = "runner-deployment"
      namespace = kubernetes_namespace.arc_system.metadata[0].name
    }
    spec = {
      replicas = 1
      template = {
        spec = {
          organization = var.github_organization
          labels       = ["self-hosted", "akamai", "linode"]
        }
      }
    }
  }

  depends_on = [helm_release.arc]
}
