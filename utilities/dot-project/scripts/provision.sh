#!/usr/bin/env bash
# provision.sh — Create and bootstrap a .project repo for a CNCF project.
#
# Usage:
#   ./scripts/provision.sh --org <org> --name <name> [--repo <repo>] [options]
#   ./scripts/provision.sh --batch <file> [options]
#
# Options:
#   --org <org>           GitHub organization (e.g., "project-copacetic")
#   --name <name>         Project display name (e.g., "Copacetic")
#   --repo <repo>         Primary repo name (defaults to org name)
#   --batch <file>        Batch mode: read org|name|repo from file (pipe-delimited)
#   --dry-run             Print what would be done without making changes
#   --skip-secrets        Skip setting repository secrets
#   --skip-protection     Skip setting branch protection rules
#   --skip-issue          Skip creating onboarding issue
#   --force               Force regeneration of scaffold files (overwrites auxiliary files)
#   --bootstrap-bin <p>   Path to bootstrap binary (default: ./bootstrap)
#   -h, --help            Show this help message
#
# Required environment variables:
#   GITHUB_TOKEN           GitHub token for gh CLI (set via gh auth login)
#   LANDSCAPE_REPO_TOKEN   Token for landscape repo PR creation
#
# Optional environment variables:
#   LFX_AUTH_TOKEN         LFX auth token for maintainer verification
#
# Batch file format (one project per line, # for comments):
#   org|name|repo
#   project-copacetic|Copacetic|copacetic
#   grpc|gRPC|grpc

set -euo pipefail

# Load .env from CWD if present (KEY=VALUE, skips comments and blank lines)
if [[ -f .env ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Strip Windows carriage returns (\r)
        line="${line//$'\r'/}"
        # Skip blank lines and comments
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
        # Export only valid KEY=VALUE lines (no eval, no command substitution)
        if [[ "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; then
            export "${line?}"
        fi
    done < .env
fi

# Defaults
DRY_RUN=false
SKIP_SECRETS=false
SKIP_PROTECTION=false
SKIP_ISSUE=false
FORCE=false
BOOTSTRAP_BIN="./bootstrap"
BATCH_FILE=""
ORG=""
NAME=""
REPO=""

die() { echo "Error: $*" >&2; exit 1; }
info() { echo "==> $*" >&2; }
warn() { echo "WARNING: $*" >&2; }
dry() { if $DRY_RUN; then echo "[dry-run] $*" >&2; return 0; fi; return 1; }

usage() {
    sed -n '2,/^$/{ s/^# //; s/^#//; p }' "$0"
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --org)          ORG="$2"; shift 2 ;;
        --name)         NAME="$2"; shift 2 ;;
        --repo)         REPO="$2"; shift 2 ;;
        --batch)        BATCH_FILE="$2"; shift 2 ;;
        --dry-run)      DRY_RUN=true; shift ;;
        --skip-secrets) SKIP_SECRETS=true; shift ;;
        --skip-protection) SKIP_PROTECTION=true; shift ;;
        --force)        FORCE=true; shift ;;
        --skip-issue)   SKIP_ISSUE=true; shift ;;
        --bootstrap-bin) BOOTSTRAP_BIN="$2"; shift 2 ;;
        -h|--help)      usage ;;
        *)              die "Unknown option: $1" ;;
    esac
done

# ──────────────────────────────────────────────
# Normalize GitHub URL inputs
# ──────────────────────────────────────────────

