#!/usr/bin/env python3
"""
Compare vendored landscape.yml to PCC and CLOMonitor snapshots in datasources/.

Canonical truth: PCC + CLOMonitor (when both agree). When they disagree, the
report calls that out alongside landscape drift.
"""

from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import sys
import datetime
import re
import urllib.request
from typing import Any, Dict, List, Optional, Tuple
from urllib.parse import urlparse

try:
    import yaml  # type: ignore
except Exception:
    print("Missing dependency: PyYAML. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(2)

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
AUDIT_ROOT = os.path.dirname(SCRIPT_DIR)
LANDSCAPE_PATH = os.path.join(AUDIT_ROOT, "datasources", "landscape.yml")
PCC_PATH = os.path.join(AUDIT_ROOT, "datasources", "pcc_projects.yaml")
CLOMONITOR_PATH = os.path.join(AUDIT_ROOT, "datasources", "clomonitor.yaml")
OUTPUT_DIR = os.path.join(AUDIT_ROOT, "audit", "landscape_data_integrity_audit")

_ldi_path = os.path.join(SCRIPT_DIR, "landscape_data_integrity_audit.py")
_spec = importlib.util.spec_from_file_location("_ldi", _ldi_path)
if _spec is None or _spec.loader is None:
    print(
        f"Failed to load helper module from: {_ldi_path}",
        file=sys.stderr,
    )
    sys.exit(2)
_ldi = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_ldi)

list_landscape_items = getattr(_ldi, "list_landscape_items", None) or getattr(
    _ldi, "iter_landscape_items"
)
get_extra = _ldi.get_extra
effective_project = _ldi.effective_project
present = _ldi.present
SCOPE_MATURITIES = _ldi.SCOPE_MATURITIES


def normalize_slug(s: Any) -> str:
    return str(s or "").strip().lower().replace("_", "-")


def normalize_key(name: str) -> str:
    return " ".join(str(name or "").lower().split())


def normalize_url(u: str) -> str:
    u = str(u or "").strip().rstrip("/")
    if u.endswith(".git"):
        u = u[:-4]
    if u.startswith("https://github.com/"):
        rest = u[19:].lower().rstrip("/")
        return f"https://github.com/{rest}"
    return u.lower().rstrip("/")


def normalize_repo_identity(u: str) -> str:
    """
    Normalize repository identity for diffing.

    For GitHub URLs, treat all repos under the same org/owner as equivalent to
    reduce noise (e.g. org root URL vs specific repo URL).
    """
    n = normalize_url(u)
    gh_prefix = "https://github.com/"
    if n.startswith(gh_prefix):
        tail = n[len(gh_prefix) :]
        owner = tail.split("/", 1)[0].strip()
        if owner:
            return f"github-org:{owner}"
    return n


def normalize_date(s: Any) -> str:
    if s is None:
        return ""
    t = str(s).strip().strip("'\"")
    if "T" in t:
        t = t.split("T", 1)[0]
    return t[:10] if len(t) >= 10 else t


def parse_iso_date(s: Any) -> Optional[datetime.date]:
    v = normalize_date(s)
    if not v:
        return None
    try:
        return datetime.date.fromisoformat(v)
    except Exception:
        return None


def dates_within_tolerance(a: Any, b: Any, tolerance_days: int) -> bool:
    da = parse_iso_date(a)
    db = parse_iso_date(b)
    if da is None or db is None:
        return False
    return abs((da - db).days) <= tolerance_days


def is_github_url(u: Any) -> bool:
    n = normalize_url(str(u or ""))
    if n.startswith("http://github.com/"):
        n = "https://github.com/" + n[len("http://github.com/") :]
    return n.startswith("https://github.com/")


def canonical_token(s: str) -> str:
    return re.sub(r"[^a-z0-9]+", "", str(s or "").lower())


