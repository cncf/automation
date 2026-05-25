terraform {
  backend "s3" {}

  required_version = ">= 1.6.0"

  required_providers {
    linode = {
      source  = "linode/linode"
      version = "~> 3.13.0"
    }
  }
}

provider "linode" {
  token       = var.linode_api_token
  api_version = "v4beta"
}
