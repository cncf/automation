## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.6.0 |
| <a name="requirement_linode"></a> [linode](#requirement\_linode) | ~> 3.13.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_linode"></a> [linode](#provider\_linode) | 3.13.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [linode_firewall.cluster](https://registry.terraform.io/providers/linode/linode/latest/docs/resources/firewall) | resource |
| [linode_lke_cluster.github_runners](https://registry.terraform.io/providers/linode/linode/latest/docs/resources/lke_cluster) | resource |
| [linode_vpc.main](https://registry.terraform.io/providers/linode/linode/latest/docs/resources/vpc) | resource |
| [linode_vpc_subnet.cluster](https://registry.terraform.io/providers/linode/linode/latest/docs/resources/vpc_subnet) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_autoscaler_max"></a> [autoscaler\_max](#input\_autoscaler\_max) | Maximum number of nodes for autoscaler | `number` | `3` | no |
| <a name="input_autoscaler_min"></a> [autoscaler\_min](#input\_autoscaler\_min) | Minimum number of nodes for autoscaler | `number` | `1` | no |
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | The name of the LKE cluster | `string` | `"github-runners"` | no |
| <a name="input_control_plane_acl_ipv4"></a> [control\_plane\_acl\_ipv4](#input\_control\_plane\_acl\_ipv4) | List of IPv4 CIDRs allowed to access the Kubernetes API server | `list(string)` | <pre>[<br/>  "0.0.0.0/0"<br/>]</pre> | no |
| <a name="input_control_plane_acl_ipv6"></a> [control\_plane\_acl\_ipv6](#input\_control\_plane\_acl\_ipv6) | List of IPv6 CIDRs allowed to access the Kubernetes API server | `list(string)` | <pre>[<br/>  "::/0"<br/>]</pre> | no |
| <a name="input_environment"></a> [environment](#input\_environment) | Environment name (e.g., dev, staging, prod) | `string` | `"dev"` | no |
| <a name="input_kubernetes_version"></a> [kubernetes\_version](#input\_kubernetes\_version) | The Kubernetes version to use for the cluster | `string` | `"1.34"` | no |
| <a name="input_linode_api_token"></a> [linode\_api\_token](#input\_linode\_api\_token) | Linode API Token | `string` | n/a | yes |
| <a name="input_node_count"></a> [node\_count](#input\_node\_count) | The initial number of nodes in the cluster | `number` | `1` | no |
| <a name="input_node_type"></a> [node\_type](#input\_node\_type) | Linode instance type for cluster nodes | `string` | `"g6-standard-1"` | no |
| <a name="input_region"></a> [region](#input\_region) | The region where the cluster will be deployed | `string` | `"us-east"` | no |
| <a name="input_vpc_subnet_cidr"></a> [vpc\_subnet\_cidr](#input\_vpc\_subnet\_cidr) | IPv4 CIDR block for the VPC subnet | `string` | `"10.0.0.0/24"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_api_endpoints"></a> [api\_endpoints](#output\_api\_endpoints) | The API endpoints for the Kubernetes cluster |
| <a name="output_cluster_id"></a> [cluster\_id](#output\_cluster\_id) | The unique ID of the LKE cluster |
| <a name="output_cluster_status"></a> [cluster\_status](#output\_cluster\_status) | The operational status of the cluster |
| <a name="output_firewall_id"></a> [firewall\_id](#output\_firewall\_id) | The ID of the cloud firewall |
| <a name="output_k8s_version"></a> [k8s\_version](#output\_k8s\_version) | The Kubernetes version running on the cluster |
| <a name="output_kubeconfig"></a> [kubeconfig](#output\_kubeconfig) | Base64-encoded kubeconfig for the LKE cluster |
| <a name="output_region"></a> [region](#output\_region) | The region where the cluster is deployed |
| <a name="output_subnet_id"></a> [subnet\_id](#output\_subnet\_id) | The ID of the VPC subnet |
| <a name="output_vpc_id"></a> [vpc\_id](#output\_vpc\_id) | The ID of the VPC |
