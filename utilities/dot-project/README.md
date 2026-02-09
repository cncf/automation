# .project - CNCF Project Metadata

Every CNCF project maintains a `.project` repository in their GitHub organization containing standardized metadata about the project. This enables maintainers to own their own data while CNCF automation can act on it for landscape updates, governance audits, staleness checks, and more.

## Quick Start

For CNCF projects adopting `.project`:

1. Copy the `template/` directory contents into your `.project` repo
2. Fill in your project details in `project.yaml` and `maintainers.yaml`
3. The included GitHub Actions workflow will validate on every PR

## Schema (v1.0.0)

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `schema_version` | string | Must be `"1.0.0"` |
| `slug` | string | Unique project identifier (lowercase, alphanumeric + hyphens) |
| `name` | string | Display name |
| `description` | string | One-line description |
| `maturity_log` | array | At least one entry with phase, date, issue URL |
| `repositories` | array | At least one valid HTTP(S) URL |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Project type (e.g., "project", "platform", "specification") |
| `project_lead` | string | GitHub handle of primary contact |
| `cncf_slack_channel` | string | CNCF Slack channel (must start with `#`) |
| `website` | string | Project website URL |
| `artwork` | string | Artwork/logo URL |
| `social` | map | Platform-name to URL mapping |
| `mailing_lists` | array | Email addresses |
| `audits` | array | Security/performance audit entries |
| `security` | object | Security policy, threat model, contact email |
| `governance` | object | Contributing, codeowners, governance doc paths |
| `legal` | object | License path |
| `documentation` | object | Readme, support, architecture, API doc paths |
| `landscape` | object | CNCF Landscape category and subcategory |

### Maturity Phases

Valid values for `maturity_log[].phase`: `sandbox`, `incubating`, `graduated`, `archived`

Entries must be in chronological order.

### Example: Minimal `project.yaml`

```yaml
schema_version: "1.0.0"
slug: "my-project"
name: "My Project"
description: "A brief description of my project"
maturity_log:
  - phase: "sandbox"
    date: "2024-01-15T00:00:00Z"
    issue: "https://github.com/cncf/toc/issues/XXX"
repositories:
  - "https://github.com/my-org/my-project"
```

### Example: Full `project.yaml`

See `yaml/example-project.yaml` or `template/project.yaml` for a complete example with all fields.

## Tools

### Validator

Validates `project.yaml` and `maintainers.yaml` files.

```bash
# Build
make build

# Validate with defaults
./bin/validator

# Validate specific files
./bin/validator -config yaml/projectlist.yaml -maintainers yaml/maintainers.yaml

# Output as JSON
./bin/validator -output json

# Skip maintainer validation
./bin/validator -maintainers ""

# Enable LFX handle verification
./bin/validator -verify-maintainers
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `yaml/projectlist.yaml` | Path to project list configuration |
| `-maintainers` | `yaml/maintainers.yaml` | Path to maintainers file (empty to skip) |
| `-base-maintainers` | | Base maintainers file for diff validation |
| `-cache` | `.cache` | Cache directory |
| `-output` | `text` | Output format: `text`, `json`, `yaml` |
| `-verify-maintainers` | `false` | Verify handles via LFX API |

### Landscape Updater

The `landscape-updater` tool automates the process of updating the CNCF Landscape YAML based on changes in project metadata.

```bash
# Dry run to see what would change
./bin/landscape-updater --project ./project.yaml --landscape ./landscape.yml --dry-run

# Apply changes and create a PR
./bin/landscape-updater --project ./project.yaml --landscape ./landscape.yml --create-pr
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | | Path to the project's `project.yaml` file (required) |
| `--landscape` | | Path to the `landscape.yml` file (required) |
| `--landscape-repo` | `cncf/landscape` | Target repository for the PR |
| `--create-pr` | `false` | Create a Pull Request with the changes |
| `--dry-run` | `false` | Print diff and PR details without executing |

### Bootstrap

The `bootstrap` tool auto-generates `project.yaml` and `maintainers.yaml` scaffolds by fetching data from CLOMonitor, GitHub, and the CNCF landscape. It discovers maintainer handles from CODEOWNERS, OWNERS, and MAINTAINERS files.

```bash
# Dry run: preview generated YAML on stdout
./bin/bootstrap -name "My Project" -github-org my-org -dry-run

# Generate scaffold files in current directory
./bin/bootstrap -name "My Project" -github-org my-org -github-repo my-repo

# Generate into a specific directory
./bin/bootstrap -name "Envoy" -github-org envoyproxy -github-repo envoy -output-dir ./envoy/.project

# Skip external API lookups (GitHub-only)
./bin/bootstrap -github-org my-org -skip-clomonitor

# Use a GitHub token for higher rate limits
GITHUB_TOKEN=ghp_xxx ./bin/bootstrap -name "My Project" -github-org my-org
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-name` | | Project display name to search for |
| `-github-org` | | GitHub organization |
| `-github-repo` | | Primary repository name (defaults to org name) |
| `-github-token` | | GitHub token (or set `GITHUB_TOKEN` env) |
| `-output-dir` | `.` | Directory to write scaffold output |
| `-skip-clomonitor` | `false` | Skip CLOMonitor API lookup |
| `-skip-github` | `false` | Skip GitHub API lookup |
| `-dry-run` | `false` | Print generated YAML without writing files |

