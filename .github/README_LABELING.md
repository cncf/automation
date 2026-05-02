# Label & ChatOps Guide

This repository uses automated labeling via **slash commands** in issue comments and pull requests. Labels help organize, prioritize, and track work across the cncf-automation project.

## Quick Start

Add labels to an issue or PR by commenting:

```
/kind bug
/priority high
/status in-progress
/area ci
/help
```

---

## Slash Commands

All slash commands are triggered by commenting on an issue or PR. Commands can be combined in a single comment.

### Kind Labels
Specify the type of issue or PR:

```
/kind bug              # Bug or issue
/kind enhancement      # Feature or improvement
/kind docs             # Documentation update
/kind chore            # Chore or maintenance
/kind question         # Question or discussion
/kind initiative       # Initiative or tracking item
/kind meeting          # Meeting-related
/kind review           # Review item
/kind subproject       # Subproject-related
/kind dd               # DD (Distributed Definitions) related
```

### Priority Labels
Set the urgency level:

```
/priority critical     # Critical - needs immediate attention
/priority high         # High priority
/priority medium       # Medium priority
/priority low          # Low priority
```

### Status Labels
Indicate the current state:

```
/status needs-review   # Needs review before proceeding
/status blocked        # Blocked or waiting on something
/status in-progress    # Currently being worked on
```

### Area Labels
Tag items by their area of the codebase:

```
/area ci               # CI/CD infrastructure and automation
/area utilities        # Utilities and helper tools
/area cloudrunners     # Cloud runner infrastructure
/area automation       # Automation scripts and workflows
/area infrastructure   # Infrastructure and deployment
/area observability    # Observability, monitoring, and logging
/area ambassadors      # CNCF Ambassadors program
/area kubestronaut     # Kubestronaut program
```

### Helper Labels
Apply special labels:

```
/help                  # Mark as help-wanted (good for contributors)
```

### Remove Commands
Remove specific labels:

```
/remove-kind bug                    # Remove kind/bug label
/remove-priority high               # Remove priority/high label
/remove-status blocked              # Remove status/blocked label
/remove-area ci                     # Remove area/ci label
/remove-help                        # Remove help-wanted label
```

---

## Auto-Labeling Behavior

### File Path Auto-Labeling
Labels are automatically applied based on the files changed in a PR:

| Path Pattern | Applied Label | Description |
|---|---|---|
| `ci/**` | `area/ci` | CI/CD infrastructure |
| `utilities/**` | `area/utilities` | Utilities and tools |
| `ci/cloudrunners/**` | `area/cloudrunners` | Cloud runners |
| `Ambassadors/**` | `area/ambassadors` | Ambassadors program |
| `Kubestronaut/**` | `area/kubestronaut` | Kubestronaut program |
| `tests/**` | `area/infrastructure` | Testing infrastructure |

### Missing Label Indicators
If certain required labels are missing, helper labels are automatically applied:

- **`needs-kind`** — Applied when a PR/issue lacks a `kind/*` label
- **`needs-priority`** — Applied when a PR/issue lacks a `priority/*` label
- **`needs-area`** — Applied when a PR/issue lacks an `area/*` label
- **`needs-status`** — Applied when a PR/issue lacks a `status/*` label
- **`needs-triage`** — Applied when a PR/issue lacks a `triage/*` label
- **`needs-group`** — Applied when lacking `toc`, `tag/*`, or `sub/*` label

These helper labels encourage proper labeling without requiring it.

---

## Label Categories & Meanings

### 🎯 Kind (Issue Type)
Indicates what type of work this is.

**Colors**: Purple (`8250DF`)

| Label | Meaning |
|---|---|
| `kind/bug` | A defect or issue that needs fixing |
| `kind/enhancement` | A feature request or improvement |
| `kind/docs` | Documentation update or new docs |
| `kind/chore` | Maintenance, cleanup, refactoring |
| `kind/question` | Questions or discussions |
| `kind/initiative` | Initiative or tracking item |
| `kind/dd` | Related to DD (Distributed Definitions) process |

### 🔴 Priority (Urgency)
Indicates how urgent this work is.

**Color Spectrum**: Red (critical) → Orange (high) → Amber (medium) → Gray (low)

