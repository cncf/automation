# Project Status Audit

This utility generates a canonical list of CNCF project statuses from the LFX PCC API and audits that source of truth against multiple public datasets (CNCF Landscape, CLOMonitor, Foundation Maintainers CSV, DevStats, Artwork) plus optional [LFX Insights](https://insights.linuxfoundation.org/) health metrics. Results are published as unified human-readable tables.

## What it does

- Fetches PCC projects from LFX and writes `datasources/pcc_projects.yaml` (source of truth)
  - Active CNCF projects grouped by category (`Graduated`, `Incubating`, `Sandbox`)
  - `forming_projects` (status: “Formation - Exploratory”)
  - `archived_projects` (anything not Active or Forming)
- Downloads a reproducible snapshot of public sources into `datasources/`:
  - `landscape.yml`, `clomonitor.yaml`, `project-maintainers.csv`, `devstats.html`, `artwork.md`
- Optionally uses **`datasources/lfx_insights_health.yaml`** when present (from the weekly Insights workflow): **Insights Health** (tier label, e.g. Excellent, or **Archived** when LFX Insights shows the project archived) and **Health Score** (numeric when applicable). Missing file or per-project gaps show “-”; these columns are **informational only** and **never** affect anomaly detection.
- Audits and writes:
  - `audit/status_audit.md` — anomalies only; sorted Graduated → Incubating → Sandbox → Forming → Archived, A–Z within each
  - `audit/all_statuses.md` — all projects; anomalies section first, then the same status sections
- Missing values are rendered as “-”. A project is included in **Anomalies** if:
  - Any **lifecycle** source is missing (“-” for Landscape, empty for CLOMonitor / Maintainers / DevStats / Artwork), OR
  - Any of those sources reports a status different from PCC.

## Files

| Path | Role |
|------|------|
| `scripts/fetch_pcc_projects.py` | Calls LFX PCC API; writes `pcc_projects.yaml` (requires `LFX_TOKEN`) |
| `scripts/fetch_lfx_insights_health.py` | Builds `lfx_insights_health.yaml` from Insights project pages + badge (no token; archived status from page) |
| `scripts/audit_landscape_status.py` | Compares sources and writes `audit/*.md` |
| `scripts/landscape_source_diff.py` | Compare `landscape.yml` to `pcc_projects.yaml` + `clomonitor.yaml`; flags landscape drift and PCC↔CLOMonitor disagreements → `audit/landscape_data_integrity_audit/landscape_source_diff.{md,json}` |
| `scripts/repo_url_landscape_healthcheck.py` | Build both PCC-focused and CLOMonitor-focused repo URL anomalies from `landscape_source_diff.json` (`repo_url` findings) using `curl` checks, GitHub org-match alignment, and final destination comparison → `audit/landscape_data_integrity_audit/repo_url_pcc_landscape_anomalies.md`, `audit/landscape_data_integrity_audit/repo_url_landscape_clomonitor_anomalies.md` |
| `.github/workflows/sync-pcc-and-audit-statuses.yml` | Manual workflow: PCC + snapshots + audit → PR |
| `.github/workflows/landscape-data-content-auditor.yml` | Manual workflow: runs both landscape audit scripts; opens a PR only if `audit/landscape_data_integrity_audit/*.{md,json}` change (no `LFX_TOKEN`) |
| `.github/workflows/sync-lfx-insights-health.yml` | Weekly (Sunday UTC) + manual: refresh `lfx_insights_health.yaml` → PR |
| `datasources/pcc_projects.yaml` | Generated PCC data |
| `datasources/lfx_insights_health.yaml` | Generated **Insights Health** tier and **Health Score** (optional until first weekly run merges) |
| `datasources/` | Other source snapshots |
| `audit/status_audit.md` | Generated anomalies table |
| `audit/all_statuses.md` | Generated full table |
| `audit/landscape_data_integrity_audit/landscape_source_diff.{md,json}` | Generated landscape vs PCC / CLOMonitor diff |
| `audit/landscape_data_integrity_audit/repo_url_pcc_landscape_anomalies.md` | Generated PCC-focused repo URL anomalies from `landscape_source_diff.json` |
| `audit/landscape_data_integrity_audit/repo_url_landscape_clomonitor_anomalies.md` | Generated CLOMonitor-focused repo URL anomalies from `landscape_source_diff.json` |

## Data sources

- **PCC:** LFX `project-service` API (see `fetch_pcc_projects.py`)
- **Landscape:** `https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml`
- **CLOMonitor:** `https://raw.githubusercontent.com/cncf/clomonitor/main/data/cncf.yaml`
- **Foundation Maintainers CSV:** `https://raw.githubusercontent.com/cncf/foundation/main/project-maintainers.csv`
- **DevStats:** `https://devstats.cncf.io/`
- **Artwork README:** `https://raw.githubusercontent.com/cncf/artwork/main/README.md`
- **LFX Insights:** Project overview HTML ([insights.linuxfoundation.org](https://insights.linuxfoundation.org/)) plus the public badge; slug candidates follow PCC **name** then **`slug`** when they differ (see Notes).

## GitHub Actions (recommended)

### PCC sync + audit

1. Add repo secret **`LFX_TOKEN`** with a valid LF PCC API token.
2. Ensure Actions can write contents and open PRs with `GITHUB_TOKEN`.
3. **Actions → “Sync PCC and Audit CNCF Project Statuses” → Run workflow**
4. Review the PR: `pcc_projects.yaml`, `datasources/**`, `audit/status_audit.md`, `audit/all_statuses.md`

### LFX Insights health snapshot (no `LFX_TOKEN`)

1. **Actions → “Sync LFX Insights health scores” → Run workflow** (or wait for the weekly Sunday schedule).
2. Merge the PR that updates `datasources/lfx_insights_health.yaml`.
3. Run the PCC audit workflow again (or merge a branch that already includes the health file) so the **Insights Health** and **Health Score** columns populate.

### Landscape data content auditor (no `LFX_TOKEN`)

1. Ensure `datasources/landscape.yml`, `datasources/pcc_projects.yaml`, and `datasources/clomonitor.yaml` are present on the default branch (typically refreshed by the PCC sync workflow).
2. **Actions → “Landscape Data Content Auditor” → Run workflow**
3. If outputs under `audit/landscape_data_integrity_audit/` change, the workflow opens or updates a PR with only those files (`landscape_source_diff.{md,json}`, `repo_url_pcc_landscape_anomalies.md`, `repo_url_landscape_clomonitor_anomalies.md`).

## Run locally

Dependencies: Python 3.11+; `pip install -r requirements.txt` (run from this directory)

**PCC YAML** (writes `datasources/pcc_projects.yaml`):

```bash
export LFX_TOKEN=your_lfx_token
cd utilities/audit_project_lifecycle_across_tools
python scripts/fetch_pcc_projects.py
```

**LFX Insights health snapshot** (writes `datasources/lfx_insights_health.yaml`; requires `pcc_projects.yaml`):

```bash
cd utilities/audit_project_lifecycle_across_tools
python scripts/fetch_lfx_insights_health.py
# Optional: python scripts/fetch_lfx_insights_health.py --max-projects 10
```

**Landscape vs PCC / CLOMonitor diff** (requires `landscape.yml`, `pcc_projects.yaml`, `clomonitor.yaml` under `datasources/`):

```bash
cd utilities/audit_project_lifecycle_across_tools
python scripts/landscape_source_diff.py
```

**Repo URL anomaly reports from source diff** (requires `landscape_source_diff.json`; no `LFX_TOKEN`):

```bash
cd utilities/audit_project_lifecycle_across_tools
python scripts/repo_url_landscape_healthcheck.py
```

Generates:
- `audit/landscape_data_integrity_audit/repo_url_pcc_landscape_anomalies.md`
- `audit/landscape_data_integrity_audit/repo_url_landscape_clomonitor_anomalies.md`

**Audit reports:**

```bash
cd utilities/audit_project_lifecycle_across_tools
python scripts/audit_landscape_status.py
```

Outputs: `audit/status_audit.md`, `audit/all_statuses.md`, and any missing `datasources/*` snapshots downloaded on first run.

## Notes and assumptions

- PCC is the source of truth for **lifecycle** labels; external sources are compared to PCC categories (Graduated, Incubating, Sandbox, Archived, Forming, Prospect as applicable).
- **Insights Health** / **Health Score** are informational (not compared to PCC lifecycle).
- TAGs are excluded from the PCC categories section in `pcc_projects.yaml`.
- PCC entries with `status: Formation - Disengaged` are excluded from audit outputs.
- PCC entries with `status: Formation - Engaged` are treated as **Forming** in the audit.
- Landscape matching uses aliases (parentheticals, suffix trimming, Unicode normalization, `lfx_slug`, hyphen/space variants) — see `audit_landscape_status.py`.
- In `repo_url_landscape_clomonitor_anomalies.md`, archived projects are intentionally ignored because CLOMonitor removes archived projects quickly.
- DevStats parsing uses page row headings (“Graduated”, “Incubating”, “Sandbox”, “Archived”).
- LFX Insights fetch loads each project’s Insights **page** first. If the page shows the project as **Archived** there, we store **Insights Health** = `Archived` and no score (the public badge alone can still show another label and is not used in that case). URL slugs are tried as: slug derived from PCC **name** (same as the audit “Project” column), then PCC **`slug`** when it differs (e.g. CubeFS → `chubaofs`).
