# Landscape vs datasources diff

**Canonical:** `datasources/pcc_projects.yaml` and `datasources/clomonitor.yaml`. 
When those two disagree, that is called out. **`landscape.yml` should be updated** to match the agreed sources (or you must reconcile PCC vs CLOMonitor first).

## Summary

- **CNCF landscape items in scope:** 251
- **With at least one drift / conflict row:** 10
- **Findings where Landscape and CLOMonitor disagree:** 8
- **No PCC and no CLOMonitor match:** 5

## Differences (sorted by field)

Each row is one detected mismatch. Sorted by `Field`, then `Project`.

| Field | Project | Maturity | Landscape | PCC | CLOMonitor | Landscape≈CLO? | Note |
|---|---|---|---|---|---|---|---|
| extra.accepted | Copa | sandbox | 2023-09-19 | — | 2023-12-19 | **No** | Landscape ('2023-09-19') ≠ CLOMonitor ('2023-12-19'). |
| extra.accepted | KubeStellar | sandbox | 2023-12-19 | — | 2023-09-19 | **No** | Landscape ('2023-12-19') ≠ CLOMonitor ('2023-09-19'). |
| extra.clomonitor_name | KAITO | sandbox | — | — | kaito | **No** | Landscape missing; CLOMonitor has 'kaito'. |
| extra.clomonitor_name | KServe | incubating | — | — | kserve | **No** | Landscape missing; CLOMonitor has 'kserve'. |
| extra.clomonitor_name | Podman Container Tools | sandbox | podman | — | podman-container-tools | **No** | Landscape ('podman') ≠ CLOMonitor ('podman-container-tool… |
| extra.clomonitor_name | Runme Notebooks | sandbox | runme | — | runme-notebooks | **No** | Landscape ('runme') ≠ CLOMonitor ('runme-notebooks'). |
| extra.dev_stats_url | OpenEverest | sandbox | — | — | https://openeverest.devstats.cncf.io/ | **No** | Landscape missing; CLOMonitor has 'https://openeverest.de… |
| extra.lfx_slug | Prometheus | graduated | prometheus_del | prometheus | — | — | Landscape ('prometheus_del') ≠ PCC ('prometheus'). |
| project (maturity) | Service Mesh Performance | archived | archived | sandbox | — | — | Landscape ('archived') ≠ PCC ('sandbox'). |
| repo_url | Drasi | sandbox | http://github.com/drasi-project/drasi-platform | https://github.com/drasi-project | https://github.com/drasi-project/drasi-platform | **No** | Landscape ('http://github.com/drasi-project/drasi-platfor… |

## No datasource match

These are in-scope landscape projects that could not be matched to PCC or CLOMonitor; they are usually candidates for upstream/source alignment PRs.

| Project | Maturity | Path |
|---------|----------|------|
| Service Mesh Interface (SMI) | archived | Orchestration & Management / Service Mesh |
| Volcano-Kthena | incubating | Inference / Framework |
| Cedar | sandbox | Provisioning / Security & Compliance |
| Higress | sandbox | Orchestration & Management / API Gateway |
| Monocle | sandbox | Observability and Analysis / Observability |