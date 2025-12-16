resource "oci_containerengine_addon" "cluster_autoscaler" {
  addon_name                       = "ClusterAutoscaler"
  cluster_id                       = oci_containerengine_cluster.service.id
  remove_addon_resources_on_delete = true

  configurations {
    key   = "authType"
    value = "instance"
  }

  configurations {
    key   = "nodes"
    value = "${var.cluster_autoscaler_min}:${var.cluster_autoscaler_max}:${oci_containerengine_node_pool.service_worker.id}"
  }
}