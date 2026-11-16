# committees.yaml: requirements and design

`committees.yaml` is the org-level counterpart of a project's
`maintainers.yaml`. It records an organization's governance bodies, the roles
inside them, who holds each role, who elected them and for which term. The
field reference is in [SCHEMA.md](SCHEMA.md#committeesyaml); this document
explains why the file looks the way it does and how CNCF data maps onto it.

## Goals

- One machine-readable source of truth for CNCF governance membership, kept in
  `cncf/.project`, so that later automation can generate the cncf.io people
  pages (TOC, TAGs, TAB, Governing Board, staff) and GitHub team membership
  instead of hand-editing `cncf/people` `people.json` and `config.yaml`.
- Generic: nothing CNCF-specific in the schema, so other organizations (for
  example Kubernetes, which runs its own GitHub orgs) can adopt it as is.
- Validated in CI like every other `.project` file.

## Non-goals (for now)

- Building the sync automation (cncf.io pages, CLOWarden/GitHub teams, LFX).
  The file is designed so that automation can be written later without a
  schema change.
- Programs that are not committees: Ambassadors, Kubestronauts, Golden
  Kubestronauts and Mentees stay in `cncf/people`.
- Contact details or other personal data.

## Requirements

| # | Requirement | How it is met |
|---|-------------|---------------|
| R1 | Every member is identified by a valid GitHub handle | `github` is required and validated; the validator also checks by default that the account exists (404 fails) |
| R2 | Record who elected/appointed each person | Required free-text `elected_by` (replaces the `source` idea from cncf/automation#744) |
| R3 | Record terms | Required `start`, optional `end` (`YYYY-MM-DD`) |
| R4 | Support roles without a fixed term (staff, board seats) | Omit `end` |
| R5 | Keep past members | `emeritus: true` teams, whose members must have an `end`; a handle may repeat in a team when terms do not overlap (re-elections, separate stints) |
| R6 | Members are objects, never bare strings | Bare strings are a validation error |
| R7 | No PII | Members accept only `github`, `role`, `liaison`, `elected_by`, `start`, `end`; anything else (for example `email`, `name` or `company`) is rejected. Names and affiliations come from LFID |
| R8 | Describe GitHub access | `github_teams` per role; a GitHub team's intended members are the current members of every non-emeritus role that lists it |
| R9 | Validation must not start failing because time passed | Expired terms and future emeritus end dates are warnings, not errors. Empty roles are errors: a role is added once it has a holder |
| R10 | Usable outside CNCF | No CNCF-specific values in the schema; `org` names the GitHub org |

## Design decisions

- **Teams are roles.** A committee has teams such as `members`, `alternates`,
  `chairs`, `tech-leads`, `leads`, `emeritus`.
- **Positions within a team are a member `role`.** The TOC, TAB and
  Governing Board chairs are members of their body, so they are marked with
  `role: chair` (or `vice-chair`, `apac-chair`) on their `members` entry
  instead of being listed again in a separate team. The role shares the
  entry's `elected_by` and term; a role change is recorded by editing the
  entry. TAG `chairs` stay a team, because there the role is the whole team.
- **Liaisons are a member `liaison` list.** TOC members are liaisons to one
  or more TAGs, so their TOC `members` entry lists the TAG ids, for example
  `liaison: [tag-infrastructure, tag-workloads-foundation]`. A TAG's liaisons
  are the members whose `liaison` names it. The validator checks every id
  exists. The field is not TOC-specific: any member may be liaison to another
  committee in the file.
- **`elected_by` rather than `source`.** "Source" was ambiguous (data source vs
  selecting body). `elected_by` names the selecting body or constituency, which
  is what the people pages show (e.g. "GB-appointed").
- **GitHub team maintainers are not modelled.** In the CNCF org, staff are
  *maintainers* of governance teams (for example `mrbobbytables` and
  `riaankleinhans` on `cncf-toc-voters`). That is an administrative role, not
  a committee seat, so a sync should only manage `role=member`.
- **Composite GitHub teams fall out of the union rule.** `tag-<x>-leads` and
  `cncf-tag-leads` are listed on both the `chairs` and `tech-leads` roles, so
  their membership is derived rather than declared separately (the
  `formation` mechanism in `cncf/people` `config.yaml`).

## CNCF bodies and how they map

| Committee `id` | Body | Roles (teams) | `elected_by` values | GitHub teams |
|---|---|---|---|---|
| `toc` | Technical Oversight Committee: 11 voting members, 2-year staggered terms; plus non-voting alternates | `members` (roles `chair`, `vice-chair`; `liaison` lists the TAGs each member is liaison to), `alternates`, `emeritus` | `Governing Board`, `End User TAB`, `TOC`, `Non-sandbox project maintainers` (charter 6(b)(iii)) | `cncf-toc` (members + alternates), `cncf-toc-voters` (members) |
| `tag-*` | Technical Advisory Groups, overseen by the TOC: chairs and tech leads serve 2-year terms | `chairs`, `tech-leads` (liaisons are the TOC members whose `liaison` names the TAG) | `TOC` (tech leads are nominated by chairs and approved by the TOC) | `tag-<x>-chairs`, `tag-<x>-tech-leads`, `tag-<x>-leads`, `cncf-tag-leads` |
| `tag-*-<sub>` | TAG subprojects: 1-year terms | `leads` | the parent TAG (super-majority of the parent body) | none |
| `mentoring-subproject` | TOC subproject: 1-year terms | `chairs`, `tech-leads` | `TOC` | none |
| `end-user-tab` | End User Technical Advisory Board | `members` (roles `chair`, `vice-chair`, `apac-chair`), `emeritus` | seat: `End User Platinum/Gold/Silver members`, `End User community (at-large)`, `TOC` | `cncf-tab` |
| `governing-board` | Governing Board | `members` (roles `chair`, `vice-chair`) | seat: member tier, `Kubernetes project`, `Developer seat`, ex-officio TOC/TAB chair | `cncf-gb` |
| `staff` | CNCF staff | `staff` | `CNCF` | `cncf-staff` |

Board and staff seats have no fixed terms, so their members omit `end`.

## Data sources and precedence

The one-time bootstrap used, in order of precedence:

1. `cncf/toc`: `README.md` (TOC members, alternates, seats, terms, liaisons),
   `EMERITUS.md`, `tags.yaml` (TAG chairs/tech leads with terms, subprojects),
   `governance/tag-governance.md` (who elects whom, term lengths)
2. `cncf/tab`: `gov.yaml`, `emeritus.md`
3. `cncf/people`: `config.yaml` (GitHub team declarations)
4. `cncf/people`: `people.json` (`toc_role`/`gb_role`/`tab_role`)
5. Live GitHub team membership

Conflicts were resolved in that order and listed in the bootstrap drift
report. Emails in `tags.yaml` were not copied. Where no start date was
recorded, the first appearance of the person in that role in the source file's
git history was used and flagged as approximate.

## Adopting it in an organization's `.project` repository

```yaml
name: Validate committees
on:
  pull_request:
    paths: [committees.yaml, .github/workflows/validate-committees.yaml]
  push:
    branches: [main]
    paths: [committees.yaml, .github/workflows/validate-committees.yaml]
  workflow_dispatch:
permissions:
  contents: read
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0 # lets the action check only handles new since the base commit
          persist-credentials: false
      - uses: cncf/automation/.github/actions/validate-committees@<sha> # pin to a commit
```

Handles and URLs are checked by default; add `with: { check_links: 'false' }`
to skip that.

## Open questions

1. Should the Governing Board and staff be synced to `cncf-gb`/`cncf-staff`?
   Today `cncf-gb` has a single member; a sync from this file would add every
   GB member with a GitHub handle.
2. Many GB members and most non-technical staff have no GitHub handle. Do they
   belong in this file (which would make `github` optional), or do their pages
   keep another source?
3. Should the Ambassadors program, the Marketing Committee and the TCGs
   (Technical Community Groups) be modelled here? There is no structured
   source for TCG membership today.
4. Who owns `elected_by` for historical TOC terms and for GB seats that
   `people.json` does not describe (bootstrapped as `unknown`)?
5. Should the cncf.io people pages read this file directly, or should a
   generator keep `people.json` as an output format?
