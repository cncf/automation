#!/usr/bin/env python3
"""
Fetch LFX Insights health tier (and optional numeric score) for CNCF projects listed in
datasources/pcc_projects.yaml.

Uses the public project page (SSR) as the source of truth for whether a project is archived
on Insights; in that case we record tier "Archived" and no score — not the badge tier (e.g.
"Stable") which can be misleading for archived projects.

Slug candidates: first a slug derived from the PCC project `name` (same as the audit "Project"
column), then the PCC `slug` when present (covers cases where the LF slug differs, e.g. CubeFS →
chubaofs). No LFX_TOKEN required.
"""

from __future__ import annotations

import argparse
import os
import re
import sys
import time
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple
from urllib.parse import parse_qs, unquote, urlparse

import requests

try:
    import yaml  # type: ignore
except Exception:
    print("Missing dependency: PyYAML. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(2)

REPO_ROOT = os.getcwd()
DATASOURCES_DIR = os.path.join(REPO_ROOT, "datasources")
PCC_PATH = os.path.join(DATASOURCES_DIR, "pcc_projects.yaml")
OUTPUT_PATH = os.path.join(DATASOURCES_DIR, "lfx_insights_health.yaml")

BADGE_URL = "https://insights.linuxfoundation.org/api/badge/health-score"
SEARCH_URL = "https://insights.linuxfoundation.org/api/search"
PROJECT_PAGE_URL = "https://insights.linuxfoundation.org/project/{slug}"

SESSION = requests.Session()
SESSION.headers.update(
    {
        "User-Agent": "cncf-automation/lfx-insights-health (+github actions)",
        "Accept": "text/html,application/json",
    }
)

# Insights renders the overall score as:
#   <span ...>NN</span><span ...>out of 100</span>
OVERALL_SCORE_RE = re.compile(
    r">\s*(\d{1,3})\s*</span>\s*<span[^>]*>\s*out of 100\s*</span>"
)
SLEEP_SECONDS = 0.35


def is_archived_insights_page(html: str) -> bool:
    """True when Insights marks the project archived (health score not shown for current period)."""
    if "Archived project are excluded from" in html:
        return True
    if 'font-bold text-neutral-500">Archived Project</' in html:
        return True
    if re.search(r'text-nowrap">Archived</span>', html):
        return True
    return False


def fetch_project_page_html_with_status(slug: str) -> Tuple[Optional[str], Optional[int]]:
    """Return (html, http_status) for an Insights project page lookup."""
    url = PROJECT_PAGE_URL.format(slug=requests.utils.quote(slug, safe=""))
    try:
        r = SESSION.get(url, timeout=45)
    except requests.RequestException:
        return None, None
    if r.status_code != 200:
        return None, r.status_code
    return r.text, r.status_code


def overall_score_from_page_html(html: str) -> Optional[int]:
    m = OVERALL_SCORE_RE.search(html)
    if not m:
        return None
    try:
        return int(m.group(1))
    except ValueError:
        return None


def slugify_from_name(name: str) -> str:
    """Fallback slug when PCC has no Slug (forming / some archived)."""
    s = re.sub(r"\([^)]*\)", "", name or "")
    s = s.lower().strip()
    s = re.sub(r"[^a-z0-9]+", "-", s)
    s = re.sub(r"-+", "-", s).strip("-")
    return s


def normalize_name_for_match(value: str) -> str:
    s = (value or "").lower().strip()
    s = re.sub(r"\([^)]*\)", "", s)
    s = re.sub(r"[^a-z0-9]+", " ", s).strip()
    return s


def search_insights_slug_by_name(name: str) -> Optional[str]:
    """
    Resolve a project slug using the same public search endpoint the Insights UI uses.
    Prefer exact normalized-name matches to avoid accidental cross-project matches.
    """
    query = (name or "").strip()
    if not query:
        return None
    try:
        r = SESSION.get(SEARCH_URL, params={"query": query}, timeout=45)
    except requests.RequestException:
        return None
    if r.status_code != 200:
        return None
    try:
        payload = r.json()
    except ValueError:
        return None

    projects = payload.get("projects")
    if not isinstance(projects, list) or not projects:
        return None

    query_norm = normalize_name_for_match(query)
    exact: List[str] = []
    for item in projects:
        if not isinstance(item, dict):
            continue
        slug = item.get("slug")
        project_name = item.get("name")
        if not isinstance(slug, str) or not slug.strip():
            continue
        if isinstance(project_name, str) and normalize_name_for_match(project_name) == query_norm:
            exact.append(slug.strip())

    if exact:
        # Deterministic pick if there are multiple exact-normalized names.
        return sorted(set(exact), key=str.lower)[0]

    # Conservative fallback: single search hit from UI endpoint.
    if len(projects) == 1 and isinstance(projects[0], dict):
        slug = projects[0].get("slug")
        if isinstance(slug, str) and slug.strip():
            return slug.strip()
    return None


def iter_pcc_projects(
    pcc: Dict[str, Any], max_projects: Optional[int] = None
) -> List[Tuple[str, Optional[str], str]]:
    """
    Yields (display_name, pcc_slug_or_none, origin) for each project row we try to match
    in Insights. Order: Graduated, Incubating, Sandbox, forming, archived.
    """
    out: List[Tuple[str, Optional[str], str]] = []
    categories = pcc.get("categories") or {}
    for cat in ("Graduated", "Incubating", "Sandbox"):
        for item in categories.get(cat) or []:
            name = (item.get("name") or "").strip()
            if not name:
                continue
            slug = item.get("slug")
            if isinstance(slug, str):
                slug = slug.strip() or None
            out.append((name, slug, f"category:{cat}"))
    for item in pcc.get("forming_projects") or []:
        name = (item.get("name") or "").strip()
        if not name:
            continue
        slug = item.get("slug")
        if isinstance(slug, str):
            slug = slug.strip() or None
        out.append((name, slug, "forming"))
    for item in pcc.get("archived_projects") or []:
        name = (item.get("name") or "").strip()
        if not name:
            continue
        slug = item.get("slug")
        if isinstance(slug, str):
            slug = slug.strip() or None
        out.append((name, slug, "archived"))
    if max_projects is not None:
        return out[: max(0, max_projects)]
    return out


def resolve_slugs_to_try(name: str, pcc_slug: Optional[str]) -> List[str]:
    """
    Insights URLs use /project/{slug}. Try slug from PCC display name first (matches user-facing
    search), then PCC slug when it differs (e.g. CubeFS → chubaofs, KEDA casing).
    """
    seen: set[str] = set()
    ordered: List[str] = []
    derived = slugify_from_name(name)
    if derived and derived not in seen:
        seen.add(derived)
        ordered.append(derived)
    if pcc_slug:
        for s in (pcc_slug, pcc_slug.lower()):
            if s and s not in seen:
                seen.add(s)
                ordered.append(s)
    return ordered


def tier_from_badge_head(slug: str) -> Tuple[Optional[str], int]:
    """
    Returns (tier_label, http_status) for the badge HEAD request.
    tier_label is parsed from shields redirect Location (message=...).
    """
    try:
        r = SESSION.head(
            BADGE_URL,
            params={"project": slug},
            allow_redirects=False,
            timeout=45,
        )
    except requests.RequestException as e:
        print(f"  badge HEAD error for slug={slug!r}: {e}", file=sys.stderr)
        return None, 0

    if r.status_code == 302 and r.headers.get("Location"):
        loc = r.headers["Location"]
        q = parse_qs(urlparse(loc).query)
        raw = (q.get("message") or [None])[0]
        if raw:
            return unquote(raw), r.status_code
    return None, r.status_code


def fetch_one(name: str, pcc_slug: Optional[str]) -> Dict[str, Any]:
    slugs = resolve_slugs_to_try(name, pcc_slug)
    row: Dict[str, Any] = {
        "name": name,
        "pcc_slug": pcc_slug,
        "insights_slug_used": None,
        "health_tier": None,
        "overall_score": None,
        "error": None,
    }
    last_badge_status: Optional[int] = None
    saw_only_404_project_pages = True

    # 1) Prefer project page: authoritative for "Archived" and numeric score.
    for slug in slugs:
        time.sleep(SLEEP_SECONDS)
        html, page_status = fetch_project_page_html_with_status(slug)
        if page_status != 404:
            saw_only_404_project_pages = False
        if not html:
            continue
        if is_archived_insights_page(html):
            row["insights_slug_used"] = slug
            row["health_tier"] = "Archived"
            row["overall_score"] = None
            return row
        score = overall_score_from_page_html(html)
        time.sleep(SLEEP_SECONDS)
        tier, last_badge_status = tier_from_badge_head(slug)
        if tier:
            row["insights_slug_used"] = slug
            row["health_tier"] = tier
            row["overall_score"] = score
            return row

    # 2) Fallback: badge-only (no successful project page).
    for slug in slugs:
        tier, last_badge_status = tier_from_badge_head(slug)
        time.sleep(SLEEP_SECONDS)
        if tier:
            row["insights_slug_used"] = slug
            row["health_tier"] = tier
            row["overall_score"] = None
            return row

    # 3) UI-like fallback: if name-based slug attempts all landed on 404, resolve via Insights search.
    if saw_only_404_project_pages:
        search_slug = search_insights_slug_by_name(name)
        if search_slug and search_slug not in slugs:
            time.sleep(SLEEP_SECONDS)
            html, _status = fetch_project_page_html_with_status(search_slug)
            if html:
                if is_archived_insights_page(html):
                    row["insights_slug_used"] = search_slug
                    row["health_tier"] = "Archived"
                    row["overall_score"] = None
                    return row
                score = overall_score_from_page_html(html)
                time.sleep(SLEEP_SECONDS)
                tier, last_badge_status = tier_from_badge_head(search_slug)
                if tier:
                    row["insights_slug_used"] = search_slug
                    row["health_tier"] = tier
                    row["overall_score"] = score
                    return row
            tier, last_badge_status = tier_from_badge_head(search_slug)
            time.sleep(SLEEP_SECONDS)
            if tier:
                row["insights_slug_used"] = search_slug
                row["health_tier"] = tier
                row["overall_score"] = None
                return row

    row["error"] = "not_found"
    if last_badge_status is not None:
        row["http_status_last"] = last_badge_status
    return row


def main() -> None:
    parser = argparse.ArgumentParser(description="Fetch LFX Insights health scores from PCC project list.")
    parser.add_argument(
        "--max-projects",
        type=int,
        default=None,
        metavar="N",
        help="Only process the first N projects (for testing).",
    )
    args = parser.parse_args()

    os.makedirs(DATASOURCES_DIR, exist_ok=True)
    if not os.path.isfile(PCC_PATH):
        print(f"Error: {PCC_PATH} not found. Run PCC sync or checkout the repo with datasources.", file=sys.stderr)
        sys.exit(1)

    with open(PCC_PATH, "r", encoding="utf-8") as f:
        pcc = yaml.safe_load(f.read())

    projects_in = iter_pcc_projects(pcc, max_projects=args.max_projects)
    results: List[Dict[str, Any]] = []
    for i, (name, pcc_slug, _origin) in enumerate(projects_in):
        print(f"[{i + 1}/{len(projects_in)}] {name!r} ...", flush=True)
        results.append(fetch_one(name, pcc_slug))
        time.sleep(SLEEP_SECONDS)

    out_doc: Dict[str, Any] = {
        "source": "LFX Insights (project page SSR + badge; archived status from page)",
        "pcc_source_file": "datasources/pcc_projects.yaml",
        "generated_at": datetime.now(timezone.utc).replace(microsecond=0).isoformat(),
        "projects": results,
    }

    with open(OUTPUT_PATH, "w", encoding="utf-8") as f:
        yaml.safe_dump(out_doc, f, sort_keys=False, allow_unicode=True)

    ok = sum(1 for r in results if r.get("health_tier"))
    missing = sum(1 for r in results if r.get("error") == "not_found")
    print(f"Wrote {OUTPUT_PATH} — {ok} with tier, {missing} not found, {len(results)} total")


if __name__ == "__main__":
    main()
