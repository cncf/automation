#!/bin/bash
#
# Local SBOM Generation Script
# This script mimics the GitHub Actions workflow for local testing
#
# Prerequisites:
#   - Go 1.22+
#   - git
#   - gh CLI (GitHub CLI) - for API access
#   - jq
#   - yq (https://github.com/mikefarah/yq)
#
# Usage:
#   ./generate-sbom-local.sh                           # Process all projects
#   ./generate-sbom-local.sh kubernetes/kubernetes     # Process specific repo
#   ./generate-sbom-local.sh --force kubernetes/kubernetes  # Force regenerate
#
# Environment variables:
#   GH_TOKEN or GITHUB_TOKEN - GitHub token for API access
#   MAX_RELEASES - Maximum releases to process per repo (default: 3)
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
DATA_FILE="$ROOT_DIR/supply-chain/util/data/repositories.yaml"
SBOM_BASE_DIR="$ROOT_DIR/supply-chain/sbom"

# Parse arguments
FORCE_REGENERATE="false"
PROJECT_FILTER=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --force|-f)
      FORCE_REGENERATE="true"
      shift
      ;;
    --help|-h)
      echo "Usage: $0 [--force] [owner/repo]"
      echo ""
      echo "Options:"
      echo "  --force, -f    Force regenerate existing SBOMs"
      echo "  --help, -h     Show this help message"
      echo ""
      echo "Examples:"
      echo "  $0                           # Process all projects"
      echo "  $0 kubernetes/kubernetes     # Process specific repo"
      echo "  $0 --force coredns/coredns   # Force regenerate for coredns"
      exit 0
      ;;
    *)
      PROJECT_FILTER="$1"
      shift
      ;;
  esac
done

# Set token
GH_TOKEN="${GH_TOKEN:-$GITHUB_TOKEN}"
if [ -z "$GH_TOKEN" ]; then
  echo "Warning: No GitHub token found. API rate limits may apply."
  echo "Set GH_TOKEN or GITHUB_TOKEN environment variable for higher limits."
fi

MAX_RELEASES="${MAX_RELEASES:-3}"

