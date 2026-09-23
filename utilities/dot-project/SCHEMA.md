# .project Schema Specification

**Version:** 1.0.0
**Status:** Active

This document defines the schema for CNCF `.project` repository metadata files.

## Repository layouts

A `.project` repository uses one of two layouts.

**Single-project (default).** `project.yaml` and `maintainers.yaml` sit at the
repository root. This is what almost every CNCF project needs, and it remains
the layout produced when nothing indicates otherwise.

```
.project/
├── project.yaml
├── maintainers.yaml
└── .github/workflows/
```

**Multi-project.** A few GitHub organizations host more than one distinct CNCF
project — `spiffe` holds both SPIFFE and SPIRE, and `spinframework` holds both
Spin and SpinKube. These are separately accepted projects with their own
maturity, their own landscape entry, and frequently their own maintainers, so
they cannot share a single `project.yaml`. Such a repository declares an
`org.yaml` index at its root and gives every project its own directory:

```
.project/
├── org.yaml
├── spiffe/
│   ├── project.yaml
│   └── maintainers.yaml
├── spire/
│   ├── project.yaml
│   └── maintainers.yaml
└── .github/workflows/
```

The presence of `org.yaml` is the only thing that selects the multi-project
layout; every tool detects it automatically. The two layouts are mutually
exclusive: when `org.yaml` exists, a root `project.yaml` or `maintainers.yaml`
is an error, because it would be ambiguous which project it described.

Project directories are exactly one level deep. That is what lets workflows use
an unambiguous `*/project.yaml` path filter rather than a recursive glob that
would also match unrelated files.

## org.yaml

`org.yaml` is deliberately *only* an index. It names the projects in the
repository and where they live, and carries no project metadata of its own.

Shared values are not hoisted into it even when several projects genuinely share
them. A value that looks shared today — a security contact, an adopters list —
is routinely project-specific tomorrow (SPIRE maintainers handle their own
security reports while the SPIFFE Steering Committee handles the rest), and
hoisting it would mean no single file fully describes a project. Every
`project.yaml` therefore stands alone, exactly as it does in the single-project
layout.

### Top-Level Fields

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `schema_version` | string | Yes | `org.yaml` schema version | Must be a supported version (currently `"1.0.0"`) |
| `org` | string | Yes | The GitHub organization that owns this repository | Non-empty |
| `projects` | OrgProjectEntry[] | Yes | The CNCF projects in this organization | At least two entries; `id` values must be unique |

`org.yaml` has its own version line, independent of `project.yaml`. The
multi-project layout introduced no change to the `project.yaml` schema, so
`project.yaml` stays at `1.0.0`.

### OrgProjectEntry

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `id` | string | Yes | Project identifier, and by default the directory name | Lowercase alphanumeric and hyphens only, no leading/trailing hyphens; unique within the file |
| `path` | string | No | Directory holding the project, when it differs from `id` | Relative, exactly one level deep, no `.`/`..` segments, not dot-prefixed |

Use `path` only when a directory has to be named something other than the
project id. Omitting it is the norm.

### Example

```yaml
schema_version: "1.0.0"
org: "spiffe"

projects:
  # SPIFFE (graduated)
  - id: "spiffe"
  # SPIRE (graduated)
  - id: "spire"
```

### Validation

When `org.yaml` is present:

- A root `project.yaml` or `maintainers.yaml` is an error.
- Every declared project must have a directory containing both `project.yaml`
  and `maintainers.yaml`.
- A directory that looks like a project directory but is not declared in
  `org.yaml` is an error — silently ignoring it would let a project drop out of
  every automation without anyone noticing.
- Each `project.yaml`'s `slug` and each `maintainers.yaml`'s `project_id` must
  match the directory's project id, so that the routing key is the same
  everywhere.
- A `slug` may not be reused across projects in the repository.

A team name may appear in more than one project. GitHub teams are org-scoped, so
one team that appears twice is one team whose membership is the union of both
definitions. This is legitimate — a shared steering committee, or an
"authentication" team staffed partly from each project — so it is reported as a
warning, never an error, when the member lists differ.

## project.yaml

