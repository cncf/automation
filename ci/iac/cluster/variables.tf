// Start: OCI provider Variables
variable "tenancy_ocid" {
  type = string
}

variable "compartment_ocid" {
  type = string
}

variable "user_ocid" {
  type = string
}

variable "private_key_path" {
  type      = string
  sensitive = true
}

variable "region" {
  type    = string
  default = "us-sanjose-1"
}

variable "fingerprint" {
  type = string
}

variable "config_file_profile" {
  type    = string
  default = "DEFAULT"
}

variable "oci_auth_type" {
  type    = string
  default = "APIKey"
}
// End: OCI provider variables

variable "cluster_name" {
  type = string
}

variable "node_pool_worker_size" {
  type = number
}

variable "kubernetes_version" {
  type = string
}

variable "cluster_autoscaler_min" {
  type = number
}

variable "cluster_autoscaler_max" {
  type = number
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