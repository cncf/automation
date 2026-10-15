#!/usr/bin/env python3
"""Generate the GitHub GPU Runners Grafana dashboard ConfigMap.

Covers every scale set in the "GPUs" runner group. runnerGroup is not a metric label, so GPU_SETS
below must be kept in sync with the AutoscalingRunnerSets that set runnerGroup: "GPUs".

Usage: python3 gen_gpu_dashboard.py > dashboards/github-gpu-runners.yaml
       (also copy to ci/cluster/cks-gha-us-west-01A/manifests/dashboards/)
"""
import json
import re

DS = {"type": "prometheus", "uid": "prometheus"}
GPU_SETS = [
    "oracle-vm-gpu-a10-1",
    "oracle-vm-gpu-a10-2",
    "cncf-c-ubuntu-24.04-16-64-x86-1xrtxpro6000",
    "cncf-c-ubuntu-24.04-16-64-x86-2xrtxpro6000",
]
# scale set name prefix -> provider
PROVIDERS = {"oracle-": "Oracle (OCI)", "cncf-c-": "CoreWeave"}
GPU_SET_RE = "(" + "|".join(re.escape(s) for s in GPU_SETS) + ")"
GPU = GPU_SET_RE + ".*"
# listener pods are named <scaleset>-<hash>-listener; strip to scale set
POD_TO_SET = GPU_SET_RE + r"-[a-z0-9]+-listener$"

ids = iter(range(1, 1000))


def target(expr, legend="__auto", instant=False, fmt=None):
    t = {"datasource": DS, "editorMode": "code", "expr": expr, "legendFormat": legend,
         "range": not instant, "instant": instant, "refId": "A"}
    if fmt:
        t["format"] = fmt
    return t


def panel(ptype, title, targets, x, y, w, h, desc="", unit=None, thresholds=None, extra=None, defaults=None):
    for i, t in enumerate(targets):
        t["refId"] = chr(ord("A") + i)
    d = {"color": {"mode": "palette-classic"}, "mappings": [], "noValue": "0"}
    if unit:
        d["unit"] = unit
    if thresholds:
        d["color"] = {"mode": "thresholds"}
        d["thresholds"] = {"mode": "absolute", "steps": thresholds}
    if defaults:
        d.update(defaults)
    p = {"id": next(ids), "type": ptype, "title": title, "description": desc, "datasource": DS,
         "gridPos": {"x": x, "y": y, "w": w, "h": h}, "targets": targets,
         "fieldConfig": {"defaults": d, "overrides": []}, "options": {}}
    if extra:
        p.update(extra)
    return p


def row(title, y, collapsed=False):
    return {"id": next(ids), "type": "row", "title": title, "collapsed": collapsed,
            "gridPos": {"x": 0, "y": y, "w": 24, "h": 1}, "panels": []}


def stat(title, expr, x, y, w=4, h=4, unit=None, thresholds=None, desc="", legend="__auto", instant=False):
    return panel("stat", title, [target(expr, legend, instant=instant)], x, y, w, h, desc, unit, thresholds,
                 extra={"options": {"reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
                                    "colorMode": "value", "graphMode": "area", "textMode": "auto"}})


def ts(title, targets, x, y, w=12, h=8, unit=None, desc="", stack=False):
    return panel("timeseries", title, targets, x, y, w, h, desc, unit,
                 defaults={"custom": {"fillOpacity": 15, "lineWidth": 1, "stacking": {"mode": "normal" if stack else "none"}}},
                 extra={"options": {"legend": {"displayMode": "table", "placement": "right", "calcs": ["lastNotNull", "max"]},
                                    "tooltip": {"mode": "multi", "sort": "desc"}}})


def heatmap(title, expr, x, y, w=12, h=8, desc=""):
    return panel("heatmap", title, [target(expr, "{{le}}", fmt="heatmap")], x, y, w, h, desc,
                 extra={"options": {"calculate": False, "yAxis": {"unit": "s"},
                                    "color": {"mode": "scheme", "scheme": "Oranges", "steps": 64},
                                    "cellGap": 1, "legend": {"show": True}}})


