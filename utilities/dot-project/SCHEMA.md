# .project Schema Specification

**Version:** 1.0.0
**Status:** Active

This document defines the schema for CNCF `.project` repository metadata files.

## project.yaml

### Top-Level Fields

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `schema_version` | string | Yes | Schema version | Must be a supported version (currently `"1.0.0"`) |
| `slug` | string | Yes | Unique project identifier | Lowercase alphanumeric and hyphens only, no leading/trailing hyphens |
| `name` | string | Yes | Project display name | Non-empty |
| `description` | string | Yes | One-line project description | Non-empty |
| `type` | string | No | Project type | Free text (e.g., `"project"`, `"platform"`, `"specification"`) |
| `package_managers` | map[string]string | No | Registry identifiers (e.g., npm, Docker Hub) | All values must be non-empty strings |
| `project_lead` | string | No | Primary contact GitHub handle | Non-empty if present; `@` prefix is stripped |
| `cncf_slack_channel` | string | No | CNCF Slack channel name | Must start with `#` if present |
| `maturity_log` | MaturityEntry[] | Yes | Maturity phase history | At least one entry; chronological order |
| `repositories` | string[] | Yes | Repository URLs | At least one valid HTTP(S) URL |
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
| `contact` | string | No | Security contact email | Valid email address (RFC 5322) if present |

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

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `type` | string | Yes | Identity agreement type | One of: `dco`, `cla`, `none` |
| `url` | PathRef | No | Link to DCO/CLA document | Path must be non-empty if present |

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

PathRef values can be either relative file paths (e.g., `SECURITY.md`) or full URLs (e.g., `https://github.com/org/repo/blob/main/SECURITY.md`).

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
| `teams` | Team[] | Yes | Team definitions | Must include `project-maintainers` team |

### Team

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `name` | string | Yes | Team name | `project-maintainers` is required |
| `members` | string[] | Yes | GitHub handles | Non-empty for `project-maintainers`; normalized (trimmed, `@` stripped) |

## Validation Rules

1. **Unknown fields are rejected** -- any field not in this schema causes a validation error
2. **URL validation** -- all URLs must have `http://` or `https://` scheme and a valid domain
3. **Email validation** -- uses RFC 5322 parsing
4. **Slug format** -- lowercase letters, digits, and hyphens; no leading/trailing hyphens
5. **Maturity ordering** -- maturity_log entries must be in chronological order
6. **Handle normalization** -- leading `@` and whitespace are stripped; duplicates are detected case-insensitively
7. **Required teams** -- every maintainer entry must include a `project-maintainers` team with at least one member
