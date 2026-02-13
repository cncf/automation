# CNCF Projects Supply Chain & SBOM Generator

## ⚠️ DISCLAIMER

**IMPORTANT NOTICE:**

The Software Bill of Materials (SBOM) files in this directory are **automatically generated** by the CNCF and are **NOT official SBOMs** provided by the individual CNCF projects themselves.

1. **No Warranty**: These SBOMs are provided "AS IS" without warranty of any kind, express or implied, including but not limited to the warranties of merchantability, fitness for a particular purpose, and noninfringement.

2. **No Guarantee of Completeness or Accuracy**: We make no representations or guarantees regarding the completeness, accuracy, reliability, or currentness of the information contained in these SBOMs. The automated generation process may miss dependencies, include incorrect versions, or contain other errors.

3. **Use at Your Own Risk**: Any use of these SBOMs is entirely at your own risk. This roject, its contributors, and maintainers shall not be liable for any claims, damages, or other liability arising from the use of these SBOMs.

4. **Not a Substitute for Official SBOMs**: For production use, compliance requirements, or security audits, please refer to the official documentation and releases of each individual CNCF project.

5. **Automated Generation**: These SBOMs are generated using the `kubernetes-sigs/bom` tool and `go-licenses` for license detection. The generation process runs weekly and only processes Go-based projects.

---

## Overview

This tool generates Software Bill of Materials (SBOM) in SPDX JSON format for CNCF projects.