def slug_aliases(s: Any) -> List[str]:
    v = str(s or "").strip().lower()
    if not v:
        return []
    out = {v}
    out.add(v.replace("_", "-"))
    out.add(v.replace("-", " "))
    out.add(v.replace("_", " "))
    out.add(v.replace(" ", "-"))
    out.add(v.replace(" ", ""))
    out.add(canonical_token(v))
    return [x for x in out if x]


def slug_equivalent(a: Any, b: Any) -> bool:
    aa = set(slug_aliases(a))
    bb = set(slug_aliases(b))
    if not aa or not bb:
        return False
    return bool(aa.intersection(bb))


def devstats_project_token(u: Any) -> str:
    n = normalize_url(str(u or ""))
    if not n:
        return ""
    m = re.match(r"^https?://([^./]+)\.(?:devstats|teststats)\.cncf\.io", n)
    return canonical_token(m.group(1)) if m else ""


# ---------------------------------------------------------------------------
# GitHub redirect resolution helpers
# ---------------------------------------------------------------------------

_redirect_cache: Dict[str, str] = {}
_redirect_curl_cache: Dict[str, Optional[str]] = {}


def github_owner_from_url(url: str) -> Optional[str]:
    """Return GitHub org/owner (first path segment) or None if not a GitHub URL."""
    if not url:
        return None
    candidate = url if "://" in url else f"https://{url}"
    parsed = urlparse(candidate)
    host = (parsed.netloc or "").lower()
    if host not in {"github.com", "www.github.com"}:
        return None
    path = (parsed.path or "").strip("/")
    if not path:
        return None
    owner = path.split("/", 1)[0].strip().lower()
    return owner or None


def curl_check_url(url: str, timeout: int = 20) -> Optional[str]:
    """
    Follow redirects with curl and return the normalized effective URL.

    Returns None when curl is unavailable or the request does not succeed.
    """
    if url in _redirect_curl_cache:
        return _redirect_curl_cache[url]

    cmd = [
        "curl",
        "-L",
        "-sS",
        "-o",
        "/dev/null",
        "-w",
        "%{http_code} %{url_effective}",
        "--max-time",
        str(timeout),
        url,
    ]
    final: Optional[str] = None
    try:
        cp = subprocess.run(
            cmd,
            check=False,
            capture_output=True,
            text=True,
        )
        out = (cp.stdout or "").strip()
        if cp.returncode == 0 and out:
            parts = out.split(" ", 1)
            code_raw = parts[0] if parts else "000"
            effective = parts[1].strip() if len(parts) > 1 else url
            try:
                code = int(code_raw)
            except ValueError:
                code = 0
            if 200 <= code < 400:
                final = normalize_url(effective or url)
    except Exception:
        final = None

    _redirect_curl_cache[url] = final
    return final


def resolve_final_url(url: str, timeout: int = 5) -> str:
    """Follow HTTP redirects and return the final (normalized) destination URL.

    Returns the original URL unchanged if the network request fails for any
    reason, so callers degrade gracefully on connectivity issues.
    """
    if url in _redirect_cache:
        return _redirect_cache[url]
    try:
        req = urllib.request.Request(
            url,
            method="HEAD",
            headers={"User-Agent": "cncf-audit-redirect-check/1.0"},
        )
        with urllib.request.urlopen(req, timeout=timeout) as resp:  # noqa: S310
            final = normalize_url(resp.geturl())
    except Exception:
        # Fallback to GET for servers that reject HEAD (e.g. some GitHub pages).
        try:
            req_get = urllib.request.Request(
                url,
                headers={"User-Agent": "cncf-audit-redirect-check/1.0"},
            )
            with urllib.request.urlopen(req_get, timeout=timeout) as resp:  # noqa: S310
                final = normalize_url(resp.geturl())
        except Exception:
            final = normalize_url(url)
    _redirect_cache[url] = final
    return final


