---
mode: agent
description: Move a CNCF project to a new maturity level across all repos in the workspace (landscape, clomonitor, foundation, artwork) — creates a branch per repo and applies the correct per-repo edits.
---

# Move a CNCF project to a new maturity level

You are automating a CNCF project maturity-level change across the four repos in
this multi-root workspace. Work carefully and idempotently. Do NOT push, and do
NOT open pull requests unless explicitly asked.

## Inputs (the form)

Collect these three values. If any are missing, ask me for them before starting.

- **Project**: ${input:project:Project name as it appears in landscape.yml (e.g. HAMi)}
- **New level**: ${input:newLevel:sandbox | incubating | graduated | archived}
- **Date moved**: ${input:dateMoved:e.g. 01 Jan 2025}

Normalize the values first and show me the normalized plan before editing:

- `PROJECT` = the exact `name:` used in landscape.yml.
- `LEVEL` = lowercase one of: `sandbox`, `incubating`, `graduated`, `archived`.
- `LEVEL_TITLE` = Title-case (`Sandbox`/`Incubating`/`Graduated`/`Archived`).
- `DATE_ISO` = the date converted to `YYYY-MM-DD`.
- `SLUG` = lowercase, hyphenated project slug (spaces → `-`).
- `BRANCH` = `move/${SLUG}-to-${LEVEL}`.

## Repos in this workspace

This prompt lives in **cncf-automation** but edits the four target repos below.
The workspace must contain all four folders for the workflow to run end to end.

| Key | Workspace folder | Level-bearing file |
|-----|------------------|--------------------|
| landscape | `cncf-landscape` | `landscape.yml` |
| clomonitor | `clomonitor` | `data/cncf.yaml` |
| foundation | `cncf-foundation` | `project-maintainers.csv` |
| artwork | `cncf-artwork/artwork` | `examples/*.md`, `README.md` |

## Step 1 — Prepare git in each repo

For EACH of the four repos, run (using the repo's own path with `git -C <repo>`):

1. `git -C <repo> status --porcelain` — if the repo has uncommitted changes,
   STOP and report; do not proceed for that repo until I confirm.
2. Detect the default branch:
   `git -C <repo> remote show origin | sed -n 's/.*HEAD branch: //p'`
3. `git -C <repo> checkout <default-branch>`
4. `git -C <repo> pull --ff-only`
5. `git -C <repo> checkout -b ${BRANCH}` (if the branch already exists, check it
   out instead and report that it existed).

Report a short table of each repo's default branch and the branch you created.

## Step 2 — Apply the per-repo edits

Only edit the file(s) listed. Preserve surrounding formatting, indentation, and
quote style exactly. Make the minimal change.

### landscape (`cncf-landscape/landscape.yml`)

Find the `- item:` whose `name:` equals `PROJECT`.

1. Set its `project:` field to `LEVEL`. If the item has no `project:` key, add
   one directly under `name:` (this is a project entering the CNCF program).
2. Under that item's `extra:` block, add a dated key for the new level using
   single-quoted ISO format, keeping any existing date keys (`accepted:`,
   earlier `incubating:`, etc.). Add whichever applies:
   - incubating → `incubating: 'DATE_ISO'`
   - graduated  → `graduated: 'DATE_ISO'`
   - archived   → `archived: 'DATE_ISO'`
   (sandbox entry typically only has `accepted:` — add `accepted:` if missing.)
   If the key already exists, update its value to `DATE_ISO`.
3. If the item has an `artwork_url:` field pointing at a maturity-specific
   examples file (e.g. `.../examples/incubating.md#<slug>-logos`), update the
   filename segment to the destination maturity file
   (`.../examples/${LEVEL}.md#<slug>-logos`; use the correct `sandbox_*` shard
   for sandbox). Leave the `#<slug>-logos` anchor unchanged.

### clomonitor (`clomonitor/data/cncf.yaml`)

Find the entry whose `name:` or `display_name:` matches `PROJECT`. Set its
`maturity:` field to `LEVEL` (double-quoted values are used for dates here, but
`maturity:` is an unquoted bareword — match the file's existing style).

### foundation (`cncf-foundation/project-maintainers.csv`)

The first CSV column holds the level (`Graduated`/`Incubating`/`Sandbox`/`Archived`).
A project usually has one header row with the level + project name, followed by
continuation rows with empty first two columns. Update the first column of the
project's header row to `LEVEL_TITLE`. Do not touch other projects' rows.

### artwork (`cncf-artwork/artwork/examples/*.md` and `README.md`)

Maturity files: `graduated.md`, `incubating.md`, `sandbox_a-j.md`,
`sandbox_k.md`, `sandbox_l-r.md`, `sandbox_s-z.md`, `archived.md`.

1. Locate the project's logo section — an `#### <Project> Logos` heading followed
   by its HTML `<table>` block — in its current maturity file.
2. Move that entire block into the destination maturity file, inserting it in
   alphabetical order among the other `####` sections. For sandbox, pick the
   correct alphabetical shard (`a-j`, `k`, `l-r`, `s-z`).
3. Update the corresponding link in `README.md` to point at the new file.
4. If `LEVEL` is `archived`: also move the directory
   `projects/${SLUG}` → `archived/${SLUG}` (use `git -C <artwork> mv`), and use
   `archived.md` as the destination.
   If the project has no logo section, note it and skip the artwork edits.

## Step 3 — Verify

- `git -C <repo> --no-pager diff` for each repo; confirm only the intended lines
  changed.
- For landscape/clomonitor, sanity-check YAML is still valid (indentation intact).
- Do NOT commit or push. Leave the changes staged/unstaged on the branch for me
  to review.

## Step 4 — Summary

Print a table: repo | branch | file(s) changed | lines changed | notes. Flag any
repo where the project was not found so I can handle it manually.

## When I later ask you to commit

Follow the cncf-automation `AGENTS.md` conventions:

- DCO sign-off is required: `git -C <repo> commit -s`.
- Present-tense subject, e.g. `Move ${PROJECT} to ${LEVEL} (${DATE_ISO})`.
- Commit each repo separately on its `${BRANCH}` branch.
- Only push when I explicitly ask. When pushing, push the feature branch to
  origin (`git -C <repo> push -u origin ${BRANCH}`) — NEVER push to a repo's
  default branch. Do not open pull requests unless asked.
