## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | ~> 1.14 |
| <a name="requirement_oci"></a> [oci](#requirement\_oci) | 8.5.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_oci"></a> [oci](#provider\_oci) | 8.5.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [oci_identity_customer_secret_key.service_user_s3_key](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/identity_customer_secret_key) | resource |
| [oci_identity_group.service_group](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/identity_group) | resource |
| [oci_identity_policy.bucket_write_policy](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/identity_policy) | resource |
| [oci_identity_user.service_user](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/identity_user) | resource |
| [oci_identity_user_group_membership.service_membership](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/identity_user_group_membership) | resource |
| [oci_objectstorage_bucket.buckets](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/objectstorage_bucket) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_buckets"></a> [buckets](#input\_buckets) | Map of bucket names to their configuration. | <pre>map(object({<br/>    access_type  = optional(string, "NoPublicAccess")<br/>    storage_tier = optional(string, "Standard")<br/>    versioning   = optional(string, "Disabled")<br/>  }))</pre> | n/a | yes |
| <a name="input_compartment_ocid"></a> [compartment\_ocid](#input\_compartment\_ocid) | The OCID of the compartment in which OCI resources will be managed. | `string` | n/a | yes |
| <a name="input_config_file_profile"></a> [config\_file\_profile](#input\_config\_file\_profile) | Profile name from the OCI CLI configuration file (~/.oci/config) to use for authentication. | `string` | `"DEFAULT"` | no |
| <a name="input_fingerprint"></a> [fingerprint](#input\_fingerprint) | Fingerprint of the public key uploaded to the OCI user account for API authentication. | `string` | n/a | yes |
| <a name="input_namespace"></a> [namespace](#input\_namespace) | OCI Object Storage namespace. | `string` | n/a | yes |
| <a name="input_oci_auth_type"></a> [oci\_auth\_type](#input\_oci\_auth\_type) | Authentication method used by the OCI provider (e.g., APIKey, InstancePrincipal, ResourcePrincipal). | `string` | `"APIKey"` | no |
| <a name="input_private_key_path"></a> [private\_key\_path](#input\_private\_key\_path) | Path to the private key file used for OCI API key authentication. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | OCI region where resources will be deployed. | `string` | `"us-sanjose-1"` | no |
| <a name="input_service_user_description"></a> [service\_user\_description](#input\_service\_user\_description) | Description for the IAM service user. | `string` | `"Service user for Object Storage write access"` | no |
| <a name="input_service_user_email"></a> [service\_user\_email](#input\_service\_user\_email) | Email address for the IAM service user. | `string` | n/a | yes |
| <a name="input_service_user_name"></a> [service\_user\_name](#input\_service\_user\_name) | Name of the IAM user for service access. | `string` | n/a | yes |
| <a name="input_tenancy_ocid"></a> [tenancy\_ocid](#input\_tenancy\_ocid) | The OCID of the Oracle Cloud Infrastructure tenancy where resources will be created. | `string` | n/a | yes |
| <a name="input_user_ocid"></a> [user\_ocid](#input\_user\_ocid) | The OCID of the OCI user associated with the API key used for authentication. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_buckets"></a> [buckets](#output\_buckets) | Map of created bucket names to their details. |
| <a name="output_s3_compatible_access_key"></a> [s3\_compatible\_access\_key](#output\_s3\_compatible\_access\_key) | S3-compatible access key ID for the service user. |
| <a name="output_s3_compatible_secret_key"></a> [s3\_compatible\_secret\_key](#output\_s3\_compatible\_secret\_key) | S3-compatible secret key for the service user. |
| <a name="output_service_group"></a> [service\_group](#output\_service\_group) | Service group details. |
| <a name="output_service_user"></a> [service\_user](#output\_service\_user) | Service user details. |