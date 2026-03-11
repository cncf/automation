data "oci_containerengine_node_pool_option" "amd64" {
  compartment_id = var.compartment_ocid
  node_pool_option_id = "all"
  node_pool_k8s_version = var.kubernetes_version
  node_pool_os_arch = "X86_64"
  node_pool_os_type = "OL8"
}

locals {
  non_gpu_images = [
    for source in data.oci_containerengine_node_pool_option.amd64.sources :
    source if !strcontains(source.source_name, "GPU")
  ]
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
    boot_volume_size_in_gbs = var.oke_node_boot_volume_size
    image_id                = local.non_gpu_images[0].image_id
    source_type             = "image"
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
