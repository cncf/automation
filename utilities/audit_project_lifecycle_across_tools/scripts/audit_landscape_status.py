#!/usr/bin/env python3
import os
import sys
from typing import Dict, Any, List, Tuple

import requests

try:
    import yaml  # type: ignore
except Exception:
    print("Missing dependency: PyYAML. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(2)

try:
    from bs4 import BeautifulSoup  # type: ignore
except Exception:
    print("Missing dependency: beautifulsoup4. Install with: pip install beautifulsoup4", file=sys.stderr)
    sys.exit(2)

RAW_LANDSCAPE_URL = "https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml"
CLOMONITOR_CNCF_URL = "https://raw.githubusercontent.com/cncf/clomonitor/main/data/cncf.yaml"
DEVSTATS_URL = "https://devstats.cncf.io/"
ARTWORK_README_URL = "https://raw.githubusercontent.com/cncf/artwork/main/README.md"
REPO_ROOT = os.getcwd()
PCC_YAML_PATH = os.path.join(REPO_ROOT, "datasources", "pcc_projects.yaml")
AUDIT_OUTPUT_PATH = os.path.join(REPO_ROOT, "audit", "status_audit.md")
ALL_AUDIT_OUTPUT_PATH = os.path.join(REPO_ROOT, "audit", "all_statuses.md")
PROJECT_HEALTH_OUTPUT_PATH = os.path.join(REPO_ROOT, "audit", "project_health.md")
DATASOURCES_DIR = os.path.join(REPO_ROOT, "datasources")
LANDSCAPE_SRC_PATH = os.path.join(DATASOURCES_DIR, "landscape.yml")
CLOMONITOR_SRC_PATH = os.path.join(DATASOURCES_DIR, "clomonitor.yaml")
DEVSTATS_SRC_PATH = os.path.join(DATASOURCES_DIR, "devstats.html")
ARTWORK_SRC_PATH = os.path.join(DATASOURCES_DIR, "artwork.md")
LFX_HEALTH_SRC_PATH = os.path.join(DATASOURCES_DIR, "lfx_insights_health.yaml")

LANDSCAPE_YML_URL = "https://github.com/cncf/landscape/blob/master/landscape.yml"
CLOMONITOR_YAML_URL = "https://github.com/cncf/clomonitor/blob/main/data/cncf.yaml"
LFX_HEALTH_REL = "../datasources/lfx_insights_health.yaml"
PCC_DATASOURCE_REL = "./pcc_projects.yaml"


def render_markdown_table(headers: List[str], rows: List[List[str]]) -> List[str]:
    lines: List[str] = []
    lines.append("| " + " | ".join(headers) + " |")
    lines.append("|" + "|".join(["---"] * len(headers)) + "|")
    for row in rows:
        cells = [str(c if c else "-") for c in row]
        lines.append("| " + " | ".join(cells) + " |")
    return lines


def ensure_dirs() -> None:
    os.makedirs(os.path.dirname(AUDIT_OUTPUT_PATH), exist_ok=True)
    os.makedirs(DATASOURCES_DIR, exist_ok=True)


def download_landscape_yaml() -> Dict[str, Any]:
    """
    Load Landscape YAML from datasources if present; otherwise fetch and persist it.
    """
    ensure_dirs()
    if os.path.exists(LANDSCAPE_SRC_PATH):
        with open(LANDSCAPE_SRC_PATH, "r", encoding="utf-8") as f:
            return yaml.safe_load(f.read())
    resp = requests.get(RAW_LANDSCAPE_URL, timeout=60)
    resp.raise_for_status()
    text = resp.text
    with open(LANDSCAPE_SRC_PATH, "w", encoding="utf-8") as f:
        f.write(text)
    return yaml.safe_load(text)

def download_clomonitor_yaml() -> Any:
    """
    Load CLOMonitor cncf.yaml from datasources if present; otherwise fetch and persist it.
    """
    ensure_dirs()
    if os.path.exists(CLOMONITOR_SRC_PATH):
        with open(CLOMONITOR_SRC_PATH, "r", encoding="utf-8") as f:
            return yaml.safe_load(f.read())
    resp = requests.get(CLOMONITOR_CNCF_URL, timeout=60)
    resp.raise_for_status()
    text = resp.text
    with open(CLOMONITOR_SRC_PATH, "w", encoding="utf-8") as f:
        f.write(text)
    return yaml.safe_load(text)

def download_devstats_html() -> str:
    """
    Load DevStats HTML from datasources if present; otherwise fetch and persist it.
    """
    ensure_dirs()
    if os.path.exists(DEVSTATS_SRC_PATH):
        with open(DEVSTATS_SRC_PATH, "r", encoding="utf-8") as f:
            return f.read()
    resp = requests.get(DEVSTATS_URL, timeout=60)
    resp.raise_for_status()
    text = resp.text
    with open(DEVSTATS_SRC_PATH, "w", encoding="utf-8") as f:
        f.write(text)
    return text

def download_artwork_readme() -> str:
    """
    Load Artwork README from datasources if present; otherwise fetch and persist it.
    """
    ensure_dirs()
    if os.path.exists(ARTWORK_SRC_PATH):
        with open(ARTWORK_SRC_PATH, "r", encoding="utf-8") as f:
            return f.read()
    resp = requests.get(ARTWORK_README_URL, timeout=60)
    resp.raise_for_status()
    text = resp.text
    with open(ARTWORK_SRC_PATH, "w", encoding="utf-8") as f:
        f.write(text)
    return text


def load_pcc_yaml() -> Dict[str, Any]:
    if not os.path.exists(PCC_YAML_PATH):
        print(f"Error: {PCC_YAML_PATH} not found. Generate it first.", file=sys.stderr)
        sys.exit(1)
    with open(PCC_YAML_PATH, "r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def build_lfx_health_map_from_yaml(doc: Dict[str, Any]) -> Dict[str, Tuple[str, str]]:
    """
    Map normalized lookup keys -> (health_tier, score_str) from lfx_insights_health.yaml.
    """
    out: Dict[str, Tuple[str, str]] = {}
    for row in doc.get("projects") or []:
        name = (row.get("name") or "").strip()
        if not name:
            continue
        tier = (row.get("health_tier") or "").strip()
        os_raw = row.get("overall_score")
        score_str = str(os_raw) if os_raw is not None else ""
        extra_slugs: List[str] = []
        for field in ("pcc_slug", "insights_slug_used"):
            s = row.get(field)
            if isinstance(s, str) and s.strip():
                extra_slugs.append(s.strip())
        seen_slug: set[str] = set()
        uniq_slugs: List[str] = []
        for s in extra_slugs:
            if s not in seen_slug:
                seen_slug.add(s)
                uniq_slugs.append(s)

        keys: set[str] = set()
        if uniq_slugs:
            keys.update(
                generate_aliases_from_landscape(name, {"lfx_slug": uniq_slugs[0]})
            )
            for s in uniq_slugs[1:]:
                keys.update(generate_aliases_from_landscape(name, {"lfx_slug": s}))
        else:
            keys.update(generate_aliases_from_landscape(name, {}))
        for k in keys:
            if k and k not in out:
                out[k] = (tier, score_str)
    return out


def load_lfx_health_map() -> Tuple[Dict[str, Tuple[str, str]], bool]:
    """
    Load LFX Insights health snapshot if present (from weekly automation).
    Returns (lookup map, file_was_present).
    """
    if not os.path.exists(LFX_HEALTH_SRC_PATH):
        return {}, False
    with open(LFX_HEALTH_SRC_PATH, "r", encoding="utf-8") as f:
        doc = yaml.safe_load(f.read()) or {}
    return build_lfx_health_map_from_yaml(doc), True


def normalize_name(name: str) -> str:
    return (name or "").strip().lower()

def _nfkd_ascii(text: str) -> str:
    text = (text or "").replace("³", "3")
    import unicodedata
    nfkd = unicodedata.normalize("NFKD", text)
    return "".join(ch for ch in nfkd if not unicodedata.combining(ch))

def normalize_key(name: str) -> str:
    s = _nfkd_ascii(name).lower().strip()
    s = s.replace("_", " ")
    s = " ".join(s.split())
    return s

def _remove_parentheticals(s: str) -> str:
    import re
    return re.sub(r"\s*\([^)]*\)", "", s).strip()

def _extract_parenthetical_tokens(s: str) -> List[str]:
    import re
    tokens: List[str] = []
    for part in re.findall(r"\(([^)]*)\)", s):
        for t in part.replace("/", " ").replace("-", " ").split():
            t = normalize_key(t)
            if t:
                tokens.append(t)
    return tokens

COMMON_SUFFIXES = (" project", " specification", " operator", " framework", " container linux")

def _remove_common_suffixes(s: str) -> List[str]:
    outs = {s}
    for suf in COMMON_SUFFIXES:
        if s.endswith(suf):
            outs.add(s[: -len(suf)].strip())
    return list(outs)

def _hyphen_space_variants(s: str) -> List[str]:
    if not s:
        return []
    v1 = " ".join(s.replace("-", " ").split())
    v2 = s.replace(" ", "-")
    return list({s, v1, v2})

def _compact_key(s: str) -> str:
    # Keep only alphanumerics; drop spaces, hyphens, punctuation and parentheses
    return "".join(ch for ch in s if ch.isalnum())

def _split_composite_tokens(s: str) -> List[str]:
    import re
    parts = re.split(r"\s*(?:/|,|&| and )\s*", s)
    out: List[str] = []
    for p in parts:
        p = p.strip()
        if p:
            out.append(p)
    return out

def _camel_to_words(s: str) -> str:
    # Insert spaces between camelCase and PascalCase boundaries
    import re
    return re.sub(r"(?<=[a-z0-9])(?=[A-Z])", " ", s)
def generate_aliases_from_landscape(name: str, extra: Any) -> List[str]:
    aliases: List[str] = []
    base = normalize_key(name)
    if not base:
        return []
    aliases.append(base)
    no_paren = normalize_key(_remove_parentheticals(name))
    if no_paren and no_paren not in aliases:
        aliases.append(no_paren)
    for tok in _extract_parenthetical_tokens(name):
        if tok and tok not in aliases:
            aliases.append(tok)
    for candidate in list(aliases):
        for trimmed in _remove_common_suffixes(candidate):
            if trimmed and trimmed not in aliases:
                aliases.append(trimmed)
            with_proj = f"{trimmed} project".strip()
            if with_proj and with_proj not in aliases:
                aliases.append(with_proj)
    for candidate in list(aliases):
        for v in _hyphen_space_variants(candidate):
            if v and v not in aliases:
                aliases.append(v)
    # Composite split (/, &, commas, " and ")
    for candidate in list(aliases):
        for part in _split_composite_tokens(candidate):
            if part and part not in aliases:
                aliases.append(part)
    # Compact (no punctuation/spaces) variants for each alias
    for candidate in list(aliases):
        compact = _compact_key(candidate)
        if compact and compact not in aliases:
            aliases.append(compact)
    # CamelCase to words variants (then normalized)
    for candidate in list(aliases):
        camel = normalize_key(_camel_to_words(candidate))
        if camel and camel not in aliases:
            aliases.append(camel)
    if isinstance(extra, dict):
        lfx_slug = normalize_key((extra.get("lfx_slug") or ""))
        if lfx_slug and lfx_slug not in aliases:
            aliases.append(lfx_slug)
    return aliases


def normalize_status(value: str) -> str:
    if not value:
        return ""
    v = value.strip().lower()
    # Map common variants
    if v in ("graduated",):
        return "graduated"
    if v in ("incubating", "incubator"):
        return "incubating"
    if v in ("sandbox",):
        return "sandbox"
    if v in ("archived", "archive", "archieve", "retired"):
        return "archived"
    if v in ("formation - exploratory", "formation - engaged", "forming", "form", "exploratory"):
        return "forming"
    if v in ("prospect",):
        return "prospect"
    return v


def normalize_slug(value: str) -> str:
    """
    Normalize slugs for equality checks across sources.
    """
    return (value or "").strip().lower()


def build_landscape_status_map(landscape_data: Dict[str, Any]) -> Dict[str, str]:
    name_to_status: Dict[str, str] = {}
    landscape_list: List[Any] = landscape_data.get("landscape") or []
    for cat in landscape_list:
        subcats = (cat.get("subcategories") or [])
        for sub in subcats:
            items = (sub.get("items") or [])
            for item in items:
                # Items may be nested lists/dicts; standardize on dicts with "name" and "project"
                name = (item.get("name") or "").strip()
                if not name:
                    continue
                status = normalize_status(item.get("project") or "")
                if not status:
                    # Non-CNCF or missing project status; skip
                    continue
                extra = item.get("extra") or {}
                # Generate robust alias keys for matching Landscape items to PCC names
                for key in generate_aliases_from_landscape(name, extra):
                    if key and key not in name_to_status:
                        name_to_status[key] = status
    return name_to_status


def build_landscape_slug_map(landscape_data: Dict[str, Any]) -> Dict[str, str]:
    """
    Map normalized lookup keys -> landscape extra.lfx_slug for each project.
    """
    name_to_slug: Dict[str, str] = {}
    landscape_list: List[Any] = landscape_data.get("landscape") or []
    for cat in landscape_list:
        subcats = (cat.get("subcategories") or [])
        for sub in subcats:
            items = (sub.get("items") or [])
            for item in items:
                name = (item.get("name") or "").strip()
                if not name:
                    continue
                extra = item.get("extra") or {}
                lfx_slug = (extra.get("lfx_slug") or "").strip() if isinstance(extra, dict) else ""
                if not lfx_slug:
                    continue
                for key in generate_aliases_from_landscape(name, extra):
                    if key and key not in name_to_slug:
                        name_to_slug[key] = lfx_slug
    return name_to_slug


def build_artwork_status_map(readme_text: str) -> Dict[str, str]:
    # Parse cncf/artwork README where projects are grouped under bullet headings.
    category_to_status = {
        "graduated projects": "graduated",
        "incubating projects": "incubating",
        "sandbox projects": "sandbox",
        "archived projects": "archived",
    }
    name_to_status: Dict[str, str] = {}
    current_status: str = ""

    def parse_bullet_text(line: str) -> str:
        # Extract text after the first '* '
        try:
            star_idx = line.index("*")
        except ValueError:
            return ""
        text = line[star_idx + 1 :].strip()
        # Handle markdown links: [Name](url)
        if text.startswith("[") and "]" in text:
            try:
                end = text.index("]")
                text = text[1:end].strip()
            except Exception:
                pass
        # Trim trailing double-space soft break markers
        text = text.split("  ")[0].strip()
        # Remove stray list markers or punctuation
        return text.strip("*-_ ").strip()

    lines = readme_text.splitlines()
    for raw in lines:
        line = raw.rstrip("\n")
        if not line.strip():
            continue
        # Zero-indent bullets define categories
        if line.startswith("* "):
            cat = parse_bullet_text(line).lower()
            if cat in category_to_status:
                current_status = category_to_status[cat]
                continue
            else:
                # A new top-level bullet that isn't a known category ends the current section
                current_status = ""
        # Indented bullets under a current category are project names (including subprojects)
        if current_status and line.lstrip().startswith("* ") and not line.startswith("* "):
            name = parse_bullet_text(line)
            if name:
                # Generate aliases for artwork project names
                for key in generate_aliases_from_landscape(name, {}):
                    if key and key not in name_to_status:
                        name_to_status[key] = current_status

    return name_to_status


def build_clomonitor_status_map(clomonitor_data: Any) -> Dict[str, str]:
    # clomonitor cncf.yaml is a list of project entries with fields:
    # - name (slug), display_name, maturity (graduated/incubating/sandbox), ...
    name_to_status: Dict[str, str] = {}
    if not isinstance(clomonitor_data, list):
        return name_to_status
    for entry in clomonitor_data:
        if not isinstance(entry, dict):
            continue
        display_name = (entry.get("display_name") or "").strip()
        slug = (entry.get("name") or "").strip()
        maturity = normalize_status(entry.get("maturity") or "")
        if not maturity:
            continue
        # Aliases from display name
        if display_name:
            for key in generate_aliases_from_landscape(display_name, {}):
                if key and key not in name_to_status:
                    name_to_status[key] = maturity
        # Aliases from slug (hyphen/space and suffix variants, plus compact)
        if slug:
            slug_key = normalize_key(slug)
            candidates = set([slug_key])
            for v in _hyphen_space_variants(slug_key) + _remove_common_suffixes(slug_key):
                candidates.add(v.strip())
            # compact variants
            for v in list(candidates):
                candidates.add(_compact_key(v))
            for k in candidates:
                if k and k not in name_to_status:
                    name_to_status[k] = maturity
    return name_to_status


def build_devstats_status_map(html: str) -> Dict[str, str]:
    soup = BeautifulSoup(html, "html.parser")
    name_to_status: Dict[str, str] = {}
    valid_statuses = {"graduated", "incubating", "sandbox", "archived"}
    status_markers = {"Graduated", "Incubating", "Sandbox", "Archived"}

    # Helper: detect if a table row is a status heading row
    def row_status(tr) -> str:
        cells = tr.find_all(["th", "td"])
        for c in cells:
            text = (c.get_text() or "").strip()
            if text in status_markers:
                return normalize_status(text)
        return ""

    # Iterate over all table rows in document order; when a status row is found,
    # collect anchors from subsequent rows until the next status row.
    all_rows = soup.find_all("tr")
    i = 0
    while i < len(all_rows):
        current = all_rows[i]
        current_status = row_status(current)
        if current_status and current_status in valid_statuses:
            i += 1
            while i < len(all_rows):
                nxt = all_rows[i]
                nxt_status = row_status(nxt)
                if nxt_status and nxt_status in valid_statuses:
                    break
                # Collect project anchors in this row
                for a in nxt.find_all("a"):
                    name = (a.get_text() or "").strip()
                    if not name:
                        continue
                    for key in generate_aliases_from_landscape(name, {}):
                        if key and key not in name_to_status:
                            name_to_status[key] = current_status
                i += 1
            continue
        i += 1

    return name_to_status

def _extract_github_path(url: str) -> str:
    """
    Return normalized GitHub path key:
    - 'org/repo' if a repo URL
    - 'org' if an org URL
    Empty string if not a GitHub URL or cannot parse.
    """
    if not url:
        return ""
    u = url.strip().lower()
    if not (u.startswith("http://") or u.startswith("https://")):
        return ""
    try:
        from urllib.parse import urlparse
        parsed = urlparse(u)
        if parsed.netloc != "github.com":
            return ""
        path = parsed.path.strip("/")
        if not path:
            return ""
        parts = [p for p in path.split("/") if p]
        if not parts:
            return ""
        if len(parts) == 1:
            return parts[0]
        repo = parts[1]
        if repo.endswith(".git"):
            repo = repo[:-4]
        return f"{parts[0]}/{repo}"
    except Exception:
        return ""


def collect_pcc_expected_statuses(pcc_data: Dict[str, Any]) -> List[Tuple[str, str, str]]:
    pairs: List[Tuple[str, str, str]] = []
    categories: Dict[str, List[Dict[str, Any]]] = pcc_data.get("categories") or {}
    for cat_name, items in categories.items():
        norm_status = normalize_status(cat_name)
        if norm_status not in ("graduated", "incubating", "sandbox"):
            continue
        for item in items or []:
            name = item.get("name") or ""
            if name:
                slug = (item.get("slug") or "").strip()
                pairs.append((name, slug, norm_status))
    # Archived projects
    for item in pcc_data.get("archived_projects") or []:
        name = item.get("name") or ""
        if not name:
            continue
        # PCC includes some entries that are not CNCF projects; exclude them from the audit output.
        raw_status = (item.get("status") or "").strip()
        if raw_status == "Formation - Disengaged":
            continue
        if raw_status == "Formation - Engaged":
            slug = (item.get("slug") or "").strip()
            pairs.append((name, slug, "forming"))
            continue
        if raw_status == "Prospect":
            slug = (item.get("slug") or "").strip()
            pairs.append((name, slug, "prospect"))
            continue
        slug = (item.get("slug") or "").strip()
        pairs.append((name, slug, "archived"))
    # Forming projects
    for item in pcc_data.get("forming_projects") or []:
        name = item.get("name") or ""
        if not name:
            continue
        slug = (item.get("slug") or "").strip()
        pairs.append((name, slug, "forming"))
    return pairs


def write_audit_markdown(
    combined_rows: List[Tuple[str, str, str, str, str, str, str, str, str, str]],
) -> None:
    lines: List[str] = []
    lines.append(f"# CNCF Project Status Audit")
    lines.append("")
    if not combined_rows:
        lines.append("_No mismatches found between PCC and external sources._")
    else:
        # Sort by PCC status: graduated, incubating, sandbox, forming, archived, prospect; then by project name
        status_order = {"graduated": 0, "incubating": 1, "sandbox": 2, "forming": 3, "archived": 4, "prospect": 5}
        def sort_key(row: Tuple[str, str, str, str, str, str, str, str, str, str]) -> Tuple[int, str]:
            name, _, _, pcc_status, *_ = row
            return (status_order.get(pcc_status, 99), name.lower())
        def fmt(v: str) -> str:
            return v if v else "-"

        core_rows: List[List[str]] = []
        for name, pcc_slug, landscape_slug, pcc_status, landscape_status, cm_status, d_status, a_status, lfx_tier, lfx_score in sorted(combined_rows, key=sort_key):
            core_rows.append([
                name,
                fmt(pcc_slug),
                fmt(landscape_slug),
                fmt(pcc_status),
                fmt(landscape_status),
                fmt(cm_status),
                fmt(d_status),
                fmt(a_status),
            ])
        lines.extend(render_markdown_table(
            [
                "Project",
                f"[PCC Slug]({PCC_DATASOURCE_REL})",
                f"[Landscape Slug]({LANDSCAPE_YML_URL})",
                f"[PCC Status]({PCC_DATASOURCE_REL})",
                f"[Landscape Status]({LANDSCAPE_YML_URL})",
                f"[CLOMonitor]({CLOMONITOR_YAML_URL})",
                f"[DevStats]({DEVSTATS_URL})",
                f"[Artwork]({ARTWORK_README_URL})",
            ],
            core_rows,
        ))

    with open(AUDIT_OUTPUT_PATH, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def write_full_status_markdown(
    all_rows: List[Tuple[str, str, str, str, str, str, str, str, str, str]],
) -> None:
    """
    Write a full report with anomalies first, then all projects grouped by PCC category
    (Graduated, Incubating, Sandbox), with projects in alphabetical order.
    """
    # Compute anomalies: include projects with ANY missing value ('-' after formatting) OR
    # any external source present and different from PCC
    anomalies: List[Tuple[str, str, str, str, str, str, str, str, str, str]] = []
    for name, pcc_slug, l_slug, pcc_status, l_status, cm_status, d_status, a_status, lfx_tier, lfx_score in all_rows:
        norm_pcc = normalize_status(pcc_status)
        missing_any = (l_status == "-") or (not cm_status) or (not d_status) or (not a_status)
        differs_any = any([
            (l_status and l_status != norm_pcc and l_status != "-"),
            (cm_status and normalize_status(cm_status) != norm_pcc),
            (d_status and normalize_status(d_status) != norm_pcc),
            (a_status and normalize_status(a_status) != norm_pcc),
        ])
        if missing_any or differs_any:
            anomalies.append((name, pcc_slug, l_slug, pcc_status, l_status, cm_status, d_status, a_status, lfx_tier, lfx_score))

    def section(title: str, rows: List[Tuple[str, str, str, str, str, str, str, str, str, str]]) -> List[str]:
        out: List[str] = []
        out.append(f"## {title}")
        out.append("")
        if not rows:
            out.append("_No entries._")
            out.append("")
            return out
        def fmt(v: str) -> str:
            return v if v else "-"
        core_rows: List[List[str]] = []
        for name, pcc_slug, landscape_slug, pcc_status, landscape_status, cm_status, d_status, a_status, lfx_tier, lfx_score in rows:
            core_rows.append([
                name,
                fmt(pcc_slug),
                fmt(landscape_slug),
                fmt(pcc_status),
                fmt(landscape_status),
                fmt(cm_status),
                fmt(d_status),
                fmt(a_status),
            ])
        out.extend(render_markdown_table(
            [
                "Project",
                f"[PCC Slug]({PCC_DATASOURCE_REL})",
                f"[Landscape Slug]({LANDSCAPE_YML_URL})",
                f"[PCC]({PCC_DATASOURCE_REL})",
                f"[Landscape]({LANDSCAPE_YML_URL})",
                f"[CLOMonitor]({CLOMONITOR_YAML_URL})",
                f"[DevStats]({DEVSTATS_URL})",
                f"[Artwork]({ARTWORK_README_URL})",
            ],
            core_rows,
        ))
        out.append("")
        return out

    # Sort helpers (match anomalies table order)
    status_order = {"graduated": 0, "incubating": 1, "sandbox": 2, "forming": 3, "archived": 4, "prospect": 5}
    def status_then_name(row: Tuple[str, str, str, str, str, str, str, str, str, str]) -> Tuple[int, str]:
        name, _, _, pcc_status, *_ = row
        return (status_order.get(normalize_status(pcc_status), 99), name.lower())

    # Sort anomalies by PCC status then name
    anomalies_sorted = sorted(anomalies, key=status_then_name)

    # Group all by PCC category (include forming, archived, and prospect too)
    by_cat: Dict[str, List[Tuple[str, str, str, str, str, str, str, str, str, str]]] = {
        "graduated": [],
        "incubating": [],
        "sandbox": [],
        "forming": [],
        "archived": [],
        "prospect": [],
    }
    for row in all_rows:
        _, _, _, pcc_status, *_ = row
        cat = normalize_status(pcc_status)
        if cat in by_cat:
            by_cat[cat].append(row)

    # Sort alphabetical within each section
    for k in list(by_cat.keys()):
        by_cat[k] = sorted(by_cat[k], key=lambda r: r[0].lower())

    lines: List[str] = []
    lines.append("# CNCF Project Statuses")
    lines.append("")
    lines.extend(section("Anomalies", anomalies_sorted))
    # Sections in the requested sort order
    lines.extend(section("Graduated", by_cat["graduated"]))
    lines.extend(section("Incubating", by_cat["incubating"]))
    lines.extend(section("Sandbox", by_cat["sandbox"]))
    lines.extend(section("Forming", by_cat["forming"]))
    lines.extend(section("Archived", by_cat["archived"]))
    lines.extend(section("Prospect", by_cat["prospect"]))

    with open(ALL_AUDIT_OUTPUT_PATH, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def write_project_health_markdown(
    all_rows: List[Tuple[str, str, str, str, str, str, str, str, str, str]],
) -> None:
    lines: List[str] = []
    lines.append("# CNCF Project Health")
    lines.append("")

    def fmt(v: str) -> str:
        return v if v else "-"

    def to_health_row(row: Tuple[str, str, str, str, str, str, str, str, str, str]) -> List[str]:
        name, _pcc_slug, _landscape_slug, pcc_status, _l_status, _cm_status, _d_status, _a_status, lfx_tier, lfx_score = row
        return [name, fmt(pcc_status), fmt(lfx_tier), fmt(lfx_score)]

    def section(title: str, rows: List[Tuple[str, str, str, str, str, str, str, str, str, str]]) -> List[str]:
        out: List[str] = []
        out.append(f"## {title}")
        out.append("")
        if not rows:
            out.append("_No entries._")
            out.append("")
            return out
        sorted_rows = sorted(rows, key=lambda r: r[0].lower())
        out.extend(render_markdown_table(
            [
                "Project",
                f"[PCC Status]({PCC_DATASOURCE_REL})",
                f"[Insights Health]({LFX_HEALTH_REL})",
                f"[Health Score]({LFX_HEALTH_REL})",
            ],
            [to_health_row(r) for r in sorted_rows],
        ))
        out.append("")
        return out

    by_cat: Dict[str, List[Tuple[str, str, str, str, str, str, str, str, str, str]]] = {
        "graduated": [],
        "incubating": [],
        "sandbox": [],
        "forming": [],
        "archived": [],
        "prospect": [],
    }
    anomalies: List[Tuple[str, str, str, str, str, str, str, str, str, str]] = []
    anomaly_statuses = {"graduated", "incubating", "sandbox"}

    for row in all_rows:
        pcc_status = normalize_status(row[3])
        if pcc_status in by_cat:
            by_cat[pcc_status].append(row)
        health_score = (row[9] or "").strip()
        if pcc_status in anomaly_statuses and not health_score:
            anomalies.append(row)

    lines.extend(section("Anomalies", anomalies))
    lines.extend(section("Graduated", by_cat["graduated"]))
    lines.extend(section("Incubating", by_cat["incubating"]))
    lines.extend(section("Sandbox", by_cat["sandbox"]))
    lines.extend(section("Forming", by_cat["forming"]))
    lines.extend(section("Archived", by_cat["archived"]))
    lines.extend(section("Prospect", by_cat["prospect"]))

    with open(PROJECT_HEALTH_OUTPUT_PATH, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def main() -> None:
    ensure_dirs()
    pcc = load_pcc_yaml()
    landscape = download_landscape_yaml()
    clomonitor = download_clomonitor_yaml()
    devstats_html = download_devstats_html()
    artwork_readme = download_artwork_readme()
    landscape_map = build_landscape_status_map(landscape)
    landscape_slug_map = build_landscape_slug_map(landscape)
    clomonitor_map = build_clomonitor_status_map(clomonitor)
    devstats_map = build_devstats_status_map(devstats_html)
    artwork_map = build_artwork_status_map(artwork_readme)
    lfx_map, _ = load_lfx_health_map()
    expected = collect_pcc_expected_statuses(pcc)

    combined_rows: List[Tuple[str, str, str, str, str, str, str, str, str, str]] = []
    all_rows: List[Tuple[str, str, str, str, str, str, str, str, str, str]] = []
    for name, pcc_slug, pcc_status in expected:
        norm_pcc = normalize_status(pcc_status)
        # Build multiple query keys for Landscape lookup
        query_keys: List[str] = []
        base_key = normalize_key(name)
        query_keys.append(base_key)
        no_paren = normalize_key(_remove_parentheticals(name))
        if no_paren and no_paren not in query_keys:
            query_keys.append(no_paren)
        for candidate in list(query_keys):
            for trimmed in _remove_common_suffixes(candidate):
                if trimmed and trimmed not in query_keys:
                    query_keys.append(trimmed)
        for candidate in list(query_keys):
            for v in _hyphen_space_variants(candidate):
                if v and v not in query_keys:
                    query_keys.append(v)
        # Add compact and camel-case-separated variants
        for candidate in list(query_keys):
            comp = _compact_key(candidate)
            if comp and comp not in query_keys:
                query_keys.append(comp)
            camel = normalize_key(_camel_to_words(candidate))
            if camel and camel not in query_keys:
                query_keys.append(camel)
        for tok in _extract_parenthetical_tokens(name):
            if tok and tok not in query_keys:
                query_keys.append(tok)
        l_status_raw = ""
        l_slug_raw = ""
        for k in query_keys:
            if k in landscape_map:
                l_status_raw = landscape_map[k]
                break
        for k in query_keys:
            if k in landscape_slug_map:
                l_slug_raw = landscape_slug_map[k]
                break
        # Use the same robust key set for other sources
        cm_status_raw = ""
        d_status_raw = ""
        a_status_raw = ""
        for k in query_keys:
            if not cm_status_raw and k in clomonitor_map:
                cm_status_raw = clomonitor_map[k]
            if not d_status_raw and k in devstats_map:
                d_status_raw = devstats_map[k]
            if not a_status_raw and k in artwork_map:
                a_status_raw = artwork_map[k]
        # For Landscape, explicitly show '-' when missing to flag anomaly
        l_status = normalize_status(l_status_raw) if l_status_raw else "-"
        l_slug = (l_slug_raw or "").strip()
        # For other sources, keep empty when missing
        cm_status = normalize_status(cm_status_raw) if cm_status_raw else ""
        d_status = normalize_status(d_status_raw) if d_status_raw else ""
        a_status = normalize_status(a_status_raw) if a_status_raw else ""

        lfx_tier_raw = ""
        lfx_score_raw = ""
        for k in query_keys:
            if k in lfx_map:
                t, s = lfx_map[k]
                lfx_tier_raw = t or ""
                lfx_score_raw = s or ""
                break
        lfx_tier = (lfx_tier_raw or "").strip()
        lfx_score = (lfx_score_raw or "").strip()

        slug_disp = l_slug if l_slug else "-"
        all_rows.append((name, pcc_slug, slug_disp, norm_pcc, l_status, cm_status, d_status, a_status, lfx_tier, lfx_score))

        # Anomaly criteria:
        # - Any missing value in any source (displayed as '-' later; Landscape missing is already '-')
        # - OR any source present and different from PCC
        landscape_mismatch = (l_status == "-") or (l_status != norm_pcc)
        landscape_slug_missing = (slug_disp == "-")
        pcc_slug_norm = normalize_slug(pcc_slug)
        landscape_slug_norm = normalize_slug(slug_disp)
        landscape_slug_mismatch = bool(pcc_slug_norm) and bool(landscape_slug_norm) and (pcc_slug_norm != landscape_slug_norm)
        clomonitor_mismatch = bool(cm_status) and (cm_status != norm_pcc)
        devstats_mismatch = bool(d_status) and (d_status != norm_pcc)
        artwork_mismatch = bool(a_status) and (a_status != norm_pcc)
        any_missing = (l_status == "-") or (not cm_status) or (not d_status) or (not a_status)

        if any_missing or landscape_mismatch or landscape_slug_missing or landscape_slug_mismatch or clomonitor_mismatch or devstats_mismatch or artwork_mismatch:
            combined_rows.append((name, pcc_slug, slug_disp, norm_pcc, l_status, cm_status, d_status, a_status, lfx_tier, lfx_score))

    write_audit_markdown(combined_rows)
    write_full_status_markdown(all_rows)
    write_project_health_markdown(all_rows)
    print(f"Wrote audit with {len(combined_rows)} mismatches to {AUDIT_OUTPUT_PATH}")


if __name__ == "__main__":
    main()