def urls_resolve_to_same_destination(urls: List[str]) -> bool:
    """Return True only when every non-empty URL resolves to the same final URL.

    Only performs the check when all provided URLs are GitHub URLs — that is
    the primary source of redirect noise (org renames, repo transfers). For
    non-GitHub URLs this returns False so the existing finding is preserved.
    """
    distinct = [u for u in urls if u]
    if len(distinct) < 2:
        return False
    if not all(is_github_url(u) for u in distinct):
        return False
    resolved = {resolve_final_url(u) for u in distinct}
    return len(resolved) == 1


def _curl_github_url(url: str, timeout: int = 20) -> str:
    """Normalize a GitHub URL for curl (http→https) and return the request URL."""
    if url.startswith("http://github.com"):
        return "https://github.com" + url[18:]
    return url


def urls_resolve_to_same_github_org(urls: List[str], timeout: int = 20) -> bool:
    """
    Return True when every non-empty GitHub URL redirects to the same GitHub org.

    Repo path differences are ignored; only the effective owner after redirects
    must match. Returns False if any URL fails to resolve or is not GitHub.
    """
    distinct = [u for u in urls if u]
    if len(distinct) < 2:
        return False
    if not all(is_github_url(u) for u in distinct):
        return False

    owners: List[str] = []
    for raw in distinct:
        final = curl_check_url(_curl_github_url(raw), timeout=timeout)
        if not final:
            return False
        owner = github_owner_from_url(final)
        if not owner:
            return False
        owners.append(owner)
    return len(set(owners)) == 1


def suppress_repo_mismatch_for_non_github_pcc(
    pcc_repo: str,
    clo_name: str,
    land_devstats: str,
    clo_devstats: str,
) -> bool:
    """
    Suppress repo_url anomaly when PCC provides a website (non-GitHub) and
    CLOMonitor identity aligns with DevStats identity.
    """
    if not pcc_repo or is_github_url(pcc_repo):
        return False
    ctoken = canonical_token(clo_name)
    if not ctoken:
        return False

    lt = devstats_project_token(land_devstats)
    ct = devstats_project_token(clo_devstats)
    if not lt and not ct:
        return False

    # Require CLOMonitor name to align with at least one DevStats project token.
    return (lt and lt == ctoken) or (ct and ct == ctoken)


def pcc_maturity_from_row(tier: str, row: Dict[str, Any]) -> str:
    t = (tier or "").strip()
    if t == "Archived":
        return "archived"
    m = {
        "Graduated": "graduated",
        "Incubating": "incubating",
        "Sandbox": "sandbox",
    }.get(t, "")
    if not m and row.get("status") == "Archived":
        return "archived"
    return m


def clo_maturity(raw: Any) -> str:
    s = str(raw or "").strip().lower()
    if s in ("graduated", "incubating", "sandbox", "archived"):
        return s
    return s


def clo_primary_repo(entry: Dict[str, Any]) -> str:
    repos = entry.get("repositories") or []
    if not isinstance(repos, list) or not repos:
        return ""
    r0 = repos[0]
    if isinstance(r0, dict):
        return str(r0.get("url") or "").strip()
    return ""


def load_pcc_indexes(path: str) -> Tuple[Dict[str, Dict[str, Any]], Dict[str, Dict[str, Any]]]:
    """by_slug, by_normalized_project_name"""
    by_slug: Dict[str, Dict[str, Any]] = {}
    by_name: Dict[str, Dict[str, Any]] = {}
    if not os.path.isfile(path):
        return by_slug, by_name
    with open(path, "r", encoding="utf-8") as f:
        doc = yaml.safe_load(f.read()) or {}
    categories = doc.get("categories") or {}
    for tier in ("Graduated", "Incubating", "Sandbox"):
        for row in categories.get(tier) or []:
            if not isinstance(row, dict):
                continue
            slug = normalize_slug(row.get("slug"))
            enriched = {**row, "_pcc_tier": tier}
            if slug:
                by_slug[slug] = enriched
            nk = normalize_key(row.get("name") or "")
            if nk and nk not in by_name:
                by_name[nk] = enriched
    for row in doc.get("archived_projects") or []:
        if not isinstance(row, dict):
            continue
        slug = normalize_slug(row.get("slug"))
        enriched = {**row, "_pcc_tier": "Archived"}
        if slug:
            by_slug[slug] = enriched
        nk = normalize_key(row.get("name") or "")
        if nk and nk not in by_name:
            by_name[nk] = enriched
    return by_slug, by_name


