# E2E Testing for GitHub Actions Runners

This document describes the automated end-to-end (E2E) testing system for CNCF's external GitHub Actions runners.

## Overview

The E2E testing system automatically validates all runner types when:
- Runner image Dockerfile changes
- Upstream dependencies are updated (GitHub Actions runner, Docker, Go, etc.)
- Weekly scheduled runs to catch any issues
- Manual triggers for testing specific runner types

## Workflows

### 1. E2E Runner Tests (`e2e-runner-tests.yml`)

**Triggers:**
- Push/PR to `main` with changes to `ci/gha-runner-image/**`
- After successful runner image builds
- Weekly schedule (Mondays at 2 AM UTC)
- Manual dispatch with scope selection

**Test Coverage:**
- **Oracle Runners**: All x86-64 and ARM64 variants
  - `oracle-4cpu-16gb-x86-64`
  - `oracle-8cpu-32gb-x86-64`
  - `oracle-16cpu-64gb-x86-64`
  - `oracle-24cpu-384gb-x86-64`
  - `oracle-2cpu-8gb-arm64`
  - `oracle-16cpu-64gb-arm64`
  - `oracle-32cpu-128gb-arm64`

- **Container Runners**: Various sizes on x86-64
  - `oracle-2cpu-8gb-x86-64`
  - `oracle-4cpu-16gb-x86-64`
  - `oracle-8cpu-32gb-x86-64`
  - `oracle-16cpu-64gb-x86-64`

- **VM Runners**: Both architectures
  - `oracle-vm-2cpu-8gb-x86-64`
  - `oracle-vm-2cpu-8gb-arm64`

- **Equinix Runners**: Basic testing (continues on error if unavailable)
  - `equinix-2cpu-8gb`

**Test Types:**
- Basic functionality (uname, system info)
- Docker functionality (run containers, build images)
- Development tools (Go, Python, Git, Make)
- Kubernetes capability (kind clusters for VM runners)
- Resource verification (CPU, memory, disk)

### 2. Upstream Dependency Monitoring (`monitor-upstream-runner-changes.yml`)

**Triggers:**
- Daily schedule (6 AM UTC)
- Manual dispatch

**Monitored Dependencies:**
- GitHub Actions Runner
- Docker CE
- Docker Buildx
- Runner Container Hooks
- Go language

**Actions on Changes:**
- Creates PR with updated versions
- Automatically triggers E2E tests
- Provides detailed change summary

### 3. Enhanced Image Publishing (`publish-runner-images.yml`)

**New Features:**
- Automatically triggers E2E tests after successful image builds
- Ensures new images are validated before deployment

## Manual Testing

### Run All Tests
```bash
gh workflow run e2e-runner-tests.yml --ref main -f test_scope=all
```

### Run Specific Runner Type
```bash
# Oracle runners only
gh workflow run e2e-runner-tests.yml --ref main -f test_scope=oracle-only

# Container runners only
gh workflow run e2e-runner-tests.yml --ref main -f test_scope=container-only

# VM runners only
gh workflow run e2e-runner-tests.yml --ref main -f test_scope=vm-only

# Equinix runners only
gh workflow run e2e-runner-tests.yml --ref main -f test_scope=equinix-only
```

### Check for Upstream Changes
```bash
gh workflow run monitor-upstream-runner-changes.yml --ref main
```

## Test Results

The E2E tests provide:
- **Individual runner validation**: Each runner type is tested independently
- **Failure isolation**: Failed tests don't block other runner types
- **Comprehensive reporting**: Summary job shows overall status
- **Detailed logs**: Each test step provides specific validation results

## Monitoring and Alerts

- **Weekly scheduled runs** catch any infrastructure issues
- **Automatic upstream monitoring** ensures timely updates
- **PR-based updates** provide review opportunity for changes
- **Test summary job** provides clear pass/fail status

## Troubleshooting

### Test Failures
1. Check the specific runner test job for detailed logs
2. Verify runner availability in the CNCF infrastructure
3. Check for upstream dependency issues
4. Review recent changes to runner images

### Upstream Update Issues
1. Review the auto-generated PR for version changes
2. Check compatibility between new versions
3. Run manual E2E tests before merging updates
4. Monitor test results after deployment

## Future Enhancements

- **Performance benchmarking**: Add performance tests for different workloads
- **Security scanning**: Integrate security scans for runner images
- **Resource utilization**: Monitor and report resource usage patterns
- **Integration testing**: Test with real CNCF project workflows