#!/usr/bin/env bash

# Generates Waybill SPDX SBOMs for this repository's own tooling:
#   - one SBOM per component (Go modules and Python utilities)
#   - one repo-wide SBOM from the committed tree (git archive HEAD)
#   - one SPDX inventory of the GitHub Actions / CI generation chain

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT_DIR/tooling-sbom-out}"
WAYBILL_VERSION="${WAYBILL_VERSION:-v0.2.0}"
REPO_URL="${REPO_URL:-https://github.com/cncf/automation.git}"
ROOT_PACKAGE_NAME="${ROOT_PACKAGE_NAME:-cncf/automation}"
SKIP_REPO_SBOM="false"
SKIP_COMPONENT_SBOMS="false"
export WAYBILL_VERSION

# Go module components (each has its own go.mod).
GO_COMPONENTS=(
  ci/cloudrunners
  ci/gha-runner-vm
  ci/gha-runner-vm-oci
  utilities/dot-project
  utilities/labeler
  utilities/landscape-mcp-server
  utilities/landscape-sync
)

# Python components. Format: <path>[:<exclude>,<exclude>...]
PY_COMPONENTS=(
  "Ambassadors"
  "Kubestronaut:Rendering,kubestronauts-coupons"
  "Kubestronaut/Rendering"
  "Kubestronaut/kubestronauts-coupons"
  "utilities/audit_project_lifecycle_across_tools"
)

usage() {
  cat <<'EOF'
Usage: .github/scripts/generate-tooling-sbom.sh [--output-dir <dir>] [--skip-repo-sbom] [--skip-component-sboms]

Generates:
  - components/<slug>.spdx.json: one SBOM per Go module / Python utility
  - tooling-repo.spdx.json:      SBOM for this repository's committed tree
  - tooling-ci.spdx.json:        SBOM for the GitHub Actions / CI generation chain
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-dir)
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --skip-repo-sbom)
      SKIP_REPO_SBOM="true"
      shift
      ;;
    --skip-component-sboms)
      SKIP_COMPONENT_SBOMS="true"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

WORKFLOW_DIR="$ROOT_DIR/.github/workflows"
ACTIONS_DIR="$ROOT_DIR/.github/actions"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
REPO_COMMIT="${GITHUB_SHA:-$(git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null || echo unknown)}"
GO_TOOLCHAIN_VERSION="$(cat "$ROOT_DIR/.go-version" 2>/dev/null || echo unknown)"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1" >&2
    exit 1
  fi
}

spdx_id() {
  echo "$1" | tr '[:lower:]' '[:upper:]' | sed 's/[^A-Z0-9]/-/g'
}

slugify() {
  echo "$1" | tr '[:upper:]' '[:lower:]' | sed 's|[/_]|-|g'
}

waybill_scan() {
  local path="$1"
  local root_name="$2"
  local output_file="$3"
  shift 3

  waybill sbom scan \
    --path "$path" \
    --format spdx-2.3-json \
    --root-name "$root_name" \
    --root-version "$REPO_COMMIT" \
    --repo "$REPO_URL" \
    --git-ref "$REPO_COMMIT" \
    --output "$output_file" \
    "$@"
}

generate_component_sboms() {
  if [[ "$SKIP_COMPONENT_SBOMS" == "true" ]]; then
    echo "Skipping per-component SBOM generation"
    return
  fi

  require_cmd waybill
  mkdir -p "$OUTPUT_DIR/components"

  local component
  for component in "${GO_COMPONENTS[@]}"; do
    echo "Generating component SBOM: $component"
    waybill_scan \
      "$ROOT_DIR/$component" \
      "${ROOT_PACKAGE_NAME}/${component}" \
      "$OUTPUT_DIR/components/$(slugify "$component").spdx.json"
  done

  local entry path excludes exclude_args
  for entry in "${PY_COMPONENTS[@]}"; do
    path="${entry%%:*}"
    excludes=""
    [[ "$entry" == *:* ]] && excludes="${entry#*:}"

    exclude_args=()
    if [[ -n "$excludes" ]]; then
      local IFS=','
      local ex
      for ex in $excludes; do
        exclude_args+=(--exclude-path "$ex")
      done
      unset IFS
    fi

    echo "Generating component SBOM: $path"
    waybill_scan \
      "$ROOT_DIR/$path" \
      "${ROOT_PACKAGE_NAME}/${path}" \
      "$OUTPUT_DIR/components/$(slugify "$path").spdx.json" \
      "${exclude_args[@]}"
  done
}

