output "cluster" {
  value = {
    kubeconfig = data.oci_containerengine_cluster_kube_config.service.content
  }
  sensitive = true
}

output "ingress_ip" {
  value       = oci_core_public_ip.ingress_ip.ip_address
  description = "Static IP address of the Ingress"
}