| Label | Meaning |
|---|---|
| `priority/critical` | **B62324** - Needs immediate attention |
| `priority/high` | **FF6B35** - High urgency, do soon |
| `priority/medium` | **FFA500** - Standard priority |
| `priority/low` | **D3D3D3** - Can be deferred |

### 🟦 Status (Current State)
Indicates the current state of work.

**Colors**: Traffic light system (Yellow → Red → Blue)

| Label | Meaning |
|---|---|
| `status/needs-review` | **FBCA04** (yellow) - Awaiting review |
| `status/blocked` | **b60205** (red) - Blocked or waiting |
| `status/in-progress` | **1F6FEB** (blue) - Currently being worked on |

### 🟣 Area (Codebase Location)
Indicates which area of the codebase this affects.

**Color**: Violet (`7057FF`)

| Label | Meaning |
|---|---|
| `area/ci` | CI/CD infrastructure, workflows, automation |
| `area/utilities` | Utility scripts, helper tools |
| `area/cloudrunners` | Cloud runner infrastructure |
| `area/automation` | Automation workflows and scripts |
| `area/infrastructure` | Infrastructure, deployment, IaC |
| `area/observability` | Observability, monitoring, logging |
| `area/ambassadors` | CNCF Ambassadors program |
| `area/kubestronaut` | Kubestronaut program |

### ⏳ Triage (TOC-specific)
Pre-existing labels for TOC (table of contents) categorization.

| Label | Meaning |
|---|---|
| `triage/valid` | Issue has sufficient detail |
| `triage/duplicate` | Duplicate of existing issue |
| `triage/needs-information` | Needs more info before action |
| `triage/not-planned` | Out of scope or won't be done |

### 🎓 Group Labels
Broader categorizations (from TOC repo):

- **`toc`** - TOC (Table of Contents) related
- **`tag/<name>`** - TAG (Technical Advisory Group) related
- **`sub/<name>`** - Subproject related

---

## Examples

### Example 1: Labeling a Bug Fix
```
This PR fixes issue #123 where the CI pipeline fails on ARM64.

/kind bug
/priority high
/area ci
/status in-progress
```

### Example 2: Feature Request from Community
```
Request: Add support for additional cloud providers in cloud runners.

/kind enhancement
/priority medium
/area cloudrunners
/help
```

### Example 3: Documentation Update
```
Updated the README with new CLI options.

/kind docs
/area utilities
```

### Example 4: Removing Incorrect Labels
```
Removing this from high priority as we've decided to defer.

/remove-priority high
/priority low
```

---

## Color Meanings at a Glance

| Color | Pattern | Meaning |
|---|---|---|
| 🟢 **Green** (`2DA44E`) | Complete | Done, finished, resolved |
| 🔵 **Blue** (`1F6FEB`) | In Progress | Actively being worked on |
| 🟠 **Orange** (`D97706`) | Not Started | Backlog, to-do |
| 🔴 **Red** (`b60205`) | Blocked/Issue | Problem, blocked, or needs attention |
| 🟡 **Yellow** (`FBCA04`) | Pending/Needs | Waiting for input, needs review |
| 🟣 **Purple** (`8250DF`) | Category | Kind/type indicators |
| 🟦 **Teal** (`0E8A8A`) | Group | Tag/group identifiers |

---

## Best Practices

1. **Be Specific**: Use the most specific label available (e.g., `priority/critical` not just `urgent`)
2. **Combine Labels**: Use multiple labels together for clarity:
   - Always include a `kind/*` label
   - Use `priority/*` for PRs and important issues
   - Add `area/*` to identify affected parts of the codebase
3. **Use Auto-Labels**: Let `needs-*` labels guide you to apply missing labels
4. **Update Status**: As work progresses, update the `status/*` label to reflect current state
5. **Remove When Done**: Remove `status/in-progress` and `priority/` labels when complete

---

## Troubleshooting

### Commands Not Working?

1. **Verify format**: Commands must start with `/` at the beginning of a line or comment
2. **Check spelling**: Label values must be valid (e.g., `/priority high` not `/priority HIGH`)
3. **One command per line**: Use separate lines for each command if combining multiple:
   ```
   /kind bug
   /priority high
   /area ci
   ```
