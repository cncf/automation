# Project Validator

A Go utility that validates CNCF project metadata and maintainer rosters. It checks project YAML manifests, reconciles maintainer lists against canonical sources, and surfaces changes via cached diffs.

## Features

- Validates project YAML files against structured schema requirements
- Detects content drift using SHA256 hashes and cached history
- Validates maintainer definitions against canonical `.project` repository data
- Stubbed third-party verification hook for maintainer identity checks
- Multiple output formats: human-readable text, JSON, YAML
- Includes GitHub Action and Makefile helpers

## Installation

### From Source

```bash
cd utilities/dot-project/
go build -o bin/validator ./cmd/validator
go build -o bin/landscape-updater ./cmd/landscape-updater
```

### Using Docker

```bash
cd utilities/dot-project/
docker build -t dot-project-validator .

# Run validator
docker run --rm -v $(pwd)/yaml:/app/yaml dot-project-validator -config /app/yaml/projectlist.yaml

# Run landscape-updater
docker run --rm --entrypoint landscape-updater dot-project-validator --help
```

## Usage

```bash
./validator [flags]
```

### Flags

- `-config string`: Path to project list configuration file (default `yaml/projectlist.yaml`)
- `-maintainers string`: Path to maintainer roster file (default `yaml/maintainers.yaml`, set empty to skip)
- `-base-maintainers string`: Path to base maintainers file for diff validation (optional)
- `-cache string`: Directory for cached validation results (default `.cache`)
- `-verify-maintainers`: Toggle stubbed external maintainer verification (default `false`)

### Examples

```bash
# Validate projects and maintainers with default configuration
./validator

# Validate only projects
./validator -maintainers "" 

# Validate projects and maintainers from custom paths with verification enabled
./validator -config configs/projectlist.yaml \
            -maintainers configs/maintainers.yaml \
            -verify-maintainers

# Validate only new/changed maintainers by excluding those in base file
./validator -maintainers maintainers.yaml \
            -base-maintainers previous-maintainers.yaml
```

## Configuration

### Project List (`yaml/projectlist.yaml`)

> **Note:** Sample configuration files rely on the `REPO_ROOT` environment variable. Set it to the repository root before running commands (e.g., `export REPO_ROOT="$(pwd)"`).

```yaml
projects:
  - url: "https://raw.githubusercontent.com/my-org/service/main/project.yaml"
    id: "service"
  - url: "file:///path/to/local/project.yaml"
    id: "local-project"
```

### Maintainers (`yaml/maintainers.yaml`)

```yaml
maintainers:
  - project_id: "service"
    teams:
      - name: "project-maintainers"
        members:
          - alice
          - bob
      - name: "other-team"
        members:
          - carol
```

Each maintainer entry must contain a `project-maintainers` team which cannot be empty. Handles are normalized (trimmed and stripped of leading `@`) before verification.

### Project YAML Format

Each project YAML file should follow this structure:

```yaml
name: "Project Name"
description: "Project description"
type: "software" # Optional
schema_version: "0.1" # Optional
maturity_log:
  - phase: "incubating"
    date: "2024-01-15T00:00:00Z"
    issue: "https://github.com/cncf/toc/issues/123"
repositories:
  - "https://github.com/project/main-repo"
social:
  twitter: "https://twitter.com/project"
  slack: "https://project.slack.com"
  youtube: "https://youtube.com/@project"
artwork: "https://github.com/project/artwork"
website: "https://project.io"
mailing_lists:
  - "project-dev@lists.cncf.io"
audits:
  - date: "2023-12-01T00:00:00Z"
    type: "security"
    url: "https://github.com/project/audits/security-2023.pdf"

# New optional sections
security:
  policy:
    path: "SECURITY.md"
  threat_model:
    path: "docs/THREAT_MODEL.md"
  contact: "security@project.io"

governance:
  contributing:
    path: "CONTRIBUTING.md"
  codeowners:
    path: ".github/CODEOWNERS"
  governance_doc:
    path: "GOVERNANCE.md"

legal:
  license:
    path: "LICENSE"

documentation:
  readme:
    path: "README.md"
  support:
    path: "SUPPORT.md"
  architecture:
    path: "docs/ARCHITECTURE.md"
  api:
    path: "docs/API.md"
```

## Maintainer Verification Stub

When `-verify-maintainers` is enabled, the validator logs stubbed calls to an external identity provider. Provide credentials and endpoint via environment variables:

- `MAINTAINER_API_ENDPOINT`: URL for the external verification service
- `MAINTAINER_API_TOKEN`: Token or credential (unused in stub but reserved)
- `MAINTAINER_API_STUB`: Set to `fail` to simulate verification failure during testing

If `MAINTAINER_API_ENDPOINT` is unset, verification is skipped with an informational log message. No network requests are sent while the stub is in place.

## Output Formats

### Text

```
Project Validation Report
========================

CHANGED: Service (https://example.com/project.yaml)
  Previous Hash: abc123
  Current Hash:  def456

INVALID: Local Project (file:///path/to/project.yaml)
  - repositories is required and cannot be empty

Summary: 2 projects validated, 1 changed, 1 with errors

Maintainers Validation Report
============================

INVALID: service
  - team 'project-maintainers' is required

Summary: 1 maintainer entries validated, 1 with issues
```

