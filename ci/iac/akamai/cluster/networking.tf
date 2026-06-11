resource "linode_vpc" "main" {
  label       = "${var.cluster_name}-vpc"
  region      = var.region
  description = "VPC for ${var.cluster_name} LKE cluster"
}

resource "linode_vpc_subnet" "cluster" {
  vpc_id = linode_vpc.main.id
  label  = "${var.cluster_name}-subnet"
  ipv4   = var.vpc_subnet_cidr
}

resource "linode_firewall" "cluster" {
  label = "${var.cluster_name}-fw"

  inbound {
    label    = "allow-kubernetes-api-443"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "443"
    ipv4     = var.control_plane_acl_ipv4
    ipv6     = var.control_plane_acl_ipv6
  }

  inbound {
    label    = "allow-kubernetes-api-6443"
    action   = "ACCEPT"
    protocol = "TCP"
    ports    = "6443"
    ipv4     = var.control_plane_acl_ipv4
    ipv6     = var.control_plane_acl_ipv6
  }

  # Not needed at the moment. Allows NodePort services (30000-32767) from anywhere - adjust as needed for security
  # inbound {
  #   label    = "allow-nodeport-services"
  #   action   = "ACCEPT"
  #   protocol = "TCP"
  #   ports    = "30000-32767"
  #   ipv4     = ["0.0.0.0/0"]
  #   ipv6     = ["::/0"]
  # }

  inbound_policy  = "DROP"
  outbound_policy = "ACCEPT"
}