### Top-Level Fields

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `schema_version` | string | Yes | Schema version | Must be a supported version (currently `"1.0.0"`) |
| `slug` | string | Yes | Unique project identifier | Lowercase alphanumeric and hyphens only, no leading/trailing hyphens |
| `name` | string | Yes | Project display name | Non-empty |
| `description` | string | Yes | One-line project description | Non-empty |
| `type` | string | No | Project type | Free text (e.g., `"project"`, `"platform"`, `"specification"`) |
| `package_managers` | map[string](string\|string[]) | No | Registry identifiers. Each value is a single string or a list of strings (e.g., multiple Docker images) | All values must be non-empty strings |
| `project_lead` | string \| string[] | No | One or more primary contact GitHub handles or teams. Accepts a plain string (single lead, backward-compatible) or a YAML list (multiple leads) | Non-empty if present; `@` prefix is stripped; each entry can be a GitHub handle (e.g., `jdoe`) or a GitHub team (e.g., `org/team-name`) |
| `slack_channels` | SlackChannel[] | No | One or more CNCF Slack channels | Each `name` must start with `#`; at most one entry may set `primary: true` |
| `maturity_log` | MaturityEntry[] | Yes | Maturity phase history | At least one entry; chronological order |
| `repositories` | (string \| RepositoryEntry)[] | Yes | Repository URLs with optional metadata. Each entry can be a plain URL string (backward-compatible) or an object with `url` and optional `tags` | At least one entry; each must have a valid HTTP(S) URL |
| `website` | string | No | Project website | Valid HTTP(S) URL if present |
| `adopters` | PathRef | No | Link to ADOPTERS.md or adopters list | Path must be non-empty if present |
| `artwork` | string | No | Artwork/logo URL | Valid HTTP(S) URL if present |
| `social` | map[string]string | No | Social platform URLs | All values must be valid HTTP(S) URLs |
| `mailing_lists` | string[] | No | Mailing list addresses | |
| `audits` | Audit[] | No | Security/performance audits | |
| `security` | SecurityConfig | No | Security policy references | |
| `governance` | GovernanceConfig | No | Governance document references | |
| `legal` | LegalConfig | No | Legal document references | |
| `documentation` | DocumentationConfig | No | Documentation references | |
| `landscape` | LandscapeConfig | No | CNCF Landscape location | Both fields required if section present |

### MaturityEntry

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `phase` | string | Yes | Maturity phase | One of: `sandbox`, `incubating`, `graduated`, `archived` |
| `date` | datetime | Yes | Date of phase transition | ISO 8601 format |
| `issue` | string | Yes | TOC issue URL | Non-empty |

### RepositoryEntry

Each entry in the `repositories` list can be either a plain URL string or an object:

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `url` | string | Yes | Repository URL | Valid HTTP(S) URL |
| `tags` | string[] | No | Labels/categories for grouping (e.g., `core`, `sig-apps`, `tooling`) | Each tag must be non-empty |
| `primary` | boolean | No | Whether this is the main project repository | At most one entry per project may be `true` |

Example:

```yaml
repositories:
  - url: "https://github.com/helm/helm"
    primary: true
  - url: "https://github.com/helm/chart-testing"
    tags: [tooling]
  - url: "https://github.com/helm/helm-www"
    tags: [docs]
```

### SlackChannel

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `name` | string | Yes | Channel name (e.g., `#kubernetes-dev`) | Must start with `#` |
| `workspace` | string | No | Slack workspace identifier (e.g., `cncf`) | |
| `link` | string | No | Invite or channel URL | Valid HTTP(S) URL if present |
| `primary` | boolean | No | Whether this is the primary channel most end-users should join | At most one entry per project may be `true` |

Example:

```yaml
slack_channels:
  - name: "#kubernetes-users"
    workspace: "cncf"
    link: "https://cloud-native.slack.com/messages/kubernetes-users"
    primary: true
  - name: "#kubernetes-dev"
    workspace: "cncf"
    link: "https://cloud-native.slack.com/messages/kubernetes-dev"
```

### Audit

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `date` | datetime | Yes | Audit date | ISO 8601 format |
| `type` | string | Yes | Audit type | Non-empty (e.g., `"security"`, `"performance"`) |
| `url` | string | Yes | Report URL | Valid HTTP(S) URL |

### SecurityConfig

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `policy` | PathRef | No | Security policy file | Path must be non-empty if present |
| `threat_model` | PathRef | No | Threat model document | Path must be non-empty if present |
| `contact` | SecurityContact | No | Security contact information | At least one of `email` or `advisory_url` required when present |

### SecurityContact

At least one of `email` or `advisory_url` must be provided when the `contact` section is present.

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `email` | string | No* | Security contact email | Valid email address (RFC 5322) if present |
| `advisory_url` | string | No* | GitHub Security Advisory form URL | Must match `https://github.com/{org}/{repo}/security/advisories/new` |

\* At least one of `email` or `advisory_url` is required when the section is present.

### GovernanceConfig

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `contributing` | PathRef | No | Contributing guide | Path must be non-empty if present |
| `codeowners` | PathRef | No | CODEOWNERS file | Path must be non-empty if present |
| `governance_doc` | PathRef | No | Governance document | Path must be non-empty if present |
| `gitvote_config` | PathRef | No | GitVote configuration | Path must be non-empty if present |
| `vendor_neutrality_statement` | PathRef | No | Vendor neutrality statement | Path must be non-empty if present |
| `decision_making_process` | PathRef | No | Decision-making process documentation | Path must be non-empty if present |
| `roles_and_teams` | PathRef | No | Roles and teams documentation | Path must be non-empty if present |
| `code_of_conduct` | PathRef | No | Code of conduct | Path must be non-empty if present |
| `sub_project_list` | PathRef | No | Subproject listing | Path must be non-empty if present |
| `sub_project_docs` | PathRef | No | Subproject documentation | Path must be non-empty if present |
| `contributor_ladder` | PathRef | No | Contributor ladder documentation | Path must be non-empty if present |
| `change_process` | PathRef | No | Change process documentation | Path must be non-empty if present |
| `comms_channels` | PathRef | No | Communication channels listing | Path must be non-empty if present |
| `community_calendar` | PathRef | No | Community calendar | Path must be non-empty if present |
| `contributor_guide` | PathRef | No | Contributor guide | Path must be non-empty if present |
| `maintainer_lifecycle` | MaintainerLifecycle | No | Maintainer lifecycle documentation | |

