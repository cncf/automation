# Cluster

Each provider that donates resources to the CNCF has a folder here where we layout the Kubernetes cluster that will 
run external GitHub Action Runners for use by Projects governed by the CNCF.


## Cluster Directory Overview

``` shell
├── [PROVIDER]     # cluster deployment artefacts for ...
│   ├── arc        # ... the Actions Runner Controller
│   ├── argocd     # ... ArgoCD used to setup the external Github Action Runners
│   ├── autoscaler # ... autoscales resources for runner based on activity 
│   ├── runners    # ... defines how the runners look from a resource point of view    
```

## References
https://github.com/actions/actions-runner-controller


### Initial Proof of concept work
https://github.com/jeefy/gha-runner - ARC Work by @jeefy
https://github.com/jeefy/automation/commit/e3524e225b14dcd4b40663b8a73ee46421f5ab51 ArgoWork
