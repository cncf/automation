# CNCF Services Cluster Infrastructure

This repository contains the necessary codes for the infrastructure setup of the OKE (Oracle Kubernetes Engine).

## Creating/updating OKE cluster

> [!IMPORTANT]
> You need to make sure that below environment variables are set before running terraform/opentofu commands:
>
> ```bash
> export TF_VAR_compartment_ocid=ocid1.compartment.oc1..xxx
> export TF_VAR_tenancy_ocid=ocid1.tenancy.oc1..xxx
> export TF_VAR_user_ocid=ocid1.user.oc1..xxx
> export TF_VAR_fingerprint=xx:xx
> export TF_VAR_private_key_path=/local/path/to/private/key
> ```

```bash
# inits terraform
make init

# Creates plan.out file
make plan

# Applies the plan.out file
make apply
```

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | ~> 1.12 |
| <a name="requirement_oci"></a> [oci](#requirement\_oci) | 7.1.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_oci"></a> [oci](#provider\_oci) | 7.1.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [oci_containerengine_addon.cluster_autoscaler](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/containerengine_addon) | resource |
| [oci_containerengine_cluster.service](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/containerengine_cluster) | resource |
| [oci_containerengine_node_pool.service_worker](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/containerengine_node_pool) | resource |
| [oci_core_internet_gateway.service](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_internet_gateway) | resource |
| [oci_core_nat_gateway.service](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_nat_gateway) | resource |
| [oci_core_public_ip.ingress_ip](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_public_ip) | resource |
| [oci_core_route_table.private](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_route_table) | resource |
| [oci_core_route_table.public](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_route_table) | resource |
| [oci_core_security_list.k8s_api_endpoint](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_security_list) | resource |
| [oci_core_security_list.node](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_security_list) | resource |
| [oci_core_security_list.svc_lb](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_security_list) | resource |
| [oci_core_service_gateway.service](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_service_gateway) | resource |
| [oci_core_subnet.k8s_api_endpoint](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_subnet) | resource |
| [oci_core_subnet.node](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_subnet) | resource |
| [oci_core_subnet.svc_lb](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_subnet) | resource |
| [oci_core_vcn.service](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/resources/core_vcn) | resource |
| [oci_containerengine_cluster_kube_config.service](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/data-sources/containerengine_cluster_kube_config) | data source |
| [oci_core_services.services](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/data-sources/core_services) | data source |
| [oci_identity_availability_domains.availability_domains](https://registry.terraform.io/providers/oracle/oci/7.1.0/docs/data-sources/identity_availability_domains) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_autoscaler_max"></a> [cluster\_autoscaler\_max](#input\_cluster\_autoscaler\_max) | n/a | `number` | `10` | no |
| <a name="input_cluster_autoscaler_min"></a> [cluster\_autoscaler\_min](#input\_cluster\_autoscaler\_min) | n/a | `number` | `3` | no |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | n/a | `string` | `"oke-cncf-services"` | no |
| <a name="input_compartment_ocid"></a> [compartment\_ocid](#input\_compartment\_ocid) | n/a | `string` | n/a | yes |
| <a name="input_config_file_profile"></a> [config\_file\_profile](#input\_config\_file\_profile) | n/a | `string` | `"DEFAULT"` | no |
| <a name="input_fingerprint"></a> [fingerprint](#input\_fingerprint) | n/a | `string` | n/a | yes |
| <a name="input_kubernetes_version"></a> [kubernetes\_version](#input\_kubernetes\_version) | n/a | `string` | `"v1.32.1"` | no |
| <a name="input_node_pool_worker_size"></a> [node\_pool\_worker\_size](#input\_node\_pool\_worker\_size) | n/a | `number` | `3` | no |
| <a name="input_oci_auth_type"></a> [oci\_auth\_type](#input\_oci\_auth\_type) | n/a | `string` | `"APIKey"` | no |
| <a name="input_oke_node_cpu"></a> [oke\_node\_cpu](#input\_oke\_node\_cpu) | OKE worker node CPUs | `number` | `8` | no |
| <a name="input_oke_node_memory"></a> [oke\_node\_memory](#input\_oke\_node\_memory) | OKE worker node memory in GBs | `number` | `32` | no |
| <a name="input_oke_node_shape"></a> [oke\_node\_shape](#input\_oke\_node\_shape) | OKE Nodepool node shape | `string` | `"VM.Standard.E4.Flex"` | no |
| <a name="input_private_key_path"></a> [private\_key\_path](#input\_private\_key\_path) | n/a | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | n/a | `string` | `"us-sanjose-1"` | no |
| <a name="input_tenancy_ocid"></a> [tenancy\_ocid](#input\_tenancy\_ocid) | Start: OCI provider Variables | `string` | n/a | yes |
| <a name="input_user_ocid"></a> [user\_ocid](#input\_user\_ocid) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_cluster"></a> [cluster](#output\_cluster) | n/a |
| <a name="output_ingress_ip"></a> [ingress\_ip](#output\_ingress\_ip) | Static IP address of the Ingress |