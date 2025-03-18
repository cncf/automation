output "api_endpoints" {
  description = "The API endpoints for the cluster"
  value       = linode_lke_cluster.cncf_poc_cluster.api_endpoints
}

output "status" {
  description = "The status of the cluster"
  value       = linode_lke_cluster.cncf_poc_cluster.status
}

output "kubeconfig_path" {
  description = "Path to the kubeconfig file"
  value       = "${path.module}/kubeconfig.yaml"
}

output "dashboard_url" {
  description = "URL to access the Kubernetes Dashboard"
  value       = "https://${linode_lke_cluster.cncf_poc_cluster.api_endpoints[0]}/kubernetes-dashboard"
}