def load_clomonitor_indexes(path: str) -> Dict[str, Dict[str, Any]]:
    """by project `name` (slug)"""
    by_name: Dict[str, Dict[str, Any]] = {}
    if not os.path.isfile(path):
        return by_name
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f.read())
    if not isinstance(data, list):
        return by_name
    for entry in data:
        if not isinstance(entry, dict):
            continue
        k = normalize_slug(entry.get("name"))
        if k:
            by_name[k] = entry
    return by_name


def resolve_pcc_clo(
    item: Dict[str, Any],
    pcc_by_slug: Dict[str, Dict[str, Any]],
    pcc_by_name: Dict[str, Dict[str, Any]],
    clo_by_name: Dict[str, Dict[str, Any]],
) -> Tuple[Optional[Dict[str, Any]], Optional[Dict[str, Any]], str]:
    """Returns (pcc_row, clo_entry, match_note)."""
    extra = get_extra(item)
    name = str(item.get("name") or "").strip()
    pcc: Optional[Dict[str, Any]] = None
    clo: Optional[Dict[str, Any]] = None
    notes: List[str] = []

    ck = normalize_slug(extra.get("clomonitor_name"))
    if ck and ck in clo_by_name:
        clo = clo_by_name[ck]
        notes.append("clomonitor_name")

    sk = normalize_slug(extra.get("lfx_slug"))
    if sk and sk in pcc_by_slug:
        pcc = pcc_by_slug[sk]
        notes.append("lfx_slug")

    if clo and not pcc:
        nk = normalize_slug(clo.get("name"))
        if nk in pcc_by_slug:
            pcc = pcc_by_slug[nk]
            notes.append("pcc_via_clo_name")

    if not pcc and not clo:
        nk = normalize_key(name)
        if nk in pcc_by_name:
            pcc = pcc_by_name[nk]
            notes.append("pcc_name")
        guess = normalize_slug(name.replace(" ", "-").replace("(", "").replace(")", ""))
        if guess in clo_by_name:
            clo = clo_by_name[guess]
            notes.append("clo_slug_guess")

    if pcc and not clo:
        nk = normalize_slug(pcc.get("slug"))
        if nk in clo_by_name:
            clo = clo_by_name[nk]
            notes.append("clo_via_pcc_slug")

    return pcc, clo, "+".join(sorted(set(notes))) if notes else "none"


