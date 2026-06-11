output "cluster_id" {
  description = "The unique ID of the LKE cluster"
  value       = linode_lke_cluster.github_runners.id
}

output "api_endpoints" {
  description = "The API endpoints for the Kubernetes cluster"
  value       = linode_lke_cluster.github_runners.api_endpoints
}

output "cluster_status" {
  description = "The operational status of the cluster"
  value       = linode_lke_cluster.github_runners.status
}

output "kubeconfig" {
  description = "Base64-encoded kubeconfig for the LKE cluster"
  value       = linode_lke_cluster.github_runners.kubeconfig
  sensitive   = true
}

output "region" {
  description = "The region where the cluster is deployed"
  value       = linode_lke_cluster.github_runners.region
}

output "k8s_version" {
  description = "The Kubernetes version running on the cluster"
  value       = linode_lke_cluster.github_runners.k8s_version
}

output "vpc_id" {
  description = "The ID of the VPC"
  value       = linode_vpc.main.id
}

output "subnet_id" {
  description = "The ID of the VPC subnet"
  value       = linode_vpc_subnet.cluster.id
}

output "firewall_id" {
  description = "The ID of the cloud firewall"
  value       = linode_firewall.cluster.id
}