generate_repo_sbom() {
  local output_file="$OUTPUT_DIR/tooling-repo.spdx.json"
  local temp_dir

  if [[ "$SKIP_REPO_SBOM" == "true" ]]; then
    echo "Skipping repository tooling SBOM generation"
    return
  fi

  require_cmd waybill
  require_cmd git

  echo "Generating repository-wide tooling SBOM"
  temp_dir="$(mktemp -d)"
  git -C "$ROOT_DIR" archive HEAD -- . ':(exclude)tooling-sbom' | tar -x -C "$temp_dir"

  if ! waybill_scan "$temp_dir" "$ROOT_PACKAGE_NAME" "$output_file"; then
    rm -rf "$temp_dir"
    return 1
  fi

  rm -rf "$temp_dir"
}

package_json() {
  local spdx_id_value="$1"
  local name="$2"
  local version="$3"
  local download_location="$4"
  local supplier="$5"
  local homepage="$6"
  local purpose="$7"

  jq -n \
    --arg spdx_id "$spdx_id_value" \
    --arg name "$name" \
    --arg version "$version" \
    --arg download_location "$download_location" \
    --arg supplier "$supplier" \
    --arg homepage "$homepage" \
    --arg purpose "$purpose" \
    '{
      SPDXID: $spdx_id,
      name: $name,
      versionInfo: $version,
      downloadLocation: $download_location,
      filesAnalyzed: false,
      supplier: $supplier,
      homepage: $homepage,
      primaryPackagePurpose: $purpose,
      licenseConcluded: "NOASSERTION",
      licenseDeclared: "NOASSERTION",
      copyrightText: "NOASSERTION"
    }'
}

