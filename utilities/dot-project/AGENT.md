# AGENT.md - dot-project Validator

## Project Overview

This is a Go-based utility for validating CNCF project metadata and maintainer rosters. It validates project YAML manifests against structured schema requirements, reconciles maintainer lists against canonical sources, surfaces changes via cached diffs, and converts project metadata to CNCF landscape format.

### Repository layouts

Every tool here handles two `.project` repository layouts, detected at runtime:

- **single** — `project.yaml` and `maintainers.yaml` at the repository root.
  This is the overwhelming majority and the default.
- **multi** — an `org.yaml` index at the root and one directory per project,
  for the few GitHub organizations that host more than one distinct CNCF
  project (for example, `spiffe` holds SPIFFE and SPIRE; `spinframework` holds Spin and
  SpinKube).

`Discover(repoRoot)` in `discovery.go` reports the layout; `ValidateRepo` in
`repo_validate.go` enforces the structural rules. Detection is based purely on
the presence of `org.yaml` — never on a flag, an org name list, or a heuristic
over directory contents. See [SCHEMA.md](SCHEMA.md#repository-layouts).

Two invariants are worth knowing before changing anything here:

1. **One set of workflows and actions serves both layouts.** There is no
   multi-project variant. Around 218 repositories are already onboarded and
   updating them is slow, so anything that would require them to change their
   workflow is not an option.
2. **`Discover` reports the layout, not correctness.** It succeeds on a
   repository that `ValidateRepo` will reject. Callers that need a valid
   repository must run both.


## Repository Structure

```
utilities/dot-project/
├── cmd/
│   ├── validator/              # Main CLI validator tool
│   ├── landscape-updater/      # Tool to convert project.yaml to landscape format
│   └── bootstrap/              # Tool to auto-generate project scaffolds from external data
├── template/                   # Template files for new .project repositories
│   ├── project.yaml
│   ├── maintainers.yaml
│   └── .github/workflows/validate.yaml
├── example/                    # Realistic filled-in example (Kubernetes-like)
│   ├── project.yaml
│   ├── maintainers.yaml
│   └── .github/workflows/validate.yaml
├── testdata/                   # Test fixtures and sample configs
├── bin/                        # Build output (gitignored)
├── .cache/                     # Validation cache directory (gitignored)
├── types.go                    # Core type definitions (Project, Maintainer, Config, etc.)
├── bootstrap_types.go          # Bootstrap intermediate types (BootstrapResult, API data structs)
├── bootstrap_parsers.go        # CODEOWNERS, OWNERS, MAINTAINERS file parsers
├── bootstrap_sources.go        # Landscape/CLOMonitor/GitHub API clients, fuzzy matching, data merge
├── bootstrap_scaffold.go       # Scaffold generator (project.yaml, maintainers.yaml templates)
├── bootstrap_multiproject.go   # Landscape scan for multi-project orgs, org.yaml generation
├── org.go                      # org.yaml types, loading, and validation
├── discovery.go                # Repository layout detection (single vs. multi-project)
├── repo_validate.go            # Whole-repository structural validation
├── validator.go                # Project validation logic
├── maintainers.go              # Maintainer validation logic with LFX integration
├── landscape.go                # Landscape entry conversion and comparison
├── validator_test.go           # Core validation tests
├── bootstrap_parsers_test.go   # CODEOWNERS/OWNERS/MAINTAINERS parser tests
├── bootstrap_sources_test.go   # Landscape/CLOMonitor/GitHub client, fuzzy match, merge tests
├── bootstrap_scaffold_test.go  # Scaffold generation and WriteScaffold tests
├── bootstrap_multiproject_test.go # Org scan, slugify, org.yaml, multi-scaffold tests
├── discovery_test.go           # Layout detection and repository validation tests
├── security_test.go            # Security contact email validation tests
├── social_test.go              # Social links URL validation tests
├── landscape_test.go           # Landscape conversion and diff tests
├── integration_test.go         # YAML fixture integration tests
├── test_helpers_test.go        # Shared test helpers (validBaseProject, etc.)
├── Dockerfile                  # Multi-stage Docker build
├── Makefile                    # Build and development tasks
├── SCHEMA.md                   # Formal schema specification
└── README.md                   # User documentation
```

### Related GitHub Actions (parent repo `.github/`)

```
.github/
├── actions/
│   ├── validate-maintainers/   # Reusable action for maintainer validation
│   └── validate-project/       # Reusable action for project validation
└── workflows/
    ├── project-validator.yml                  # Main CI workflow for this tool
    ├── validate-maintainers.yaml              # Validates maintainers on PR
```

## Build and Development

### Prerequisites

- Go 1.24+
- Docker (optional, for containerized builds)

### Build Commands

```bash
# Build validator binary (outputs to bin/)
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Build Docker image
docker build -t dot-project-validator .

# Clean build artifacts
make clean
```

Note: The Makefile `build` target builds the `validator`, `landscape-updater`, and `bootstrap` binaries.

### Running the Validator

```bash
# Default run (validates projects and maintainers)
make run

# Or directly:
./bin/validator --config testdata/projectlist.yaml --maintainers testdata/maintainers.yaml

# Skip maintainer validation
./bin/validator --config testdata/projectlist.yaml --maintainers ""

# Validate a whole .project repository, in either layout
./bin/validator --repo-root .

# Repository-scoped, projects only / maintainers only
./bin/validator --repo-root . --maintainers=
./bin/validator --repo-root . --config=/dev/null

# With external verification enabled
./bin/validator --verify-maintainers

# Diff validation (only verify new/changed maintainers)
./bin/validator --maintainers maintainers.yaml --base-maintainers previous-maintainers.yaml

# Output formats: text (default), json, yaml
./bin/validator --config testdata/projectlist.yaml --output json
```

### Running the Landscape Updater

Converts a `project.yaml` to CNCF landscape entry format. Validates the project first, then outputs the landscape entry.

```bash
./bin/landscape-updater --project path/to/project.yaml

# With landscape file for comparison (comparison not yet fully implemented)
./bin/landscape-updater --project project.yaml --landscape landscape.yml

# Output formats: text (default), json, yaml
./bin/landscape-updater --project project.yaml --output yaml

# Dry run is on by default
./bin/landscape-updater --project project.yaml --dry-run=false

# Process every project in a .project repository (one branch and one PR each)
./bin/landscape-updater --repo-root . --landscape landscape.yml --create-pr
```

### Running the Bootstrap Tool

Auto-generates `project.yaml` and `maintainers.yaml` scaffolds by fetching data from CLOMonitor, GitHub API, and the CNCF landscape. Discovers maintainer handles from CODEOWNERS, OWNERS, and MAINTAINERS files.

```bash
# Dry run: preview generated YAML
./bin/bootstrap -name "My Project" -github-org my-org -dry-run

# Generate scaffold in current directory
./bin/bootstrap -name "My Project" -github-org my-org -github-repo my-repo

# Generate into a specific directory
./bin/bootstrap -name "Envoy" -github-org envoyproxy -github-repo envoy -output-dir /tmp/envoy

# A multi-project org. Detected automatically by scanning the landscape for
# every active CNCF project in the org; writes org.yaml plus one directory each.
./bin/bootstrap -github-org spiffe -output-dir /tmp/spiffe

# Force a layout when the landscape does not reflect reality yet
./bin/bootstrap -github-org fluent -name Fluentd -layout multi

# Skip landscape fetch (CLOMonitor + GitHub only)
./bin/bootstrap -name "My Project" -github-org my-org -skip-landscape

# Skip CLOMonitor (landscape + GitHub only)
./bin/bootstrap -github-org my-org -skip-clomonitor

# With GitHub token for higher rate limits
GITHUB_TOKEN=ghp_xxx ./bin/bootstrap -name "My Project" -github-org my-org

# Or store the token in a .env file (auto-loaded; real env vars win)
echo 'GITHUB_TOKEN=ghp_xxx' > .env
./bin/bootstrap -name "My Project" -github-org my-org
```

**bootstrap** (`cmd/bootstrap/main.go`):
- `-name` - Project display name to search for
- `-github-org` - GitHub organization
- `-github-repo` - Primary repository name (defaults to org name)
- `-github-token` - GitHub token. Resolution order: flag → `GITHUB_TOKEN` → the env file specified by `-env-file` (default: `.env`)
- `-env-file` - Path to a `.env` file to load (default: `.env`; real env vars take precedence)
- `-output-dir` - Directory for scaffold output (default: `.`)
- `-layout` - `auto` (default; detect from the landscape), `single`, or `multi`
- `-skip-landscape` - Skip CNCF landscape YAML lookup (default: false)
- `-skip-clomonitor` - Skip CLOMonitor API lookup (default: false)
- `-skip-github` - Skip GitHub API lookup (default: false)
- `-dry-run` - Print generated YAML without writing files (default: false)

## Testing

### Test Commands

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestValidator

# Run with coverage
go test -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Test Files

- `validator_test.go` - Core validation tests (project structure, maturity log, repositories, hashing)
- `security_test.go` - Security contact email validation tests
- `social_test.go` - Social links URL validation tests
- `landscape_test.go` - Landscape entry conversion and diff comparison tests
- `integration_test.go` - YAML fixture integration tests (loads files from `testdata/` and `example/`)
- `test_helpers_test.go` - Shared test helpers (`validBaseProject()` factory function)

### Test Patterns

Tests use table-driven patterns with `validBaseProject()` as a baseline. Tests modify only the fields relevant to their case:

```go
func TestSomething(t *testing.T) {
    project := validBaseProject()
    project.Name = "" // override to trigger validation error

    errors := ValidateProjectStruct(project)
    expectedErrors := []string{
        "expected error message",
    }

    for _, expectedError := range expectedErrors {
        found := false
        for _, err := range errors {
            if err == expectedError {
                found = true
                break
            }
        }
        if !found {
            t.Errorf("Expected error '%s' not found in: %v", expectedError, errors)
        }
    }
}
```

## Code Style and Conventions

### Go Conventions

- Package name: `projects`
- Go version: 1.24+ (see `go.mod`)
- Use standard Go formatting (`go fmt`)
- Error messages should be lowercase and descriptive
- Use `fmt.Errorf` with `%w` for error wrapping
- Struct tags include both `json` and `yaml` for serialization
- YAML decoding uses `decoder.KnownFields(true)` to reject unknown fields

### Type Definitions

All core types are defined in `types.go`:
- `Project` - Main project metadata structure with nested config types
- `SecurityConfig`, `GovernanceConfig`, `LegalConfig`, `DocumentationConfig` - Nested project config sections
- `LandscapeConfig` - CNCF landscape category/subcategory mapping
- `PathRef` - Reusable path reference (used by security, governance, documentation configs)
- `MaturityEntry` - Phase, date, and issue URL for maturity log entries
- `Audit` - Security audit record (date, type, URL)
- `MaintainerEntry` / `MaintainersConfig` - Maintainer definitions with teams
- `Team` - GitHub team name and member handles
- `ValidationResult` / `MaintainerValidationResult` - Validation output types
- `Config`, `Cache`, `CacheEntry` - Configuration and caching types
- `ProjectValidator` - Main validator struct (wraps config, cache, HTTP client)
- `ProjectListEntry` / `ProjectListConfig` - Project list configuration

Additional types in domain-specific files:
- `LandscapeEntry`, `LandscapeDiff`, `LandscapeChange` - in `landscape.go`
- `StalenessResult` - in `staleness.go`
- `BootstrapConfig`, `BootstrapResult`, `CLOMonitorProject`, `CLOMonitorRepo`, `CLOMonitorReport`, `CLOMonitorScore` - in `bootstrap_types.go`
- `GitHubRepoData`, `GitHubOrgData`, `GitHubCommunityProfile`, `GitHubContentEntry` - in `bootstrap_types.go`
- `GitHubData`, `LandscapeData` - in `bootstrap_sources.go`

### Validation Logic

- `validator.go` contains project validation (`ValidateProjectStruct`) and the `ProjectValidator` type with `ValidateAll`, `FormatResults`, `NewValidator`
- `maintainers.go` contains maintainer validation with optional LFX integration and handle normalization
- `landscape.go` contains `ProjectToLandscapeEntry`, `CompareLandscapeEntries`, `LoadProjectFromFile`
- `org.go` contains `LoadOrgFromFile` and `ValidateOrgStruct` for the `org.yaml` index
- `discovery.go` contains `Discover`, which returns the layout and the projects in the repository. It reports the layout, not correctness — it succeeds on repositories `ValidateRepo` rejects
- `repo_validate.go` contains `ValidateRepo`, which enforces every cross-file rule (no root metadata in multi mode, declared vs. present directories, slug/`project_id` matching the directory, unique slugs) and returns errors *and* warnings. Shared team names across projects are a warning, never an error
- Handle normalization strips whitespace and leading `@` symbols
- All URLs are validated for proper format
- Email addresses use `net/mail.ParseAddress` for validation

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `REPO_ROOT` | Repository root for resolving relative `file://` paths in project list config |
| `LFX_AUTH_TOKEN` | Token for LFX API maintainer verification |
| `MAINTAINER_API_ENDPOINT` | External maintainer verification endpoint URL |
| `MAINTAINER_API_STUB` | Set to "fail" to simulate verification failures in testing |

### CLI Flags

**validator** (`cmd/validator/main.go`):
- `--repo-root` - Path to a `.project` repository; validates its structure and every project in it. Mutually exclusive with `--config`/`--maintainers`
- `--config` - Path to project list configuration file (default: `testdata/projectlist.yaml`)
- `--cache` - Directory to store cached validation results (default: `.cache`)
- `--maintainers` - Path to maintainers file, set empty to skip (default: `testdata/maintainers.yaml`)
- `--base-maintainers` - Path to a base maintainers file *or directory* for diff validation
- `--verify-maintainers` - Verify maintainer handles via external service (default: false)
- `--output` - Output format: text, json, yaml (default: `text`)

**landscape-updater** (`cmd/landscape-updater/main.go`):
- `--project` - Path to project.yaml file (required unless `--repo-root` is set)
- `--repo-root` - Path to a `.project` repository; processes every project, one branch and one PR each
- `--landscape` - Path to landscape.yml for comparison (optional)
- `--output` - Output format: text, json, yaml (default: `text`)
- `--dry-run` - Show changes without applying (default: true)

## Docker

### Build

```bash
docker build -t dot-project-validator .
```

### Run

```bash
# Run validator
docker run --rm -v $(pwd)/testdata:/app/testdata dot-project-validator --config /app/testdata/projectlist.yaml
```

The Dockerfile uses a multi-stage build:
1. `golang:1.24-alpine` builder stage (builds only the `validator` binary)
2. `alpine:3.20` runtime with `git` and `ca-certificates`

Note: The Docker image includes `validator` and `landscape-updater` binaries. The other CLI tools (bootstrap, etc.) are not built in the Dockerfile.

## Configuration Files

### Project List (`testdata/projectlist.yaml`)

```yaml
projects:
  - url: "https://raw.githubusercontent.com/org/repo/main/project.yaml"
    id: "project-id"
  - url: "file://${REPO_ROOT}/path/to/project.yaml"
    id: "local-project"
```

### Maintainers (`testdata/maintainers.yaml`)

```yaml
maintainers:
  - project_id: "project-id"
    org: "github-org"  # optional
    teams:
      - name: "maintainers"  # managed: true by default; at least one managed team is required
        members:
          - alice
          - bob
      - name: "emeritus"
        managed: false        # excluded from handle verification and CNCF resource provisioning
        members:
          - carol
```

### Project Schema (`project.yaml`)

```yaml
schema_version: "1.0.0"
slug: "project-name"
name: "Project Name"
description: "Project description"
project_lead: "github-handle"
slack_channels:
  - name: "#project-name"
    workspace: "cncf"
    link: "https://cloud-native.slack.com/messages/project-name"
    primary: true
maturity_log:
  - phase: "incubating"
    date: 2024-01-15
    issue: "https://github.com/cncf/toc/issues/123"
repositories:
  - "https://github.com/org/repo"
website: "https://project.io"
artwork: "https://project.io/artwork"
social:
  twitter: "https://twitter.com/project"
  slack: "https://slack.project.io"
security:
  policy:
    path: "SECURITY.md"
  contact: "security@project.io"
governance:
  contributing:
    path: "CONTRIBUTING.md"
  governance_doc:
    path: "GOVERNANCE.md"
documentation:
  readme:
    path: "README.md"
landscape:
  category: "App Definition and Development"
  subcategory: "Database"
audits:
  - date: 2023-12-01
    type: "security"
    url: "https://project.io/audit.pdf"
```

### Org Index (`org.yaml`)

Present only in multi-project repositories. Pure index — no project metadata:

```yaml
schema_version: "1.0.0"
org: "spiffe"

projects:
  - id: "spiffe"
  - id: "spire"
    path: "spire-project"   # optional; defaults to id
```

Shared values are deliberately *not* hoisted here. A value that looks shared
(security contact, adopters list) is routinely project-specific, and hoisting it
would mean no single file fully describes a project.

### Example Files (`example/`)

The `example/` directory contains starter files for new `.project` repositories
(a filled-in, realistic Kubernetes example — copy and replace values):
- `project.yaml` - Example project metadata
- `maintainers.yaml` - Example maintainers configuration
- `projectlist.yaml` - Example project list entry (used by validator tests)
- `.github/workflows/validate.yaml` - CI workflow to validate project files
- `.github/workflows/update-landscape.yml` - CI workflow to sync changes to the CNCF Landscape

## Common Tasks

### Adding a New Validation Rule

1. Add error check in `ValidateProjectStruct()` in `validator.go`
2. Add corresponding test case in `validator_test.go`
3. Update type definition in `types.go` if adding new fields

### Adding a New Maintainer Validation

1. Add check in `validateMaintainerEntry()` in `maintainers.go`
2. Handle normalization in `normalizeHandles()` if needed
3. Add test cases

### Modifying CLI Flags

Edit the corresponding `cmd/*/main.go` file. All CLIs use the standard `flag` package.

### Touching Anything Layout-Aware

The workflows and composite actions are shared by every onboarded `.project`
repository and by both layouts. Before changing them:

1. Keep the file inputs on the composite actions optional. Unset means
   "discover the layout"; that is what makes one workflow serve both layouts.
2. Keep workflow `paths:` filters covering `org.yaml`, `project.yaml`,
   `maintainers.yaml`, `*/project.yaml`, and `*/maintainers.yaml`. GitHub
   Actions does **not** support YAML anchors, so the list has to be repeated.
3. The workflows exist in three places that must stay in sync: `template/`,
   `example/`, and the embedded Go string literals in `bootstrap_scaffold.go`.
4. Per-project loops must isolate failures — one project's error must be
   reported and must not abort the remaining projects, and the command exits
   non-zero if any project failed.
5. The `uses:` references in `template/` and in the embedded literals currently
   point at `@main` so that a workflow and the action it calls are never out of
   step. Pin them back to a commit SHA once the change has landed on `main`, and
   bump all of them together — a workflow that omits an input the pinned action
   still declares `required: true` fails before it runs anything.

### Adding a New CLI Tool

1. Create `cmd/<tool-name>/main.go` with a `package main` and `main()` function
2. Import `"projects"` to use library functions from the package root
3. Use `flag` for argument parsing
4. Support `--output` with text/json/yaml formats for consistency

## Exit Codes

- `0` - All checks passed
- `1` - One or more checks failed (validation errors, stale data, failed URL checks)
- Non-zero for other errors (file not found, parse errors, etc.)
