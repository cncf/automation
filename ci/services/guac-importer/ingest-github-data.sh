#!/bin/sh

# Exit script on any error, treat unset variables as an error, and propagate exit status through pipes
set -e -u -o pipefail


# --- Expected Environment Variables ---
# GUAC_GQL_ADDR: URL for the GUAC GraphQL endpoint.
#   Example: "http://guac-graphql.guac.svc.cluster.local:8080/query"
# GITHUB_REPO_URL_FULL: Full HTTPS URL for the Git repository to clone.
#   Example: "https://github.com/owner/repo.git"
# GIT_BRANCH_OR_TAG: The branch, tag, or commit hash to checkout and analyze.
#   Example: "main", "v1.2.3", "abcdef1234567890"
# GITHUB_TOKEN: (Optional) GitHub Personal Access Token for cloning private repositories
#   or for `guacone collect github` if used. This variable should contain the token itself.

# Optional variables for the 'guacone collect github' step:
# RUN_GUACONE_COLLECT_GITHUB: Set to "true" to run the 'guacone collect github' step. Default: "false"
# GITHUB_REPO_SHORT_NAME: Short repository name for 'guacone collect github' (owner/repo).
#   Example: "owner/repo"
# GITHUB_COLLECT_MODE: Mode for 'guacone collect github' ('release' or 'workflow'). Default: "release"

MANDATORY_VARS="GUAC_GQL_ADDR GITHUB_REPO_URL_FULL GIT_BRANCH_OR_TAG"
for VAR_NAME in $MANDATORY_VARS; do
  if [ -z "${!VAR_NAME:-}" ]; then
    echo "Error: Mandatory environment variable ${VAR_NAME} is not set."
    echo "Please ensure it is defined in your .env file with 'export' or passed directly."
    exit 1
  fi
done
echo "All mandatory variables are set."

# --- Configuration with Defaults ---
GUAC_GQL_ADDR="${GUAC_GQL_ADDR}"
GITHUB_REPO_URL_FULL="${GITHUB_REPO_URL_FULL}"
GIT_BRANCH_OR_TAG="${GIT_BRANCH_OR_TAG}"
# GITHUB_TOKEN is used directly if set, no default here.

RUN_GUACONE_COLLECT_GITHUB="${RUN_GUACONE_COLLECT_GITHUB:-false}"
GITHUB_REPO_SHORT_NAME="${GITHUB_REPO_SHORT_NAME:-}" # Required if RUN_GUACONE_COLLECT_GITHUB is true
GITHUB_COLLECT_MODE="${GITHUB_COLLECT_MODE:-release}"

# --- Tool Paths (expected to be in PATH within the Docker image) ---
GIT_BIN="git"
SYFT_BIN="syft"
GUACONE_BIN="guacone"

WORKSPACE_BASE="tmp/guac_ingest_ws" # Defined here!
WORKSPACE_DIR=""

# --- Helper Functions ---
check_command() {
  if ! command -v "$1" > /dev/null; then
    echo "Error: Command '$1' not found in PATH. Please ensure it's installed."
    exit 1
  fi
}

cleanup_workspace() {
  # Nur aufräumen, wenn WORKSPACE_DIR gesetzt wurde und ein Verzeichnis ist
  if [ -n "${WORKSPACE_DIR:-}" ] && [ -d "${WORKSPACE_DIR}" ]; then
    echo "Cleaning up unique workspace: ${WORKSPACE_DIR}"
   # rm -rf "${WORKSPACE_DIR}"
  else
    echo "No unique workspace directory to clean up, or WORKSPACE_DIR was not properly set."
  fi
}
# Trap to ensure cleanup happens on script exit
trap cleanup_workspace EXIT


echo "Starting GUAC Ingestion Process..."

echo "Step 0: Validating tools..."
check_command "${GIT_BIN}"
check_command "${SYFT_BIN}"
check_command "${GUACONE_BIN}"
echo "Tool validation successful."
echo ""

echo "Step 1: Preparing workspace..."
echo "  Current user: $(id)"
echo "  Current working directory: $(pwd)"
echo "  Value of WORKSPACE_BASE before mkdir: '${WORKSPACE_BASE}'"

mkdir -p "${WORKSPACE_BASE}"
MKDIR_EXIT_CODE=$? # Exit-Code von mkdir speichern
echo "  mkdir -p '${WORKSPACE_BASE}' exit code: ${MKDIR_EXIT_CODE}"

echo "Base workspace directory ${WORKSPACE_BASE} ensured."

if [ ${MKDIR_EXIT_CODE} -ne 0 ]; then
    echo "  FATAL: mkdir -p command failed. Exiting."
    exit 1
fi

WORKSPACE_DIR=$(mktemp -d "${WORKSPACE_BASE}/run.XXXXXX")
if [ ! -d "${WORKSPACE_DIR}" ]; then # Double-check if mktemp succeeded
    echo "Error: Could not create temporary workspace directory in ${WORKSPACE_BASE} using mktemp."
    exit 1
fi
echo "Unique workspace for this run created: ${WORKSPACE_DIR}"
# Source repo will be cloned into WORKSPACE_DIR directly or a subdir of it.
REPO_DIR="${WORKSPACE_DIR}/source_repo" # Adjusted to be inside the unique WORKSPACE_DIR
SBOM_OUTPUT_FILE="${WORKSPACE_DIR}/sbom.cyclonedx.json" # Adjusted

