# .project - CNCF Project Metadata

Every CNCF project maintains a `.project` repository in their GitHub organization containing standardized metadata about the project. This enables maintainers to own their own data while CNCF automation can act on it for landscape updates, governance audits, and more.

## Quick Start

For CNCF projects adopting `.project`:

1. Copy the `example/` directory contents into your `.project` repo
2. Replace the example values in `project.yaml` and `maintainers.yaml` with your project's details
3. The included GitHub Actions workflows (`.github/workflows/`) will validate on every PR and sync changes to the CNCF Landscape

## Repository Layouts

A `.project` repository uses one of two layouts, and every tool here detects
which one it is looking at.

**Single-project** is the default and what almost every project needs:
`project.yaml` and `maintainers.yaml` at the repository root.

**Multi-project** is for the few GitHub organizations that host more than one
distinct CNCF project — `spiffe` holds both SPIFFE and SPIRE, `spinframework`
holds both Spin and SpinKube. These are separately accepted projects with their
own maturity and their own landscape entry, so they each need their own
metadata. Such a repository declares an `org.yaml` index at its root and gives
every project a directory:

```
.project/
├── org.yaml            # index: which projects live here
├── spiffe/
│   ├── project.yaml
│   └── maintainers.yaml
├── spire/
│   ├── project.yaml
│   └── maintainers.yaml
└── .github/workflows/
```

`org.yaml` is the only marker; its presence switches the repository to the
multi-project layout. When it exists, a root `project.yaml` or
`maintainers.yaml` is an error, since it would be ambiguous which project it
described. Project directories are exactly one level deep.

The same workflows and the same actions serve both layouts — there is no
multi-project variant to adopt. See [SCHEMA.md](SCHEMA.md) for the `org.yaml`
fields and the full validation rules.

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
| `slack_channels` | SlackChannel[] | One or more CNCF Slack channels; mark the main one with `primary: true` |
| `website` | string | Project website URL |
| `adopters` | PathRef | Link to ADOPTERS.md or adopters list |
| `artwork` | string | Artwork/logo URL |
| `social` | map | Platform-name to URL mapping |
| `mailing_lists` | array | Email addresses |
| `audits` | array | Security/performance audit entries |
| `package_managers` | map | Registry-name to identifier mapping |
| `security` | object | Security policy, threat model, contact email |
| `governance` | object | Contributing, codeowners, governance doc, governance DD items, maintainer lifecycle paths |
| `legal` | object | License path, identity type (DCO/CLA) |
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

See `example/project.yaml` for a complete filled-in example to copy and adapt.

## Tools

### Validator

Validates `project.yaml` and `maintainers.yaml` files.

```bash
# Build
make build

# Validate with defaults
./bin/validator

# Validate a .project repository, whichever layout it uses
./bin/validator -repo-root .

# Validate specific files
./bin/validator -config testdata/projectlist.yaml -maintainers testdata/maintainers.yaml

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
| `-repo-root` | | Path to a `.project` repository. Detects the layout and validates every project it contains, plus the repository structure itself. Mutually exclusive with `-config`/`-maintainers` |
| `-config` | `testdata/projectlist.yaml` | Path to project list configuration |
| `-maintainers` | `testdata/maintainers.yaml` | Path to maintainers file (empty to skip) |
| `-base-maintainers` | | Base maintainers file or directory, for diff validation |
| `-cache` | `.cache` | Cache directory |
| `-output` | `text` | Output format: `text`, `json`, `yaml` |
| `-verify-maintainers` | `false` | Verify handles via LFX API |

`-repo-root` is what the GitHub Actions use. It reports the detected layout,
checks the repository structure (see [SCHEMA.md](SCHEMA.md#validation)), then
validates each project found. Pass `-config=/dev/null` alongside it to validate
maintainers only, or `-maintainers=` to validate projects only.

### Landscape Updater

The `landscape-updater` tool automates the process of updating the CNCF Landscape YAML based on changes in project metadata.

```bash
# Dry run to see what would change
./bin/landscape-updater --project ./project.yaml --landscape ./landscape.yml --dry-run

