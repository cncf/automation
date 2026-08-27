# Landscape vs datasources diff

**Canonical:** `datasources/pcc_projects.yaml` and `datasources/clomonitor.yaml`. 
When those two disagree, that is called out. **`landscape.yml` should be updated** to match the agreed sources (or you must reconcile PCC vs CLOMonitor first).

## Summary

- **CNCF landscape items in scope:** 256
- **With at least one drift / conflict row:** 6
- **Findings where Landscape and CLOMonitor disagree:** 6
- **No PCC and no CLOMonitor match:** 2

## Differences (sorted by field)

Each row is one detected mismatch. Sorted by `Field`, then `Project`.

| Field | Project | Maturity | Landscape | PCC | CLOMonitor | Landscape≈CLO? | Note |
|---|---|---|---|---|---|---|---|
| extra.clomonitor_name | Cedar | sandbox | cedar | — | cedar-policy | **No** | Landscape ('cedar') ≠ CLOMonitor ('cedar-policy'). |
| extra.clomonitor_name | llm-d | sandbox | — | — | llm-d | **No** | Landscape missing; CLOMonitor has 'llm-d'. |
| extra.clomonitor_name | Podman Container Tools | sandbox | podman | — | podman-container-tools | **No** | Landscape ('podman') ≠ CLOMonitor ('podman-container-tool… |
| extra.dev_stats_url | Cedar | sandbox | — | — | https://cedarpolicy.devstats.cncf.io/ | **No** | Landscape missing; CLOMonitor has 'https://cedarpolicy.de… |
| extra.dev_stats_url | llm-d | sandbox | — | — | https://llmd.devstats.cncf.io/ | **No** | Landscape missing; CLOMonitor has 'https://llmd.devstats.… |
| extra.lfx_slug | llm-d | sandbox | — | llm-d | — | — | Landscape missing; PCC has 'llm-d'. |
| project (maturity) | Service Mesh Performance | archived | archived | sandbox | — | — | Landscape ('archived') ≠ PCC ('sandbox'). |
| repo_url | container2wasm | sandbox | https://github.com/container2wasm/container2wasm | https://github.com/ktock/container2wasm | https://github.com/container2wasm/container2wasm | Yes | PCC ('https://github.com/ktock/container2wasm') and CLOMo… |
| repo_url | Drasi | sandbox | http://github.com/drasi-project/drasi-platform | https://github.com/drasi-project | https://github.com/drasi-project/drasi-platform | **No** | Landscape ('http://github.com/drasi-project/drasi-platfor… |

## No datasource match

These are in-scope landscape projects that could not be matched to PCC or CLOMonitor; they are usually candidates for upstream/source alignment PRs.

| Project | Maturity | Path |
|---------|----------|------|
| Service Mesh Interface (SMI) | archived | Orchestration & Management / Service Mesh |
| Monocle | sandbox | Observability and Analysis / Observability |