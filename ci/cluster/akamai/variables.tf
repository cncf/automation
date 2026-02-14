# Input variables for configuring the GitHub Actions runners infrastructure
# These can be set via environment variables (TF_VAR_*) or terraform.tfvars file

# Cluster Configuration
variable "cluster_name" {
  description = "The name of the LKE cluster"
  type        = string
  default     = "github-runners"

  validation {
    condition     = length(var.cluster_name) > 0 && length(var.cluster_name) <= 32
    error_message = "Cluster name must be between 1 and 32 characters."
  }
}

variable "environment" {
  description = "Environment name (e.g., dev, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "kubernetes_version" {
  description = "The Kubernetes version to use for the cluster"
  type        = string
  default     = "1.34"

  validation {
    condition     = can(regex("^[0-9]+\\.[0-9]+$", var.kubernetes_version))
    error_message = "Kubernetes version must be in format X.Y (e.g., 1.34)."
  }
}

variable "region" {
  description = "The region where the cluster will be deployed"
  type        = string
  default     = "us-east"

  validation {
    condition     = length(var.region) > 0
    error_message = "Region must be specified."
  }
}

variable "node_count" {
  description = "The initial number of nodes in the cluster"
  type        = number
  default     = 1

  validation {
    condition     = var.node_count >= 1 && var.node_count <= 100
    error_message = "Node count must be between 1 and 100."
  }
}

# Credentials and Authentication
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

  validation {
    condition     = length(var.github_organization) > 0
    error_message = "GitHub organization must be specified."
  }
}

# Node Pool Configuration
variable "node_type" {
  description = "Linode instance type for cluster nodes"
  type        = string
  default     = "g6-standard-1"
}

variable "autoscaler_min" {
  description = "Minimum number of nodes for autoscaler"
  type        = number
  default     = 1

  validation {
    condition     = var.autoscaler_min >= 1
    error_message = "Autoscaler minimum must be at least 1."
  }
}

variable "autoscaler_max" {
  description = "Maximum number of nodes for autoscaler"
  type        = number
  default     = 3

  validation {
    condition     = var.autoscaler_max >= var.autoscaler_min
    error_message = "Autoscaler maximum must be greater than or equal to minimum."
  }
}

# Actions Runner Controller Configuration
variable "arc_version" {
  description = "Version of Actions Runner Controller Helm chart"
  type        = string
  default     = "0.23.7"

  validation {
    condition     = can(regex("^[0-9]+\\.[0-9]+\\.[0-9]+$", var.arc_version))
    error_message = "ARC version must be in semantic versioning format (X.Y.Z)."
  }
}