4. **Workflow needs token**: The slash-commands workflow requires the `SLASH_COMMANDS_PAT` secret to be configured

### Labels Not Applying?

1. **Check issue permissions**: The workflow needs write access to the repository
2. **Verify label exists**: The label must be defined in `.github/labels.yaml`
3. **Workflow status**: Check the Actions tab to see if `slash-commands.yml` executed successfully

---

## For Repository Maintainers

### Adding New Labels

1. Edit `.github/labels.yaml`
2. Add new label definition in the `labels:` section
3. Add slash command rules in the `ruleset:` section if needed
4. Commit and push — labels will auto-sync on next workflow run

### Modifying Slash Commands

1. Edit `.github/labels.yaml` and update the ruleset
2. Edit `.github/workflows/slash-commands.yml` to add new commands if needed
3. Commit and push

### Monitoring Labels

Check the **Labels** section in repository settings to see:
- All available labels and their current colors
- Label usage across issues and PRs
- Stale or unused labels

---

## Label Governance & Validation

We use automated workflows to keep labels in sync and validate their correctness.

### Auto-Sync Labels
**Workflow**: `.github/workflows/auto-sync-labels.yml`

Automatically synchronizes labels between `.github/labels.yaml` and GitHub repository:
- **When**: Triggered on push to main when `labels.yaml` changes, or manually via workflow dispatch
- **Actions**:
  - Creates new labels from `labels.yaml`
  - Updates colors and descriptions for existing labels
  - **Deletes** labels not in `labels.yaml` (if `autoDeleteLabels: true`)
- **Configuration**: Controlled by top-level `autoCreateLabels` and `autoDeleteLabels` flags in `labels.yaml`

### Validate Label Schema
**Workflow**: `.github/workflows/validate-labels.yml`

Validates that `labels.yaml` has correct structure and format:
- **When**: On PRs and pushes to main that modify `labels.yaml`
- **Checks**:
  - Valid YAML syntax
  - All labels have required fields (name, color)
  - Color values are valid hex codes
  - Label names follow naming conventions (lowercase, hyphens, slashes)
  - All referenced labels in rules exist
- **Comments** on PRs with validation results
- **Fails** the check if structural errors found (warnings allowed)

### Detect Label Drift
**Workflow**: `.github/workflows/detect-label-drift.yml`

Monitors for unintended divergence between defined labels and GitHub repository:
- **When**: Daily at 2 AM UTC, or manually via workflow dispatch
- **Detection**:
  - Labels in GitHub not defined in `labels.yaml`
  - Defined labels missing from GitHub
  - Color or description differences (drift)
- **Reports**:
  - Summary to workflow logs
  - Creates an issue if drift detected with detailed report
  - Uploads drift report artifact
- **Use Case**: Catches accidental label changes in GitHub UI that bypass the configuration

---

## Best Practices for Label Management

1. **Always edit `labels.yaml`** — Never manually edit labels in GitHub UI
2. **Use validation** — Push to PR first to catch schema errors
3. **Monitor drift** — Check drift detection reports weekly
4. **Update together** — Sync labels, rules, and documentation in same commit
5. **Test commands** — Try new slash commands in a test issue before promoting
6. **Document changes** — Note why label categories were added/modified

---

## Troubleshooting Label Issues

### Labels Not Syncing
- Check that push was to `main` branch
- Verify `labels.yaml` path is correct (must be `.github/labels.yaml`)
- Check workflow run logs in Actions tab

### Validation Fails
- Review the validation workflow logs for specific errors
- Most common: Invalid hex color (must be 6-digit, e.g., `2DA44E`)
- Ensure all label names are unique

### Drift Detected
- Compare the drift report to `labels.yaml`
- If changes were intentional, update `labels.yaml` to match
- If changes were accidental, revert GitHub labels and run auto-sync
- Re-run drift detection after sync

---

## Related Documentation

- [GitHub Labels Settings](https://github.com/cncf/automation/labels)
- [Repository Issues](https://github.com/cncf/automation/issues)
- [Contributing Guide](./CONTRIBUTING.md)
- [Configuration File](./labels.yaml)
