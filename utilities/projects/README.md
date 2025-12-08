# Project Validator

A Go utility that validates CNCF project metadata and maintainer rosters. It checks project YAML manifests, reconciles maintainer lists against canonical sources, and surfaces changes via cached diffs.

## Features

- Validates project YAML files against structured schema requirements
- Detects content drift using SHA256 hashes and cached history
- Validates maintainer definitions against canonical `.project` repository data
- Stubbed third-party verification hook for maintainer identity checks
- Multiple output formats: human-readable text, JSON, YAML
- Includes GitHub Actions workflow and Makefile helpers

## Installation

```bash
cd utilities/projects
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
- `-format string`: Output format (`text`, `json`, or `yaml`; default `text`)
- `-verify-maintainers`: Toggle stubbed external maintainer verification (default `false`)

### Examples

```bash
# Validate projects and maintainers with default configuration
./validator

# Validate only projects and emit JSON
./validator -maintainers "" -format json

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
    repository: ".project"     # Optional (defaults to .project)
    branch: "main"             # Optional (defaults to main)
    path: "MAINTAINERS.yaml"   # Optional (defaults to MAINTAINERS.yaml)
    canonical_url: "https://raw.githubusercontent.com/my-org/.project/main/MAINTAINERS.yaml" # Optional override
    handles:
      - alice
      - bob
      - carol
```

Each maintainer entry compares the local list of GitHub handles against the canonical roster sourced from the referenced `.project` repository (or explicit `canonical_url`). Handles are normalized (trimmed and stripped of leading `@`) before comparison.

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

INVALID: service (canonical: https://...)
  Missing handles:
    - carol

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
    missing_handles:
      - carol
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
