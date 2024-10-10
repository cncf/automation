# ArgoCD config 

ArgoCD will be used to deploy new external GitHub Action runners

## Cluster Prep 

```
# Create required namespaces
kubectl create ns arc-systems
kubectl create ns argocd

# Populate required secrets
kubectl apply -f secrets/ccm-secret.yaml
kubectl apply -f secrets/cluster-autoscaler-secret.yaml
kubectl apply -f secrets/github-arc-secret.yaml

# Install ArgoCD
export ARGOCD_VERSION="v2.12.4"
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/$ARGOCD_VERSION/manifests/core-install.yaml

# ArgoCD go brrrrrrr
kubectl apply -n argocd -f cncf-automation.yaml
```