terraform {
  backend "oci" {
    bucket    = "tf-state-bucket"
    namespace = "axtwf1hkrwcy"
    key       = "oke-cncf-services-state/terraform.tfstate"
  }

  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "7.1.0"
    }
  }

  required_version = "~> 1.12"
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

data "oci_identity_availability_domains" "availability_domains" {
  compartment_id = var.compartment_ocid
}
