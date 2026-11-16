# Label & ChatOps Guide

Labels and slash commands are handled by [cncf/prow-github-actions](https://github.com/cncf/prow-github-actions)
via [`workflows/prow.yml`](./workflows/prow.yml). Labels, commands and `needs-*` rules are defined in
[`prow.yaml`](./prow.yaml); reviewers and approvers come from `OWNERS` files.

## Slash Commands

Comment on an issue or PR, one command per line.

| Command | Effect |
|---------|--------|
| `/kind <value>` | Adds `kind/<value>` (stacks) |
| `/area <value>` | Sets `area/<value>` (exclusive) |
| `/priority <value>` | Sets `priority/<value>` (exclusive) |
| `/status <value>` | Sets `status/<value>` (exclusive) |
| `/triage <value>` | Sets `triage/<value>` (exclusive, issues only) |
| `/remove-<family> <value>` | Removes the label |
| `/lgtm`, `/lgtm cancel` | Review approval, bound to the current commit |
| `/approve`, `/approve cancel` | Approver sign-off per `OWNERS` coverage |
| `/hold`, `/unhold` | Blocks / unblocks merging (`do-not-merge/hold`) |
| `/assign`, `/cc`, `/uncc` | Assignees and review requests |
| `/retest`, `/test` | Re-run the PR's workflow runs |
| `/close`, `/reopen`, `/retitle`, `/milestone`, `/lifecycle` | Issue management |
| `/good-first-issue`, `/help` | Applies `good first issue` / `help wanted` |

Allowed values per family are listed in [`prow.yaml`](./prow.yaml).

## Automatic Behavior

- `needs-kind`, `needs-area`, `needs-priority`, `needs-status` are applied until a matching label exists (`needs-triage` on issues only).
- PRs touching a directory with an `OWNERS` `labels:` block get those labels (`ci/` → `area/ci`, `utilities/` → `area/utilities`, `.github/` → `kind/github-actions`, ...).
- Two reviewers from the relevant `OWNERS` files are requested when a PR opens (drafts wait for ready-for-review; Dependabot skipped).
- A PR merges (squash) once it carries `lgtm` **and** `approved` and none of `do-not-merge/*`, `needs-rebase`, `hold`.
- New commits drop `lgtm`.

Fork PRs are processed by a 5-minute sweep rather than instantly, since the `pull_request` event gives them a read-only token.

## Adding Labels or Commands

1. Edit [`prow.yaml`](./prow.yaml) — a new key under `labels:` becomes a `/<key>` command.
2. Merge; the `label-sync` job creates the labels on push (or run *Actions → Prow → Run workflow*). It never deletes labels.

## OWNERS

Any directory may hold an `OWNERS` file; it applies to that directory and below and is unioned with its parents.

```yaml
approvers:
  - alice
reviewers:
  - alice
  - bob
labels:
  - area/ci
```