### MaintainerLifecycle

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `onboarding_doc` | PathRef | No | Maintainer onboarding documentation | Path must be non-empty if present |
| `progression_ladder` | PathRef | No | Maintainer advancement path (committer → maintainer → lead) | Path must be non-empty if present |
| `mentoring_program` | string[] | No | URLs to mentoring/support program documentation | All must be valid HTTP(S) URLs if present |
| `offboarding_policy` | PathRef | No | Emeritus/offboarding policy documentation | Path must be non-empty if present |

### LegalConfig

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `license` | PathRef | No | License file | Path must be non-empty if present |
| `identity_type` | IdentityType | No | Contributor identity agreement | |

### IdentityType

DCO can be used alone, or DCO + CLA together. By default, CLA requires DCO (the baseline requirement). Some projects have an exception to use CLA without DCO; set `cla_only: true` for those.

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `has_dco` | boolean | No | Whether the project uses DCO | Defaults to false |
| `has_cla` | boolean | No | Whether the project uses CLA | Requires `has_dco` to be true unless `cla_only` is true |
| `cla_only` | boolean | No | Exception: allows CLA without DCO | Requires `has_cla` to be true |
| `dco_url` | PathRef | No | Link to DCO document | Path must be non-empty if present |
| `cla_url` | PathRef | No | Link to CLA document | Path must be non-empty if present |

### DocumentationConfig

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `readme` | PathRef | No | README file | Path must be non-empty if present |
| `support` | PathRef | No | Support document | Path must be non-empty if present |
| `architecture` | PathRef | No | Architecture document | Path must be non-empty if present |
| `api` | PathRef | No | API documentation | Path must be non-empty if present |

### LandscapeConfig

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `category` | string | Yes* | Landscape category | Required when section is present |
| `subcategory` | string | Yes* | Landscape subcategory | Required when section is present |

### PathRef

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | File path or URL |

PathRef values should be **full GitHub URLs** (e.g., `https://github.com/org/repo/blob/main/SECURITY.md`). Relative file paths (e.g., `SECURITY.md`) are accepted for backward compatibility but full URLs are the standard for new repos. The bootstrap tool always generates full URLs.

## maintainers.yaml

### Top-Level

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `maintainers` | MaintainerEntry[] | Yes | At least one entry |

### MaintainerEntry

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `project_id` | string | Yes | Project slug | Must match `slug` in project.yaml |
| `org` | string | No | GitHub organization | |
| `teams` | Team[] | Yes | Team definitions | At least one managed team required |

### Team

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `name` | string | Yes | Team name | Any descriptive name (e.g., `maintainers`, `committers`, `reviewers`, `emeritus`). In a multi-project repository the same name may appear in several projects; GitHub teams are org-scoped, so the team's membership is the union of those definitions |
| `members` | string[] | Yes | GitHub handles | At least one managed team must have members; normalized (trimmed, `@` stripped) |
| `managed` | bool | No | Whether team is provisioned to CNCF resources | Defaults to `true` if omitted. Teams with `managed: false` are excluded from handle verification, mailing lists, service desk, and Copilot seat provisioning |

## Validation Rules

1. **Unknown fields are rejected** -- any field not in this schema causes a validation error
2. **URL validation** -- all URLs must have `http://` or `https://` scheme and a valid domain
3. **Email validation** -- uses RFC 5322 parsing
4. **Slug format** -- lowercase letters, digits, and hyphens; no leading/trailing hyphens
5. **Maturity ordering** -- maturity_log entries must be in chronological order
6. **Handle normalization** -- leading `@` and whitespace are stripped; duplicates are detected case-insensitively
7. **Required managed team** -- every maintainer entry must include at least one team with `managed: true` (or `managed` omitted) that has at least one member
8. **`name` is the landscape key** -- the landscape updater matches a landscape item by `name` *and* repository URL. If `name` does not match the item's name in `cncf/landscape`, the updater finds nothing to update and emits a warning rather than silently succeeding. This matters most in a multi-project repository, where two sibling projects have similar names and it is easy to give one of them the other's landscape name.

### Multi-project repositories

These rules apply additionally when `org.yaml` is present; see
[Repository layouts](#repository-layouts) for the full list.

9. **No root metadata** -- a root `project.yaml` or `maintainers.yaml` is an error
10. **Declared and present must agree** -- every declared project needs a directory with both files, and every undeclared project-shaped directory is an error
11. **Identity must match the directory** -- `slug` and `project_id` must both equal the project's `id`
12. **Slugs are unique** -- no two projects in the repository may share a slug