def table(title, expr, x, y, w=12, h=8, desc="", value_name="Jobs"):
    return panel("table", title, [target(expr, instant=True, fmt="table")], x, y, w, h, desc,
                 extra={"options": {"sortBy": [{"displayName": value_name, "desc": True}]},
                        "transformations": [
                            {"id": "organize", "options": {"excludeByName": {"Time": True},
                                                            "renameByName": {"Value": value_name}}}]})


green_red = [{"color": "green"}, {"color": "red", "value": 1}]
util_steps = [{"color": "green"}, {"color": "orange", "value": 70}, {"color": "red", "value": 90}]

by_set = lambda m: f'sum by (runner) (label_replace({m}{{pod=~"{GPU}"}}, "runner", "$1", "pod", "{POD_TO_SET}"))'
def by_prov(m):
    expr = f'{m}{{pod=~"{GPU}"}}'
    for prefix, name in PROVIDERS.items():
        expr = f'label_replace({expr}, "provider", "{name}", "pod", "^{prefix}.*")'
    return f'sum by (provider) ({expr})'
tot = lambda m: f'sum({m}{{pod=~"{GPU}"}})'
sel = lambda m: f'{m}{{pod=~"{GPU}", pod=~"$Scaleset.*"}}'

panels = []
y = 0

# ---------------- Overview ----------------
panels.append(row("GPU Fleet Overview", y)); y += 1
panels += [
    stat("Running GPU Jobs", tot("gha_running_jobs"), 0, y, desc="Jobs currently executing on any GPU runner."),
    stat("Queued GPU Jobs", f'clamp_min({tot("gha_assigned_jobs")} - {tot("gha_running_jobs")}, 0)', 4, y,
         thresholds=green_red, desc="Jobs assigned to a GPU scale set but not yet running: waiting for a VM/pod."),
    stat("Busy Runners", tot("gha_busy_runners"), 8, y),
    stat("Idle Runners", tot("gha_idle_runners"), 12, y),
    stat("Fleet Capacity Used", f'{tot("gha_desired_runners")} / {tot("gha_max_runners")} * 100', 16, y,
         unit="percent", thresholds=util_steps, desc="Desired runners / max runners across all GPU scale sets."),
    stat("GPU Scale Sets", f'count(count by (pod) (gha_max_runners{{pod=~"{GPU}"}}))', 20, y,
         desc=f"GPU scale sets with a running listener (expected: {len(GPU_SETS)} across all clusters)."),
]
y += 4
panels += [
    ts("Capacity by Scale Set (desired / max)",
       [target(by_set("gha_desired_runners"), "{{runner}} desired"),
        target(by_set("gha_max_runners"), "{{runner}} max")], 0, y, 12, 8,
       desc="Hitting max means jobs queue. For scarce GPU shapes consider raising maxRunners or adding a provider."),
    ts("Running Jobs by Provider", [target(by_prov("gha_running_jobs"), "{{provider}}")], 12, y, 12, 8, stack=True),
]
y += 8

