# AGENTS.md — CNCF Automation

Guidance for AI agents working in this repo. Every item here answers "would an agent miss this without help?"

---

## Repo shape

Polyglot automation monorepo. **There is no root build, no Go workspace, no shared package manager.** Each component is independent; work inside the relevant subdirectory.

| Pillar | Path | Languages / tools |
|--------|------|-------------------|
| Self-hosted runner infra | `ci/` | Go, Terraform/OpenTofu, Packer (runtime), K8s YAML / Helm / ArgoCD |
| Standalone utilities | `utilities/` | Go, Python |
| Community scripts | `Ambassadors/`, `Kubestronaut/` | Python (Google Sheets, Shopify ops) |
| CI test | `tests/syntax_check.py` | Python — syntax-only, mocks credentials |

---

## Go — 7 independent modules (no go.work)

Module roots (each has its own `go.mod`, build, and test):

```
ci/cloudrunners/
ci/gha-runner-vm/
ci/gha-runner-vm-oci/
utilities/dot-project/
utilities/labeler/
utilities/landscape-mcp-server/
utilities/landscape-sync/
```

### Go version guard — critical

`.go-version` pins **Go 1.26.1**. `.github/scripts/check-go-version.sh` is the guard script that enforces this — it **fails if any `go.mod` or any `FROM golang:` Dockerfile line doesn't match exactly**. When bumping Go, update `.go-version` plus every `go.mod` and every `Dockerfile` in one commit.

### Per-module test commands

Most modules: `go test -v ./...`

Exceptions:
- `utilities/labeler/` — **no Makefile `test` target**; run `go test -v` directly (flat single package).
- `utilities/landscape-mcp-server/` — also flat single package; `go test -v` or `go test ./...` both work.
- `utilities/dot-project/` — Makefile uses `go test -v ./...`; **CI runs flat `go test -v -coverprofile=coverage.out`** (no `./...`). Note: `REPO_ROOT`, `MAINTAINER_API_ENDPOINT`, and `MAINTAINER_API_STUB=true` are set on the validator binary run step in CI, not the test step — unit tests do not require them.

### Per-module quirks

**`utilities/dot-project/`** — Read `utilities/dot-project/AGENT.md` (482 lines) and `SCHEMA.md` before editing. Key facts:
- `make build` builds only 3 of 7 `cmd/` binaries (validator, landscape-updater, bootstrap). Build others explicitly: `go build -o bin/<name> ./cmd/<name>`.
- Requires `REPO_ROOT` env var for `file://` config path resolution.
- Lint: `golangci-lint run`; security: `gosec ./...` (both must be installed separately).
- Docker image entrypoint is `validator`; override with `--entrypoint landscape-updater`.

**`utilities/landscape-mcp-server/`** — stdlib-only, **no `go.sum`, no Makefile**. Build: `go build -o landscape2-mcp-server`. Note: binary is `landscape2-mcp-server`; published image is `ghcr.io/cncf/landscape-mcp-server` (different names).

**`utilities/labeler/`** — Makefile has only `run` and `image` targets, no `build` or `test`. Local test image tags as `gha-labeler:latest`; published image is `ghcr.io/cncf/gha-labeler`.

**`ci/cloudrunners/`** — Dockerfile builds only `oci` and `kubevirt` binaries (not `gcp`). `CLOUDRUNNER_PROVIDER` env selects binary at runtime (`oci` default). OCI images are always created as `rc-<name>` (release candidates); a CI workflow promotes them to production names after tests pass.

---

## Terraform / OpenTofu

Terraform and OpenTofu are interchangeable here. Makefiles default to `TF ?= terraform`; override with `TF=tofu make ...`.

### Makefile targets (`ci/iac/oracle/` and `ci/iac/akamai/`)

```sh
# Oracle — requires OKE_CLUSTER=<name> or BUCKETS=<name>
make cluster-init OKE_CLUSTER=oke-cncf-gha-phx
make cluster-plan OKE_CLUSTER=oke-cncf-gha-phx
make cluster-apply   # no OKE_CLUSTER needed; consumes plan.out
make cluster-destroy OKE_CLUSTER=oke-cncf-gha-phx

# Akamai — requires LKE_CLUSTER=<name>
make cluster-init LKE_CLUSTER=lke-cncf-gha-iad2
make cluster-plan LKE_CLUSTER=lke-cncf-gha-iad2
make cluster-apply
```

