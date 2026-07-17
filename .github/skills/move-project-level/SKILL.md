---
name: move-project-level
description: Move a CNCF project to a new maturity level (sandbox, incubating, graduated, archived) across the landscape, clomonitor, foundation, and artwork repos. Use whenever someone wants to promote, graduate, archive, or otherwise change the maturity/level of a CNCF project and have the changes applied consistently across all the relevant repos. Triggers on phrases like "move X to incubating", "graduate project Y", "archive project Z", "promote a CNCF project", or "project moving levels".
---

# Move a CNCF project to a new maturity level

This skill drives a repeatable, multi-repo workflow for changing a CNCF project's
maturity level. The full, form-driven procedure lives in the prompt file:

- `.github/prompts/move-project-level.prompt.md`

## How to run it

The recommended entry point is the prompt file. In Copilot Chat (agent mode) run:

```
/move-project-level
```

Fill in the form when asked:

- **Project** — name exactly as it appears in `landscape.yml` (e.g. `HAMi`)
- **New level** — `sandbox`, `incubating`, `graduated`, or `archived`
- **Date moved** — e.g. `01 Jan 2025`

## Prerequisites

- The VS Code workspace must include all four target repos as folders:
  `cncf-landscape`, `clomonitor`, `cncf-foundation`, and `cncf-artwork/artwork`
  (alongside `cncf-automation`, which hosts this skill).
- SSH access to each repo's remote, ideally non-interactive (see the keychain
  setup below) so the per-repo `git pull` steps don't stall on a passphrase.

## What it changes (per repo)

| Repo | File | Change |
|------|------|--------|
| cncf-landscape | `landscape.yml` | `project:` field, dated `extra:` key, and maturity-specific `artwork_url` |
| clomonitor | `data/cncf.yaml` | `maturity:` field |
| cncf-foundation | `project-maintainers.csv` | level column (col 1) for the project |
| artwork | `examples/*.md`, `README.md` | move logo block to the new maturity file |

## Guardrails

- Creates a `move/<slug>-to-<level>` branch per repo off the detected default branch.
- Does **not** commit or push by default — stops at the diff for review.
- When asked to commit, follows `AGENTS.md`: DCO sign-off (`git commit -s`) and a
  present-tense subject.
- When asked to push, pushes the feature branch to origin only — **never** to a
  repo's default branch.

For the authoritative step-by-step instructions, open and follow the prompt file
referenced above.
