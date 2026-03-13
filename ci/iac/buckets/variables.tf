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

variable "replica_region" {
  type        = string
  description = "OCI region for bucket replication."
  default     = "us-ashburn-1"
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

variable "namespace" {
  type        = string
  description = "OCI Object Storage namespace."
}

variable "buckets" {
  type = map(object({
    access_type  = optional(string, "NoPublicAccess")
    storage_tier = optional(string, "Standard")
    versioning   = optional(string, "Disabled")
  }))
  description = "Map of bucket names to their configuration."
}

variable "service_user_name" {
  type        = string
  description = "Name of the IAM user for service access."
}

variable "service_user_description" {
  type        = string
  description = "Description for the IAM service user."
  default     = "Service user for Object Storage write access"
}

variable "service_user_email" {
  type        = string
  description = "Email address for the IAM service user."
}
