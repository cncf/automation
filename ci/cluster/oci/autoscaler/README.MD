# Cluster autoscaler configuration

Configuration files for the cluster autoscaler for the OKE cluster running
external GitHub Actions.

REFS:
Step by Step
https://docs.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengusingclusterautoscaler_topic-Working_with_the_Cluster_Autoscaler.htm#Working_with_the_Cluster_Autoscaler

OKE Workload Identity: Greater control of access
https://blogs.oracle.com/cloud-infrastructure/post/oke-workload-identity-greater-control-access

## Step 1: Setting Up an Instance Principal or Workload Identity Principal to Enable Cluster Autoscaler Access to Node Pools

### Using instance principals to enable access to node pools🔗
Created Instance Principal

### Create a new compartment-level dynamic group containing the worker nodes (compute instances) in the cluster:

https://cloud.oracle.com/identity/domains/ocid1.domain.oc1..aaaaaaaaqlvbp36i7exr5phcr4jy4o33fn7vw5vtd4h4rxmwzzfpf4dtylea/dynamic-groups/ocid1.dynamicgroup.oc1..aaaaaaaa7qbdtn3zbnph3yy62gjyr5i2ls7cvwe3pzoimmjckzg5cyki3bzq/application-roles?region=us-sanjose-1

### Policy to allow work nodes to manage nodes pools:

https://cloud.oracle.com/identity/domains/policies/ocid1.policy.oc1..aaaaaaaanawfi3j4otvdhlefhgf5fogr2wnhjzljpxmf4afjwufd3zknmk7q?region=us-sanjose-1

### Using workload identity principals to enable access to node pools

https://cloud.oracle.com/identity/domains/policies/ocid1.policy.oc1..aaaaaaaaqbjexxhyrjdjf2py2vchiz6dg7ewt4qburayq7n35k4fnuoirg7q?region=us-sanjose-1S

Step 2: Copy and customize the Cluster Autoscaler configuration file

