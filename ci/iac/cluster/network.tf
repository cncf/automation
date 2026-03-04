resource "oci_core_vcn" "service" {
  cidr_block     = "10.0.0.0/16"
  compartment_id = var.compartment_ocid
  display_name   = "${var.cluster_name}-vcn"
}

resource "oci_core_internet_gateway" "service" {
  compartment_id = var.compartment_ocid
  display_name   = "${var.cluster_name}-igw"
  vcn_id         = oci_core_vcn.service.id
}

# Get all network services:
# oci network service list
data "oci_core_services" "services" {}

resource "oci_core_service_gateway" "service" {
  compartment_id = var.compartment_ocid
  display_name   = "${var.cluster_name}-sgw"
  vcn_id         = oci_core_vcn.service.id
  services {
    service_id = data.oci_core_services.services.services[0].id
  }
}

resource "oci_core_nat_gateway" "service" {
  compartment_id = var.compartment_ocid
  display_name   = "${var.cluster_name}-ngw"
  vcn_id         = oci_core_vcn.service.id
}

resource "oci_core_route_table" "public" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "${var.cluster_name}-public-routes"

  route_rules {
    description       = "traffic to/from internet"
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.service.id
  }
}

resource "oci_core_route_table" "private" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "${var.cluster_name}-private-routes"

  route_rules {
    description       = "traffic to OCI services"
    destination       = "all-sjc-services-in-oracle-services-network"
    destination_type  = "SERVICE_CIDR_BLOCK"
    network_entity_id = oci_core_service_gateway.service.id
  }

  route_rules {
    description       = "traffic to the internet"
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_nat_gateway.service.id
  }
}

resource "oci_core_security_list" "k8s_api_endpoint" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "${var.cluster_name}-k8s-api-endpoint-seclist"

  dynamic "egress_security_rules" {
    for_each = var.k8s_api_endpoint_egress_rules
    content {
      description      = egress_security_rules.value.description
      destination      = egress_security_rules.value.destination
      destination_type = egress_security_rules.value.destination_type
      protocol         = egress_security_rules.value.protocol
      stateless        = egress_security_rules.value.stateless

      dynamic "tcp_options" {
        for_each = egress_security_rules.value.tcp_min != null ? [1] : []
        content {
          min = egress_security_rules.value.tcp_min
          max = egress_security_rules.value.tcp_max
        }
      }

      dynamic "icmp_options" {
        for_each = egress_security_rules.value.icmp_type != null ? [1] : []
        content {
          type = egress_security_rules.value.icmp_type
          code = egress_security_rules.value.icmp_code
        }
      }
    }
  }

  dynamic "ingress_security_rules" {
    for_each = var.k8s_api_endpoint_ingress_rules
    content {
      description = ingress_security_rules.value.description
      source      = ingress_security_rules.value.source
      source_type = ingress_security_rules.value.source_type
      protocol    = ingress_security_rules.value.protocol
      stateless   = ingress_security_rules.value.stateless

      dynamic "tcp_options" {
        for_each = ingress_security_rules.value.tcp_min != null ? [1] : []
        content {
          min = ingress_security_rules.value.tcp_min
          max = ingress_security_rules.value.tcp_max
        }
      }

      dynamic "icmp_options" {
        for_each = ingress_security_rules.value.icmp_type != null ? [1] : []
        content {
          type = ingress_security_rules.value.icmp_type
          code = ingress_security_rules.value.icmp_code
        }
      }
    }
  }
}

resource "oci_core_security_list" "svc_lb" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "${var.cluster_name}-svclb-seclist"

  dynamic "egress_security_rules" {
    for_each = var.svc_lb_egress_rules
    content {
      description      = egress_security_rules.value.description
      destination      = egress_security_rules.value.destination
      destination_type = egress_security_rules.value.destination_type
      protocol         = egress_security_rules.value.protocol
      stateless        = egress_security_rules.value.stateless

      dynamic "tcp_options" {
        for_each = egress_security_rules.value.tcp_min != null ? [1] : []
        content {
          min = egress_security_rules.value.tcp_min
          max = egress_security_rules.value.tcp_max
        }
      }

      dynamic "icmp_options" {
        for_each = egress_security_rules.value.icmp_type != null ? [1] : []
        content {
          type = egress_security_rules.value.icmp_type
          code = egress_security_rules.value.icmp_code
        }
      }
    }
  }

  dynamic "ingress_security_rules" {
    for_each = var.svc_lb_ingress_rules
    content {
      description = ingress_security_rules.value.description
      source      = ingress_security_rules.value.source
      source_type = ingress_security_rules.value.source_type
      protocol    = ingress_security_rules.value.protocol
      stateless   = ingress_security_rules.value.stateless

      dynamic "tcp_options" {
        for_each = ingress_security_rules.value.tcp_min != null ? [1] : []
        content {
          min = ingress_security_rules.value.tcp_min
          max = ingress_security_rules.value.tcp_max
        }
      }

      dynamic "icmp_options" {
        for_each = ingress_security_rules.value.icmp_type != null ? [1] : []
        content {
          type = ingress_security_rules.value.icmp_type
          code = ingress_security_rules.value.icmp_code
        }
      }
    }
  }
}