The SBOM generator:
- Automatically syncs the list of CNCF projects from the [CNCF Landscape](https://landscape.cncf.io)
- Fetches stable releases (major, minor, patch) from CNCF project repositories
- Only processes Go-based projects (repositories containing `go.mod` or `go.sum`)
- Skips alpha, beta, RC, and other pre-release versions
- Generates SPDX-compliant SBOM files using the [kubernetes-sigs/bom](https://github.com/kubernetes-sigs/bom) tool
- Enriches SBOMs with license information using [google/go-licenses](https://github.com/google/go-licenses)
- Stores SBOMs in a structured directory format: `sbom/{projectname}/{reponame}/{version}/{reponame}.json`

## Directory Structure

```
supply-chain/
├── README.md                           # This file
├── sbom/                               # Generated SBOM files
│   ├── index.json                      # Index of all generated SBOMs
│   ├── <project-name>/                 # Official CNCF projects
│   │   └── <repo-name>/
│   │       └── <version>/
│   │           └── <repo>.json         # SPDX SBOM file
│   └── subprojects/                    # Subproject repos from CNCF orgs
│       └── <owner>/
│           └── <repo>/
│               └── <version>/
│                   └── <repo>.json     # SPDX SBOM file
└── util/
    ├── data/
    │   ├── cncf-projects.yaml          # Auto-synced CNCF project list (DO NOT EDIT)
    │   └── discovered-repos.yaml       # Subproject repos found in CNCF orgs (DO NOT EDIT)
    ├── extract-projects/               # Go tool to sync projects from CNCF landscape
    ├── discover-repos/                 # Go tool to find subproject repos in CNCF orgs
    ├── cleanup-sbom/                   # Go tool to remove orphaned SBOM folders
    ├── generate-index/                 # Go tool to generate index.json
    ├── generate-sbom-local.sh          # Local testing script (Linux/macOS)
    └── generate-sbom-local.ps1         # Local testing script (Windows)
```
```

## GitHub Actions Workflows

### 1. Sync CNCF Projects (`.github/workflows/sync-cncf-projects.yml`)

Automatically syncs the list of CNCF projects from the official landscape.

- **Scheduled**: Daily at 03:00 UTC
- **Manual trigger**: Via workflow_dispatch
- **Output**: `util/data/cncf-projects.yaml`

### 2. Discover Additional Repos (`.github/workflows/discover-cncf-repos.yml`)

Scans GitHub organizations of CNCF projects to find additional subproject repositories with releases.

- **Scheduled**: Weekly on Monday at 04:00 UTC
- **Manual trigger**: Via workflow_dispatch
- **Output**: `util/data/discovered-repos.yaml`

This workflow finds subproject repositories that:
- Belong to the same GitHub org/user as a CNCF project
- Have at least one release
- Contain a `go.mod` file (Go-based project)
- Are not forks, archived, or disabled

### 3. Generate SBOMs (`.github/workflows/generate-sbom.yml`)

Generates SBOMs for CNCF projects.

- **Scheduled**: Weekly on Sunday at 02:00 UTC (processes only releases from the past week)
- **Manual trigger**: Via workflow_dispatch with optional filters

| Input | Description | Default |
|-------|-------------|---------|
| `project_filter` | Filter by owner/repo (e.g., "coredns/coredns") | empty (all projects) |
| `force_regenerate` | Force regenerate existing SBOMs | false |
| `releases_mode` | `recent` (past week) or `latest` (N latest releases) | recent |
| `max_releases` | Max releases per repo (only for `latest` mode) | 3 |

### Running the Workflow Manually

1. Go to Actions tab in GitHub
2. Select "Generate SBOM for CNCF Projects"
3. Click "Run workflow"
4. Optionally specify a project filter or change the releases mode

## Utility Tools

### extract-projects

Go tool that downloads the CNCF landscape and extracts all projects with status `graduated`, `incubating`, or `sandbox`.

```bash
cd supply-chain/util/extract-projects
go run . ../data/cncf-projects.yaml
```

### discover-repos

Go tool that scans GitHub organizations of CNCF projects to find additional subproject repositories with releases.

```bash
cd supply-chain/util/discover-repos
go run . /path/to/cncf-automation
```

The tool will:
- Read the list of CNCF projects from `cncf-projects.yaml`
- Scan each unique GitHub organization/user
- Find subproject repos that have releases and contain `go.mod`
- Output results to `discovered-repos.yaml`

### cleanup-sbom

Go tool that removes SBOM folders for projects no longer in the CNCF list.

```bash
cd supply-chain/util/cleanup-sbom
go run . /path/to/cncf-automation --dry-run  # Preview changes
go run . /path/to/cncf-automation            # Execute cleanup
```

### generate-index

Go tool that generates the `index.json` file for all SBOMs.

```bash
cd supply-chain/util/generate-index
go run . /path/to/cncf-automation
```

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
./supply-chain/util/generate-sbom-local.sh

# Process specific repo
./supply-chain/util/generate-sbom-local.sh coredns/coredns

# Force regenerate
./supply-chain/util/generate-sbom-local.sh --force coredns/coredns

# Set max releases per repo (default: 3)
MAX_RELEASES=5 ./supply-chain/util/generate-sbom-local.sh
```

### PowerShell Script (Windows)

```powershell
# Process all projects
.\supply-chain\util\generate-sbom-local.ps1

# Process specific repo
.\supply-chain\util\generate-sbom-local.ps1 -ProjectFilter "coredns/coredns"

# Force regenerate
.\supply-chain\util\generate-sbom-local.ps1 -Force -ProjectFilter "coredns/coredns"

# Set max releases per repo
.\supply-chain\util\generate-sbom-local.ps1 -MaxReleases 5
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GH_TOKEN` or `GITHUB_TOKEN` | GitHub token for API access (recommended for higher rate limits) |
| `MAX_RELEASES` | Maximum releases to process per repo (default: 3, bash only) |

## Project List

CNCF projects are **automatically synced** from the official [CNCF Landscape](https://landscape.cncf.io). 

Projects with the following status are included:
- `graduated`
- `incubating`  
- `sandbox`

> **Note:** You do not need to manually add projects. The sync workflow runs daily and updates the project list automatically. If a project is missing, ensure it is properly listed in the CNCF Landscape with the correct project status.

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
- Package information with CNCF project metadata
- File listings with checksums
- Dependency relationships
- License information extracted via go-licenses

### Example SBOM Metadata

Each SBOM includes enriched metadata:
- CNCF project name and status
- GitHub repository URL
- Release tag and URL
- Generation timestamp
- Tool information

## Index File

The `sbom/index.json` file provides a searchable index of all generated SBOMs:

```json
{
  "generated_at": "2026-02-13T10:30:00Z",
  "sboms": [
    {
      "project": "coredns",
      "repo": "coredns",
      "version": "1.12.0",
      "path": "coredns/coredns/1.12.0/coredns.json"
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

### Non-Go Projects Skipped

The SBOM generator currently only supports Go-based projects. Projects without `go.mod` or `go.sum` in their root directory are automatically skipped.

## Contributing

1. Fork the repository
2. Make your changes
3. Test locally with the provided scripts
4. Submit a pull request

For issues with specific CNCF projects, please contact the respective project maintainers directly.
