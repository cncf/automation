## Cluster Prep 

```
kubectl apply -f ccm-secret.yaml

RELEASE=v3.8.1 \
kubectl apply -f https://github.com/equinix/cloud-provider-equinix-metal/releases/download/${RELEASE}/deployment.yaml

kubectl apply -f cluster-autoscaler-secret.yaml 
kubectl apply -f cluster-autoscaler-svcaccount.yaml 
kubectl apply -f cluster-autoscaler-deployment.yaml

kubectl create ns argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/$ARGOCD_VERSION/manifests/core-install.yaml

kubectl apply -n argocd -f cncf-automation.yaml
```