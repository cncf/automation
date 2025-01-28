# Setting Up a Local KIND Cluster with GitHub Actions Integration

This guide walks you through setting up a local KIND cluster and integrating it with GitHub Actions using the Actions Runner Controller (ARC). 

---

## Table of Contents

##### Step 1: Set Up a Local KIND Cluster
1. Install KIND:
    - Follow the installation guide from [KIND official documentation.](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
2. Create a Cluster
    ```bash
    kind create cluster --name cncf-cluster
    ```
     
##### Step 2: Configure GitHub Actions to Use KIND
1. Set Up Runner:
    - Refer to the [ARC Work by @jeefy](https://github.com/jeefy/gha-runner) for setting up the Actions Runner Controller (ARC).

2. Apply Cluster Configuration:
    - Navigate to the directory where your configuration files are located.
    - Apply the configuration:
        ```bash
        kubectl apply -f path/to/your/configuration.yaml
        ```

##### Step 3: Test GitHub Actions Configuration
- Run a Workflow Locally:
    - Use the following command to run a GitHub Actions workflow locally:
        ```bash
        act -j your_workflow_job
        ```
- Verify the Setup:
    - Ensure the runners are working as expected by checking the logs and the status of the pods:
        ```bash
        kubectl get pods -n arc-systems
        ```

##### References
- [GitHub Actions Runner Controller](https://github.com/actions/actions-runner-controller)
- [Kubernetes KIND](https://kind.sigs.k8s.io/)

##### Additional Resources
- [Cluster Directory Overview](https://github.com/cncf/automation/blob/aa2b88357be3c5d815ef87fc68c4fda2e3f6076f/ci/cluster/README.MD#L1-L40)
- [Autoscaler Deployment](https://github.com/cncf/automation/blob/aa2b88357be3c5d815ef87fc68c4fda2e3f6076f/ci/cluster/equinix/autoscaler/deployment.yaml#L51-L154)
