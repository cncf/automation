region                 = "us-sanjose-1"
cluster_name           = "oke-cncf-services"
node_pool_worker_size  = 4
kubernetes_version     = "v1.34.1"
cluster_autoscaler_min = 3
cluster_autoscaler_max = 10
oke_node_shape         = "VM.Standard.E6.Flex"
oke_node_memory        = 64
oke_node_cpu           = 16
deploy_kcp             = true
ingress_private_ip_id  = "ocid1.privateip.oc1.us-sanjose-1.abzwuljr4i5var577r2aykc7eusqn3rpy5ciwqkvk67xpw7wmxhwdn3yw4cq"
kcp_lb_private_ip_id   = "ocid1.privateip.oc1.us-sanjose-1.abzwuljrhuxff5dzmmwpovrwddlmh44m72b7cqgie2bvv3kkpbpq4aok2cpa"
vcn_cidr               = "10.0.0.0/16"
k8s_api_cidr           = "10.0.0.0/28"
svc_cidr               = "10.0.20.0/24"
node_cidr              = "10.0.10.0/23"

# CIDR_BLOCK   : is overriden to node_cidr in networks.tf file
# K8S_API_CIDR : is overriden to k8s_api_cidr in networks.tf file
# INTERNET     : is overriden to "0.8.8.8/0" in networks.tf file
svc_lb_egress_rules = [
  {
    description      = "kube-proxy access"
    destination      = "NODE_CIDR"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 10256
    tcp_min          = 10256
  },
  {
    description      = "NodePort service access"
    destination      = "NODE_CIDR"
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
    source      = "INTERNET"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 443
    tcp_min     = 443
  },
  {
    description = "Access port 80"
    protocol    = "6"
    source      = "INTERNET"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 80
    tcp_min     = 80
  },
  {
    description = "Access port 8443"
    protocol    = "6"
    source      = "INTERNET"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 8443
    tcp_min     = 8443
  }
]

k8s_api_endpoint_egress_rules = [
  {
    description      = "All traffic to worker nodes"
    destination      = "NODE_CIDR"
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
    destination      = "NODE_CIDR"
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
    source      = "INTERNET"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 6443
    tcp_min     = 6443
  },
  {
    description = "Kubernetes worker to Kubernetes API endpoint communication"
    protocol    = "6"
    source      = "NODE_CIDR"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 6443
    tcp_min     = 6443
  },
  {
    description = "Kubernetes worker to control plane communication"
    protocol    = "6"
    source      = "NODE_CIDR"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 12250
    tcp_min     = 12250
  },
  {
    description = "Path discovery"
    protocol    = "1"
    source      = "NODE_CIDR"
    source_type = "CIDR_BLOCK"
    stateless   = false
    icmp_code   = 4
    icmp_type   = 3
  }
]

node_egress_rules = [
  {
    description      = "Access to Kubernetes API Endpoint"
    destination      = "K8S_API_CIDR"
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
    destination      = "NODE_CIDR"
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
    destination      = "INTERNET"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false
    icmp_code        = 4
    icmp_type        = 3
  },
  {
    description      = "Kubernetes worker to control plane communication"
    destination      = "K8S_API_CIDR"
    destination_type = "CIDR_BLOCK"
    protocol         = "6"
    stateless        = false
    tcp_max          = 12250
    tcp_min          = 12250
  },
  {
    description      = "Path discovery"
    destination      = "K8S_API_CIDR"
    destination_type = "CIDR_BLOCK"
    protocol         = "1"
    stateless        = false
    icmp_code        = 4
    icmp_type        = 3
  },
  {
    description      = "Worker Nodes access to Internet"
    destination      = "INTERNET"
    destination_type = "CIDR_BLOCK"
    protocol         = "all"
    stateless        = false
  }
]

node_ingress_rules = [
  {
    description = "Allow pods on one worker node to communicate with pods on other worker nodes"
    protocol    = "all"
    source      = "NODE_CIDR"
    source_type = "CIDR_BLOCK"
    stateless   = false
  },
  {
    description = "Inbound SSH traffic to worker nodes"
    protocol    = "6"
    source      = "INTERNET"
    source_type = "CIDR_BLOCK"
    stateless   = false
    tcp_max     = 22
    tcp_min     = 22
  },
  {
    description = "Path discovery"
    protocol    = "1"
    source      = "K8S_API_CIDR"
    source_type = "CIDR_BLOCK"
    stateless   = false
    icmp_code   = 4
    icmp_type   = 3
  },
  {
    description = "TCP access from Kubernetes Control Plane"
    protocol    = "6"
    source      = "K8S_API_CIDR"
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