# Check prerequisites
check_prerequisites() {
  local missing=()

  if ! command -v go &> /dev/null; then
    missing+=("go")
  fi

  if ! command -v git &> /dev/null; then
    missing+=("git")
  fi

  if ! command -v gh &> /dev/null; then
    missing+=("gh (GitHub CLI)")
  fi

  if ! command -v jq &> /dev/null; then
    missing+=("jq")
  fi

  if ! command -v yq &> /dev/null; then
    missing+=("yq")
  fi

  if [ ${#missing[@]} -gt 0 ]; then
    echo "Error: Missing required tools: ${missing[*]}"
    echo ""
    echo "Installation:"
    echo "  go:  https://golang.org/dl/"
    echo "  gh:  https://cli.github.com/"
    echo "  jq:  https://stedolan.github.io/jq/"
    echo "  yq:  https://github.com/mikefarah/yq"
    exit 1
  fi
}

# Install bom tool if not present
install_bom() {
  if ! command -v bom &> /dev/null; then
    echo "Installing bom tool..."
    go install sigs.k8s.io/bom/cmd/bom@latest
    export PATH="$PATH:$(go env GOPATH)/bin"
  fi

  if ! command -v bom &> /dev/null; then
    echo "Error: Failed to install bom tool"
    exit 1
  fi

  echo "Using bom: $(which bom)"
}

# Generate SBOM for a specific tag
generate_sbom() {
  local OWNER="$1"
  local REPO="$2"
  local PROJECT_NAME="$3"
  local TAG="$4"

  local SANITIZED_PROJECT=$(echo "$PROJECT_NAME" | tr '[:upper:]' '[:lower:]' | tr ' ' '-' | tr -cd '[:alnum:]-')
  local VERSION=$(echo "$TAG" | sed 's/^v//')
  local SBOM_DIR="${SBOM_BASE_DIR}/${SANITIZED_PROJECT}/${REPO}/${VERSION}"
  local SBOM_FILE="${SBOM_DIR}/${REPO}.json"

  # Check if SBOM already exists
  if [ -f "$SBOM_FILE" ] && [ "$FORCE_REGENERATE" != "true" ]; then
    echo "  SBOM already exists: $SBOM_FILE, skipping..."
    return 1
  fi

  echo "  Generating SBOM for $OWNER/$REPO@$TAG..."

  # Clone the repository at specific tag
  local TEMP_DIR=$(mktemp -d)
  trap "rm -rf '$TEMP_DIR'" EXIT

  if ! git clone --depth 1 --branch "$TAG" "https://github.com/${OWNER}/${REPO}.git" "$TEMP_DIR" 2>/dev/null; then
    echo "  Failed to clone $OWNER/$REPO@$TAG, skipping..."
    rm -rf "$TEMP_DIR"
    return 1
  fi

  # Create output directory
  mkdir -p "$SBOM_DIR"

  # Generate SBOM with bom tool
  if bom generate --format json --output "$SBOM_FILE" "$TEMP_DIR" 2>/dev/null; then
    echo "  Successfully generated SBOM: $SBOM_FILE"
    rm -rf "$TEMP_DIR"
    return 0
  else
    echo "  Failed to generate SBOM for $OWNER/$REPO@$TAG"
    rm -rf "$TEMP_DIR"
    return 1
  fi
}

# Process a single repository
process_repository() {
  local OWNER="$1"
  local REPO="$2"
  local PROJECT_NAME="$3"
  local PROCESSED=0

  echo ""
  echo "=========================================="
  echo "Processing: $PROJECT_NAME ($OWNER/$REPO)"
  echo "=========================================="

  # Get releases from GitHub API
  local RELEASES
  RELEASES=$(gh api "repos/${OWNER}/${REPO}/releases" --paginate -q '.[0:50]' 2>/dev/null || echo "[]")

  if [ "$RELEASES" == "[]" ] || [ -z "$RELEASES" ]; then
    echo "No releases found, trying tags..."
    local TAGS
    TAGS=$(gh api "repos/${OWNER}/${REPO}/tags" --paginate -q '.[0:20] | .[].name' 2>/dev/null || echo "")

    if [ -z "$TAGS" ]; then
      echo "No tags found, skipping..."
      return 0
    fi

    # Process tags as releases
    for TAG in $TAGS; do
      # Filter out pre-releases
      if echo "$TAG" | grep -qiE '[-\.](alpha|beta|rc|pre|dev|snapshot|nightly|canary|test|draft|wip)[0-9]*'; then
        echo "  Skipping pre-release tag: $TAG"
        continue
      fi

      # Only process semver-like tags
      if ! echo "$TAG" | grep -qE '^v?[0-9]+\.[0-9]+'; then
        echo "  Skipping non-semver tag: $TAG"
        continue
      fi

      if generate_sbom "$OWNER" "$REPO" "$PROJECT_NAME" "$TAG"; then
        PROCESSED=$((PROCESSED + 1))
      fi

      if [ "$PROCESSED" -ge "$MAX_RELEASES" ]; then
        echo "  Processed $MAX_RELEASES releases, stopping..."
        break
      fi
    done
  else
    # Process releases - filter stable releases
    readarray -t RELEASE_TAGS < <(echo "$RELEASES" | jq -r '.[] | select(.draft == false and .prerelease == false) | .tag_name')

    for TAG in "${RELEASE_TAGS[@]}"; do
      # Additional filter for pre-release patterns
      if echo "$TAG" | grep -qiE '[-\.](alpha|beta|rc|pre|dev|snapshot|nightly|canary|test|draft|wip)[0-9]*'; then
        echo "  Skipping pre-release tag: $TAG"
        continue
      fi

      if generate_sbom "$OWNER" "$REPO" "$PROJECT_NAME" "$TAG"; then
        PROCESSED=$((PROCESSED + 1))
      fi

      if [ "$PROCESSED" -ge "$MAX_RELEASES" ]; then
        echo "  Processed $MAX_RELEASES releases, stopping..."
        break
      fi
    done
  fi

  echo "Processed $PROCESSED releases for $OWNER/$REPO"
}

# Generate index of all SBOMs
generate_index() {
  echo ""
  echo "=========================================="
  echo "Generating SBOM index"
  echo "=========================================="

  local INDEX_FILE="$SBOM_BASE_DIR/index.json"

  # Check if there are any SBOMs
  local SBOM_COUNT
  SBOM_COUNT=$(find "$SBOM_BASE_DIR" -name "*.json" -type f ! -name "index.json" 2>/dev/null | wc -l)

  if [ "$SBOM_COUNT" -eq 0 ]; then
    echo "No SBOMs found, creating empty index..."
    echo '{"generated_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'", "sboms": []}' > "$INDEX_FILE"
    return 0
  fi

  # Generate index using jq
  find "$SBOM_BASE_DIR" -name "*.json" -type f ! -name "index.json" | sort | while read -r SBOM; do
    REL_PATH="${SBOM#$SBOM_BASE_DIR/}"
    PROJECT=$(echo "$REL_PATH" | cut -d'/' -f1)
    REPO=$(echo "$REL_PATH" | cut -d'/' -f2)
    VERSION=$(echo "$REL_PATH" | cut -d'/' -f3)
    echo "{\"project\": \"$PROJECT\", \"repo\": \"$REPO\", \"version\": \"$VERSION\", \"path\": \"$REL_PATH\"}"
  done | jq -s '{"generated_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'", "sboms": .}' > "$INDEX_FILE"

  echo "Index generated: $INDEX_FILE"
  echo "Total SBOMs: $SBOM_COUNT"
}

# Main execution
main() {
  echo "SBOM Generator for CNCF Projects"
  echo "================================="
  echo ""
  echo "Settings:"
  echo "  Force regenerate: $FORCE_REGENERATE"
  echo "  Project filter: ${PROJECT_FILTER:-all}"
  echo "  Max releases per repo: $MAX_RELEASES"
  echo "  Output directory: $SBOM_BASE_DIR"
  echo ""

  check_prerequisites
  install_bom

  # Ensure data file exists
  if [ ! -f "$DATA_FILE" ]; then
    echo "Error: Repository data file not found: $DATA_FILE"
    exit 1
  fi

  # Get repositories to process
  if [ -n "$PROJECT_FILTER" ]; then
    OWNER=$(echo "$PROJECT_FILTER" | cut -d'/' -f1)
    REPO=$(echo "$PROJECT_FILTER" | cut -d'/' -f2)
    REPOS=$(yq -o=json '.repositories | map(select(.owner == "'"$OWNER"'" and .repo == "'"$REPO"'"))' "$DATA_FILE")
  else
    REPOS=$(yq -o=json '.repositories' "$DATA_FILE")
  fi

  # Process each repository
  local REPO_COUNT
  REPO_COUNT=$(echo "$REPOS" | jq 'length')

  if [ "$REPO_COUNT" -eq 0 ]; then
    echo "No repositories found matching filter: $PROJECT_FILTER"
    exit 1
  fi

  echo "Found $REPO_COUNT repositories to process"

  echo "$REPOS" | jq -c '.[]' | while read -r REPO_JSON; do
    OWNER=$(echo "$REPO_JSON" | jq -r '.owner')
    REPO=$(echo "$REPO_JSON" | jq -r '.repo')
    NAME=$(echo "$REPO_JSON" | jq -r '.name')

    process_repository "$OWNER" "$REPO" "$NAME"
  done

  generate_index

  echo ""
  echo "=========================================="
  echo "SBOM generation complete!"
  echo "=========================================="
}

main "$@"