def compare_field(
    field_label: str,
    land_raw: Any,
    pcc_raw: Any,
    clo_raw: Any,
    normalize_fn,
) -> Optional[Dict[str, Any]]:
    """
    Emit a finding when landscape is out of sync with either source, or PCC and CLOMonitor disagree.
    """

    def norm(x: Any) -> str:
        if not present(x):
            return ""
        return normalize_fn(str(x).strip())

    lv, pv, cv = norm(land_raw), norm(pcc_raw), norm(clo_raw)
    has_l, has_p, has_c = bool(lv), bool(pv), bool(cv)

    if not has_p and not has_c:
        return None

    sources_agree = True
    if has_p and has_c:
        sources_agree = pv == cv

    land_ok_p = not has_p or (has_l and lv == pv)
    land_ok_c = not has_c or (has_l and lv == cv)

    if sources_agree and land_ok_p and land_ok_c:
        return None

    msgs: List[str] = []
    if has_p and has_c and not sources_agree:
        msgs.append(f"PCC ({pcc_raw!r}) and CLOMonitor ({clo_raw!r}) disagree.")
    if has_p and not land_ok_p:
        if has_l:
            msgs.append(f"Landscape ({land_raw!r}) ≠ PCC ({pcc_raw!r}).")
        else:
            msgs.append(f"Landscape missing; PCC has {pcc_raw!r}.")
    if has_c and not land_ok_c:
        if has_l:
            msgs.append(f"Landscape ({land_raw!r}) ≠ CLOMonitor ({clo_raw!r}).")
        else:
            msgs.append(f"Landscape missing; CLOMonitor has {clo_raw!r}.")

    landscape_clomonitor_agree: Optional[bool] = None
    if has_c:
        landscape_clomonitor_agree = bool(has_l and lv == cv)

    return {
        "field": field_label,
        "landscape": land_raw if present(land_raw) else None,
        "pcc": pcc_raw if present(pcc_raw) else None,
        "clomonitor": clo_raw if present(clo_raw) else None,
        "pcc_clomonitor_agree": sources_agree if (has_p and has_c) else None,
        "landscape_clomonitor_agree": landscape_clomonitor_agree,
        "message": " ".join(msgs),
    }


def compare_slug_field(
    field_label: str,
    land_raw: Any,
    pcc_raw: Any,
    clo_raw: Any,
) -> Optional[Dict[str, Any]]:
    """
    Compare slug-like identifiers with alias-equivalent matching to reduce noise.
    """
    has_l = present(land_raw)
    has_p = present(pcc_raw)
    has_c = present(clo_raw)
    if not has_p and not has_c:
        return None

    pcc_clo_agree = True
    if has_p and has_c:
        pcc_clo_agree = slug_equivalent(pcc_raw, clo_raw)

    land_ok_p = (not has_p) or (has_l and slug_equivalent(land_raw, pcc_raw))
    land_ok_c = (not has_c) or (has_l and slug_equivalent(land_raw, clo_raw))

    if pcc_clo_agree and land_ok_p and land_ok_c:
        return None

    msgs: List[str] = []
    if has_p and has_c and not pcc_clo_agree:
        msgs.append(f"PCC ({pcc_raw!r}) and CLOMonitor ({clo_raw!r}) disagree.")
    if has_p and not land_ok_p:
        msgs.append(
            f"Landscape ({land_raw!r}) ≠ PCC ({pcc_raw!r})."
            if has_l
            else f"Landscape missing; PCC has {pcc_raw!r}."
        )
    if has_c and not land_ok_c:
        msgs.append(
            f"Landscape ({land_raw!r}) ≠ CLOMonitor ({clo_raw!r})."
            if has_l
            else f"Landscape missing; CLOMonitor has {clo_raw!r}."
        )

    landscape_clomonitor_agree: Optional[bool] = None
    if has_c:
        landscape_clomonitor_agree = bool(has_l and slug_equivalent(land_raw, clo_raw))

    return {
        "field": field_label,
        "landscape": land_raw if has_l else None,
        "pcc": pcc_raw if has_p else None,
        "clomonitor": clo_raw if has_c else None,
        "pcc_clomonitor_agree": pcc_clo_agree if (has_p and has_c) else None,
        "landscape_clomonitor_agree": landscape_clomonitor_agree,
        "message": " ".join(msgs),
    }


