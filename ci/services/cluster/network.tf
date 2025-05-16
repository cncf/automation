resource "oci_core_vcn" "service" {
  cidr_block     = "10.0.0.0/16"
  compartment_id = var.oci_compartment_ocid
  display_name   = "Service Network"
}

resource "oci_core_internet_gateway" "service" {
  compartment_id = var.oci_compartment_ocid
  display_name   = "Service Internet Gateway"
  vcn_id         = oci_core_vcn.service.id
}

resource "oci_core_route_table" "service_worker" {
  compartment_id = var.oci_compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "Service Worker Route Table"

  route_rules {
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.service.id
  }
}

# Rules based on https://docs.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengnetworkconfig.htm#securitylistconfig.

resource "oci_core_security_list" "node_network" {
  compartment_id = var.oci_compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "Service Worker Cross-Node Access"

  # Allow access from node/pod CIDR to all TCP ports.
  ingress_security_rules {
    protocol = "6" # TCP

    source      = "10.0.64.0/18"
    source_type = "CIDR_BLOCK"
  }

  # Allow access from node/pod CIDR to all UDP ports.
  ingress_security_rules {
    protocol = "17" # UDP

    source      = "10.0.64.0/18"
    source_type = "CIDR_BLOCK"
  }
}

resource "oci_core_security_list" "kubernetes_api" {
  compartment_id = var.oci_compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "Kubernetes API Security List"

  # Allow access from anywhere to Kubernetes API port.
  ingress_security_rules {
    protocol  = "6" # TCP
    stateless = false

    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"

    tcp_options {
      min = 6443
      max = 6443
    }
  }

  # Allow node pool CIDR to access port for proxymux.
  ingress_security_rules {
    protocol  = "6" # TCP
    stateless = false

    source      = "10.0.64.0/18"
    source_type = "CIDR_BLOCK"

    tcp_options {
      min = 12250
      max = 12250
    }
  }

  # Path Discovery.
  ingress_security_rules {
    protocol  = "1"
    stateless = false

    source      = "10.0.64.0/18"
    source_type = "CIDR_BLOCK"
  }
}


resource "oci_core_security_list" "control_plane_node_access" {
  compartment_id = var.oci_compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "Control Plane Worker Access"

  # Allow VPC CIDR to access kubelet API.
  ingress_security_rules {
    protocol = "6" # TCP

    source      = "10.0.0.0/16"
    source_type = "CIDR_BLOCK"

    tcp_options {
      min = 10250
      max = 10250
    }
  }
}


resource "oci_core_subnet" "service_worker_nodes" {
  availability_domain = null
  cidr_block          = "10.0.64.0/18"
  compartment_id      = var.oci_compartment_ocid
  vcn_id              = oci_core_vcn.service.id

  security_list_ids = [oci_core_vcn.service.default_security_list_id, oci_core_security_list.control_plane_node_access.id, oci_core_security_list.node_network.id]
  route_table_id    = "ocid1.routetable.oc1.us-sanjose-1.aaaaaaaaguv25vmwy4j5k2hunwcmjx6tpbbfnmuhzmxowstl76mj4ui7zpxq"
  display_name      = "Service Nodes/Pods Subnet"
}

resource "oci_core_subnet" "service_worker_cluster" {
  availability_domain = null
  cidr_block          = "10.0.10.0/24"
  compartment_id      = var.oci_compartment_ocid
  vcn_id              = oci_core_vcn.service.id

  security_list_ids = [oci_core_vcn.service.default_security_list_id, oci_core_security_list.kubernetes_api.id]
  route_table_id    = oci_core_route_table.service_worker.id
  dhcp_options_id   = oci_core_vcn.service.default_dhcp_options_id
  display_name      = "Service Cluster Subnet"
}