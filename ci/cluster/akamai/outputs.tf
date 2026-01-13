# Output cluster connection information
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

output "kubeconfig_path" {
  description = "Path to the generated kubeconfig file for kubectl access"
  value       = local_file.kubeconfig.filename
}

output "region" {
  description = "The region where the cluster is deployed"
  value       = linode_lke_cluster.github_runners.region
}

output "k8s_version" {
  description = "The Kubernetes version running on the cluster"
  value       = linode_lke_cluster.github_runners.k8s_version
}