# ---------------- Per-scale-set ----------------
panels.append(row("Scale Set: $Scaleset", y)); y += 1
panels += [
    stat("Desired", f'sum({sel("gha_desired_runners")})', 0, y, 3),
    stat("Max", f'sum({sel("gha_max_runners")})', 3, y, 3),
    stat("Registered", f'sum({sel("gha_registered_runners")})', 6, y, 3),
    stat("Busy", f'sum({sel("gha_busy_runners")})', 9, y, 3),
    stat("Assigned Jobs", f'sum({sel("gha_assigned_jobs")})', 12, y, 3),
    stat("Pending Runners", f'sum(gha_controller_pending_ephemeral_runners{{name=~"$Scaleset"}})', 15, y, 3,
         thresholds=[{"color": "green"}, {"color": "orange", "value": 1}, {"color": "red", "value": 3}],
         desc="EphemeralRunners the controller has created but which have not registered yet (VM still booting, or capacity unavailable)."),
    stat("Failed Runners", f'sum(gha_controller_failed_ephemeral_runners{{name=~"$Scaleset"}})', 18, y, 3, thresholds=green_red),
    stat("Running Runners", f'max(gha_controller_running_ephemeral_runners{{name=~"$Scaleset"}})', 21, y, 3),
]
y += 4
panels += [
    heatmap("Startup Duration (queue -> runner picks up job)",
            f'sum by (le) (increase({sel("gha_job_startup_duration_seconds_bucket")}[$__rate_interval]))', 0, y,
            desc="Time from job assignment to a GPU runner starting it. Dominated by VM boot and GPU capacity availability."),
    heatmap("Job Execution Duration",
            f'sum by (le) (increase({sel("gha_job_execution_duration_seconds_bucket")}[$__rate_interval]))', 12, y),
]
y += 8
panels += [
    ts("Startup Duration Percentiles",
       [target(f'histogram_quantile(0.50, sum by (le) (rate({sel("gha_job_startup_duration_seconds_bucket")}[$__rate_interval])))', "p50"),
        target(f'histogram_quantile(0.95, sum by (le) (rate({sel("gha_job_startup_duration_seconds_bucket")}[$__rate_interval])))', "p95"),
        target(f'histogram_quantile(0.99, sum by (le) (rate({sel("gha_job_startup_duration_seconds_bucket")}[$__rate_interval])))', "p99")],
       0, y, 12, 8, unit="s"),
    ts("Job Throughput",
       [target(f'sum(rate({sel("gha_started_jobs_total")}[$__rate_interval])) * 3600', "started / h"),
        target(f'sum by (job_result) (rate({sel("gha_completed_jobs_total")}[$__rate_interval])) * 3600', "completed: {{job_result}}")],
       12, y, 12, 8, unit="short"),
]
y += 8

# ---------------- Consumers ----------------
panels.append(row("Who Is Using GPUs ($__range)", y)); y += 1
panels += [
    table("Jobs by Repository",
          f'sort_desc(sum by (repository) (increase(gha_completed_jobs_total{{pod=~"{GPU}", pod=~"$Scaleset.*"}}[$__range])) > 0)',
          0, y, 8, 10),
    table("Jobs by Workflow Job Name",
          f'sort_desc(sum by (repository, job_name) (increase(gha_completed_jobs_total{{pod=~"{GPU}", pod=~"$Scaleset.*"}}[$__range])) > 0)',
          8, y, 8, 10),
    table("Runner Hours by Repository",
          f'sort_desc(sum by (repository) (increase(gha_job_execution_duration_seconds_sum{{pod=~"{GPU}", pod=~"$Scaleset.*"}}[$__range])) / 3600 > 0)',
          16, y, 8, 10, value_name="Hours",
          desc="Wall-clock runner hours, a proxy for GPU spend. Multiply by GPUs-per-runner for GPU-hours (A10.2 and 2xrtxpro6000 = 2)."),
]
y += 10
panels += [
    ts("Completed Jobs by Result", [target(f'sum by (job_result) (increase({sel("gha_completed_jobs_total")}[$__rate_interval]))', "{{job_result}}")],
       0, y, 12, 7, stack=True),
    ts("Jobs by Trigger Event", [target(f'sum by (event_name) (increase({sel("gha_started_jobs_total")}[$__rate_interval]))', "{{event_name}}")],
       12, y, 12, 7, stack=True),
]
y += 7

# ---------------- Launcher / controller health ----------------
panels.append(row("Launcher & Controller Health", y)); y += 1
panels += [
    ts("Launcher Pod Terminations (arc-systems)",
       [target(f'sum by (reason) (increase(kube_pod_container_status_terminated_reason{{namespace="arc-systems", pod=~"{GPU}", reason!="Completed"}}[$__rate_interval]))', "{{reason}}")],
       0, y, 12, 7, desc="Non-Completed exits of GPU launcher pods (the cloudrunner container that creates the VM). Error here usually means the provider could not allocate a GPU shape."),
    ts("Launcher Pod Restarts",
       [target(f'sum by (pod) (increase(kube_pod_container_status_restarts_total{{namespace="arc-systems", pod=~"{GPU}"}}[$__rate_interval])) > 0', "{{pod}}")],
       12, y, 12, 7),
]
y += 7
panels += [
    ts("Listener Scrape Health", [target(f'up{{pod=~"{GPU}"}}', "{{pod}}")], 0, y, 12, 6,
       desc="1 = listener metrics endpoint scraped successfully."),
    ts("Controller Reconcile Errors", [target('sum by (controller) (rate(controller_runtime_reconcile_errors_total{namespace="arc-systems"}[$__rate_interval]))', "{{controller}}")],
       12, y, 12, 6),
]
y += 6

