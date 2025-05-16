output "cluster" {
  value = {
    kubeconfig = data.oci_containerengine_cluster_kube_config.service.content
  }
  sensitive = true
}