variable "oci_tenant_ocid" {
  type = string
}

variable "oci_compartment_ocid" {
  type = string
}

/*
variable "oci_user_ocid" {
  type = string
}

variable "oci_private_key" {
  type      = string
  sensitive = true
}
*/

variable "oci_region" {
  type = string
}

variable "node_pool_ssh_public_key" {
  type = string
}

variable "node_pool_worker_size" {
  type    = number
  default = 3
}

variable "kubernetes_version" {
  type    = string
  default = "v1.31.1"
}

variable "oci_config_file_profile" {
  type    = string
  default = "DEFAULT"
}

variable "oci_auth_type" {
  type    = string
  default = "SecurityToken"
}