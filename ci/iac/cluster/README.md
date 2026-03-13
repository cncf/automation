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
| [oci_containerengine_addon.cluster_autoscaler](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/containerengine_addon) | resource |
| [oci_containerengine_cluster.service](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/containerengine_cluster) | resource |
| [oci_containerengine_node_pool.service_worker](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/containerengine_node_pool) | resource |
| [oci_core_internet_gateway.service](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_internet_gateway) | resource |
| [oci_core_nat_gateway.service](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_nat_gateway) | resource |
| [oci_core_public_ip.ingress_ip](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_public_ip) | resource |
| [oci_core_public_ip.kcp_lb_ip](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_public_ip) | resource |
| [oci_core_route_table.private](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_route_table) | resource |
| [oci_core_route_table.public](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_route_table) | resource |
| [oci_core_security_list.k8s_api_endpoint](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_security_list) | resource |
| [oci_core_security_list.node](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_security_list) | resource |
| [oci_core_security_list.svc_lb](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_security_list) | resource |
| [oci_core_service_gateway.service](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_service_gateway) | resource |
| [oci_core_subnet.k8s_api_endpoint](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_subnet) | resource |
| [oci_core_subnet.node](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_subnet) | resource |
| [oci_core_subnet.svc_lb](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_subnet) | resource |
| [oci_core_vcn.service](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/resources/core_vcn) | resource |
| [oci_containerengine_cluster_kube_config.service](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/data-sources/containerengine_cluster_kube_config) | data source |
| [oci_containerengine_node_pool_option.amd64](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/data-sources/containerengine_node_pool_option) | data source |
| [oci_core_services.services](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/data-sources/core_services) | data source |
| [oci_identity_availability_domains.availability_domains](https://registry.terraform.io/providers/oracle/oci/8.5.0/docs/data-sources/identity_availability_domains) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_autoscaler_max"></a> [cluster\_autoscaler\_max](#input\_cluster\_autoscaler\_max) | Maximum number of nodes for the cluster autoscaler | `number` | n/a | yes |
| <a name="input_cluster_autoscaler_min"></a> [cluster\_autoscaler\_min](#input\_cluster\_autoscaler\_min) | Minimum number of nodes for the cluster autoscaler | `number` | n/a | yes |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Name of the OKE cluster (also used for networking and other resources) | `string` | n/a | yes |
| <a name="input_compartment_ocid"></a> [compartment\_ocid](#input\_compartment\_ocid) | The OCID of the compartment in which OCI resources will be managed. | `string` | n/a | yes |
| <a name="input_config_file_profile"></a> [config\_file\_profile](#input\_config\_file\_profile) | Profile name from the OCI CLI configuration file (~/.oci/config) to use for authentication. | `string` | `"DEFAULT"` | no |
| <a name="input_deploy_kcp"></a> [deploy\_kcp](#input\_deploy\_kcp) | Deploy KCP to the cluster and create LB IP for it | `bool` | `false` | no |
| <a name="input_fingerprint"></a> [fingerprint](#input\_fingerprint) | Fingerprint of the public key uploaded to the OCI user account for API authentication. | `string` | n/a | yes |
| <a name="input_ingress_private_ip_id"></a> [ingress\_private\_ip\_id](#input\_ingress\_private\_ip\_id) | n/a | `string` | `""` | no |
| <a name="input_k8s_api_cidr"></a> [k8s\_api\_cidr](#input\_k8s\_api\_cidr) | CIDR for the Kubernetes API network | `string` | n/a | yes |
| <a name="input_k8s_api_endpoint_egress_rules"></a> [k8s\_api\_endpoint\_egress\_rules](#input\_k8s\_api\_endpoint\_egress\_rules) | Egress security rules for the Kubernetes API Endpoint security list | <pre>list(object({<br/>    description      = string<br/>    destination      = string<br/>    destination_type = string<br/>    protocol         = string<br/>    stateless        = bool<br/>    tcp_min          = optional(number)<br/>    tcp_max          = optional(number)<br/>    icmp_type        = optional(number)<br/>    icmp_code        = optional(number)<br/>  }))</pre> | `[]` | no |
| <a name="input_k8s_api_endpoint_ingress_rules"></a> [k8s\_api\_endpoint\_ingress\_rules](#input\_k8s\_api\_endpoint\_ingress\_rules) | Ingress security rules for the Kubernetes API Endpoint security list | <pre>list(object({<br/>    description = string<br/>    source      = string<br/>    source_type = string<br/>    protocol    = string<br/>    stateless   = bool<br/>    tcp_min     = optional(number)<br/>    tcp_max     = optional(number)<br/>    icmp_type   = optional(number)<br/>    icmp_code   = optional(number)<br/>  }))</pre> | `[]` | no |
| <a name="input_kcp_lb_private_ip_id"></a> [kcp\_lb\_private\_ip\_id](#input\_kcp\_lb\_private\_ip\_id) | n/a | `string` | `""` | no |
| <a name="input_kubernetes_version"></a> [kubernetes\_version](#input\_kubernetes\_version) | Kubernetes version for OKE | `string` | n/a | yes |
| <a name="input_node_cidr"></a> [node\_cidr](#input\_node\_cidr) | CIDR for the worker nodes network | `string` | n/a | yes |
| <a name="input_node_egress_rules"></a> [node\_egress\_rules](#input\_node\_egress\_rules) | Egress security rules for the worker node security list | <pre>list(object({<br/>    description      = string<br/>    destination      = string<br/>    destination_type = string<br/>    protocol         = string<br/>    stateless        = bool<br/>    tcp_min          = optional(number)<br/>    tcp_max          = optional(number)<br/>    icmp_type        = optional(number)<br/>    icmp_code        = optional(number)<br/>  }))</pre> | `[]` | no |
| <a name="input_node_ingress_rules"></a> [node\_ingress\_rules](#input\_node\_ingress\_rules) | Ingress security rules for the worker node security list | <pre>list(object({<br/>    description = string<br/>    source      = string<br/>    source_type = string<br/>    protocol    = string<br/>    stateless   = bool<br/>    tcp_min     = optional(number)<br/>    tcp_max     = optional(number)<br/>    icmp_type   = optional(number)<br/>    icmp_code   = optional(number)<br/>  }))</pre> | `[]` | no |
| <a name="input_node_pool_worker_size"></a> [node\_pool\_worker\_size](#input\_node\_pool\_worker\_size) | Default number of worker nodes | `number` | n/a | yes |
| <a name="input_oci_auth_type"></a> [oci\_auth\_type](#input\_oci\_auth\_type) | Authentication method used by the OCI provider (e.g., APIKey, InstancePrincipal, ResourcePrincipal). | `string` | `"APIKey"` | no |
| <a name="input_oke_node_boot_volume_size"></a> [oke\_node\_boot\_volume\_size](#input\_oke\_node\_boot\_volume\_size) | The size of the boot volume in GBs | `number` | `50` | no |
| <a name="input_oke_node_cpu"></a> [oke\_node\_cpu](#input\_oke\_node\_cpu) | OKE worker node CPUs | `number` | n/a | yes |
| <a name="input_oke_node_memory"></a> [oke\_node\_memory](#input\_oke\_node\_memory) | OKE worker node memory in GBs | `number` | n/a | yes |
| <a name="input_oke_node_shape"></a> [oke\_node\_shape](#input\_oke\_node\_shape) | OKE Nodepool node shape | `string` | n/a | yes |
| <a name="input_private_key_path"></a> [private\_key\_path](#input\_private\_key\_path) | Path to the private key file used for OCI API key authentication. | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | OCI region where resources will be deployed. | `string` | `"us-sanjose-1"` | no |
| <a name="input_svc_cidr"></a> [svc\_cidr](#input\_svc\_cidr) | CIDR for the Service Network | `string` | n/a | yes |
| <a name="input_svc_lb_egress_rules"></a> [svc\_lb\_egress\_rules](#input\_svc\_lb\_egress\_rules) | Egress security rules for the service LB security list | <pre>list(object({<br/>    description      = string<br/>    destination      = string<br/>    destination_type = string<br/>    protocol         = string<br/>    stateless        = bool<br/>    tcp_min          = optional(number)<br/>    tcp_max          = optional(number)<br/>    icmp_type        = optional(number)<br/>    icmp_code        = optional(number)<br/>  }))</pre> | `[]` | no |
| <a name="input_svc_lb_ingress_rules"></a> [svc\_lb\_ingress\_rules](#input\_svc\_lb\_ingress\_rules) | Ingress security rules for the service LB security list | <pre>list(object({<br/>    description = string<br/>    source      = string<br/>    source_type = string<br/>    protocol    = string<br/>    stateless   = bool<br/>    tcp_min     = optional(number)<br/>    tcp_max     = optional(number)<br/>    icmp_type   = optional(number)<br/>    icmp_code   = optional(number)<br/>  }))</pre> | `[]` | no |
| <a name="input_tenancy_ocid"></a> [tenancy\_ocid](#input\_tenancy\_ocid) | The OCID of the Oracle Cloud Infrastructure tenancy where resources will be created. | `string` | n/a | yes |
| <a name="input_user_ocid"></a> [user\_ocid](#input\_user\_ocid) | The OCID of the OCI user associated with the API key used for authentication. | `string` | n/a | yes |
| <a name="input_vcn_cidr"></a> [vcn\_cidr](#input\_vcn\_cidr) | CIDR for the VCN | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_cluster"></a> [cluster](#output\_cluster) | Cluster Kubeconfig |
| <a name="output_ingress_ip"></a> [ingress\_ip](#output\_ingress\_ip) | Static IP address of the Ingress |
| <a name="output_kcp_lb_ip"></a> [kcp\_lb\_ip](#output\_kcp\_lb\_ip) | Static IP address of the KCP Front Proxy Service |