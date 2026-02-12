# Hacks

Operational workarounds and maintenance CronJobs for the OCI amd64 runner cluster.

## Overview

This directory contains DaemonSets and CronJobs that address platform-level issues or perform periodic cleanup tasks that are not covered by standard Kubernetes operators.

## Files

| File | Description |
|------|-------------|
| `cgroups-v2-enabler-ds.yaml` | DaemonSet that ensures cgroups v2 is enabled on all nodes |
| `ephemeralrunner-cleanup-cj.yaml` | CronJob that cleans up failed EphemeralRunner resources every 10 minutes |
| `vm-cleaner.yaml` | CronJob that terminates stale OCI VM-based runners older than 2 days |

## cgroups v2 Enabler

A DaemonSet running in `kube-system` that checks whether cgroups v2 is active on each node. If cgroups v1 is detected, it uses `grubby` to update the kernel boot parameters and reboots the node.

- **Namespace**: `kube-system`
- **Runs as**: Privileged container with `hostPID` and `hostNetwork`
- **Image**: `docker.io/library/alpine`

## EphemeralRunner Cleanup

A CronJob in the `arc-systems` namespace that runs every 10 minutes to delete `EphemeralRunner` resources that are in a `Failed` state or have recorded failures in their status.

- **Schedule**: `*/10 * * * *`
- **Namespace**: `arc-systems`
- **Image**: `docker.io/bitnamilegacy/kubectl:1.32`
- **RBAC**: Dedicated ServiceAccount with a Role granting full access to `ephemeralrunners` resources

## VM Cleaner

A CronJob that runs daily at 00:10 UTC to terminate OCI compute instances with names starting with `gha-runner-` that have been running for more than 2 days. This prevents orphaned VM-based runners from accumulating.

- **Schedule**: `10 00 * * *` (daily)
- **Namespace**: `arc-systems`
- **Image**: `ghcr.io/oracle/oci-cli` (tag is pinned to a specific release in the manifest)
- **Region**: Configured per deployment
- **Requires**: `oci-config` and `oci-api-key` Secrets mounted for OCI CLI authentication

## Relationship to Other Components

- Deployed at sync-wave `5` via [`../argo-automation.yaml`](../argo-automation.yaml)
- The EphemeralRunner cleanup supports the container-based runners in [`../runners/`](../runners/)
- The VM cleaner supports the VM-based runners in [`../vm-runners/`](../vm-runners/)
