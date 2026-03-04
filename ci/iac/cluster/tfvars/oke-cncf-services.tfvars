region                 = "us-sanjose-1"
cluster_name           = "oke-cncf-services"
node_pool_worker_size  = 4
kubernetes_version     = "v1.34.1"
cluster_autoscaler_min = 3
cluster_autoscaler_max = 10
oke_node_shape         = "VM.Standard.E6.Flex"
oke_node_memory        = 64
oke_node_cpu           = 16

svc_lb_egress_rules = [
  {
    description      = "kube-proxy access"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 10256
    tcp_min          = 10256
  },
  {
    description      = "NodePort service access"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 32767
    tcp_min          = 30000
  }
]

svc_lb_ingress_rules = [
  {
    description = "Access port 443"
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 443
    tcp_min     = 443
  },
  {
    description = "Access port 80"
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 80
    tcp_min     = 80
  },
  {
    description = "Access port 8443"
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 8443
    tcp_min     = 8443
  }
]

k8s_api_endpoint_egress_rules = [
  {
    description      = "All traffic to worker nodes"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
  },
  {
    description      = "Allow Kubernetes Control Plane to communicate with OKE"
    destination      = "all-sjc-services-in-oracle-services-network"
    destination_type = "SERVICE_CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 443
    tcp_min          = 443
  },
  {
    description      = "Path discovery"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false
    icmp_code        = 4
    icmp_type        = 3
  }
]

k8s_api_endpoint_ingress_rules = [
  {
    description = "External access to Kubernetes API endpoint"
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 6443
    tcp_min     = 6443
  },
  {
    description = "Kubernetes worker to Kubernetes API endpoint communication"
    protocol    = "6"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 6443
    tcp_min     = 6443
  },
  {
    description = "Kubernetes worker to control plane communication"
    protocol    = "6"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 12250
    tcp_min     = 12250
  },
  {
    description = "Path discovery"
    protocol    = "1"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false
    icmp_code   = 4
    icmp_type   = 3
  }
]

node_egress_rules = [
  {
    description      = "Access to Kubernetes API Endpoint"
    destination      = "10.0.0.0/28"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 6443
    tcp_min          = 6443
  },
  {
    description      = "Allow nodes to communicate with OKE to ensure correct start-up and continued functioning"
    destination      = "all-sjc-services-in-oracle-services-network"
    destination_type = "SERVICE_CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 443
    tcp_min          = 443

  },
  {
    description      = "Allow pods on one worker node to communicate with pods on other worker nodes"
    destination      = "10.0.10.0/23"
    destination_type = "CIDR_BLOCK"
    protocol         = "all"
    stateless        = false
  },
  {
    description      = "Allow pods on one worker node to communicate with pods on other worker nodes"
    destination      = "10.0.11.0/24"
    destination_type = "CIDR_BLOCK"
    protocol         = "all"
    stateless        = false
  },
  {
    description      = "ICMP Access from Kubernetes Control Plane"
    destination      = "0.0.0.0/0"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false
    icmp_code        = 4
    icmp_type        = 3
  },
  {
    description      = "Kubernetes worker to control plane communication"
    destination      = "10.0.0.0/28"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 12250
    tcp_min          = 12250
  },
  {
    description      = "Path discovery"
    destination      = "10.0.0.0/28"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false
    icmp_code        = 4
    icmp_type        = 3
  },
  {
    description      = "Worker Nodes access to Internet"
    destination      = "0.0.0.0/0"
    destination_type = "CIDR_BLOCK"
    protocol         = "all"
    stateless        = false
  }
]

node_ingress_rules = [
  {
    description = "Allow pods on one worker node to communicate with pods on other worker nodes"
    protocol    = "all"
    source      = "10.0.10.0/23"
    source_type = "CIDR_BLOCK"
    stateless   = false
  },
  {
    description = "Inbound SSH traffic to worker nodes"
    protocol    = "6"
    source      = "0.0.0.0/0"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 22
    tcp_min     = 22
  },
  {
    description = "Path discovery"
    protocol    = "1"
    source      = "10.0.0.0/28"
    source_type = "CIDR_BLOCK"
    stateless   = false
    icmp_code   = 4
    icmp_type   = 3
  },
  {
    description = "TCP access from Kubernetes Control Plane"
    protocol    = "6"
    source      = "10.0.0.0/28"
    source_type = "CIDR_BLOCK"
    stateless   = false
  },
  {
    description = "Access kube-proxy"
    protocol    = "6"
    source      = "10.0.20.0/24"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 10256
    tcp_min     = 10256
  },
  {
    description = "NodePort service access"
    protocol    = "6"
    source      = "10.0.20.0/24"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 32767
    tcp_min     = 30000
  }
]
