terraform {
  required_providers {
    oci = {
      source  = "oracle/oci"
      version = "6.33.0"
    }
  }

  # make sure to set AWS_ENDPOINT_URL_S3 to 'https://<object namespace>.compat.objectstorage.us-sanjose-1.oraclecloud.com'.
  backend "s3" {
    bucket = "cncf-service-opentofu-state"
    region = "us-sanjose-1"
    key    = "cncf-service/tf.tfstate"

    skip_region_validation      = true
    skip_credentials_validation = true
    skip_requesting_account_id  = true
    use_path_style              = true
    skip_metadata_api_check     = true
  }
}