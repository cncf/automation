cluster_name        = "lke-cncf-gha-iad2"
kubernetes_version  = "1.35"
region              = "us-iad-2"
node_count          = 3
node_type           = "g6-dedicated-16"
autoscaler_min      = 1
autoscaler_max      = 5
environment         = "prod"
