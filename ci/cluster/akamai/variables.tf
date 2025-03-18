variable "cluster_name" {
  description = "The name of the LKE cluster"
  type        = string
  default     = "cncf-poc-runners"
}

variable "kubernetes_version" {
  description = "The Kubernetes version to use for the cluster"
  type        = string
  default     = "1.32" # Updated to the most recent version
}

variable "region" {
  description = "The region where the cluster will be deployed"
  type        = string
  default     = "us-east" # Choose an appropriate region
}

variable "node_count" {
  description = "The number of nodes in the cluster"
  type        = number
  default     = 1 # Start with minimal resources for PoC
}

variable "github_token" {
  description = "GitHub Personal Access Token for Actions Runner Controller"
  type        = string
  sensitive   = true
}

variable "linode_api_token" {
  description = "Linode API Token"
  type        = string
  sensitive   = true
}

variable "github_organization" {
  description = "GitHub organization for runners"
  type        = string
  default     = "cncf"
}
