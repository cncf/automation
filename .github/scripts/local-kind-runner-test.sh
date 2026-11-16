#!/bin/bash
# Local counterpart to .github/actions/run-kind-with-arc: start a throwaway
# kind cluster, install ARC, and deploy a patched copy of a committed runner
# scale set manifest pointing at a locally-built (or just-pushed) runner
# image. This replaces ad hoc personal-fork PR dry-runs as the way to
# smoke-test a runner image change before opening a PR.
#
# Unlike the CI composite action, this script does not poll a GitHub Actions
# run for a matching test-runner job (there isn't one locally) — instead it
# deploys the scale set and leaves it running so you can either:
#   - push a commit to a repo ARC is configured against with a job using
#     `runs-on: <printed scale-set name>`, and watch it get picked up, or
#   - just confirm the runner pod registers and goes Idle.
#
# Usage:
#   local-kind-runner-test.sh deploy --runner-type container \
#     --base-image resolute \
#     --manifest-file ci/cluster/lke-gha-iad2/manifests/container-runners/cncf-c-ubuntu-2-8-x86.yaml \
#     --runner-image ghcr.io/cncf/external-gha-runner-pr:pr-123-resolute \
#     --github-repo your-user/automation
#
#   local-kind-runner-test.sh deploy --runner-type vm \
#     --arch x86 --os-version 24.04 \
#     --manifest-file ci/cluster/oci/vm-runners/cncf-ubuntu-2-8-x86.yaml \
#     --runner-image ghcr.io/cncf/gha-cloudrunner-pr:pr-123 \
#     --github-repo your-user/automation
#
#   local-kind-runner-test.sh cleanup
#
# Required environment variables:
#   GHA_PAT          - GitHub PAT used by ARC to register runners (same scopes
#                       as the GHA_PAT secret used in CI)
#   OCI_CLI_USER, OCI_CLI_TENANCY, OCI_CLI_FINGERPRINT, OCI_CLI_REGION,
#   OCI_CLI_KEY_CONTENT   - only required for --runner-type vm

set -euo pipefail

COMMAND="${1:-}"
shift || true

RUNNER_TYPE="container"
ARCH=""
OS_VERSION="24.04"
BASE_IMAGE=""
MANIFEST_FILE=""
RUNNER_IMAGE=""
GITHUB_REPO=""

while [ $# -gt 0 ]; do
  case "$1" in
    --runner-type) RUNNER_TYPE="$2"; shift 2 ;;
    --arch) ARCH="$2"; shift 2 ;;
    --os-version) OS_VERSION="$2"; shift 2 ;;
    --base-image) BASE_IMAGE="$2"; shift 2 ;;
    --manifest-file) MANIFEST_FILE="$2"; shift 2 ;;
    --runner-image) RUNNER_IMAGE="$2"; shift 2 ;;
    --github-repo) GITHUB_REPO="$2"; shift 2 ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

CLUSTER_NAME="arc-local-test"

cleanup() {
  kubectl delete autoscalingrunnerset -n arc-systems --all --timeout=120s 2>/dev/null || true
  kind delete cluster --name "${CLUSTER_NAME}" || true
}

deploy() {
  : "${GHA_PAT:?set GHA_PAT to a GitHub PAT with repo administration scope}"
  : "${MANIFEST_FILE:?pass --manifest-file}"
  : "${RUNNER_IMAGE:?pass --runner-image}"
  : "${GITHUB_REPO:?pass --github-repo owner/repo}"

  if ! kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
    kind create cluster --name "${CLUSTER_NAME}"
  fi
  kubectl wait --for=condition=Ready nodes --all --timeout=120s

  if ! helm status cncf-gha-controller -n arc-systems >/dev/null 2>&1; then
    helm install cncf-gha-controller \
      --namespace arc-systems \
      --create-namespace \
      --version 0.14.1 \
      oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set-controller
    kubectl wait --for=condition=Ready pod -l app.kubernetes.io/component=controller-manager --timeout=600s -n arc-systems
  fi

  kubectl delete secret github-arc-secret -n arc-systems --ignore-not-found
  kubectl create secret generic github-arc-secret --from-literal=github_token="${GHA_PAT}" -n arc-systems

  RUN_ID=$(date +%s)

  if [ "${RUNNER_TYPE}" = "vm" ]; then
    : "${ARCH:?pass --arch for runner-type vm}"
    : "${OCI_CLI_USER:?}" "${OCI_CLI_TENANCY:?}" "${OCI_CLI_FINGERPRINT:?}" "${OCI_CLI_REGION:?}" "${OCI_CLI_KEY_CONTENT:?}"

    kubectl delete secret oci-api-key oci-config -n arc-systems --ignore-not-found
    kubectl create secret generic oci-api-key --from-literal=oci_api_key.pem="${OCI_CLI_KEY_CONTENT}" -n arc-systems
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: oci-config
  namespace: arc-systems
type: Opaque
stringData:
  config: |
    [DEFAULT]
    user=${OCI_CLI_USER}
    fingerprint=${OCI_CLI_FINGERPRINT}
    tenancy=${OCI_CLI_TENANCY}
    region=${OCI_CLI_REGION}
    key_file=/oci/oci_api_key.pem
EOF

    SCALE_SET="local-test-${RUN_ID}-${OS_VERSION}-${ARCH}"
    sed -E "s|image: ghcr.io/cncf/gha-cloudrunner[@:][^[:space:]]+|image: ${RUNNER_IMAGE}|" "${MANIFEST_FILE}" \
      | sed "s/cncf-ubuntu-2-8/${SCALE_SET}/g" \
      | sed '/nodeSelector/,+1d' \
      | sed '/runnerScaleSetLabels/,+2d' | kubectl apply -f -
  else
    : "${BASE_IMAGE:?pass --base-image for runner-type container}"

    SCALE_SET="local-test-${RUN_ID}-${BASE_IMAGE}"
    OLD_NAME=$(grep -m1 'scale-set-name:' "${MANIFEST_FILE}" | awk '{print $2}')
    sed -e "s|${OLD_NAME}|${SCALE_SET}|g" \
        -e "s|image: ghcr.io/cncf/external-gha-runner[@:][^[:space:]]*|image: ${RUNNER_IMAGE}|" \
      "${MANIFEST_FILE}" | kubectl apply -f -
  fi

  # githubConfigUrl is baked into the committed manifests for cncf/automation;
  # patch it so the runner registers against your own repo instead.
  kubectl patch autoscalingrunnerset -n arc-systems "${SCALE_SET}" --type merge \
    -p "{\"spec\":{\"githubConfigUrl\":\"https://github.com/${GITHUB_REPO}\"}}"

  cat <<MSG

Deployed scale set: ${SCALE_SET}
Watch it come up with:   kubectl get pods -n arc-systems -w
Or k9s, namespace arc-systems.

To exercise it end-to-end, push a workflow to ${GITHUB_REPO} with a job like:

  jobs:
    smoke:
      runs-on: ${SCALE_SET}
      steps:
        - run: echo hello

When done: $0 cleanup
MSG
}

case "${COMMAND}" in
  deploy) deploy ;;
  cleanup) cleanup ;;
  *)
    echo "Usage: $0 {deploy|cleanup} [--runner-type vm|container] [--arch ARCH] [--os-version VER] [--base-image IMG] --manifest-file FILE --runner-image IMAGE --github-repo OWNER/REPO" >&2
    exit 1
    ;;
esac
