# Deploy and Update ArgoCD

In order to deploy (or version update by changing [kustomization.yaml](./kustomization.yaml#L6)) ArgoCD on the cluster, run this command:

```bash
kubectl apply -k argocd
```

## Deploy app-of-apps

Pick your cluster and deploy the related application (e.g. `oci`):

```bash
# cluster manifests: cluster/<cluster-name>/argo-automation.yaml
kubectl apply -f cluster/oci/argo-automation.yaml
```