### Critical Terraform gotchas

- **Per-cluster files must exist before `make`:** `cluster/tfbackends/<name>.tfbackend` and `cluster/tfvars/<name>.tfvars` (or `buckets/...`). Existing names: oracle clusters `oke-cncf-gha-chi`, `oke-cncf-gha-phx`, `oke-cncf-gha-runners`, `oke-cncf-services`; akamai cluster `lke-cncf-gha-iad2`.
- **`*-apply` consumes `plan.out` and ignores var files** — always run `*-plan` immediately before `*-apply`. Never apply a stale plan.
- **Run `make clean` when switching cluster names** — a single `.terraform/` dir is shared; reuse across clusters silently applies the wrong state.
- **Akamai kubeconfig:** `ci/iac/akamai/` outputs a base64-encoded kubeconfig — retrieve it with `make cluster-output | jq -r '.kubeconfig.value'` (it is not written to a file automatically).

### Required env vars per stack

| Stack | Required vars |
|-------|--------------|
| `iac/oracle/` | `TF_VAR_compartment_ocid`, `TF_VAR_tenancy_ocid`, `TF_VAR_user_ocid`, `TF_VAR_fingerprint`, `TF_VAR_private_key_path` |
| `iac/akamai/` | `TF_VAR_linode_api_token`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |

---

## Python

No central manifest. Install per-folder:

| Directory | Install |
|-----------|---------|
| `Ambassadors/` | `pip install -r Ambassadors/requirements.txt` |
| `Kubestronaut/` | `pip install -r Kubestronaut/requirements.txt` |
| `Kubestronaut/Rendering/` | `pip install -r Kubestronaut/Rendering/requirements.txt` |
| `Kubestronaut/kubestronauts-coupons/` | `pip install -r Kubestronaut/kubestronauts-coupons/requirements.txt` |
| `utilities/audit_project_lifecycle_across_tools/` | **No requirements.txt** — `pip install requests pyyaml beautifulsoup4` |

`utilities/audit_project_lifecycle_across_tools/scripts/fetch_pcc_projects.py` requires `LFX_TOKEN` env (short-lived; see CI secret). Run scripts from the subdirectory: `python scripts/<name>.py`.

`tests/syntax_check.py` is the CI-run test — it only does `py_compile` checks on two Kubestronaut files, mocking cred-dependent imports. It is not a general test suite.

---

## CI / GitHub Actions

- **`ci-test.yaml`** is `workflow_dispatch`-only and requires OCI secrets + self-hosted runners. It cannot be run locally.
- **GitHub Actions are expected to be SHA-pinned** (Kusari Inspector via `kusari.yaml` flags unpinned ones); most are, though some reusable workflows are tag-pinned (e.g. `slsa-github-generator@v2.1.0`). When editing `.github/actions/labeler-action`, manually bump its pinned SHA in `slash-commands.yml` and any other workflow that references it by SHA.
- PR labels are applied via slash commands: `/kind`, `/area`, `/priority`, `/status` — see `.github/CONTRIBUTING.md` and `.github/README_LABELING.md`.

---

## Never commit

These exist on disk but are gitignored — do not stage them:

- `terraform.tfstate`, `terraform.tfstate.*`, `.terraform/`, `.terraform.lock.hcl` (under `ci/iac/`)
- `**/kubeconfig.yaml`, `**/kubeconfig.yml`
- `credentials.json`, `token.json`
- `python_venv/`, `python_env_cncfpeople`, `python_venv_ambassadors`
- `.env`, `.env.*`

---

## Conventions

- **DCO required:** all commits must be signed off — `git commit -s` adds `Signed-off-by:`.
- **Commit subjects:** present-tense verb, e.g. `Add AGENTS.md` not `Added AGENTS.md`. Common prefixes from history: `fix:`, `feat:`, `chore:`, `refactor:`, `docs:`.
- **Go formatting:** `gofmt` before committing; `go vet ./...` per module.
- **YAML:** 2-space indentation, lines under 120 characters.
- **Packer templates:** not checked in. `ci/gha-runner-vm/` and `ci/gha-runner-vm-oci/` download and rewrite upstream `actions/runner-images` Packer HCL at runtime.
