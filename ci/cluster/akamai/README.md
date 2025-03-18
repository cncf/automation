# Akamai Provider for CNCF Self-Hosted Runners (PoC)

This directory contains automation tools and configurations for deploying and managing CNCF GitHub self-hosted runners on Akamai infrastructure.

> **Note:** This implementation is currently in Proof of Concept (PoC) stage.

## Overview

The Akamai provider enables CNCF projects to leverage Akamai's cloud infrastructure for running CI/CD workflows with GitHub Actions. These self-hosted runners offer enhanced performance, customized environments, and dedicated resources tailored to CNCF project needs.

This automation specifically provisions and manages **Linode managed Kubernetes clusters** and deploys **Actions Runner Controller (ARC)** to handle GitHub Actions workloads efficiently.

## Features

- Automated provisioning of managed Kubernetes clusters on Linode
- Deployment and configuration of Actions Runner Controller (ARC)
- Runner configuration and registration with GitHub
- Auto-scaling capabilities based on workflow demand
- Monitoring and maintenance utilities
- Support for multiple GitHub organizations and repositories

## Prerequisites

- Akamai cloud account with appropriate permissions
- Linode API credentials for Kubernetes cluster management
- Service account credentials configured for automation
- GitHub Personal Access Token (PAT) with appropriate permissions

## Configuration

Configuration is managed through environment variables and config files:

- `AKAMAI_API_KEY`: API key for accessing Akamai services
- `AKAMAI_API_SECRET`: API secret for authentication
- `LINODE_API_TOKEN`: API token for Linode Kubernetes service
- `GITHUB_PAT`: GitHub Personal Access Token for runner registration

See the sample configuration file in `config-example.yaml` for detailed settings.

## Usage

Detailed usage instructions for provisioning and managing runners are coming soon.

### Proof of Concept Deployment

This PoC uses an intentionally cost-effective setup with spot instances to demonstrate the functionality at minimal expense. The configuration is not intended for production use without appropriate adjustments.

## Kubernetes Deployment

This provider automatically:
1. Creates a Kubernetes cluster in Linode
2. Installs and configures Actions Runner Controller using Helm
3. Sets up runner scale sets for GitHub repositories/organizations
4. Configures auto-scaling based on workflow demand

## Troubleshooting

Common issues and their solutions will be documented as they are encountered.

## Contributing

Contributions to improve the Akamai provider are welcome! Please follow the contributing guidelines in the root of this repository.
