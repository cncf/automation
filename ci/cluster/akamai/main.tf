terraform {
  required_providers {
    linode = {
      source  = "linode/linode"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.9"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.21"
    }
  }
}

resource "linode_lke_cluster" "cncf_poc_cluster" {
  label       = var.cluster_name
  k8s_version = var.kubernetes_version
  region      = var.region
  tags        = ["cncf", "poc", "runners"]

  pool {
    type  = "g6-standard-1" # Cheapest shared CPU instance type
    count = var.node_count
    
    # Use autoscaler for spot instances if available
    autoscaler {
      min = 1
      max = 3
    }
  }
}

resource "local_file" "kubeconfig" {
  content  = base64decode(linode_lke_cluster.cncf_poc_cluster.kubeconfig)
  filename = "${path.module}/kubeconfig.yaml"
}

# Configure Kubernetes provider with the cluster's kubeconfig
provider "kubernetes" {
  host                   = yamldecode(base64decode(linode_lke_cluster.cncf_poc_cluster.kubeconfig)).clusters[0].cluster.server
  cluster_ca_certificate = base64decode(yamldecode(base64decode(linode_lke_cluster.cncf_poc_cluster.kubeconfig)).clusters[0].cluster.certificate-authority-data)
  token                  = yamldecode(base64decode(linode_lke_cluster.cncf_poc_cluster.kubeconfig)).users[0].user.token
}

# Configure Helm provider with the cluster's kubeconfig
provider "helm" {
  kubernetes {
    host                   = yamldecode(base64decode(linode_lke_cluster.cncf_poc_cluster.kubeconfig)).clusters[0].cluster.server
    cluster_ca_certificate = base64decode(yamldecode(base64decode(linode_lke_cluster.cncf_poc_cluster.kubeconfig)).clusters[0].cluster.certificate-authority-data)
    token                  = yamldecode(base64decode(linode_lke_cluster.cncf_poc_cluster.kubeconfig)).users[0].user.token
  }
}

# Create namespace for ARC
resource "kubernetes_namespace" "arc_system" {
  metadata {
    name = "arc-system"
  }

  depends_on = [linode_lke_cluster.cncf_poc_cluster]
}

# Create secret for GitHub token
resource "kubernetes_secret" "github_token" {
  metadata {
    name      = "github-token"
    namespace = kubernetes_namespace.arc_system.metadata[0].name
  }

  data = {
    github_token = var.github_token
  }

  depends_on = [kubernetes_namespace.arc_system]
}

# Install Actions Runner Controller using Helm
resource "helm_release" "arc" {
  name       = "arc"
  repository = "https://actions-runner-controller.github.io/actions-runner-controller"
  chart      = "actions-runner-controller"
  namespace  = kubernetes_namespace.arc_system.metadata[0].name
  version    = "0.23.0"  # Specify the version you want to use

  set {
    name  = "authSecret.create"
    value = "false"  # We create the secret separately
  }

  set {
    name  = "authSecret.name"
    value = kubernetes_secret.github_token.metadata[0].name
  }

  depends_on = [
    kubernetes_secret.github_token,
    linode_lke_cluster.cncf_poc_cluster
  ]
}

# Optional: Deploy a runner scale set
resource "kubernetes_manifest" "runner_scale_set" {
  manifest = {
    apiVersion = "actions.summerwind.dev/v1alpha1"
    kind       = "RunnerDeployment"
    metadata = {
      name      = "cncf-runner-deployment"
      namespace = kubernetes_namespace.arc_system.metadata[0].name
    }
    spec = {
      replicas = 1
      template = {
        spec = {
          organization = var.github_organization
          labels       = ["cncf-runner"]
        }
      }
    }
  }

  depends_on = [helm_release.arc]
}