# Apply changes and create a PR
./bin/landscape-updater --project ./project.yaml --landscape ./landscape.yml --create-pr

# Update every project in a .project repository, whichever layout it uses
./bin/landscape-updater --repo-root . --landscape ./landscape.yml --create-pr
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | | Path to the project's `project.yaml` file (required unless `--repo-root` is given) |
| `--repo-root` | | Path to a `.project` repository; processes every project it contains, one branch and one PR per project |
| `--landscape` | | Path to the `landscape.yml` file (required) |
| `--landscape-repo` | `cncf/landscape` | Target repository for the PR |
| `--create-pr` | `false` | Create a Pull Request with the changes |
| `--dry-run` | `false` | Print diff and PR details without executing |

A project whose `name` matches no landscape item produces a warning annotation
rather than a silent no-op, and does not stop the remaining projects from being
processed.

### Bootstrap

The `bootstrap` tool auto-generates a complete `.project` scaffold by fetching data from CLOMonitor, GitHub, and the CNCF landscape. It discovers maintainer handles from CODEOWNERS, OWNERS, and MAINTAINERS files.

**Generated files (8 total):**

| File | Description |
|------|-------------|
| `project.yaml` | Core project metadata |
| `maintainers.yaml` | Maintainer roster |
| `README.md` | .project directory documentation |
| `SECURITY.md` | Security reporting policy |
| `CODEOWNERS` | PR review requirements |
| `.gitignore` | Build/OS artifact exclusions |
| `.github/workflows/validate.yaml` | CI validation workflow |
| `.github/workflows/update-landscape.yml` | Landscape sync workflow |

All generated files use:
- **Full GitHub URLs** for all path references (not relative paths)
- **SHA-pinned action refs** for deterministic CI
- **`LANDSCAPE_REPO_TOKEN`** as the standardized secret name

