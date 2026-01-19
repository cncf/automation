#!/usr/bin/env python3
import os
import sys
import time
import json
from typing import Dict, List, Any

import requests

try:
    import yaml  # type: ignore
except Exception:
    print("Missing dependency: PyYAML. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(2)


API_URL = "https://api-gw.platform.linuxfoundation.org/project-service/v1/projects"
FOUNDATION_ID_CNCF = "a0941000002wBz4AAE"
PAGE_SIZE = 100
SLEEP_BETWEEN_CALLS_SECONDS = 0.2
DATASOURCES_DIR = os.path.join(os.getcwd(), "datasources")
OUTPUT_PATH = os.path.join(DATASOURCES_DIR, "pcc_projects.yaml")


def get_lfx_token() -> str:
    token = os.getenv("LFX_TOKEN", "").strip()
    if not token:
        print("Error: LFX_TOKEN environment variable is not set.", file=sys.stderr)
        sys.exit(1)
    return token


def fetch_page(session: requests.Session, offset: int, limit: int) -> Dict[str, Any]:
    params = {"offset": offset, "limit": limit}
    response = session.get(API_URL, params=params, timeout=30)
    response.raise_for_status()
    return response.json()


def project_to_record(p: Dict[str, Any]) -> Dict[str, Any]:
    return {
        "name": p.get("Name"),
        "slug": p.get("Slug"),
        "category": p.get("Category"),
        "status": p.get("Status"),
        "project_logo": p.get("ProjectLogo"),
        "repository_url": p.get("RepositoryURL"),
    }


def category_rank(category: Any) -> int:
    if category == "TAG":
        return 1
    if category == "Graduated":
        return 2
    if category == "Incubating":
        return 3
    if category == "Sandbox":
        return 4
    # Unknown/None go last
    return 99


def main() -> None:
    # Ensure datasources directory exists
    os.makedirs(DATASOURCES_DIR, exist_ok=True)
    token = get_lfx_token()
    session = requests.Session()
    session.headers.update(
        {
            "Authorization": f"Bearer {token}",
            "Accept": "application/json",
            "User-Agent": "project-status-audit/0.1 (+github actions)",
        }
    )

    offset = 0
    active_records: List[Dict[str, Any]] = []
    forming_records: List[Dict[str, Any]] = []
    archived_records: List[Dict[str, Any]] = []

    while True:
        data = fetch_page(session, offset=offset, limit=PAGE_SIZE)
        items: List[Dict[str, Any]] = data.get("Data") or []
        if not items:
            break

        for p in items:
            try:
                foundation = (p.get("Foundation") or {}).get("ID")
                if foundation != FOUNDATION_ID_CNCF:
                    continue
                status = p.get("Status")
                if status == "Active":
                    record = project_to_record(p)
                    active_records.append(record)
                elif status == "Formation - Exploratory":
                    # Forming projects are tracked separately with a reduced schema
                    forming_records.append(
                        {
                            "name": p.get("Name"),
                            "status": p.get("Status"),
                            "project_logo": p.get("ProjectLogo"),
                            "repository_url": p.get("RepositoryURL"),
                        }
                    )
                else:
                    # Anything not Active or Forming is considered archived/retired/other
                    archived_records.append(
                        {
                            "name": p.get("Name"),
                            "status": p.get("Status"),
                            "category": p.get("Category"),
                            "project_logo": p.get("ProjectLogo"),
                            "repository_url": p.get("RepositoryURL"),
                        }
                    )
            except Exception:
                # Skip malformed entries but continue
                continue

        offset += len(items)
        time.sleep(SLEEP_BETWEEN_CALLS_SECONDS)

    # Sort for stable output
    active_records.sort(key=lambda r: (category_rank(r.get("category")), (r.get("name") or "").lower()))
    forming_records.sort(key=lambda r: (r.get("name") or "").lower())
    archived_records.sort(key=lambda r: (r.get("name") or "").lower())

    # Group active projects by category, preserving desired order
    # Exclude "TAG" from categories per requirements
    category_keys = ["Graduated", "Incubating", "Sandbox"]
    categories: Dict[str, List[Dict[str, Any]]] = {k: [] for k in category_keys}
    for rec in active_records:
        cat = rec.get("category")
        if cat in categories:
            categories[cat].append(rec)
        else:
            # unknown categories ignored from grouping to mimic calendar focus
            pass

    output: Dict[str, Any] = {
        "source": "LFX PCC project-service",
        "foundation_id": FOUNDATION_ID_CNCF,
        "categories": categories,
        "forming_projects": forming_records,
        "archived_projects": archived_records,
    }

    with open(OUTPUT_PATH, "w", encoding="utf-8") as f:
        yaml.safe_dump(output, f, sort_keys=False, allow_unicode=True)

    print(
        f"Wrote {sum(len(v) for v in categories.values())} active projects, "
        f"{len(forming_records)} forming projects, and {len(archived_records)} archived projects to {OUTPUT_PATH}"
    )


if __name__ == "__main__":
    try:
        main()
    except requests.HTTPError as http_err:
        # Attempt to show API error payload for easier debugging
        try:
            payload = http_err.response.json()
            print(json.dumps(payload, indent=2), file=sys.stderr)
        except Exception:
            pass
        raise


