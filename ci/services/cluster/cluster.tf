resource "oci_containerengine_cluster" "service" {
  name               = "oci-service-worker"
  kubernetes_version = var.kubernetes_version

  cluster_pod_network_options {
    cni_type = "OCI_VCN_IP_NATIVE"
  }

  endpoint_config {
    is_public_ip_enabled = true
    subnet_id            = oci_core_subnet.service_worker_cluster.id
  }

  options {
    service_lb_subnet_ids = [oci_core_subnet.service_worker_cluster.id]
  }

  compartment_id = var.oci_compartment_ocid
  vcn_id         = oci_core_vcn.service.id
}

data "oci_containerengine_cluster_kube_config" "service" {
  cluster_id = oci_containerengine_cluster.service.id
}

resource "oci_containerengine_node_pool" "service_worker" {
  cluster_id     = oci_containerengine_cluster.service.id
  compartment_id = var.oci_compartment_ocid

  kubernetes_version = var.kubernetes_version
  name               = "service-worker"
  ssh_public_key     = var.node_pool_ssh_public_key

  # this matches t3.2xlarge sizings.
  node_shape = "VM.Standard.A1.Flex"
  node_shape_config {
    memory_in_gbs = 32
    ocpus         = 8
  }


  # Using image Oracle-Linux-8.x-<date>
  # Find image OCID for your region from https://docs.oracle.com/iaas/images/
  # For now aarch64 latest k/k 1.31 image is used.
  node_source_details {
    image_id    = "ocid1.image.oc1.us-sanjose-1.aaaaaaaadsern2rllfhao7uyg3t5gafy7xh63apdywrcs3hpryrgnpbgh7sa"
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
        subnet_id           = oci_core_subnet.service_worker_nodes.id
      }
    }

    node_pool_pod_network_option_details {
      cni_type       = "OCI_VCN_IP_NATIVE"
      pod_nsg_ids    = []
      pod_subnet_ids = [oci_core_subnet.service_worker_nodes.id]
    }
  }
}