echo ""


echo "Step 2: Cloning repository..."
mkdir -p "${REPO_DIR}"
echo "  Target directory: ${REPO_DIR}"
echo "  Repository: ${GITHUB_REPO_URL_FULL}"
echo "  Branch/Tag/Commit: ${GIT_BRANCH_OR_TAG}"


EFFECTIVE_CLONE_URL="${GITHUB_REPO_URL_FULL}"
# Check if GITHUB_TOKEN is set and not empty
if [ -n "${GITHUB_TOKEN:-}" ]; then
  echo "  Using GITHUB_TOKEN for authentication."
  # Modify URL for HTTPS token authentication: https://<token>@github.com/owner/repo.git
  # This handles the case where GITHUB_REPO_URL_FULL starts with "https://github.com/"
  EFFECTIVE_CLONE_URL=$(echo "${GITHUB_REPO_URL_FULL}" | sed "s|https://github.com/|https://oauth2:${GITHUB_TOKEN}@github.com/|")
  # Add git config to prevent git from asking for credentials interactively on error
  "${GIT_BIN}" config --global credential.helper '!f() { echo "username=oauth2"; echo "password=$GITHUB_TOKEN"; }; f'
else
  echo "  No GITHUB_TOKEN provided. Cloning publicly or relying on instance roles if applicable."
fi

"${GIT_BIN}" clone --depth 1 --branch "${GIT_BRANCH_OR_TAG}" "${EFFECTIVE_CLONE_URL}" "${REPO_DIR}"
echo "Repository cloned successfully."
echo ""

# 2. Analyze Source Code and Generate SBOM using Syft
echo "Step 2: Generating SBOM from source code..."
echo "  Scanning directory: ${REPO_DIR}"
echo "  Output SBOM file: ${SBOM_OUTPUT_FILE}"
#important bit json 1.5 is required for guac
"${SYFT_BIN}" scan "dir:${REPO_DIR}" -o spdx-json="${SBOM_OUTPUT_FILE}" --enrich all --source-name ${GITHUB_REPO_SHORT_NAME} --source-version ${GIT_BRANCH_OR_TAG}"

if [ ! -s "${SBOM_OUTPUT_FILE}" ]; then # -s checks if file exists and is not empty
  echo "Warning: SBOM file was generated but is empty, or generation failed silently."
  # Depending on requirements, you might choose to exit 1 here or proceed.
  # For now, we'll proceed but log a warning.
else
  echo "SBOM generated successfully: ${SBOM_OUTPUT_FILE}"
fi
echo ""

# 3. Ingest the SBOM into GUAC
echo "Step 3: Ingesting SBOM into GUAC..."
if [ -s "${SBOM_OUTPUT_FILE}" ]; then # Only ingest if SBOM is not empty
  "${GUACONE_BIN}" collect files --gql-addr "${GUAC_GQL_ADDR}" "${SBOM_OUTPUT_FILE}" --add-depsdev-on-ingest  --add-eol-on-ingest --add-vuln-on-ingest
  echo "SBOM ingestion into GUAC completed."
else
  echo "Skipping SBOM ingestion as the SBOM file is empty or missing."
fi
echo ""

# 4. Optional: Run 'guacone collect github' for release/workflow metadata
if [ "${RUN_GUACONE_COLLECT_GITHUB}" = "true" ]; then
  echo "Step 4: Running 'guacone collect github' (Optional Step)..."
  if [ -z "${GITHUB_REPO_SHORT_NAME}" ]; then
    echo "Error: GITHUB_REPO_SHORT_NAME is required when RUN_GUACONE_COLLECT_GITHUB is true."
    # We'll log an error and skip this step rather than exiting the whole script.
  else
    echo "  Repository (short name): ${GITHUB_REPO_SHORT_NAME}"
    echo "  Collection Mode: ${GITHUB_COLLECT_MODE}"
    COLLECT_GITHUB_CMD_ARGS="--gql-addr \"${GUAC_GQL_ADDR}\" --github-mode \"${GITHUB_COLLECT_MODE}\" \"${GITHUB_REPO_SHORT_NAME}\""

    if [ -n "${GITHUB_TOKEN:-}" ]; then
      echo "  Using GITHUB_TOKEN for 'guacone collect github'."
      COLLECT_GITHUB_CMD_ARGS="${COLLECT_GITHUB_CMD_ARGS} --github-token \"${GITHUB_TOKEN}\""
    else
      echo "  No GITHUB_TOKEN provided for 'guacone collect github'."
    fi

    echo "  Executing: ${GUACONE_BIN} collect github ${COLLECT_GITHUB_CMD_ARGS}"
    # Use eval to correctly interpret quotes if COMMAND_ARGS contains them.
    # Ensure GUACONE_BIN and other parts of the command are trusted.
    eval "${GUACONE_BIN} collect github ${COLLECT_GITHUB_CMD_ARGS}"
    echo "'guacone collect github' step completed."
  fi
else
  echo "Step 4: Skipping 'guacone collect github' as RUN_GUACONE_COLLECT_GITHUB is not 'true'."
fi
echo ""

echo "-----------------------------------------------------"
echo "GUAC Ingestion Process Finished Successfully."
echo "Timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
# Trap will handle cleanup