def build_report() -> Dict[str, Any]:
    with open(LANDSCAPE_PATH, "r", encoding="utf-8") as f:
        land_doc = yaml.safe_load(f.read()) or {}

    pcc_by_slug, pcc_by_name = load_pcc_indexes(PCC_PATH)
    clo_by_name = load_clomonitor_indexes(CLOMONITOR_PATH)

    projects_out: List[Dict[str, Any]] = []

    for cat_name, sub_name, item in list_landscape_items(land_doc):
        name = str(item.get("name") or "").strip()
        if not name:
            continue
        eff = effective_project(item)
        if eff not in SCOPE_MATURITIES:
            continue

        extra = get_extra(item)
        path = f"{cat_name} / {sub_name}" if cat_name or sub_name else ""
        pcc, clo, match_note = resolve_pcc_clo(item, pcc_by_slug, pcc_by_name, clo_by_name)

        land_repo = str(item.get("repo_url") or "").strip()
        land_slug = str(extra.get("lfx_slug") or "").strip()
        land_clomon = str(extra.get("clomonitor_name") or "").strip()
        land_dev = str(extra.get("dev_stats_url") or "").strip()
        land_acc = str(extra.get("accepted") or "").strip()

        pcc_repo = str(pcc.get("repository_url") or "").strip() if pcc else ""
        clo_repo = clo_primary_repo(clo) if clo else ""
        pcc_slug = str(pcc.get("slug") or "").strip() if pcc else ""
        clo_name = str(clo.get("name") or "").strip() if clo else ""
        clo_dev = str(clo.get("devstats_url") or "").strip() if clo else ""
        clo_acc = str(clo.get("accepted_at") or "").strip() if clo else ""

        pcc_mat = ""
        if pcc:
            pcc_mat = pcc_maturity_from_row(str(pcc.get("_pcc_tier") or ""), pcc)
        clo_mat = clo_maturity(clo.get("maturity")) if clo else ""

        findings: List[Dict[str, Any]] = []

        suppress_repo = suppress_repo_mismatch_for_non_github_pcc(
            pcc_repo=pcc_repo,
            clo_name=clo_name,
            land_devstats=land_dev,
            clo_devstats=clo_dev,
        )
        if not suppress_repo:
            repo_urls = [u for u in [land_repo, pcc_repo, clo_repo] if u]
            f1 = compare_field("repo_url", land_repo, pcc_repo, clo_repo, normalize_repo_identity)
            if f1 and not urls_resolve_to_same_destination(repo_urls) and not urls_resolve_to_same_github_org(
                repo_urls
            ):
                findings.append(f1)

        f2 = compare_slug_field(
            "extra.lfx_slug",
            land_slug,
            pcc_slug,
            None,  # CLOMonitor does not provide lfx_slug; PCC is the only authority
        )
        if f2:
            findings.append(f2)

        f3 = compare_field(
            "extra.clomonitor_name",
            land_clomon,
            None,
            clo_name,
            normalize_slug,
        )
        if f3:
            findings.append(f3)

        f4 = compare_field("extra.dev_stats_url", land_dev, None, clo_dev, normalize_url)
        if f4:
            findings.append(f4)

        # Accepted date can legitimately differ by publication lag; ignore <= 30 days.
        if not dates_within_tolerance(land_acc, clo_acc, tolerance_days=30):
            f5 = compare_field(
                "extra.accepted",
                land_acc,
                None,
                clo_acc,
                lambda x: normalize_date(x),
            )
            if f5:
                findings.append(f5)

        if pcc_mat or clo_mat:
            f6 = compare_field(
                "project (maturity)",
                eff,
                pcc_mat,
                clo_mat,
                normalize_slug,
            )
            if f6:
                findings.append(f6)

        projects_out.append(
            {
                "name": name,
                "path": path,
                "maturity": eff,
                "match_note": match_note,
                "matched_pcc": pcc is not None,
                "matched_clomonitor": clo is not None,
                "findings": findings,
            }
        )

    return {
        "source": "datasources vs landscape.yml",
        "projects": projects_out,
    }


def fmt_val(v: Any) -> str:
    if v is None:
        return "—"
    s = str(v).replace("|", "\\|")
    if len(s) > 60:
        return s[:57] + "…"
    return s


