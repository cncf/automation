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

  egress_security_rules {
    description      = "All traffic to worker nodes"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
  }

  egress_security_rules {
    description      = "Allow Kubernetes Control Plane to communicate with OKE"
    destination      = "all-sjc-services-in-oracle-services-network"
    destination_type = "SERVICE_CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 443
      min = 443
    }
  }

  egress_security_rules {
    description      = "Path discovery"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false

    icmp_options {
      code = 4
      type = 3
    }
  }

  ingress_security_rules {
    description = "External access to Kubernetes API endpoint"
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 6443
      min = 6443
    }
  }

  ingress_security_rules {
    description = "Kubernetes worker to Kubernetes API endpoint communication"
    protocol    = "6"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 6443
      min = 6443
    }
  }

  ingress_security_rules {
    description = "Kubernetes worker to control plane communication"
    protocol    = "6"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 12250
      min = 12250
    }
  }

  ingress_security_rules {
    description = "Path discovery"
    protocol    = "1"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false

    icmp_options {
      code = 4
      type = 3
    }
  }
}

resource "oci_core_security_list" "svc_lb" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "${var.cluster_name}-svclb-seclist"

  egress_security_rules {
    description      = "kube-proxy access"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 10256
      min = 10256
    }
  }

  egress_security_rules {
    description      = "NodePort service access"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 30330
      min = 30330
    }
  }

  egress_security_rules {
    description      = "NodePort service access"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 32170
      min = 32170
    }
  }

  egress_security_rules {
    description      = "NodePort service access"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 32709
      min = 32709
    }
  }

  ingress_security_rules {
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 443
      min = 443
    }
  }

  ingress_security_rules {
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 80
      min = 80
    }
  }
  ingress_security_rules {
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 8443
      min = 8443
    }
  }
}

resource "oci_core_security_list" "node" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.service.id
  display_name   = "${var.cluster_name}-node-seclist"

  egress_security_rules {
    description      = "Access to Kubernetes API Endpoint"
    destination      = "10.0.0.0/28"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 6443
      min = 6443
    }
  }

  egress_security_rules {
    description      = "Allow nodes to communicate with OKE to ensure correct start-up and continued functioning"
    destination      = "all-sjc-services-in-oracle-services-network"
    destination_type = "SERVICE_CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 443
      min = 443
    }
  }

  egress_security_rules {
    description      = "Allow pods on one worker node to communicate with pods on other worker nodes"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "all"
    stateless        = false
  }

  egress_security_rules {
    description      = "Allow pods on one worker node to communicate with pods on other worker nodes"
    destination      = "10.0.11.0/24"
    destination_type = "CIDR_BLOCK"
    protocol         = "all"
    stateless        = false
  }

  egress_security_rules {
    description      = "ICMP Access from Kubernetes Control Plane"
    destination      = "0.0.0.0/0"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false

    icmp_options {
      code = 4
      type = 3
    }
  }

  egress_security_rules {
    description      = "Kubernetes worker to control plane communication"
    destination      = "10.0.0.0/28"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false

    tcp_options {
      max = 12250
      min = 12250
    }
  }

  egress_security_rules {
    description      = "Path discovery"
    destination      = "10.0.0.0/28"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false

    icmp_options {
      code = 4
      type = 3
    }
  }

  egress_security_rules {
    description      = "Worker Nodes access to Internet"
    destination      = "0.0.0.0/0"
    destination_type = "CIDR_BLOCK"
    protocol         = "all"
    stateless        = false
  }

  ingress_security_rules {
    description = "Allow pods on one worker node to communicate with pods on other worker nodes"
    protocol    = "all"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false
  }

  ingress_security_rules {
    description = "Inbound SSH traffic to worker nodes"
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 22
      min = 22
    }
  }

  ingress_security_rules {
    description = "Path discovery"
    protocol    = "1"
    source      = "10.0.0.0/28"
    source_type = "CIDR_BLOCK"
    stateless   = false

    icmp_options {
      code = 4
      type = 3
    }
  }

  ingress_security_rules {
    description = "TCP access from Kubernetes Control Plane"
    protocol    = "6"
    source      = "10.0.0.0/28"
    source_type = "CIDR_BLOCK"
    stateless   = false
  }

  ingress_security_rules {
    description = "Access kube-proxy"
    protocol    = "6"
    source      = "10.0.20.0/24"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 10256
      min = 10256
    }
  }

  ingress_security_rules {
    description = "NodePort service access"
    protocol    = "6"
    source      = "10.0.20.0/24"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 30330
      min = 30330
    }
  }

  ingress_security_rules {
    description = "NodePort service access"
    protocol    = "6"
    source      = "10.0.20.0/24"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 32170
      min = 32170
    }
  }

  ingress_security_rules {
    description = "NodePort service access"
    protocol    = "6"
    source      = "10.0.20.0/24"
    source_type = "CIDR_BLOCK"
    stateless   = false

    tcp_options {
      max = 32709
      min = 32709
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
  private_ip_id  = "ocid1.privateip.oc1.us-sanjose-1.abzwuljrkimbtnfaj5jjpepkmp4ifttqcltnmdldwvzviicmsk5foxp4oiwa"
}

resource "oci_core_public_ip" "kcp_lb_ip" {
  compartment_id = var.compartment_ocid
  lifetime       = "RESERVED"
  display_name   = "${var.cluster_name}-kcp-lp-ip"
  private_ip_id  = "ocid1.privateip.oc1.us-sanjose-1.abzwuljrzbthnlqawsumhrda7i7ucfivjcirrw565fuomenlknsmcxpvn2ka"
}
