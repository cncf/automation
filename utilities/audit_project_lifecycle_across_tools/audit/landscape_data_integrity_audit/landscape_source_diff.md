# Landscape vs datasources diff

**Canonical:** `datasources/pcc_projects.yaml` and `datasources/clomonitor.yaml`. 
When those two disagree, that is called out. **`landscape.yml` should be updated** to match the agreed sources (or you must reconcile PCC vs CLOMonitor first).

## Summary

- **CNCF landscape items in scope:** 256
- **With at least one drift / conflict row:** 2
- **Findings where Landscape and CLOMonitor disagree:** 0
- **No PCC and no CLOMonitor match:** 2

## Differences (sorted by field)

Each row is one detected mismatch. Sorted by `Field`, then `Project`.

| Field | Project | Maturity | Landscape | PCC | CLOMonitor | Landscape≈CLO? | Note |
|---|---|---|---|---|---|---|---|
| project (maturity) | Service Mesh Performance | archived | archived | sandbox | — | — | Landscape ('archived') ≠ PCC ('sandbox'). |
| repo_url | container2wasm | sandbox | https://github.com/container2wasm/container2wasm | https://github.com/ktock/container2wasm | https://github.com/container2wasm/container2wasm | Yes | PCC ('https://github.com/ktock/container2wasm') and CLOMo… |

## No datasource match

These are in-scope landscape projects that could not be matched to PCC or CLOMonitor; they are usually candidates for upstream/source alignment PRs.

| Project | Maturity | Path |
|---------|----------|------|
| Service Mesh Interface (SMI) | archived | Orchestration & Management / Service Mesh |
| Monocle | sandbox | Observability and Analysis / Observability |