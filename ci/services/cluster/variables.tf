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
  type    = string
  default = "oke-cncf-services"
}

variable "node_pool_worker_size" {
  type    = number
  default = 3
}

variable "kubernetes_version" {
  type    = string
  default = "v1.34.1"
}

variable "cluster_autoscaler_min" {
  type    = number
  default = 3
}

variable "cluster_autoscaler_max" {
  type    = number
  default = 10
}

variable "oke_node_shape" {
  type        = string
  description = "OKE Nodepool node shape"
  default     = "VM.Standard.E4.Flex"
}

variable "oke_node_memory" {
  type        = number
  description = "OKE worker node memory in GBs"
  default     = 32
}

variable "oke_node_cpu" {
  type        = number
  description = "OKE worker node CPUs"
  default     = 8
}
