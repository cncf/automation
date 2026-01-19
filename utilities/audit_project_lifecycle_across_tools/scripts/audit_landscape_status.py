#!/usr/bin/env python3
import os
import sys
from typing import Dict, Any, List, Tuple

import csv
import io
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
FOUNDATION_MAINTAINERS_CSV_URL = "https://raw.githubusercontent.com/cncf/foundation/main/project-maintainers.csv"
DEVSTATS_URL = "https://devstats.cncf.io/"
ARTWORK_README_URL = "https://raw.githubusercontent.com/cncf/artwork/main/README.md"
REPO_ROOT = os.getcwd()
PCC_YAML_PATH = os.path.join(REPO_ROOT, "datasources", "pcc_projects.yaml")
AUDIT_OUTPUT_PATH = os.path.join(REPO_ROOT, "audit", "status_audit.md")
ALL_AUDIT_OUTPUT_PATH = os.path.join(REPO_ROOT, "audit", "all_statuses.md")
DATASOURCES_DIR = os.path.join(REPO_ROOT, "datasources")
LANDSCAPE_SRC_PATH = os.path.join(DATASOURCES_DIR, "landscape.yml")
CLOMONITOR_SRC_PATH = os.path.join(DATASOURCES_DIR, "clomonitor.yaml")
MAINTAINERS_SRC_PATH = os.path.join(DATASOURCES_DIR, "project-maintainers.csv")
DEVSTATS_SRC_PATH = os.path.join(DATASOURCES_DIR, "devstats.html")
ARTWORK_SRC_PATH = os.path.join(DATASOURCES_DIR, "artwork.md")


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

def download_foundation_maintainers_csv() -> List[Dict[str, str]]:
    """
    Load Maintainers CSV from datasources if present; otherwise fetch and persist it.
    """
    ensure_dirs()
    if os.path.exists(MAINTAINERS_SRC_PATH):
        with open(MAINTAINERS_SRC_PATH, "r", encoding="utf-8") as f:
            text = f.read()
    else:
        resp = requests.get(FOUNDATION_MAINTAINERS_CSV_URL, timeout=60)
        resp.raise_for_status()
        text = resp.text
        with open(MAINTAINERS_SRC_PATH, "w", encoding="utf-8") as f:
            f.write(text)
    # The CSV has a header row where first column header is empty, second is "Project"
    reader = csv.reader(io.StringIO(text))
    rows: List[Dict[str, str]] = []
    header = None
    for i, r in enumerate(reader):
        if i == 0:
            header = r
            continue
        # Map to fields by position we care about: 0=status, 1=project
        status = (r[0] if len(r) > 0 else "").strip()
        project = (r[1] if len(r) > 1 else "").strip()
        url = ""
        if len(r) >= 6:
            url_candidate = (r[-1] or "").strip()
            if url_candidate.startswith("http"):
                url = url_candidate
        if not project:
            continue
        rows.append({"status": status, "project": project, "url": url})
    return rows

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
    if v in ("formation - exploratory", "forming", "form", "exploratory"):
        return "forming"
    return v


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


def build_foundation_status_map(entries: List[Dict[str, str]]) -> Dict[str, str]:
    name_to_status: Dict[str, str] = {}
    for e in entries:
        project = (e.get("project") or "").strip()
        status = e.get("status") or ""
        url = (e.get("url") or "").strip()
        if not project or not status:
            continue
        norm_status = normalize_status(status)
        # Filter to statuses we track; skip steering/maintainers pseudo-projects if not in PCC
        if norm_status in ("graduated", "incubating", "sandbox", "archived", "forming"):
            # Base aliases
            alias_candidates: List[str] = generate_aliases_from_landscape(project, {})
            # Add colon-left alias (e.g., "Istio: Steering Committee" -> "Istio")
            if ":" in project:
                lhs = project.split(":", 1)[0].strip()
                if lhs:
                    alias_candidates.extend(generate_aliases_from_landscape(lhs, {}))
            # Add first-word alias (e.g., "Kubernetes steering" -> "Kubernetes")
            first_word = project.split()[0] if project.split() else ""
            if first_word:
                alias_candidates.extend(generate_aliases_from_landscape(first_word, {}))
            # Add '-ai' stripped variant if present (e.g., 'k8sgpt-ai' -> 'k8sgpt')
            for a in list(alias_candidates):
                if a.endswith("-ai"):
                    alias_candidates.append(a[:-3])
            for key in alias_candidates:
                if key and key not in name_to_status:
                    name_to_status[key] = norm_status
            # GitHub URL aliases (org and org/repo)
            gh = _extract_github_path(url)
            if gh:
                parts = gh.split("/")
                org = parts[0]
                if org and org not in name_to_status:
                    name_to_status[org] = norm_status
                if len(parts) >= 2 and gh not in name_to_status:
                    name_to_status[gh] = norm_status
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