**Multi-project organizations.** Before generating anything, bootstrap scans the
CNCF landscape for every active project in the given organization. If it finds
more than one, it runs the whole pipeline once per project and writes the
[multi-project layout](#repository-layouts) instead: an `org.yaml` index, a
directory per project, and one shared set of repository-level files. Generated
team names are always prefixed with the project slug (`spire-maintainers`, not
`maintainers`), because GitHub teams are org-scoped and two projects that both
scaffolded a team called `maintainers` would silently share it.

The landscape is the only reliable signal here — nothing in a GitHub
organization itself says "these two repositories are separately accepted CNCF
projects". Use `-layout` to override the detection: `-layout single` forces the
flat layout, and `-layout multi` forces the directory layout for an
organization whose split the CNCF has not recorded in the landscape yet, leaving
the remaining projects to be added to `org.yaml` by hand.

```bash
# Dry run: preview generated YAML on stdout
./bin/bootstrap -name "My Project" -github-org my-org -dry-run

# Generate scaffold files in current directory
./bin/bootstrap -name "My Project" -github-org my-org -github-repo my-repo

# Generate into a specific directory
./bin/bootstrap -name "Envoy" -github-org envoyproxy -github-repo envoy -output-dir ./envoy/.project

# A multi-project org: detected from the landscape, no extra flags needed
./bin/bootstrap -github-org spiffe -output-dir ./spiffe/.project

# Skip external API lookups (GitHub-only)
./bin/bootstrap -github-org my-org -skip-clomonitor

# Use a GitHub token for higher rate limits
GITHUB_TOKEN=ghp_xxx ./bin/bootstrap -name "My Project" -github-org my-org

# Or put the token in a .env file (auto-loaded; never overrides real env vars)
echo 'GITHUB_TOKEN=ghp_xxx' > .env
./bin/bootstrap -name "My Project" -github-org my-org
```

> **Tip:** A token is recommended — unauthenticated GitHub API requests are
> rate-limited (HTTP 403 once the limit is exceeded). The token is read from
> (in priority order): `-github-token` flag → `GITHUB_TOKEN` env → the env file
> specified by `-env-file` (default: `.env`).

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-name` | | Project display name to search for |
| `-github-org` | | GitHub organization |
| `-github-repo` | | Primary repository name (defaults to org name) |
| `-github-token` | | GitHub token (or set `GITHUB_TOKEN` env, or a `.env` file) |
| `-env-file` | `.env` | Path to a `.env` file to load (real env vars take precedence) |
| `-output-dir` | `.` | Directory to write scaffold output |
| `-layout` | `auto` | Repository layout: `auto` (detect from the landscape), `single`, or `multi` |
| `-skip-landscape` | `false` | Skip CNCF landscape YAML lookup |
| `-skip-clomonitor` | `false` | Skip CLOMonitor API lookup |
| `-skip-github` | `false` | Skip GitHub API lookup |
| `-maintainers-csv` | | Optional path to a local `project-maintainers.csv` (default: fetch fresh from `cncf/foundation`) |
| `-dry-run` | `false` | Print generated YAML without writing files |

#### Data Sources and Priority

The bootstrap tool fetches data from multiple sources and merges them with this priority order:

1. **CNCF Landscape** (highest priority) - fetches `landscape.yml` from `cncf/landscape` repo for project name, description, website, repo URL, logo, twitter, maturity, category/subcategory
2. **CLOMonitor** - project metadata, scores, repository list
3. **GitHub API** (fallback) - repo description, org info, community health profile

Maintainer discovery uses the CNCF foundation maintainers CSV
([`cncf/foundation/project-maintainers.csv`](https://github.com/cncf/foundation/blob/main/project-maintainers.csv))
as the source of truth. The project being bootstrapped is matched against the
CSV's project labels case-insensitively, trying several name variants (landscape
name, project name, GitHub org, repo, and CLOMonitor name) and gathering any
per-project sub-groups (e.g. `Kubernetes steering` + `Kubernetes maintainers`).
Pass `-maintainers-csv` to read a local copy instead of fetching it. If no match
is found, the maintainer roster is left empty with a TODO.

### Provisioning

The `provision.sh` script automates the end-to-end process of creating a `.project` repo for a CNCF project: repo creation, bootstrap, push, secrets, and branch protection.

```bash
# Single project (dry run)
./scripts/provision.sh --org project-copacetic --name Copacetic --dry-run

# Single project (live)
LANDSCAPE_REPO_TOKEN=ghp_xxx ./scripts/provision.sh --org project-copacetic --name Copacetic

# Batch mode
LANDSCAPE_REPO_TOKEN=ghp_xxx ./scripts/provision.sh --batch scripts/example-batch.txt

# Via Makefile
make provision ORG=project-copacetic NAME=Copacetic DRY_RUN=1
```

#### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `LANDSCAPE_REPO_TOKEN` | Token with write access to `cncf/landscape` for PR creation |
| `LFX_AUTH_TOKEN` | (Optional) Token for LFX maintainer handle verification |

#### Options

| Flag | Description |
|------|-------------|
| `--org <org>` | GitHub organization |
| `--name <name>` | Project display name |
| `--repo <repo>` | Primary repo name (defaults to org) |
| `--batch <file>` | Pipe-delimited file: `org\|name\|repo` |
| `--dry-run` | Print actions without executing |
| `--skip-secrets` | Skip setting repo secrets |
| `--skip-protection` | Skip branch protection setup |
| `--bootstrap-bin <path>` | Path to bootstrap binary |

#### Batch File Format

```
# Comments start with #
# Format: org|name|repo
project-copacetic|Copacetic|copacetic
grpc|gRPC|grpc
```

#### Required Secrets (per .project repo)

| Secret | Purpose |
|--------|---------|
| `LANDSCAPE_REPO_TOKEN` | Token for `update-landscape.yml` to open PRs against `cncf/landscape` |
| `LFX_AUTH_TOKEN` | Token for `validate.yaml` to verify maintainer handles via LFX |

### Audit Checker

Verifies all URLs referenced in a project are accessible.

```bash
./bin/audit-checker -project project.yaml

# Check every project in a .project repository (the default when -project is omitted)
./bin/audit-checker -repo-root .
```

Both checkers default `-repo-root` to `.` when `-project` is omitted, so a bare
invocation inside a `.project` repository does the right thing in either layout.
Each project is reported separately and the command fails if any project fails,
so one project's broken link never hides another's result.

## GitHub Actions

All action references should be **SHA-pinned** for reproducibility.

Each action's file inputs are optional. Leave them unset and the action
discovers the repository layout for itself, validating a root `project.yaml` or
every project directory as appropriate. Setting them pins the action to one
file, which in a multi-project repository would silently skip the other
projects — so the actions emit a warning and fall back to discovery if a file
input is set in a repository that has an `org.yaml`.

### Using the Validate Project Action

```yaml
- uses: cncf/automation/.github/actions/validate-project@85e0bcd298817a6e26e286d6b22615f8c81b4e4b
```

### Using the Validate Maintainers Action

```yaml
- uses: cncf/automation/.github/actions/validate-maintainers@85e0bcd298817a6e26e286d6b22615f8c81b4e4b
  with:
    # Disabled until the LFX LLT issue is resolved. Validation is done manually for now.
    verify_maintainers: 'false'
  env:
    LFX_AUTH_TOKEN: ${{ secrets.LFX_AUTH_TOKEN }}
```

### Landscape Update Action

```yaml
name: Update Landscape
on:
  push:
    branches: [main]
    paths:
      - 'org.yaml'
      - 'project.yaml'
      - '*/project.yaml'
  workflow_dispatch:

permissions:
  contents: read

jobs:
  update:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - name: Checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0

      - name: Update Landscape
        uses: cncf/automation/.github/actions/landscape-update@85e0bcd298817a6e26e286d6b22615f8c81b4e4b
        with:
          token: ${{ secrets.LANDSCAPE_REPO_TOKEN }}
```

The `*/project.yaml` filter covers the multi-project layout; it is harmless in a
single-project repository and costs nothing to keep. In a multi-project
repository the action opens one pull request per changed project.

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
make provision      # Provision a .project repo (prints usage)
make fmt            # Format code
make lint           # Run linter (requires golangci-lint)
make security       # Run security checks (requires gosec)
make clean          # Clean build artifacts
```

### Docker

```bash
docker build -t dot-project-validator .

# Run validator
docker run --rm -v $(pwd)/testdata:/app/testdata dot-project-validator -config /app/testdata/projectlist.yaml

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
      - name: "your-project-maintainers"
        members:
          - githubuser1
          - githubuser2
```

Each maintainer entry must contain at least one team with `managed: true` (the default when `managed` is omitted) that has at least one member. Team names are free-form (e.g. `maintainers`, `committers`, `reviewers`, `emeritus`); set `managed: false` on teams that should be tracked but excluded from CNCF resource provisioning (handle verification, mailing lists, service desk, Copilot seats). Handles are normalized (trimmed and stripped of leading `@`) before verification.

Prefixing the team name with the project slug is what the bootstrap tool
generates. GitHub teams are org-scoped, so in an organization that hosts (or one
day may host) more than one CNCF project, an unprefixed `maintainers` team would
collide.

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

In a [multi-project repository](#repository-layouts), `project.yaml` and
`maintainers.yaml` live in each project's directory rather than at the root, and
an `org.yaml` index at the root is required.

## Schema Versioning

The `schema_version` field is required and validated. Currently supported: `1.0.0`.

`org.yaml` carries its own independent `schema_version`, also currently `1.0.0`.
The multi-project layout introduced no change to the `project.yaml` schema, so
existing files did not need a version bump.

New schema versions will be added as the format evolves. The validator supports multiple versions simultaneously to allow gradual migration.

## Support

For questions or issues with the validation tools:
- Open an issue in [cncf/automation](https://github.com/cncf/automation)
- Check existing examples in `utilities/dot-project/example/`
