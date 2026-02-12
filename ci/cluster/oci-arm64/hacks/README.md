# Hacks

Operational workarounds for the OCI ARM64 runner cluster.

## Overview

This directory contains DaemonSets that address platform-level issues not covered by standard Kubernetes operators.

## Files

| File | Description |
|------|-------------|
| `cgroups-v2-enabler-ds.yaml` | DaemonSet that ensures cgroups v2 is enabled on all nodes |

## cgroups v2 Enabler

A DaemonSet running in `kube-system` that checks whether cgroups v2 is active on each node. If cgroups v1 is detected, it uses `grubby` to update the kernel boot parameters and reboots the node.

- **Namespace**: `kube-system`
- **Runs as**: Privileged container with `hostPID` and `hostNetwork`
- **Image**: `docker.io/library/alpine`
- **Behavior**: If `/sys/fs/cgroup/cgroup.controllers` exists, cgroups v2 is confirmed. Otherwise, the node kernel is updated and rebooted.

## Relationship to Other Components

- Deployed at sync-wave `5` via [`../argo-automation.yaml`](../argo-automation.yaml)
- Ensures cgroups v2 compatibility required by the container runners in [`../runners/`](../runners/)
