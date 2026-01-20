# Project Status Audit
Project Status Audit

This repository generates a canonical list of CNCF project statuses from the LFX PCC API and audits that source of truth against multiple public datasets (CNCF Landscape, CLOMonitor, Foundation Maintainers CSV, and DevStats). Results are published as a unified human-readable table.

## What it does

- Fetches PCC projects from LFX and writes `pcc_projects.yaml` (source of truth)
  - Includes active CNCF projects grouped by category (`Graduated`, `Incubating`, `Sandbox`)
  - Includes `forming_projects` (status: “Formation - Exploratory”)
  - Includes `archived_projects` (anything not Active or Forming)
- Downloads a reproducible snapshot of public sources into `datasources/`:
  - `datasources/landscape.yml`, `datasources/clomonitor.yaml`, `datasources/project-maintainers.csv`, `datasources/devstats.html`, `datasources/artwork.md`
- Audits against those sources and writes:
  - `audit/status_audit.md` (Anomalies only; sorted: Graduated → Incubating → Sandbox → Forming → Archived, A–Z within each)
  - `audit/all_statuses.md` (All projects; Anomalies section first, then the same status sections)
- Missing values are rendered as “-”. A project is included in Anomalies if:
  - Any source is missing (“-” for Landscape, empty for others), OR
  - Any source reports a status different from PCC.

## Files

- `scripts/fetch_pcc_projects.py`: Fetches LFX PCC and writes `pcc_projects.yaml`
- `scripts/audit_landscape_status.py`: Audits external sources and writes both reports
- `.github/workflows/sync-pcc-and-audit-statuses.yml`: Manual workflow that fetches PCC + sources, runs audits, and opens a PR
- `pcc_projects.yaml`: Generated, canonical PCC data (no timestamp to avoid noisy diffs)
- `datasources/`: Snapshot of audited source files (captured by the workflow)
- `audit/status_audit.md`: Generated anomalies table (mismatches or missing data)
- `audit/all_statuses.md`: Generated full table of all projects and sources

## Data sources

- Landscape: `https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml`
- CLOMonitor: `https://raw.githubusercontent.com/cncf/clomonitor/main/data/cncf.yaml`
- Foundation Maintainers CSV: `https://raw.githubusercontent.com/cncf/foundation/main/project-maintainers.csv`
- DevStats: `https://devstats.cncf.io/`
- Artwork README: `https://raw.githubusercontent.com/cncf/artwork/main/README.md`

## GitHub Actions (recommended)

1. Add a repo secret `LFX_TOKEN` with a valid LF PCC API token.
2. Ensure Actions permissions allow “Read and write permissions” and PR creation for the `GITHUB_TOKEN`.
3. Trigger the workflow:
   - GitHub → Actions → “Sync PCC and Audit CNCF Project Statuses” → “Run workflow”
4. Review the PR with updates to:
   - `pcc_projects.yaml`
   - `datasources/**`
   - `audit/status_audit.md`
   - `audit/all_statuses.md`

## Run locally

Dependencies:
- Python 3.11+
- pip packages: `requests`, `pyyaml`, `beautifulsoup4`

Generate PCC YAML (writes to `datasources/pcc_projects.yaml`):

```bash
export LFX_TOKEN=your_lfx_token
python scripts/fetch_pcc_projects.py
```

Run the audits:

```bash
python scripts/audit_landscape_status.py
```

Outputs:
- `datasources/pcc_projects.yaml`
- `datasources/**` snapshot (if files were not already present for other sources)
- `audit/status_audit.md` (anomalies only, missing data as “-”)
- `audit/all_statuses.md` (all projects: anomalies first, then by status group)

## Notes and assumptions

- PCC is the source of truth; we compare maturity/status labels from external sources to PCC categories:
  - Graduated, Incubating, Sandbox, Archived, Forming (Formation - Exploratory)
- TAGs are intentionally excluded from the PCC categories section.
- The workflow snapshots audited sources into `datasources/**` and includes them in the PR for reproducibility.
- Landscape matching improvements to reduce name variance issues:
  - Aliases from parentheses/acronyms (e.g., “Open Policy Agent (OPA)”, “KAITO”, “ORAS”)
  - Common suffix trimming (“Project”, “Specification”, “Operator”, “Framework”)
  - Unicode normalization (e.g., “Metal³” → “metal3”), lfx_slug alias, and hyphen/space variants
- DevStats parsing uses the page’s row headings (“Graduated”, “Incubating”, “Sandbox”, “Archived”) to derive statuses.