resource "oci_core_security_list" "node" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "${var.cluster_name}-node-seclist"

  dynamic "egress_security_rules" {
    for_each = var.node_egress_rules
    content {
      description      = egress_security_rules.value.description
      destination      = egress_security_rules.value.destination
      destination_type = egress_security_rules.value.destination_type
      protocol         = egress_security_rules.value.protocol
      stateless        = egress_security_rules.value.stateless

      dynamic "tcp_options" {
        for_each = egress_security_rules.value.tcp_min != null ? [1] : []
        content {
          min = egress_security_rules.value.tcp_min
          max = egress_security_rules.value.tcp_max
        }
      }

      dynamic "icmp_options" {
        for_each = egress_security_rules.value.icmp_type != null ? [1] : []
        content {
          type = egress_security_rules.value.icmp_type
          code = egress_security_rules.value.icmp_code
        }
      }
    }
  }

  dynamic "ingress_security_rules" {
    for_each = var.node_ingress_rules
    content {
      description = ingress_security_rules.value.description
      source      = ingress_security_rules.value.source
      source_type = ingress_security_rules.value.source_type
      protocol    = ingress_security_rules.value.protocol
      stateless   = ingress_security_rules.value.stateless

      dynamic "tcp_options" {
        for_each = ingress_security_rules.value.tcp_min != null ? [1] : []
        content {
          min = ingress_security_rules.value.tcp_min
          max = ingress_security_rules.value.tcp_max
        }
      }

      dynamic "icmp_options" {
        for_each = ingress_security_rules.value.icmp_type != null ? [1] : []
        content {
          type = ingress_security_rules.value.icmp_type
          code = ingress_security_rules.value.icmp_code
        }
      }
    }
  }
}

resource "oci_core_subnet" "k8s_api_endpoint" {
  availability_domain = null
  cidr_block          = "10.0.0.0/28"
  compartment_id      = var.compartment_ocid
  vcn_id              = oci_core_vcn.service.id

  security_list_ids = [oci_core_security_list.k8s_api_endpoint.id]
  route_table_id    = oci_core_route_table.public.id
  display_name      = "${var.cluster_name}-k8sApiEndpoint-subnet"
}

resource "oci_core_subnet" "svc_lb" {
  availability_domain = null
  cidr_block          = "10.0.20.0/24"
  compartment_id      = var.compartment_ocid
  vcn_id              = oci_core_vcn.service.id

  security_list_ids = [oci_core_security_list.svc_lb.id]
  route_table_id    = oci_core_route_table.public.id
  display_name      = "${var.cluster_name}-svclb-subnet"
}

resource "oci_core_subnet" "node" {
  availability_domain = null
  cidr_block          = "10.0.10.0/23"
  compartment_id      = var.compartment_ocid
  vcn_id              = oci_core_vcn.service.id

  prohibit_public_ip_on_vnic = true

  security_list_ids = [oci_core_security_list.node.id]
  route_table_id    = oci_core_route_table.private.id
  display_name      = "${var.cluster_name}-node-subnet"
}

resource "oci_core_public_ip" "ingress_ip" {
  compartment_id = var.compartment_ocid
  lifetime       = "RESERVED"
  display_name   = "${var.cluster_name}-ingress-ip"
  private_ip_id  = "ocid1.privateip.oc1.us-sanjose-1.abzwuljr4i5var577r2aykc7eusqn3rpy5ciwqkvk67xpw7wmxhwdn3yw4cq"
}

resource "oci_core_public_ip" "kcp_lb_ip" {
  compartment_id = var.compartment_ocid
  lifetime       = "RESERVED"
  display_name   = "${var.cluster_name}-kcp-lp-ip"
  private_ip_id  = "ocid1.privateip.oc1.us-sanjose-1.abzwuljrhuxff5dzmmwpovrwddlmh44m72b7cqgie2bvv3kkpbpq4aok2cpa"
}
