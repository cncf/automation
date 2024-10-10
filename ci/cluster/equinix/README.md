# ArgoCD config 

ArgoCD will be used to deploy new external GitHub Action runners

## Cluster

Using a `c2.medium.x86` for the Control Plane node.

```
export API_IP=$(jq -r '.network.addresses | map(select(.public==true and .management==true)) | first | .address' /tmp/metadata.json)
echo $API_IP
export INSTALL_K3S_EXEC="\
    --bind-address ${API_IP} \
    --advertise-address ${API_IP} \
    --node-ip ${API_IP} \
    --tls-san ${API_IP} \
    --no-deploy servicelb \
    --cluster-cidr=10.42.0.0/16,2001:cafe:42::/56 \
    --service-cidr=10.43.0.0/16,2001:cafe:43::/112 \
    --kubelet-arg cloud-provider=external \
    --disable-cloud-controller"
curl -sfL https://get.k3s.io | sh -
```

Get the K3s Join token: `cat /var/lib/rancher/k3s/server/node-token` and use it as part of the Autoscaler's cloud-init (part of `secrets/cluster-autoscaler-secret.yaml`)

Get the K3s admin.conf: `cat /etc/rancher/k3s/k3s.yaml` and store it in 1Password

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