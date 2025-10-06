locals {
  # it can be aquired from:
  # oci ce node-pool-options get --node-pool-option-id all | jq '.data.sources.[] | select(."source-name" | match("Oracle-Linux-8.10-2025.*OKE-1.32.*"))'
  oci_image_id = "ocid1.image.oc1.us-sanjose-1.aaaaaaaa3rkpji3z32yqk5jaydjnxrlfwrgb5rqmyzak2ytcctjm7jrwdgya"
}

resource "oci_containerengine_cluster" "service" {
  name               = var.cluster_name
  kubernetes_version = var.kubernetes_version

  cluster_pod_network_options {
    cni_type = "OCI_VCN_IP_NATIVE"
  }

  endpoint_config {
    is_public_ip_enabled = true
    subnet_id            = oci_core_subnet.k8s_api_endpoint.id
  }

  options {
    service_lb_subnet_ids = [oci_core_subnet.svc_lb.id]
  }

  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  # it has to be enhanced cluster for addons
  type = "ENHANCED_CLUSTER"
}

data "oci_containerengine_cluster_kube_config" "service" {
  cluster_id = oci_containerengine_cluster.service.id
}

resource "oci_containerengine_node_pool" "service_worker" {
  cluster_id     = oci_containerengine_cluster.service.id
  compartment_id = var.compartment_ocid

  kubernetes_version = var.kubernetes_version
  name               = "${var.cluster_name}-pool1"

  # this matches t3.2xlarge sizings.
  node_shape = var.oke_node_shape
  node_shape_config {
    memory_in_gbs = var.oke_node_memory
    ocpus         = var.oke_node_cpu
  }

  node_source_details {
    image_id    = local.oci_image_id
    source_type = "image"
  }

  node_config_details {
    size = var.node_pool_worker_size

    # create placement_configs for each availability domain.
    # There happens to be only a single one in us-sanjose-1.
    dynamic "placement_configs" {
      for_each = data.oci_identity_availability_domains.availability_domains.availability_domains
      content {
        availability_domain = placement_configs.value.name
        subnet_id           = oci_core_subnet.node.id
      }
    }

    node_pool_pod_network_option_details {
      cni_type       = "OCI_VCN_IP_NATIVE"
      pod_nsg_ids    = []
      pod_subnet_ids = [oci_core_subnet.node.id]
    }
  }
}
