# CNCF Projects SBOM Generator

This tool generates Software Bill of Materials (SBOM) in SPDX JSON format for CNCF projects.

## Overview

The SBOM generator:
- Fetches stable releases (major, minor, patch) from CNCF project repositories
- Skips alpha, beta, RC, and other pre-release versions
- Generates SPDX-compliant SBOM files using the [kubernetes-sigs/bom](https://github.com/kubernetes-sigs/bom) tool
- Stores SBOMs in a structured directory format: `supply-chain/sbom/{projectname}/{reponame}/{version}/{reponame}.json`

## Directory Structure

```
supply-chain/sbom/
├── index.json                          # Index of all generated SBOMs
├── kubernetes/
│   └── kubernetes/
│       ├── 1.29.0/
│       │   └── kubernetes.json
│       ├── 1.28.5/
│       │   └── kubernetes.json
│       └── 1.27.10/
│           └── kubernetes.json
├── prometheus/
│   └── prometheus/
│       └── 2.48.0/
│           └── prometheus.json
└── ...
```

## GitHub Actions Workflow

The workflow (`.github/workflows/generate-sbom.yml`) runs:
- **Scheduled**: Weekly on Sunday at 02:00 UTC
- **Manual trigger**: Via workflow_dispatch with optional filters

### Workflow Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `project_filter` | Filter by owner/repo (e.g., "kubernetes/kubernetes") | empty (all projects) |
| `force_regenerate` | Force regenerate existing SBOMs | false |

### Running the Workflow

1. Go to Actions tab in GitHub
2. Select "Generate SBOM for CNCF Projects"
3. Click "Run workflow"
4. Optionally specify a project filter

## Local Testing

### Prerequisites

- Go 1.22+
- git
- [GitHub CLI (gh)](https://cli.github.com/)
- [jq](https://stedolan.github.io/jq/) (for bash script)
- [yq](https://github.com/mikefarah/yq)

### Bash Script (Linux/macOS/WSL)

```bash
# Process all projects
./supply-chain/sbom/generate-sbom-local.sh

# Process specific repo
./supply-chain/sbom/generate-sbom-local.sh kubernetes/kubernetes

# Force regenerate
./supply-chain/sbom/generate-sbom-local.sh --force coredns/coredns

# Set max releases per repo (default: 3)
MAX_RELEASES=5 ./supply-chain/sbom/generate-sbom-local.sh
```

### PowerShell Script (Windows)

```powershell
# Process all projects
.\supply-chain\sbom\generate-sbom-local.ps1

# Process specific repo
.\supply-chain\sbom\generate-sbom-local.ps1 -ProjectFilter "kubernetes/kubernetes"

# Force regenerate
.\supply-chain\sbom\generate-sbom-local.ps1 -Force -ProjectFilter "coredns/coredns"

# Set max releases per repo
.\supply-chain\sbom\generate-sbom-local.ps1 -MaxReleases 5
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GH_TOKEN` or `GITHUB_TOKEN` | GitHub token for API access (optional but recommended for higher rate limits) |
| `MAX_RELEASES` | Maximum releases to process per repo (default: 3, bash only) |

## Adding New Projects

To add new CNCF projects for SBOM generation, edit `supply-chain/util/data/repositories.yaml`:

```yaml
repositories:
  - owner: organization-name
    repo: repository-name
    name: Project Display Name
    category: category-name
```

## Release Filtering

The generator only processes stable releases:

**Included:**
- Full releases (e.g., v1.0.0, v2.5.3)
- Releases marked as non-prerelease and non-draft in GitHub

**Excluded:**
- Alpha releases (e.g., v1.0.0-alpha.1)
- Beta releases (e.g., v1.0.0-beta.2)
- Release candidates (e.g., v1.0.0-rc1)
- Development versions (e.g., v1.0.0-dev)
- Snapshots, nightly, canary builds
- Draft releases

## SBOM Format

Generated SBOMs are in SPDX 2.3 JSON format, containing:
- Package information
- File listings with checksums
- Dependency relationships
- License information (when available)

## Index File

The `index.json` file provides a searchable index of all generated SBOMs:

```json
{
  "generated_at": "2024-01-15T10:30:00Z",
  "sboms": [
    {
      "project": "kubernetes",
      "repo": "kubernetes",
      "version": "1.29.0",
      "path": "kubernetes/kubernetes/1.29.0/kubernetes.json"
    }
  ]
}
```

## Troubleshooting

### Rate Limiting

If you encounter GitHub API rate limits:
1. Set the `GH_TOKEN` environment variable with a valid GitHub token
2. Run `gh auth login` to authenticate the GitHub CLI

### bom Tool Installation Fails

Ensure Go is properly installed and `GOPATH/bin` is in your PATH:

```bash
go install sigs.k8s.io/bom/cmd/bom@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

### Clone Failures

Some repositories may have protected tags or require authentication. The script will skip these and continue with other releases.

## Contributing

1. Fork the repository
2. Add new projects to `supply-chain/util/data/repositories.yaml`
3. Test locally with the provided scripts
4. Submit a pull request
