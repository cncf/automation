// Start: OCI provider Variables
variable "tenancy_ocid" {
  type        = string
  description = "The OCID of the Oracle Cloud Infrastructure tenancy where resources will be created."
}

variable "compartment_ocid" {
  type        = string
  description = "The OCID of the compartment in which OCI resources will be managed."
}

variable "user_ocid" {
  type        = string
  description = "The OCID of the OCI user associated with the API key used for authentication."
}

variable "private_key_path" {
  type        = string
  description = "Path to the private key file used for OCI API key authentication."
  sensitive   = true
}

variable "region" {
  type        = string
  description = "OCI region where resources will be deployed."
  default     = "us-sanjose-1"
}

variable "fingerprint" {
  type        = string
  description = "Fingerprint of the public key uploaded to the OCI user account for API authentication."
}

variable "config_file_profile" {
  type        = string
  description = "Profile name from the OCI CLI configuration file (~/.oci/config) to use for authentication."
  default     = "DEFAULT"
}

variable "oci_auth_type" {
  type        = string
  description = "Authentication method used by the OCI provider (e.g., APIKey, InstancePrincipal, ResourcePrincipal)."
  default     = "APIKey"
}
// End: OCI provider variables

variable "cluster_name" {
  type        = string
  description = "Name of the OKE cluster (also used for networking and other resources)"
}

variable "node_pool_worker_size" {
  type        = number
  description = "Default number of worker nodes"
}

variable "kubernetes_version" {
  type        = string
  description = "Kubernetes version for OKE"
}

variable "cluster_autoscaler_min" {
  type        = number
  description = "Minimum number of nodes for the cluster autoscaler"
}

variable "cluster_autoscaler_max" {
  type        = number
  description = "Maximum number of nodes for the cluster autoscaler"
}

variable "oke_node_shape" {
  type        = string
  description = "OKE Nodepool node shape"
}

variable "oke_node_memory" {
  type        = number
  description = "OKE worker node memory in GBs"
}

variable "oke_node_cpu" {
  type        = number
  description = "OKE worker node CPUs"
}

variable "oke_node_boot_volume_size" {
  type        = number
  description = "The size of the boot volume in GBs"
  default     = 50
}

variable "vcn_cidr" {
  type        = string
  description = "CIDR for the VCN"
}

variable "k8s_api_cidr" {
  type        = string
  description = "CIDR for the Kubernetes API network"
}

variable "svc_cidr" {
  type        = string
  description = "CIDR for the Service Network"
}

variable "node_cidr" {
  type        = string
  description = "CIDR for the worker nodes network"
}

variable "svc_lb_egress_rules" {
  description = "Egress security rules for the service LB security list"
  type = list(object({
    description      = string
    destination      = string
    destination_type = string
    protocol         = string
    stateless        = bool
    tcp_min          = optional(number)
    tcp_max          = optional(number)
    icmp_type        = optional(number)
    icmp_code        = optional(number)
  }))
  default = []
}

variable "svc_lb_ingress_rules" {
  description = "Ingress security rules for the service LB security list"
  type = list(object({
    description = string
    source      = string
    source_type = string
    protocol    = string
    stateless   = bool
    tcp_min     = optional(number)
    tcp_max     = optional(number)
    icmp_type   = optional(number)
    icmp_code   = optional(number)
  }))
  default = []
}

variable "k8s_api_endpoint_egress_rules" {
  description = "Egress security rules for the Kubernetes API Endpoint security list"
  type = list(object({
    description      = string
    destination      = string
    destination_type = string
    protocol         = string
    stateless        = bool
    tcp_min          = optional(number)
    tcp_max          = optional(number)
    icmp_type        = optional(number)
    icmp_code        = optional(number)
  }))
  default = []
}

variable "k8s_api_endpoint_ingress_rules" {
  description = "Ingress security rules for the Kubernetes API Endpoint security list"
  type = list(object({
    description = string
    source      = string
    source_type = string
    protocol    = string
    stateless   = bool
    tcp_min     = optional(number)
    tcp_max     = optional(number)
    icmp_type   = optional(number)
    icmp_code   = optional(number)
  }))
  default = []
}

variable "node_egress_rules" {
  description = "Egress security rules for the worker node security list"
  type = list(object({
    description      = string
    destination      = string
    destination_type = string
    protocol         = string
    stateless        = bool
    tcp_min          = optional(number)
    tcp_max          = optional(number)
    icmp_type        = optional(number)
    icmp_code        = optional(number)
  }))
  default = []
}

variable "node_ingress_rules" {
  description = "Ingress security rules for the worker node security list"
  type = list(object({
    description = string
    source      = string
    source_type = string
    protocol    = string
    stateless   = bool
    tcp_min     = optional(number)
    tcp_max     = optional(number)
    icmp_type   = optional(number)
    icmp_code   = optional(number)
  }))
  default = []
}

variable "deploy_kcp" {
  type        = bool
  description = "Deploy KCP to the cluster and create LB IP for it"
  default     = false
}

variable "ingress_private_ip_id" {
  type    = string
  default = ""
}

variable "kcp_lb_private_ip_id" {
  type    = string
  default = ""
}
