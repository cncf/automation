# OCI GitHub Actions Runners - Quick Reference

Quick commands and references for managing the Oracle Cloud GitHub Actions infrastructure.

## Quick Commands

### Check Cluster Status

```bash
# Get all nodes
kubectl get nodes

# Check runner pods
kubectl get pods -n arc-systems

# Check ArgoCD applications
kubectl get applications -n argocd

# View runner logs
kubectl logs -n arc-systems <pod-name>
```

### ArgoCD Operations

```bash
# List all applications
argocd app list

# Sync specific application
argocd app sync github-runners

# Get application details
argocd app get github-runners

# Force sync (ignore differences)
argocd app sync github-runners --force

# Refresh application (re-check Git)
argocd app refresh github-runners
```

### Monitoring

```bash
# Port-forward Grafana
kubectl port-forward -n default svc/kube-prometheus-stack-grafana 3000:80

# Port-forward ArgoCD UI
kubectl port-forward -n argocd svc/argocd-server 8080:443

# Check Prometheus
kubectl port-forward -n default svc/kube-prometheus-stack-prometheus 9090:9090
```

### Debugging

```bash
# Describe runner pod
kubectl describe pod <pod-name> -n arc-systems

# Get events
kubectl get events -n arc-systems --sort-by='.lastTimestamp'

# Check ARC controller logs
kubectl logs -n arc-systems -l app.kubernetes.io/name=gha-runner-scale-set-controller

# Check Karpenter logs
kubectl logs -n karpenter -l app.kubernetes.io/name=karpenter
```

## Sync Wave Order

Applications deploy in this order:

| Wave | Component | Purpose |
|------|-----------|---------|
| -1 | External Secrets | Secrets must exist first |
| 2 | ARC Controller | Manages runner lifecycle |
| 3 | Runners | Actual runner pods |
| 4 | Monitoring | Observability stack |
| 5 | Hacks | Cleanup utilities |
| 10 | Karpenter | Node autoscaling |

## Available Runner Labels

### AMD64 Runners
- `oracle-2cpu-8gb-x86-64`
- `oracle-4cpu-16gb-x86-64`
- `oracle-8cpu-32gb-x86-64`
- `oracle-16cpu-64gb-x86-64`
- `oracle-24cpu-384gb-x86-64`

### ARM64 Runners
- `oracle-2cpu-8gb-arm64`
- `oracle-4cpu-16gb-arm64`
- `oracle-8cpu-32gb-arm64`
- `oracle-16cpu-64gb-arm64`
- `oracle-32cpu-128gb-arm64`

## Common Fixes

### Restart ARC Controller

```bash
kubectl rollout restart deployment -n arc-systems -l app.kubernetes.io/name=gha-runner-scale-set-controller
```

### Force Sync All Applications

```bash
argocd app sync -l cluster=oci-gha-amd64-runners
```

### Delete Stuck Runner Pod

```bash
kubectl delete pod <pod-name> -n arc-systems --force --grace-period=0
```

### Recreate Failed Application

```bash
argocd app delete <app-name>
kubectl apply -f ci/cluster/oci/argo-automation.yaml
```

## Useful Prometheus Queries

```promql
# Active runners
sum(kube_pod_status_phase{namespace="arc-systems", phase="Running"})

# Pending runners
sum(kube_pod_status_phase{namespace="arc-systems", phase="Pending"})

# Failed pods in last hour
count(kube_pod_status_phase{namespace="arc-systems", phase="Failed"})

# CPU usage by runner
sum(rate(container_cpu_usage_seconds_total{namespace="arc-systems"}[5m])) by (pod)

# Memory usage by runner
sum(container_memory_working_set_bytes{namespace="arc-systems"}) by (pod)
```

## Secrets

### Required Secrets

```yaml
# GitHub token secret
apiVersion: v1
kind: Secret
metadata:
  name: github-arc-secret
  namespace: arc-systems
type: Opaque
data:
  github_token: <base64-encoded-token>
```

### Create Secret Manually

```bash
kubectl create secret generic github-arc-secret \
  --from-literal=github_token=ghp_xxxxxxxxxxxx \
  --namespace=arc-systems
```

## Important Files

| File | Purpose |
|------|---------|
| `argo-automation.yaml` | Main ArgoCD app-of-apps |
| `arc/arc.yaml` | ARC controller config |
| `arc/values.yaml` | ARC Helm values |
| `runners/*/` | Runner configurations |
| `karpenter/nodepool.yaml` | Node autoscaling config |
| `monitoring/values.yaml` | Prometheus/Grafana config |

## Update Workflows

### Update Runner Image

1. Edit `ci/gha-runner-image/Dockerfile`
2. Build: `docker build -t <image> .`
3. Push: `docker push <image>`
4. Update runner manifests with new image tag
5. Commit and push - ArgoCD syncs automatically

### Update ARC Version

1. Edit `ci/cluster/oci/arc/arc.yaml`
2. Change `targetRevision: 0.11.0` to new version
3. Commit and push
4. ArgoCD syncs automatically

### Add New Runner Size

1. Copy existing runner directory: `cp -r runners/8cpu-32gb runners/12cpu-48gb`
2. Edit resource specifications
3. Update runner labels
4. Commit and push

## Emergency Procedures

### All Runners Down

```bash
# Check ARC controller
kubectl get pods -n arc-systems -l app.kubernetes.io/name=gha-runner-scale-set-controller

# Restart if needed
kubectl rollout restart deployment -n arc-systems

# Check GitHub token
kubectl get secret github-arc-secret -n arc-systems -o yaml
```

### Cluster Out of Resources

```bash
# Check node status
kubectl top nodes

# Check Karpenter
kubectl logs -n karpenter -l app.kubernetes.io/name=karpenter --tail=50

# Manually scale node pool (if needed)
# Use OCI console or CLI
```

### ArgoCD Not Syncing

```bash
# Check ArgoCD status
kubectl get pods -n argocd

# Restart ArgoCD
kubectl rollout restart deployment -n argocd argocd-server
kubectl rollout restart statefulset -n argocd argocd-application-controller

# Force refresh
argocd app sync --force --prune -l cluster=oci-gha-amd64-runners
```

## Escalation

If issues persist:

1. Check #cncf-ci Slack channel
2. Review recent commits to `cncf/automation`
3. Check OCI console for infrastructure issues
4. Create GitHub issue with logs and details

## Quick Links

- ArgoCD UI: `kubectl port-forward -n argocd svc/argocd-server 8080:443` then https://localhost:8080
- Grafana: `kubectl port-forward -n default svc/kube-prometheus-stack-grafana 3000:80` then http://localhost:3000
- GitHub Actions: https://github.com/cncf/automation/actions
- OCI Console: https://cloud.oracle.com/

---

Tip: Bookmark this page for quick access during incidents!
