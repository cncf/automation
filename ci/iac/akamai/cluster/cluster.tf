locals {
  common_tags = [
    "github-actions-runners",
    "environment:${var.environment}"
  ]
}

resource "linode_lke_cluster" "github_runners" {
  label       = var.cluster_name
  k8s_version = var.kubernetes_version
  region      = var.region
  tags        = local.common_tags
  vpc_id      = linode_vpc.main.id
  subnet_id   = linode_vpc_subnet.cluster.id

  control_plane {
    high_availability = true
    acl {
      enabled = true
      addresses {
        ipv4 = var.control_plane_acl_ipv4
        ipv6 = var.control_plane_acl_ipv6
      }
    }
  }

  pool {
    type            = var.node_type
    count           = var.node_count
    disk_encryption = "enabled"
    firewall_id     = linode_firewall.cluster.id

    autoscaler {
      min = var.autoscaler_min
      max = var.autoscaler_max
    }

    labels = {
      "role" = "github-runner"
    }
  }
}