#### Data Sources and Priority

The bootstrap tool fetches data from multiple sources and merges them with this priority order:

1. **CNCF Landscape** (highest priority) - project name, description, website, repos, maturity
2. **CLOMonitor** - project metadata, scores, repository list
3. **GitHub API** (fallback) - repo description, org info, community health profile

Maintainer discovery checks these files (in the repo root, `.github/`, and org `.github` repo):
- `CODEOWNERS` - extracts `@handle` references
- `OWNERS` - parses Kubernetes-style YAML (approvers/reviewers)
- `MAINTAINERS` / `MAINTAINERS.md` - heuristic extraction of handles, tables, GitHub URLs

### Staleness Checker

Checks if maintainer data hasn't been updated within a threshold.

```bash
./bin/staleness-checker -project project.yaml -threshold 180
```

### Audit Checker

Verifies all URLs referenced in a project are accessible.

```bash
./bin/audit-checker -project project.yaml
```

## GitHub Actions

### Using the Validate Project Action

```yaml
- uses: cncf/automation/.github/actions/validate-project@main
  with:
    project_file: 'project.yaml'
```

### Using the Validate Maintainers Action

```yaml
- uses: cncf/automation/.github/actions/validate-maintainers@main
  with:
    maintainers_file: 'maintainers.yaml'
    verify_maintainers: 'true'
  env:
    LFX_AUTH_TOKEN: ${{ secrets.LFX_AUTH_TOKEN }}
```

### Reusable Workflow

```yaml
jobs:
  validate:
    uses: cncf/automation/.github/workflows/reusable-validate-maintainers.yaml@main
    with:
      maintainers-file: 'maintainers.yaml'
    secrets:
      LFX_AUTH_TOKEN: ${{ secrets.LFX_AUTH_TOKEN }}
```

### Landscape Update Action

```yaml
name: Update Landscape
on:
  push:
    paths:
      - 'project.yaml'
    branches:
      - main

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Update Landscape
        uses: cncf/automation/.github/actions/landscape-update@main
        with:
          project_file: './project.yaml'
          token: ${{ secrets.LANDSCAPE_REPO_TOKEN }}
```

## Maintainer Verification

When `-verify-maintainers` is enabled, handles are verified against the Linux Foundation's LFX platform. Set `LFX_AUTH_TOKEN` for production use.

Environment variables:

| Variable | Description |
|----------|-------------|
| `LFX_AUTH_TOKEN` | Bearer token for LFX API |
| `MAINTAINER_API_ENDPOINT` | Alternative verification endpoint |
| `MAINTAINER_API_STUB` | Set to `fail` to simulate verification failure |
| `REPO_ROOT` | Repository root for resolving relative config paths |

## Development

```bash
make build          # Build all binaries to bin/
make docker-build   # Build Docker image
make test           # Run tests
make test-coverage  # Run tests with coverage report
make fmt            # Format code
make lint           # Run linter (requires golangci-lint)
make security       # Run security checks (requires gosec)
make clean          # Clean build artifacts
```

### Docker

```bash
docker build -t dot-project-validator .

# Run validator
docker run --rm -v $(pwd)/yaml:/app/yaml dot-project-validator -config /app/yaml/projectlist.yaml

# Run landscape-updater
docker run --rm --entrypoint landscape-updater dot-project-validator --help
```

## Implementation Guide for CNCF Projects

### Quick Start for Projects

1. **Create a `project.yaml` file** in your repository root with your project metadata (see Schema section above).

2. **Create a `MAINTAINERS.yaml` file** (optional but recommended):

```yaml
maintainers:
  - project_id: "your-project"
    teams:
      - name: "project-maintainers"
        members:
          - githubuser1
          - githubuser2
```

Each maintainer entry must contain a `project-maintainers` team which cannot be empty. Handles are normalized (trimmed and stripped of leading `@`) before verification.

3. **Add GitHub Actions** to automatically validate changes (see GitHub Actions section above).

### Benefits for Projects

- **Automated validation**: Catch metadata errors before they propagate
- **Landscape sync**: Automatically update CNCF Landscape when your metadata changes
- **Maintainer verification**: Optional validation of maintainer GitHub handles
- **Change detection**: SHA256-based caching detects only meaningful changes

### Required Files

| File | Required | Purpose |
|------|----------|---------|
| `project.yaml` | Yes | Core project metadata and references |
| `MAINTAINERS.yaml` | Recommended | Maintainer roster for verification |
| `SECURITY.md` | Recommended | Security policy (referenced in project.yaml) |
| `CONTRIBUTING.md` | Recommended | Contribution guidelines |
| `GOVERNANCE.md` | Recommended | Project governance document |

## Schema Versioning

The `schema_version` field is required and validated. Currently supported: `1.0.0`.

New schema versions will be added as the format evolves. The validator supports multiple versions simultaneously to allow gradual migration.

## Support

For questions or issues with the validation tools:
- Open an issue in [cncf/automation](https://github.com/cncf/automation)
- Check existing examples in `utilities/dot-project/yaml/`