generate_ci_sbom() {
  local output_file="$OUTPUT_DIR/tooling-ci.spdx.json"
  local temp_dir
  temp_dir="$(mktemp -d)"

  require_cmd jq

  local packages_file="$temp_dir/packages.jsonl"
  local relationships_file="$temp_dir/relationships.jsonl"
  : > "$packages_file"
  : > "$relationships_file"

  echo "Generating CI generation-chain SBOM"

  local root_spdx="SPDXRef-Package-Tooling-CI"
  package_json \
    "$root_spdx" \
    "${ROOT_PACKAGE_NAME}-ci-generation-chain" \
    "$REPO_COMMIT" \
    "$REPO_URL" \
    "Organization: CNCF" \
    "${REPO_URL%.git}" \
    "APPLICATION" >> "$packages_file"

  jq -n --arg source "SPDXRef-DOCUMENT" --arg target "$root_spdx" \
    '{spdxElementId:$source, relationshipType:"DESCRIBES", relatedSpdxElement:$target}' >> "$relationships_file"

  add_dependency() {
    local spdx_ref="$1"
    jq -n --arg source "$root_spdx" --arg target "$spdx_ref" \
      '{spdxElementId:$source, relationshipType:"DEPENDS_ON", relatedSpdxElement:$target}' >> "$relationships_file"
  }

  # Every `uses:` reference across workflows and composite actions.
  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue

    local spdx_ref name version download_location homepage supplier
    local purpose="FRAMEWORK"

    if [[ "$ref" == ./* ]]; then
      spdx_ref="SPDXRef-$(spdx_id "$ref")"
      name="Local action ${ref}"
      version="$REPO_COMMIT"
      download_location="$REPO_URL"
      homepage="${REPO_URL%.git}/blob/${REPO_COMMIT}/${ref#./}"
      supplier="Organization: CNCF"
      purpose="SOURCE"
    else
      local action_name="${ref%@*}"
      local action_version="${ref##*@}"
      spdx_ref="SPDXRef-$(spdx_id "$action_name-$action_version")"
      name="GitHub Action ${action_name}"
      version="$action_version"
      download_location="https://github.com/$(echo "$action_name" | cut -d/ -f1-2)"
      homepage="$download_location"
      supplier="Organization: GitHub"
    fi

    package_json \
      "$spdx_ref" \
      "$name" \
      "$version" \
      "$download_location" \
      "$supplier" \
      "$homepage" \
      "$purpose" >> "$packages_file"

    add_dependency "$spdx_ref"
  done < <(
    { sed -nE 's/^[[:space:]-]*uses:[[:space:]]*([^[:space:]#]+).*$/\1/p' \
        "$WORKFLOW_DIR"/*.yml "$WORKFLOW_DIR"/*.yaml 2>/dev/null
      find "$ACTIONS_DIR" -name 'action.yml' -o -name 'action.yaml' 2>/dev/null | \
        xargs -r sed -nE 's/^[[:space:]-]*uses:[[:space:]]*([^[:space:]#]+).*$/\1/p'
    } | sort -u
  )

  package_json \
    "SPDXRef-Package-ToolingWorkflow" \
    "Local workflow .github/workflows/generate-tooling-sbom.yml" \
    "$REPO_COMMIT" \
    "$REPO_URL" \
    "Organization: CNCF" \
    "${REPO_URL%.git}/blob/${REPO_COMMIT}/.github/workflows/generate-tooling-sbom.yml" \
    "SOURCE" >> "$packages_file"
  add_dependency "SPDXRef-Package-ToolingWorkflow"

  package_json \
    "SPDXRef-Package-ToolingScript" \
    "Local script .github/scripts/generate-tooling-sbom.sh" \
    "$REPO_COMMIT" \
    "$REPO_URL" \
    "Organization: CNCF" \
    "${REPO_URL%.git}/blob/${REPO_COMMIT}/.github/scripts/generate-tooling-sbom.sh" \
    "SOURCE" >> "$packages_file"
  add_dependency "SPDXRef-Package-ToolingScript"

  package_json \
    "SPDXRef-Package-Waybill" \
    "Waybill" \
    "$WAYBILL_VERSION" \
    "https://github.com/kusari-oss/waybill/releases/download/${WAYBILL_VERSION}/waybill-${WAYBILL_VERSION}-x86_64-unknown-linux-gnu.tar.gz" \
    "Organization: Kusari" \
    "https://github.com/kusari-oss/waybill" \
    "APPLICATION" >> "$packages_file"
  add_dependency "SPDXRef-Package-Waybill"

  package_json \
    "SPDXRef-Package-GoToolchain" \
    "Go toolchain" \
    "$GO_TOOLCHAIN_VERSION" \
    "https://go.dev/dl/" \
    "Organization: Go team" \
    "https://go.dev/" \
    "APPLICATION" >> "$packages_file"
  add_dependency "SPDXRef-Package-GoToolchain"

  jq -s \
    --arg timestamp "$TIMESTAMP" \
    --arg name "${ROOT_PACKAGE_NAME} CI generation chain" \
    --arg namespace "${REPO_URL%.git}/tooling-ci/${REPO_COMMIT}" \
    --slurpfile packages "$packages_file" \
    --slurpfile relationships "$relationships_file" \
    -n \
    '{
      spdxVersion: "SPDX-2.3",
      dataLicense: "CC0-1.0",
      SPDXID: "SPDXRef-DOCUMENT",
      name: $name,
      documentNamespace: $namespace,
      creationInfo: {
        created: $timestamp,
        creators: [
          "Tool: .github/scripts/generate-tooling-sbom.sh",
          ("Tool: Waybill " + env.WAYBILL_VERSION)
        ]
      },
      packages: $packages,
      relationships: $relationships
    }' > "$output_file"

  rm -rf "$temp_dir"
}

main() {
  require_cmd jq
  mkdir -p "$OUTPUT_DIR"

  generate_component_sboms
  generate_repo_sbom
  generate_ci_sbom

  echo "Generated tooling SBOM artifacts in: $OUTPUT_DIR"
  find "$OUTPUT_DIR" -name '*.json' | sort
}

main "$@"
