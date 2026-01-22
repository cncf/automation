# Project Validator

A Go utility that validates CNCF project metadata and maintainer rosters. It checks project YAML manifests, reconciles maintainer lists against canonical sources, and surfaces changes via cached diffs.

## Features

- Validates project YAML files against structured schema requirements
- Detects content drift using SHA256 hashes and cached history
- Validates maintainer definitions against canonical `.project` repository data
- **Extension mechanism for third-party tools** (schema version 1.1.0+)
- Stubbed third-party verification hook for maintainer identity checks
- Multiple output formats: human-readable text, JSON, YAML
- Includes GitHub Actions workflow and Makefile helpers

## Installation

```bash
cd utilities/dot-project/
go build ./cmd/validator
```

## Usage

```bash
./validator [flags]
```

### Flags

- `-config string`: Path to project list configuration file (default `yaml/projectlist.yaml`)
- `-maintainers string`: Path to maintainer roster file (default `yaml/maintainers.yaml`, set empty to skip)
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
    org: "my-org"              # Optional if canonical_url provided
    teams:
      - name: "project-maintainers"
        members:
          - alice
          - bob
      - name: "other-team"
        members:
          - carol

### Project YAML Format
```

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
  twitter: "@project"
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
  policy: { path: "SECURITY.md" }
  threat_model: { path: "docs/THREAT_MODEL.md" }

governance:
  contributing: { path: "CONTRIBUTING.md" }
  codeowners: { path: ".github/CODEOWNERS" }
  governance_doc: { path: "GOVERNANCE.md" }

legal:
  license: { path: "LICENSE" }

documentation:
  readme: { path: "README.md" }
  support: { path: "SUPPORT.md" }
  architecture: { path: "docs/ARCHITECTURE.md" }
  api: { path: "docs/API.md" }
```

### Extensions (schema_version >= 1.1.0)

Extensions allow third-party tools to store their configuration within the `.project` file without conflicting with core fields. Each extension is namespaced by tool name.

```yaml
schema_version: "1.1.0"  # Required for extensions
# ... other fields ...

extensions:
  # Tool-specific configuration
  scorecard:
    metadata:
      author: "OSSF"
      homepage: "https://securityscorecards.dev"
      repository: "https://github.com/ossf/scorecard"
      license: "Apache-2.0"
      version: "4.0.0"
    config:
      checks:
        - Binary-Artifacts
        - Branch-Protection
      threshold: 7.0

  clomonitor:
    metadata:
      author: "CNCF"
      homepage: "https://clomonitor.io"
    config:
      category: "platform"
```

**Extension naming rules:**
- Use alphanumeric characters, hyphens, underscores, and dots
- Reserved names (core field names) cannot be used
- Namespacing with organization prefix is recommended (e.g., `my-org.tool-name`)

Each maintainer entry must contain a `project-maintainers` team which cannot be empty. Handles are normalized (trimmed and stripped of leading `@`) before verification.

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
make help
make build
make test
make run
make run-json
```

## GitHub Actions

`.github/workflows/project-validator.yml` schedules daily runs, executes tests, builds the CLI, validates projects and maintainers, and uploads reports. Maintainer verification is stubbed and ready for integration with a third-party service via environment variables.

## Testing

```bash
make test
```

Tests cover project schema validation, URL validation, hash calculation, maintainer normalization, roster reconciliation, and verification stubs.
