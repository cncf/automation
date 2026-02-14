#!/bin/bash

CLUSTER_NAME="kind-test"
KIND_CONFIG="kind-config.yaml"

sudo sysctl fs.inotify.max_user_instances=1280
sudo sysctl fs.inotify.max_user_watches=655360

cat <<EOF > $KIND_CONFIG
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
EOF

echo "[*] Creating Kind cluster..."
kind create cluster --name $CLUSTER_NAME --config $KIND_CONFIG

kubectl wait --for=condition=Ready nodes --all --timeout=120s

echo "[*] Creating pod ..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:latest
    resources:
      requests:
        memory: "256Mi"
        cpu: "100m"
      limits:
        memory: "256Mi"
        cpu: "500m"
EOF

echo "[*] Waiting for pods to be ready..."
kubectl wait --for=condition=Ready pod/nginx --timeout=300s

echo "[*] Pods running, doing test workload..."
sleep 60

echo "[*] Deleting pods..."
kubectl delete pod nginx

echo "[*] Deleting Kind cluster..."
kind delete cluster --name $CLUSTER_NAME

echo "[*] Done!"