def render_markdown(data: Dict[str, Any]) -> str:
    lines: List[str] = []
    lines.append("# Landscape vs datasources diff")
    lines.append("")
    lines.append("**Canonical:** `datasources/pcc_projects.yaml` and `datasources/clomonitor.yaml`. ")
    lines.append("When those two disagree, that is called out. **`landscape.yml` should be updated** to match the agreed sources (or you must reconcile PCC vs CLOMonitor first).")
    lines.append("")

    projects = data["projects"]
    with_findings = [p for p in projects if p["findings"]]
    unmatched = [p for p in projects if not p["matched_pcc"] and not p["matched_clomonitor"]]
    landscape_clo_mismatch = [
        f
        for p in projects
        for f in p["findings"]
        if f.get("landscape_clomonitor_agree") is False
    ]

    lines.append("## Summary")
    lines.append("")
    lines.append(f"- **CNCF landscape items in scope:** {len(projects)}")
    lines.append(f"- **With at least one drift / conflict row:** {len(with_findings)}")
    lines.append(
        f"- **Findings where Landscape and CLOMonitor disagree:** {len(landscape_clo_mismatch)}"
    )
    lines.append(f"- **No PCC and no CLOMonitor match:** {len(unmatched)}")
    lines.append("")

    lines.append("## Differences (sorted by field)")
    lines.append("")
    lines.append(
        "Each row is one detected mismatch. Sorted by `Field`, then `Project`."
    )
    lines.append("")
    lines.append(
        "| Field | Project | Maturity | Landscape | PCC | CLOMonitor | Landscape≈CLO? | Note |"
    )
    lines.append(
        "|---|---|---|---|---|---|---|---|"
    )

    flat_rows: List[Tuple[str, str, str, Any, Any, Any, Any, str]] = []
    for p in with_findings:
        for f in p["findings"]:
            agree = f.get("landscape_clomonitor_agree")
            agree_s = "—" if agree is None else ("Yes" if agree else "**No**")
            flat_rows.append(
                (
                    f.get("field", ""),
                    p["name"],
                    p["maturity"],
                    f.get("landscape"),
                    f.get("pcc"),
                    f.get("clomonitor"),
                    agree_s,
                    f.get("message", ""),
                )
            )

    flat_rows.sort(key=lambda r: (str(r[0]).lower(), str(r[1]).lower()))
    for fld, name, mat, lv, pv, cv, land_clo_s, msg in flat_rows:
        lines.append(
            f"| {fmt_val(fld)} | {fmt_val(name)} | {fmt_val(mat)} | "
            f"{fmt_val(lv)} | {fmt_val(pv)} | {fmt_val(cv)} | {land_clo_s} | {fmt_val(msg)} |"
        )
    lines.append("")

    lines.append("## No datasource match")
    lines.append("")
    lines.append(
        "These are in-scope landscape projects that could not be matched to PCC or CLOMonitor; "
        "they are usually candidates for upstream/source alignment PRs."
    )
    lines.append("")
    if not unmatched:
        lines.append("_All in-scope items resolved to at least PCC or CLOMonitor._")
    else:
        lines.append("| Project | Maturity | Path |")
        lines.append("|---------|----------|------|")
        for p in sorted(unmatched, key=lambda x: (x["maturity"], x["name"].lower())):
            lines.append(f"| {p['name']} | {p['maturity']} | {p['path']} |")

    return "\n".join(lines)


def main() -> int:
    if not os.path.isfile(LANDSCAPE_PATH):
        print(f"Missing {LANDSCAPE_PATH}", file=sys.stderr)
        return 1

    data = build_report()
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    md_path = os.path.join(OUTPUT_DIR, "landscape_source_diff.md")
    json_path = os.path.join(OUTPUT_DIR, "landscape_source_diff.json")

    with open(md_path, "w", encoding="utf-8") as f:
        f.write(render_markdown(data))

    with open(json_path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)

    print(f"Wrote {md_path}")
    print(f"Wrote {json_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