# ---------------- GPU telemetry (DCGM) ----------------
panels.append(row("GPU Telemetry (DCGM exporter; CoreWeave nodes)", y, collapsed=True))
dcgm = panels[-1]["panels"]
dy = y + 1
dcgm.append(ts("GPU Utilization", [target('avg by (Hostname, gpu) (DCGM_FI_DEV_GPU_UTIL)', "{{Hostname}} gpu{{gpu}}")], 0, dy, 12, 8, unit="percent",
               desc="Requires a DCGM exporter on GPU nodes (CoreWeave runners are in-cluster pods). OCI VM runners run outside the cluster and show no data here."))
dcgm.append(ts("GPU Memory Used", [target('avg by (Hostname, gpu) (DCGM_FI_DEV_FB_USED)', "{{Hostname}} gpu{{gpu}}")], 12, dy, 12, 8, unit="decmbytes"))
dy += 8
dcgm.append(ts("GPU Temperature", [target('avg by (Hostname, gpu) (DCGM_FI_DEV_GPU_TEMP)', "{{Hostname}} gpu{{gpu}}")], 0, dy, 12, 7, unit="celsius"))
dcgm.append(ts("GPU Power Draw", [target('avg by (Hostname, gpu) (DCGM_FI_DEV_POWER_USAGE)', "{{Hostname}} gpu{{gpu}}")], 12, dy, 12, 7, unit="watt"))
for p in dcgm:
    p["fieldConfig"]["defaults"]["noValue"] = "no DCGM exporter"

dashboard = {
    "title": "GitHub GPU Runners",
    "uid": "github-gpu-runners",
    "description": "Every scale set in the GPUs runner group: Oracle VM (oracle-vm-gpu-*) and CoreWeave container runners "
                   "(cncf-c-ubuntu-*rtxpro6000). Built from ARC metrics; the DCGM row needs a GPU exporter on the nodes.",
    "tags": ["github", "arc", "gpu", "runners"],
    "editable": True, "graphTooltip": 1, "liveNow": False, "schemaVersion": 41, "version": 1,
    "time": {"from": "now-24h", "to": "now"},
    "timepicker": {"refresh_intervals": ["30s", "1m", "5m", "15m", "30m", "1h"]},
    "refresh": "1m", "timezone": "", "links": [
        {"title": "GitHub ARC Monitoring", "type": "link", "url": "/d/af21e938-2151-4bf2-b798-8cf9232f947a", "icon": "dashboard"}],
    "annotations": {"list": []},
    "templating": {"list": [{
        "name": "Scaleset", "label": "GPU Scale Set", "type": "query", "datasource": DS,
        "description": "Scale sets in the GPUs runner group (Oracle VM and CoreWeave)",
        "definition": "label_values(gha_max_runners, name)",
        "query": {"qryType": 1, "query": "label_values(gha_max_runners, name)", "refId": "PrometheusVariableQueryEditor-VariableQuery"},
        "regex": "/^" + GPU_SET_RE + "$/", "includeAll": True, "multi": True, "refresh": 2,
        "current": {"text": "All", "value": "$__all"}, "options": [],
    }]},
    "panels": panels,
}

cm = {
    "apiVersion": "v1", "kind": "ConfigMap",
    "metadata": {"name": "github-gpu-runners", "namespace": "monitoring", "labels": {"grafana_dashboard": "1"}},
    "data": {"github-gpu-runners.json": json.dumps(dashboard, indent=2)},
}

import yaml  # noqa: E402

class Dumper(yaml.SafeDumper):
    pass

def str_presenter(dumper, data):
    if "\n" in data:
        return dumper.represent_scalar("tag:yaml.org,2002:str", data, style="|")
    return dumper.represent_scalar("tag:yaml.org,2002:str", data)

Dumper.add_representer(str, str_presenter)
print("# Generated by hack/gen_gpu_dashboard.py; edit the generator, not this file.")
print(yaml.dump(cm, Dumper=Dumper, sort_keys=False, width=1000), end="")