def collect_pcc_expected_statuses(pcc_data: Dict[str, Any]) -> List[Tuple[str, str]]:
    pairs: List[Tuple[str, str]] = []
    categories: Dict[str, List[Dict[str, Any]]] = pcc_data.get("categories") or {}
    for cat_name, items in categories.items():
        norm_status = normalize_status(cat_name)
        if norm_status not in ("graduated", "incubating", "sandbox"):
            continue
        for item in items or []:
            name = item.get("name") or ""
            if name:
                pairs.append((name, norm_status))
    # Archived projects
    for item in pcc_data.get("archived_projects") or []:
        name = item.get("name") or ""
        if not name:
            continue
        pairs.append((name, "archived"))
    # Forming projects
    for item in pcc_data.get("forming_projects") or []:
        name = item.get("name") or ""
        if not name:
            continue
        pairs.append((name, "forming"))
    return pairs


def write_audit_markdown(
    combined_rows: List[Tuple[str, str, str, str, str, str, str]],
) -> None:
    lines: List[str] = []
    lines.append(f"# CNCF Project Status Audit")
    lines.append("")
    if not combined_rows:
        lines.append("_No mismatches found between PCC and external sources._")
    else:
        # Column headers hyperlinked to their respective sources for quick reference
        lines.append("| Project | [PCC status](./pcc_projects.yaml) | [Landscape status](https://github.com/cncf/landscape/blob/master/landscape.yml) | [CLOMonitor status](https://github.com/cncf/clomonitor/blob/main/data/cncf.yaml) | [Maintainers CSV status](https://github.com/cncf/foundation/blob/main/project-maintainers.csv) | [DevStats status](https://devstats.cncf.io/) | [Artwork status](https://github.com/cncf/artwork/blob/main/README.md) |")
        lines.append("|---|---|---|---|---|---|---|")
        # Sort by PCC status: graduated, incubating, sandbox, forming, archived; then by project name
        status_order = {"graduated": 0, "incubating": 1, "sandbox": 2, "forming": 3, "archived": 4}
        def sort_key(row: Tuple[str, str, str, str, str, str, str]) -> Tuple[int, str]:
            name, pcc_status, *_ = row
            return (status_order.get(pcc_status, 99), name.lower())
        def fmt(v: str) -> str:
            return v if v else "-"
        for name, pcc_status, landscape_status, cm_status, m_status, d_status, a_status in sorted(combined_rows, key=sort_key):
            lines.append(f"| {name} | {fmt(pcc_status)} | {fmt(landscape_status)} | {fmt(cm_status)} | {fmt(m_status)} | {fmt(d_status)} | {fmt(a_status)} |")

    with open(AUDIT_OUTPUT_PATH, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def write_full_status_markdown(
    all_rows: List[Tuple[str, str, str, str, str, str, str]],
) -> None:
    """
    Write a full report with anomalies first, then all projects grouped by PCC category
    (Graduated, Incubating, Sandbox), with projects in alphabetical order.
    """
    # Compute anomalies: include projects with ANY missing value ('-' after formatting) OR
    # any external source present and different from PCC
    anomalies: List[Tuple[str, str, str, str, str, str, str]] = []
    for name, pcc_status, l_status, cm_status, m_status, d_status, a_status in all_rows:
        norm_pcc = normalize_status(pcc_status)
        missing_any = (l_status == "-") or (not cm_status) or (not m_status) or (not d_status) or (not a_status)
        differs_any = any([
            (l_status and l_status != norm_pcc and l_status != "-"),
            (cm_status and normalize_status(cm_status) != norm_pcc),
            (m_status and normalize_status(m_status) != norm_pcc),
            (d_status and normalize_status(d_status) != norm_pcc),
            (a_status and normalize_status(a_status) != norm_pcc),
        ])
        if missing_any or differs_any:
            anomalies.append((name, pcc_status, l_status, cm_status, m_status, d_status, a_status))

    def section(title: str, rows: List[Tuple[str, str, str, str, str, str, str]]) -> List[str]:
        out: List[str] = []
        out.append(f"## {title}")
        out.append("")
        if not rows:
            out.append("_No entries._")
            out.append("")
            return out
        out.append("| Project | PCC | [Landscape](https://github.com/cncf/landscape/blob/master/landscape.yml) | [CLOMonitor](https://github.com/cncf/clomonitor/blob/main/data/cncf.yaml) | [Maintainers](https://github.com/cncf/foundation/blob/main/project-maintainers.csv) | [DevStats](https://devstats.cncf.io/) | [Artwork](https://github.com/cncf/artwork/blob/main/README.md) |")
        out.append("|---|---|---|---|---|---|---|")
        def fmt(v: str) -> str:
            return v if v else "-"
        for name, pcc_status, l_status, cm_status, m_status, d_status, a_status in rows:
            out.append(f"| {name} | {fmt(pcc_status)} | {fmt(l_status)} | {fmt(cm_status)} | {fmt(m_status)} | {fmt(d_status)} | {fmt(a_status)} |")
        out.append("")
        return out

    # Sort helpers (match anomalies table order)
    status_order = {"graduated": 0, "incubating": 1, "sandbox": 2, "forming": 3, "archived": 4}
    def status_then_name(row: Tuple[str, str, str, str, str, str, str]) -> Tuple[int, str]:
        name, pcc_status, *_ = row
        return (status_order.get(normalize_status(pcc_status), 99), name.lower())

    # Sort anomalies by PCC status then name
    anomalies_sorted = sorted(anomalies, key=status_then_name)

    # Group all by PCC category (include forming and archived too)
    by_cat: Dict[str, List[Tuple[str, str, str, str, str, str, str]]] = {
        "graduated": [],
        "incubating": [],
        "sandbox": [],
        "forming": [],
        "archived": [],
    }
    for row in all_rows:
        _, pcc_status, *_ = row
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

    with open(ALL_AUDIT_OUTPUT_PATH, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n")


def main() -> None:
    ensure_dirs()
    pcc = load_pcc_yaml()
    landscape = download_landscape_yaml()
    clomonitor = download_clomonitor_yaml()
    maintainers_csv = download_foundation_maintainers_csv()
    devstats_html = download_devstats_html()
    artwork_readme = download_artwork_readme()
    landscape_map = build_landscape_status_map(landscape)
    clomonitor_map = build_clomonitor_status_map(clomonitor)
    maintainers_map = build_foundation_status_map(maintainers_csv)
    devstats_map = build_devstats_status_map(devstats_html)
    artwork_map = build_artwork_status_map(artwork_readme)
    expected = collect_pcc_expected_statuses(pcc)

    combined_rows: List[Tuple[str, str, str, str, str, str, str]] = []
    all_rows: List[Tuple[str, str, str, str, str, str, str]] = []
    for name, pcc_status in expected:
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
        for k in query_keys:
            if k in landscape_map:
                l_status_raw = landscape_map[k]
                break
        # Use the same robust key set for other sources
        cm_status_raw = ""
        m_status_raw = ""
        d_status_raw = ""
        a_status_raw = ""
        for k in query_keys:
            if not cm_status_raw and k in clomonitor_map:
                cm_status_raw = clomonitor_map[k]
            if not m_status_raw and k in maintainers_map:
                m_status_raw = maintainers_map[k]
            if not d_status_raw and k in devstats_map:
                d_status_raw = devstats_map[k]
            if not a_status_raw and k in artwork_map:
                a_status_raw = artwork_map[k]
        # For Landscape, explicitly show '-' when missing to flag anomaly
        l_status = normalize_status(l_status_raw) if l_status_raw else "-"
        # For other sources, keep empty when missing
        cm_status = normalize_status(cm_status_raw) if cm_status_raw else ""
        m_status = normalize_status(m_status_raw) if m_status_raw else ""
        d_status = normalize_status(d_status_raw) if d_status_raw else ""
        a_status = normalize_status(a_status_raw) if a_status_raw else ""

        all_rows.append((name, norm_pcc, l_status, cm_status, m_status, d_status, a_status))

        # Anomaly criteria:
        # - Any missing value in any source (displayed as '-' later; Landscape missing is already '-')
        # - OR any source present and different from PCC
        landscape_mismatch = (l_status == "-") or (l_status != norm_pcc)
        clomonitor_mismatch = bool(cm_status) and (cm_status != norm_pcc)
        maintainers_mismatch = bool(m_status) and (m_status != norm_pcc)
        devstats_mismatch = bool(d_status) and (d_status != norm_pcc)
        artwork_mismatch = bool(a_status) and (a_status != norm_pcc)
        any_missing = (l_status == "-") or (not cm_status) or (not m_status) or (not d_status) or (not a_status)

        if any_missing or landscape_mismatch or clomonitor_mismatch or maintainers_mismatch or devstats_mismatch or artwork_mismatch:
            combined_rows.append((name, norm_pcc, l_status, cm_status, m_status, d_status, a_status))

    write_audit_markdown(combined_rows)
    write_full_status_markdown(all_rows)
    print(f"Wrote audit with {len(combined_rows)} mismatches to {AUDIT_OUTPUT_PATH}")


if __name__ == "__main__":
    main()