### JSON

```json
{
  "projects": [ ... ],
  "maintainers": [ ... ]
}
```

### YAML

```yaml
projects:
  - url: "..."
    valid: true
maintainers:
  - project_id: service
    valid: false
    errors:
      - "team 'project-maintainers' is required"
```

## Makefile Targets

```bash
make help           # Show all available targets
make build          # Build both validator and landscape-updater binaries
make docker-build   # Build Docker image
make test           # Run tests
make test-coverage  # Run tests with coverage report
make clean          # Clean build artifacts and caches
make install        # Install Go dependencies
make run            # Run validator with default settings
make run-changes    # Show only changes and summary
make fmt            # Format Go code
make lint           # Run linter (requires golangci-lint)
make security       # Run security checks (requires gosec)
```

## GitHub Actions

`.github/workflows/project-validator.yml` schedules daily runs, executes tests, builds the CLI, validates projects and maintainers, and uploads reports. Maintainer verification is stubbed and ready for integration with a third-party service via environment variables.

## Testing

```bash
make test
```

Tests cover project schema validation, URL validation, hash calculation, maintainer normalization, roster reconciliation, and verification stubs.

## Implementation Guide for CNCF Projects

This section provides a concise guide for CNCF project maintainers to adopt the dot-project validation tools in their repositories.

### Quick Start for Projects

1. **Create a `project.yaml` file** in your repository root with your project metadata:

```yaml
name: "Your Project Name"
description: "Brief description of your project"
schema_version: "1.0.0"
type: "platform"  # or "library", "tool", etc.

maturity_log:
  - phase: "sandbox"  # or "incubating", "graduated"
    date: 2024-01-15T00:00:00Z
    issue: "https://github.com/cncf/toc/issues/XXX"

repositories:
  - "https://github.com/your-org/your-project"

website: "https://your-project.io"
artwork: "https://github.com/cncf/artwork/tree/master/projects/your-project"

social:
  twitter: "https://twitter.com/yourproject"
  slack: "https://yourproject.slack.com"

security:
  policy:
    path: "SECURITY.md"
  contact: "security@yourproject.io"

governance:
  contributing:
    path: "CONTRIBUTING.md"
  governance_doc:
    path: "GOVERNANCE.md"

legal:
  license:
    path: "LICENSE"
```

2. **Create a `MAINTAINERS.yaml` file** (optional but recommended):

```yaml
maintainers:
  - project_id: "your-project"
    teams:
      - name: "project-maintainers"
        members:
          - githubuser1
          - githubuser2
          - githubuser3
```

3. **Add GitHub Actions** to automatically validate changes:

**.github/workflows/validate-project.yml:**
```yaml
name: Validate Project Metadata

on:
  pull_request:
    paths:
      - 'project.yaml'
      - 'MAINTAINERS.yaml'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      # Validate maintainers
      - name: Validate Maintainers
        uses: cncf/automation/.github/actions/validate-maintainers@main
        with:
          maintainers_file: './MAINTAINERS.yaml'
          verify_maintainers: 'true'
```

**.github/workflows/update-landscape.yml:**
```yaml
name: Update CNCF Landscape

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
      - uses: actions/checkout@v4
      
      - name: Update Landscape
        uses: cncf/automation/.github/actions/landscape-update@main
        with:
          project_file: './project.yaml'
          token: ${{ secrets.GITHUB_TOKEN }}
```

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

### Support

For questions or issues with the validation tools:
- Open an issue in [cncf/automation](https://github.com/cncf/automation)
- Check existing examples in `utilities/dot-project/yaml/`

## Landscape Updater

The `landscape-updater` tool automates the process of updating the CNCF Landscape YAML based on changes in project metadata.

### Usage

```bash
./landscape-updater --project <path-to-project.yaml> --landscape <path-to-landscape.yml> [flags]
```

### Flags

- `--project`: Path to the project's `project.yaml` file (required).
- `--landscape`: Path to the `landscape.yml` file (required).
- `--landscape-repo`: Target repository for the PR (default: "cncf/landscape").
- `--create-pr`: If set, creates a Pull Request with the changes.
- `--dry-run`: If set, prints the diff and PR details to stdout without modifying files or creating a PR.

### Example

```bash
# Dry run to see what would change
./landscape-updater --project ./project.yaml --landscape ./landscape.yml --dry-run

# Apply changes and create a PR
./landscape-updater --project ./project.yaml --landscape ./landscape.yml --create-pr
```

### GitHub Action

You can use the `landscape-update` action in your GitHub Workflows to automatically update the landscape when your `project.yaml` changes.

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

### Validate Maintainers Action

You can use the `validate-maintainers` action to validate your `MAINTAINERS.yaml` file.

```yaml
name: Validate Maintainers
on:
  pull_request:
    paths:
      - 'MAINTAINERS.yaml'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Validate Maintainers
        uses: cncf/automation/.github/actions/validate-maintainers@main
        with:
          maintainers_file: './MAINTAINERS.yaml'
          base_maintainers_file: './base-MAINTAINERS.yaml'  # Optional: exclude these handles
          verify_maintainers: 'true'
```
