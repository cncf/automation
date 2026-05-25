cluster_name        = "lke-cncf-gha-iad2"
kubernetes_version  = "1.35"
region              = "us-iad-2"
node_count          = 3
node_type           = "g6-standard-8"
autoscaler_min      = 1
autoscaler_max      = 3
environment         = "prod"
