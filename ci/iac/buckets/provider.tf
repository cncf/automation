terraform {
  backend "oci" {}

  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "8.5.0"
    }
  }

  required_version = "~> 1.14"
}

provider "oci" {
  tenancy_ocid        = var.tenancy_ocid
  user_ocid           = var.user_ocid
  region              = var.region
  auth                = var.oci_auth_type
  private_key_path    = var.private_key_path
  fingerprint         = var.fingerprint
  config_file_profile = var.config_file_profile
}