# normalize_github_url strips a full GitHub URL down to its org (and optionally repo) parts.
# Accepts: https://github.com/org, https://github.com/org/repo, or plain "org"
# Sets ORG (and REPO if a repo segment is present and REPO wasn't explicitly provided).
normalize_github_url() {
    local value="$1"
    local field="$2"  # "org" or "repo"

    # Strip trailing slashes
    value="${value%/}"

    if [[ "$value" =~ ^https?://github\.com/([^/]+)(/([^/]+))?$ ]]; then
        local parsed_org="${BASH_REMATCH[1]}"
        local parsed_repo="${BASH_REMATCH[3]}"

        if [[ "$field" == "org" ]]; then
            ORG="$parsed_org"
            # If a repo segment was in the URL and --repo wasn't explicitly set, use it
            if [[ -n "$parsed_repo" && -z "$REPO" ]]; then
                REPO="$parsed_repo"
                info "Extracted --repo '${REPO}' from GitHub URL"
            fi
            info "Extracted --org '${ORG}' from GitHub URL"
        elif [[ "$field" == "repo" ]]; then
            if [[ -n "$parsed_repo" ]]; then
                REPO="$parsed_repo"
            else
                REPO="$parsed_org"  # URL was github.com/org, treat as repo name
            fi
            info "Extracted --repo '${REPO}' from GitHub URL"
        fi
    elif [[ "$value" =~ ^https?:// ]]; then
        die "--${field} looks like a URL but is not a valid GitHub URL (expected https://github.com/<org>[/<repo>]): ${value}"
    fi
    # Otherwise it's already a plain name — no transformation needed
}

normalize_github_url "$ORG" "org"
if [[ -n "$REPO" ]]; then
    normalize_github_url "$REPO" "repo"
fi

# ──────────────────────────────────────────────
# Prerequisites
# ──────────────────────────────────────────────

check_prerequisites() {
    # gh CLI
    if ! command -v gh &>/dev/null; then
        die "gh CLI not found. Install from https://cli.github.com/"
    fi
    if ! gh auth status &>/dev/null; then
        die "gh CLI not authenticated. Run 'gh auth login' first."
    fi

    # Bootstrap binary
    if [[ ! -x "$BOOTSTRAP_BIN" ]]; then
        # Try building it
        if [[ -f "cmd/bootstrap/main.go" ]]; then
            info "Building bootstrap binary..."
            if ! dry "would build bootstrap binary"; then
                go build -o bootstrap ./cmd/bootstrap
                BOOTSTRAP_BIN="./bootstrap"
            fi
        else
            die "Bootstrap binary not found at '$BOOTSTRAP_BIN'. Build with: go build -o bootstrap ./cmd/bootstrap"
        fi
    fi

    # Required secrets for non-dry-run, non-skip-secrets
    if ! $DRY_RUN && ! $SKIP_SECRETS; then
        if [[ -z "${LANDSCAPE_REPO_TOKEN:-}" ]]; then
            die "LANDSCAPE_REPO_TOKEN environment variable is required (or use --skip-secrets)"
        fi
    fi
}

# ──────────────────────────────────────────────
# Secret management
# ──────────────────────────────────────────────

# set_secret sets a GitHub Actions secret on a repo.
# If it fails (e.g., enterprise policy blocks API access), prints manual instructions.
set_secret() {
    local repo="$1"
    local secret_name="$2"
    local secret_value="$3"

    if echo "$secret_value" | gh secret set "$secret_name" --repo "$repo" 2>/dev/null; then
        return 0
    fi

    warn "Could not set secret $secret_name on ${repo}"
    warn "This is likely due to an enterprise policy blocking API-based secret management."
    warn "Set it manually:"
    warn "  Repo:  https://github.com/${repo}/settings/secrets/actions"
    warn "  Org:   https://github.com/organizations/${repo%%/*}/settings/secrets/actions"
    return 1
}

# ──────────────────────────────────────────────
# Provision a single project
# ──────────────────────────────────────────────

provision_project() {
    local org="$1"
    local name="$2"
    local repo="${3:-$org}"

    # Normalize org if it's a GitHub URL (e.g., https://github.com/tokenetes/tokenetes)
    if [[ "$org" =~ ^https?://github\.com/([^/]+)(/([^/]+))?/?$ ]]; then
        org="${BASH_REMATCH[1]}"
        if [[ -n "${BASH_REMATCH[3]}" && ( -z "$repo" || "$repo" == "$1" ) ]]; then
            repo="${BASH_REMATCH[3]}"
        fi
    elif [[ "$org" =~ ^https?:// ]]; then
        warn "org '${org}' looks like a URL but is not a recognized GitHub URL; using as-is"
    fi

    # Normalize repo if it's a GitHub URL
    if [[ "$repo" =~ ^https?://github\.com/([^/]+)(/([^/]+))?/?$ ]]; then
        if [[ -n "${BASH_REMATCH[3]}" ]]; then
            repo="${BASH_REMATCH[3]}"
        else
            repo="${BASH_REMATCH[1]}"
        fi
    elif [[ "$repo" =~ ^https?:// ]]; then
        warn "repo '${repo}' looks like a URL but is not a recognized GitHub URL; using as-is"
    fi

    local target_repo="${org}/.project"

    info "Provisioning: ${target_repo} (name: ${name}, primary repo: ${repo})"

    # Step 1: Create repo if it doesn't exist
    if gh repo view "$target_repo" &>/dev/null; then
        info "  Repo ${target_repo} already exists, skipping creation"
    else
        if dry "would create repo: ${target_repo}"; then
            :
        else
            info "  Creating repo: ${target_repo}"
            gh repo create "$target_repo" \
                --public \
                --description "Project metadata for ${name} - CNCF .project automation" \
                || die "Failed to create repo ${target_repo}"
        fi
    fi

    # Step 2: Clone/init to temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf '$tmp_dir'" EXIT

    if dry "would clone ${target_repo} to ${tmp_dir}"; then
        :
    else
        info "  Cloning ${target_repo}..."
        if ! gh repo clone "$target_repo" "$tmp_dir" -- --depth=1 2>/dev/null; then
            # Empty repo - initialize manually
            info "  Empty repo detected, initializing..."
            git -C "$tmp_dir" init -b main
            git -C "$tmp_dir" remote add origin "https://github.com/${target_repo}.git"
        fi
    fi

    # Step 3: Run bootstrap
    if dry "would run bootstrap: ${BOOTSTRAP_BIN} -name '${name}' -github-org '${org}' -github-repo '${repo}' -output-dir '${tmp_dir}'"; then
        :
    else
        info "  Running bootstrap..."
        local bootstrap_args=(-name "$name" -github-org "$org" -github-repo "$repo" -output-dir "$tmp_dir")
        if $FORCE; then
            bootstrap_args+=(-force)
        fi
        "$BOOTSTRAP_BIN" "${bootstrap_args[@]}" \
            || die "Bootstrap failed for ${name}"
    fi

    # Step 4: Commit and push
    if dry "would commit and push to ${target_repo}"; then
        :
    else
        info "  Committing and pushing..."
        git -C "$tmp_dir" add -A
        git -C "$tmp_dir" \
            -c user.name="cncf-automation[bot]" \
            -c user.email="projects@cncf.io" \
            commit -m "Initial .project scaffold for ${name}" \
            || { info "  Nothing to commit (already up to date)"; }
        git -C "$tmp_dir" push -u origin main \
            || die "Failed to push to ${target_repo}"
    fi

    # Step 5: Set secrets
    if ! $SKIP_SECRETS; then
        if dry "would set secrets on ${target_repo}"; then
            :
        else
            info "  Setting secrets..."
            set_secret "$target_repo" "LANDSCAPE_REPO_TOKEN" "$LANDSCAPE_REPO_TOKEN" \
                || warn "Could not set LANDSCAPE_REPO_TOKEN (set manually via GitHub UI)"
            if [[ -n "${LFX_AUTH_TOKEN:-}" ]]; then
                set_secret "$target_repo" "LFX_AUTH_TOKEN" "$LFX_AUTH_TOKEN" \
                    || warn "Could not set LFX_AUTH_TOKEN (set manually via GitHub UI)"
            else
                warn "LFX_AUTH_TOKEN not set; skipping (maintainer verification won't work)"
            fi
        fi
    fi

    # Step 6: Branch protection
    if ! $SKIP_PROTECTION; then
        if dry "would set branch protection on ${target_repo}"; then
            :
        else
            info "  Setting branch protection..."
            gh api -X PUT "repos/${target_repo}/branches/main/protection" \
                --input - <<'PROTECTION' || warn "Branch protection failed (may require admin access)"
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["validate-project", "validate-maintainers"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1
  },
  "restrictions": null
}
PROTECTION
        fi
    fi

    # Step 7: Trigger validation workflow
    if dry "would trigger validation workflow on ${target_repo}"; then
        :
    else
        info "  Triggering validation workflow..."
        gh workflow run validate.yaml --repo "$target_repo" 2>/dev/null \
            || warn "Could not trigger workflow (may need a push event first)"
    fi

    info "  Done: https://github.com/${target_repo}"
    echo ""

    # Step 8: Create onboarding issue if there are TODOs
    if ! $SKIP_ISSUE; then
        create_onboarding_issue "$org" "$name" "$tmp_dir" "$target_repo"
    fi

    # Clean up trap for this iteration
    rm -rf "$tmp_dir"
    trap - EXIT
}

# ──────────────────────────────────────────────
# Onboarding issue
# ──────────────────────────────────────────────

create_onboarding_issue() {
    local org="$1"
    local name="$2"
    local tmp_dir="$3"
    local target_repo="$4"

    # Collect TODO lines from project.yaml header (lines before schema_version:)
    local todos=()
    if [[ -f "${tmp_dir}/project.yaml" ]]; then
        while IFS= read -r line; do
            [[ "$line" =~ ^schema_version: ]] && break
            if [[ "$line" =~ "#"[[:space:]]"TODO:" ]]; then
                # Strip leading "# TODO: " prefix
                todos+=("${line#*TODO: }")
            fi
        done < "${tmp_dir}/project.yaml"
    fi

    if [[ ${#todos[@]} -eq 0 ]]; then
        info "  No TODOs found in project.yaml — skipping onboarding issue"
        return 0
    fi

    if dry "would create onboarding issue on ${target_repo}"; then
        return 0
    fi

    # Fetch up to 3 org owners; fall back to maintainers.yaml handles
    local handles=()
    while IFS= read -r login; do
        handles+=("$login")
        [[ ${#handles[@]} -ge 3 ]] && break
    done < <(gh api "orgs/${org}/members?role=admin&per_page=10" --jq '.[].login' 2>/dev/null || true)

    if [[ ${#handles[@]} -eq 0 ]] && [[ -f "${tmp_dir}/maintainers.yaml" ]]; then
        while IFS= read -r login; do
            handles+=("${login#@}")
            [[ ${#handles[@]} -ge 3 ]] && break
        done < <(grep -E '^\s+-\s+github:' "${tmp_dir}/maintainers.yaml" | sed 's/.*github:[[:space:]]*//' || true)
    fi

    # Build mention string
    local mentions=""
    for h in "${handles[@]}"; do
        mentions+="@${h} "
    done

    # Ensure labels exist (--force is idempotent)
    gh label create "onboarding" --repo "$target_repo" \
        --color "0075ca" --description "Project onboarding tasks" --force 2>/dev/null || true
    gh label create "metadata" --repo "$target_repo" \
        --color "e4e669" --description "Project metadata" --force 2>/dev/null || true

    # Dedup: skip if open issue with same title already exists
    local title="Onboarding: complete ${name} .project setup"
    local existing
    existing=$(gh issue list --repo "$target_repo" --state open \
        --search "\"${title}\"" --json number --jq 'length' 2>/dev/null || echo "0")
    if [[ "$existing" -gt 0 ]]; then
        info "  Onboarding issue already exists — skipping"
        return 0
    fi

    # Build checklist body
    local checklist=""
    for todo in "${todos[@]}"; do
        checklist+="- [ ] ${todo}"$'\n'
    done

    local body
    body=$(cat <<EOF
Hi ${mentions}👋

The \`.project\` repo has been provisioned for **${name}**. A few items still need your attention:

## Checklist

${checklist}
Please open a PR against this repo to address each item. The validators will block merge until \`project.yaml\` and \`maintainers.yaml\` are complete.

> This issue was auto-generated by the CNCF .project provisioning tool.
EOF
)

    info "  Creating onboarding issue on ${target_repo}..."
    gh issue create \
        --repo "$target_repo" \
        --title "$title" \
        --body "$body" \
        --label "onboarding" \
        --label "metadata" \
        || warn "Could not create onboarding issue on ${target_repo}"
}

# ──────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────

main() {
    check_prerequisites

    if [[ -n "$BATCH_FILE" ]]; then
        # Batch mode
        if [[ ! -f "$BATCH_FILE" ]]; then
            die "Batch file not found: ${BATCH_FILE}"
        fi

        local count=0
        local failed=0
        while IFS='|' read -r b_org b_name b_repo; do
            # Skip comments and empty lines
            [[ "$b_org" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$b_org" ]] && continue

            # Trim whitespace
            b_org=$(echo "$b_org" | xargs)
            b_name=$(echo "$b_name" | xargs)
            b_repo=$(echo "${b_repo:-}" | xargs)

            if provision_project "$b_org" "$b_name" "$b_repo"; then
                ((count++))
            else
                warn "Failed to provision ${b_org}/${b_name}"
                ((failed++))
            fi
        done < "$BATCH_FILE"

        info "Batch complete: ${count} succeeded, ${failed} failed"
    else
        # Single mode
        [[ -z "$ORG" ]] && die "--org is required (or use --batch)"
        [[ -z "$NAME" ]] && die "--name is required (or use --batch)"

        provision_project "$ORG" "$NAME" "${REPO:-$ORG}"
    fi
}

main
