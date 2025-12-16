# Linode provider for managing LKE infrastructure
provider "linode" {
  token = var.linode_api_token
}

# Kubernetes provider for managing cluster resources
# Configured to use credentials from the LKE cluster kubeconfig
provider "kubernetes" {
  host                   = local.k8s_host
  cluster_ca_certificate = local.k8s_ca_cert
  token                  = local.k8s_token
}

# Helm provider for deploying charts (Actions Runner Controller)
# Configured to use credentials from the LKE cluster kubeconfig
provider "helm" {
  kubernetes {
    host                   = local.k8s_host
    cluster_ca_certificate = local.k8s_ca_cert
    token                  = local.k8s_token
  }
}
