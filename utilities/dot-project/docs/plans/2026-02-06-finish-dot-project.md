# Finish `.project` Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete the `.project` validator and ecosystem so that every CNCF project can own a `.project` repo in their GitHub org with standardized, validated metadata -- and CNCF automation can act on that data.

**Architecture:** A Go CLI validator (already ~70% built) validates `project.yaml` and `maintainers.yaml` schemas, with SHA256-based change detection. The remaining work falls into five areas: (1) schema hardening and versioning, (2) missing schema fields, (3) code quality fixes, (4) CI/CD and GitHub Actions, and (5) documentation. The validator is consumed by GitHub Actions workflows -- both internally for the `cncf/automation` repo and as a reusable workflow for external `.project` repos.

**Tech Stack:** Go 1.24, `gopkg.in/yaml.v3`, GitHub Actions, Makefile

---

## Phase 1: Housekeeping and Code Quality

These tasks fix bugs, inconsistencies, and security issues in the current codebase. They are independent of each other and can be done in any order.

### Task 1: Fix `.gitignore` to exclude sensitive/generated files

**Files:**
- Modify: `utilities/dot-project/.gitignore`

**Rationale:** `.env` contains a real JWT token and is tracked by git. `.cache/cache.json` contains developer-local file paths. Both should be ignored. Also ignore build artifacts (`bin/`, `coverage.*`).

**Step 1: Update `.gitignore`**

```gitignore
validator
bin/
.cache/
.env
coverage.out
coverage.html
*.test
.test-cache/
```

**Step 2: Remove tracked sensitive files from git index**

```bash
git rm --cached .env 2>/dev/null || true
git rm --cached -r .cache/ 2>/dev/null || true
```

**Step 3: Commit**

```bash
git add .gitignore
git commit -m "fix: exclude sensitive and generated files from git tracking"
```

---

### Task 2: Move inline types to `types.go` and add missing JSON tags

**Files:**
- Modify: `utilities/dot-project/types.go`
- Modify: `utilities/dot-project/validator.go`

**Rationale:** `ProjectListEntry` and `ProjectListConfig` are defined inline in `validator.go` instead of `types.go`. `MaintainersConfig`, `MaintainerEntry`, and `Team` lack `json` tags, inconsistent with all other types. `ProjectList` is unused and should be removed.

**Step 1: Add `json` tags to maintainer types and move project list types**

In `types.go`:
- Remove `type ProjectList []string` (unused)
- Add `json` struct tags to `MaintainersConfig`, `MaintainerEntry`, `Team`
- Add `ProjectListEntry` and `ProjectListConfig` types (moved from `validator.go`)

In `validator.go`:
- Remove the `ProjectListEntry` and `ProjectListConfig` type definitions

**Step 2: Run tests**

```bash
go test -v
```

Expected: All tests pass, no behavior change.

**Step 3: Commit**

```bash
git commit -am "refactor: consolidate types into types.go, add json tags to maintainer types"
```

---

### Task 3: Fix `example-project.yaml` misplaced governance fields

**Files:**
- Modify: `utilities/dot-project/yaml/example-project.yaml`

**Rationale:** The example file has `contributing`, `codeowners`, and `governance_doc` nested under `security:` instead of `governance:`. This would silently fail validation because YAML unmarshaling ignores unknown fields. The example should be correct since projects will copy from it.

**Step 1: Fix the structure**

Move the `contributing`, `codeowners`, `governance_doc` fields out of `security:` and into a new `governance:` section.

**Step 2: Run validator against example**

```bash
go run ./cmd/validator -config /dev/null -maintainers ""
```

Or write a quick test that validates the example YAML file parses correctly with the expected governance fields populated.

**Step 3: Commit**

```bash
git commit -am "fix: move governance fields to correct section in example-project.yaml"
```

---

### Task 4: Fix Makefile inconsistencies

**Files:**
- Modify: `utilities/dot-project/Makefile`

**Rationale:** The `help` target references `run-json` which doesn't exist. The build output goes to `./validator` but AGENT.md references `./bin/validator`. Clean target doesn't clean `bin/`.

**Step 1: Remove `run-json` from help text and add `--output` flag once it exists (or remove reference)**

**Step 2: Standardize build output to `./bin/validator`**

Update `build` target:
```makefile
build:
	@echo "Building project validator..."
	@mkdir -p bin
	go build -o bin/validator ./cmd/validator
```

Update `clean`:
```makefile
clean:
	@echo "Cleaning build artifacts..."
	rm -f validator
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf .cache
	rm -rf .test-cache
```

Update `run` and `run-changes` to use `./bin/validator`.

**Step 3: Run `make build && make test`**

**Step 4: Commit**

```bash
git commit -am "fix: standardize Makefile build output and remove phantom targets"
```

---

### Task 5: Add `--output` flag to CLI

**Files:**
- Modify: `utilities/dot-project/cmd/validator/main.go`

**Rationale:** The library supports JSON/YAML/text output but the CLI always outputs text. Add a flag.

**Step 1: Add flag**

```go
outputFormat = flag.String("output", "text", "Output format: text, json, yaml")
```

**Step 2: Use it in the `FormatResults` and `FormatMaintainersResults` calls**

**Step 3: Write a test or manually verify**

```bash
./bin/validator --output json -maintainers ""
```

**Step 4: Commit**

```bash
git commit -am "feat: add --output flag for json/yaml/text format selection"
```

---

## Phase 2: Schema Versioning and Hardening

### Task 6: Define schema version and validate it

**Files:**
- Modify: `utilities/dot-project/types.go`
- Modify: `utilities/dot-project/validator.go`
- Modify: `utilities/dot-project/validator_test.go`

**Rationale:** `schema_version` exists in the struct but is optional and never validated. For a system where automation depends on the schema, versioning is critical. We should:
- Make `schema_version` required
- Validate it against a known set of supported versions
- The current schema is `v1.0.0`

**Step 1: Write failing test**

```go
func TestSchemaVersionValidation(t *testing.T) {
    tests := []struct {
        name          string
        version       string
        expectError   bool
        errorContains string
    }{
        {"valid version", "1.0.0", false, ""},
        {"missing version", "", true, "schema_version is required"},
        {"unsupported version", "99.0.0", true, "unsupported schema_version"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            project := validBaseProject() // helper with all required fields
            project.SchemaVersion = tt.version
            errors := validateProjectStruct(project)
            // assert based on tt.expectError and tt.errorContains
        })
    }
}
```

**Step 2: Run test, confirm failure**

**Step 3: Implement in `validateProjectStruct`**

Add a `SupportedSchemaVersions` variable (e.g., `[]string{"1.0.0"}`) and validation logic at the top of `validateProjectStruct`.

**Step 4: Run tests, confirm pass**

**Step 5: Update all YAML fixtures to include `schema_version: "1.0.0"` and ensure `bad-project.yaml` omits it**

**Step 6: Commit**

```bash
git commit -am "feat: require and validate schema_version field"
```

---

### Task 7: Validate `maturity_log` phase values against allowed set

**Files:**
- Modify: `utilities/dot-project/validator.go`
- Modify: `utilities/dot-project/validator_test.go`

**Rationale:** Phase values should be constrained to the CNCF lifecycle: `sandbox`, `incubating`, `graduated`, `archived`. Currently any string is accepted.

**Step 1: Write failing test**

```go
func TestMaturityPhaseValues(t *testing.T) {
    project := validBaseProject()
    project.MaturityLog = []MaturityEntry{
        {Phase: "invalid-phase", Date: time.Now(), Issue: "https://github.com/cncf/toc/issues/1"},
    }
    errors := validateProjectStruct(project)
    // Expect error containing "invalid maturity phase"
}
```

**Step 2: Implement validation**

Define `var validPhases = map[string]bool{"sandbox": true, "incubating": true, "graduated": true, "archived": true}`

**Step 3: Run tests, confirm pass**

**Step 4: Commit**

```bash
git commit -am "feat: validate maturity_log phase against allowed values"
```

---

### Task 8: Validate maturity log ordering (dates ascending)

**Files:**
- Modify: `utilities/dot-project/validator.go`
- Modify: `utilities/dot-project/validator_test.go`

**Rationale:** A project's maturity log should be chronologically ordered. Out-of-order entries indicate a data error.

**Step 1: Write failing test**

```go
func TestMaturityLogOrdering(t *testing.T) {
    project := validBaseProject()
    project.MaturityLog = []MaturityEntry{
        {Phase: "incubating", Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), Issue: "..."},
        {Phase: "sandbox", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "..."},
    }
    errors := validateProjectStruct(project)
    // Expect error about chronological ordering
}
```

**Step 2: Implement**

**Step 3: Run tests, confirm pass**

**Step 4: Commit**

```bash
git commit -am "feat: validate maturity_log entries are chronologically ordered"
```

---

### Task 9: Strengthen URL validation

**Files:**
- Modify: `utilities/dot-project/validator.go`
- Modify: `utilities/dot-project/validator_test.go`

**Rationale:** Current `isValidURL()` accepts `https://x` as valid. Should use `net/url.Parse` for proper validation.

**Step 1: Write failing test cases**

```go
{"https://x", false},          // No domain
{"https://.", false},           // Invalid domain
{"https://example.com", true}, // Valid
```

**Step 2: Replace `isValidURL` implementation**

```go
func isValidURL(str string) bool {
    u, err := url.Parse(str)
    if err != nil {
        return false
    }
    if u.Scheme != "http" && u.Scheme != "https" {
        return false
    }
    if u.Host == "" || !strings.Contains(u.Host, ".") {
        return false
    }
    return true
}
```

**Step 3: Run tests, fix any test cases that depended on the permissive behavior**

**Step 4: Commit**

```bash
git commit -am "fix: strengthen URL validation using net/url"
```

---

## Phase 3: Missing Schema Fields

### Task 10: Add `slug` / project identifier field

**Files:**
- Modify: `utilities/dot-project/types.go`
- Modify: `utilities/dot-project/validator.go`
- Modify: `utilities/dot-project/validator_test.go`
- Modify: YAML fixtures

**Rationale:** Projects need a stable unique identifier (slug) that maps to the CNCF landscape, LFX, and other systems. This is different from `name` (display name). Example: `"kubernetes"`, `"envoy"`, `"argo"`.

**Step 1: Add field to `Project` struct**

```go
Slug string `json:"slug" yaml:"slug"` // Unique project identifier (lowercase, alphanumeric + hyphens)
```

**Step 2: Write validation test (required, format: lowercase alphanumeric + hyphens)**

**Step 3: Implement validation**

**Step 4: Update YAML fixtures**

**Step 5: Commit**

```bash
git commit -am "feat: add required slug field for stable project identification"
```

---

### Task 11: Add `accepted_date` and `project_lead` / primary contact fields

**Files:**
- Modify: `utilities/dot-project/types.go`
- Modify: `utilities/dot-project/validator.go`
- Modify: `utilities/dot-project/validator_test.go`

**Rationale:** For the "ping every 6 months if no maintainer changes" automation, we need to know who to contact and when the project was accepted. The maturity_log provides dates but having an explicit "accepted into CNCF" date and a primary contact simplifies automation. Consider whether `project_lead` should just be the first member of `project-maintainers` or a separate field.

**Step 1: Add field(s) to `Project` struct**

```go
ProjectLead string `json:"project_lead,omitempty" yaml:"project_lead,omitempty"` // GitHub handle of primary contact
```

**Step 2: Write validation (if present, must be non-empty string)**

**Step 3: Commit**

```bash
git commit -am "feat: add project_lead contact field"
```

---

### Task 12: Add `channel` field for CNCF Slack workspace

**Files:**
- Modify: `utilities/dot-project/types.go`
- Modify: `utilities/dot-project/validator.go`

**Rationale:** Many CNCF automations need to notify project channels. Having a structured `cncf_slack_channel` field (rather than burying it in the freeform `social` map) enables automated notifications (like the 6-month maintainer ping).

**Step 1: Add to `Project`**

```go
CNCFSlackChannel string `json:"cncf_slack_channel,omitempty" yaml:"cncf_slack_channel,omitempty"`
```

**Step 2: Validate format if present (starts with `#`)**

**Step 3: Commit**

```bash
git commit -am "feat: add cncf_slack_channel field for automated notifications"
```

---

### Task 13: Add `landscape` section for CNCF Landscape integration

**Files:**
- Modify: `utilities/dot-project/types.go`
- Modify: `utilities/dot-project/validator.go`
- Modify: `utilities/dot-project/validator_test.go`

**Rationale:** For the "automatically update CNCF Landscape if they change levels" use case, the project YAML needs to reference where it lives in the landscape. This bridges the gap to the landscape-updater tool referenced in AGENT.md.

**Step 1: Define types**

```go
type LandscapeConfig struct {
    Category    string `json:"category" yaml:"category"`         // Landscape category
    Subcategory string `json:"subcategory" yaml:"subcategory"`   // Landscape subcategory
}
```

**Step 2: Add to `Project`**

```go
Landscape *LandscapeConfig `json:"landscape,omitempty" yaml:"landscape,omitempty"`
```

**Step 3: Write validation (if present, both fields required)**

**Step 4: Commit**

```bash
git commit -am "feat: add landscape section for CNCF Landscape integration"
```

---

## Phase 4: Strict/Unknown Field Rejection

### Task 14: Reject unknown YAML fields (strict parsing)

**Files:**
- Modify: `utilities/dot-project/validator.go`
- Add: `utilities/dot-project/strict_test.go`

**Rationale:** Currently, typos or misplaced fields (like the `example-project.yaml` issue with governance fields under security) are silently ignored. YAML should fail validation if unknown fields are present. Use `yaml.Decoder` with `KnownFields(true)`.

**Step 1: Write failing test**

```go
func TestUnknownFieldsRejected(t *testing.T) {
    yamlContent := `
name: "Test"
description: "Test"
unknown_field: "should fail"
`
    // Parse and validate, expect error about unknown field
}
```

**Step 2: Implement using `yaml.NewDecoder` with `KnownFields(true)`**

Update the YAML unmarshaling in `validateProject` to use:
```go
decoder := yaml.NewDecoder(strings.NewReader(content))
decoder.KnownFields(true)
var project Project
if err := decoder.Decode(&project); err != nil {
    // report error
}
```

**Step 3: Run full test suite, fix any fixtures with extra fields**

**Step 4: Commit**

```bash
git commit -am "feat: reject unknown YAML fields for strict schema validation"
```

---

## Phase 5: CI/CD and GitHub Actions

### Task 15: Fix Go version in existing workflows

**Files:**
- Modify: `.github/workflows/project-validator.yml`
- Modify: `.github/workflows/validate-maintainers.yaml`
- Modify: `.github/workflows/reusable-validate-maintainers.yaml`

**Rationale:** All three workflows specify `go-version: '1.21'` but `go.mod` requires `1.24.5`. The workflows will fail to build.

**Step 1: Update all three workflows to `go-version: '1.24'`**

Also update `actions/setup-go` from `v4` to `v5` in the two that still use `v4`.

**Step 2: Commit**

```bash
git commit -am "fix: update Go version in CI workflows to match go.mod"
```

---

### Task 16: Add CI test workflow for PRs

**Files:**
- Modify: `.github/workflows/project-validator.yml` (or create a dedicated test workflow)

**Rationale:** The existing workflow runs tests, builds, and validates. But it should also run `go vet`, `golangci-lint`, and test coverage reporting for PRs.

**Step 1: Add lint and vet steps to the existing workflow**

```yaml
    - name: Vet
      working-directory: utilities/dot-project/
      run: go vet ./...

    - name: Run tests with coverage
      working-directory: utilities/dot-project/
      run: |
        go test -v -coverprofile=coverage.out
        go tool cover -func=coverage.out
```

**Step 2: Commit**

```bash
git commit -am "feat: add go vet and coverage reporting to CI workflow"
```

---

### Task 17: Create composite GitHub Action for maintainer validation

**Files:**
- Create: `.github/actions/validate-maintainers/action.yml`

**Rationale:** The AGENT.md references a reusable action at `cncf/automation/.github/actions/validate-maintainers@main` but it doesn't exist. External repos should be able to use a simple composite action to validate their maintainers file.

**Step 1: Create the action**

```yaml
name: 'Validate Maintainers'
description: 'Validate maintainers.yaml file against CNCF schema'
inputs:
  maintainers_file:
    description: 'Path to maintainers.yaml'
    required: true
  base_maintainers_file:
    description: 'Path to base maintainers file for diff validation'
    required: false
    default: ''
  verify_maintainers:
    description: 'Whether to verify handles via LFX'
    required: false
    default: 'false'
runs:
  using: 'composite'
  steps:
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'
    - name: Build validator
      shell: bash
      run: |
        cd ${{ github.action_path }}/../../../utilities/dot-project
        go build -o validator ./cmd/validator
    - name: Validate
      shell: bash
      run: |
        # ... validation logic
```

**Step 2: Commit**

```bash
git commit -am "feat: add composite GitHub Action for maintainer validation"
```

---

### Task 18: Create composite GitHub Action for project validation

**Files:**
- Create: `.github/actions/validate-project/action.yml`

**Rationale:** External `.project` repos need a way to validate their `project.yaml` on every PR. This action should be a single line to add to a workflow.

**Step 1: Create the action**

Similar to Task 17 but for project.yaml validation.

**Step 2: Commit**

```bash
git commit -am "feat: add composite GitHub Action for project YAML validation"
```

---

### Task 19: Create template `.project` repository structure

**Files:**
- Create: `utilities/dot-project/template/project.yaml`
- Create: `utilities/dot-project/template/maintainers.yaml`
- Create: `utilities/dot-project/template/.github/workflows/validate.yaml`
- Create: `utilities/dot-project/template/README.md`

**Rationale:** When a CNCF project creates their `.project` repo, they need a starting template. This template should include a pre-configured workflow that calls the reusable validation workflow.

**Step 1: Create template project.yaml with placeholder values and comments**

```yaml
# Project metadata for [PROJECT NAME]
# See https://github.com/cncf/automation/tree/main/utilities/dot-project for docs
schema_version: "1.0.0"
slug: "my-project"     # CHANGE: lowercase project identifier
name: "My Project"     # CHANGE: display name
description: ""        # CHANGE: one-line description
# ... etc with comments explaining each field
```

**Step 2: Create template workflow**

```yaml
name: Validate Project Metadata
on:
  pull_request:
    paths: ['project.yaml', 'maintainers.yaml']
jobs:
  validate:
    uses: cncf/automation/.github/workflows/reusable-validate-maintainers.yaml@main
    # ...
```

**Step 3: Commit**

```bash
git commit -am "feat: add template .project repository structure"
```

---

## Phase 6: Documentation

### Task 20: Rewrite README.md

**Files:**
- Modify: `utilities/dot-project/README.md`

**Rationale:** Current README has a Markdown formatting error (unclosed code fence at line 79) and is missing several sections. It should be the single source of truth for maintainers adopting this system.

New structure:
1. What is `.project`? (high-level vision: project-owned metadata)
2. Quick Start (for a CNCF project adopting this)
3. Schema Reference (all fields, required vs optional, format constraints)
4. Validator Usage (CLI, flags, output formats)
5. GitHub Actions Integration (how to use the reusable workflows)
6. Development (building, testing, contributing)
7. Schema Versioning Policy

**Step 1: Rewrite**

**Step 2: Commit**

```bash
git commit -am "docs: rewrite README with complete schema reference and adoption guide"
```

---

### Task 21: Update AGENT.md to match reality

**Files:**
- Modify: `utilities/dot-project/AGENT.md`

**Rationale:** AGENT.md references things that don't exist yet (`cmd/landscape-updater/`, `Dockerfile`) and has stale paths (`bin/validator` vs `./validator`). Update it to accurately reflect the codebase after all the above changes.

**Step 1: Update to match actual file structure and commands**

**Step 2: Commit**

```bash
git commit -am "docs: update AGENT.md to reflect current codebase"
```

---

### Task 22: Create `SCHEMA.md` formal schema documentation

**Files:**
- Create: `utilities/dot-project/SCHEMA.md`

**Rationale:** A formal, versioned schema document that serves as the specification. Different from the README which is a user guide.

Structure:
- Schema Version: 1.0.0
- Required Fields table (name, description, slug, schema_version, maturity_log, repositories)
- Optional Fields table (all others)
- Type definitions with constraints
- Example: minimal valid project.yaml
- Example: full project.yaml

**Step 1: Write SCHEMA.md**

**Step 2: Commit**

```bash
git commit -am "docs: add formal SCHEMA.md specification"
```

---

## Phase 7: Test Improvements

### Task 23: Add test helper `validBaseProject()` and refactor tests

**Files:**
- Create: `utilities/dot-project/test_helpers_test.go`
- Modify: `utilities/dot-project/validator_test.go`

**Rationale:** Many tests repeat the same valid base project setup. A helper reduces duplication and makes tests clearer about what they're actually testing.

**Step 1: Create helper**

```go
func validBaseProject() Project {
    return Project{
        Name:          "Test Project",
        Description:   "A valid test project",
        SchemaVersion: "1.0.0",
        Slug:          "test-project",
        MaturityLog: []MaturityEntry{
            {
                Phase: "sandbox",
                Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
                Issue: "https://github.com/cncf/toc/issues/123",
            },
        },
        Repositories: []string{"https://github.com/test/repo"},
    }
}
```

**Step 2: Refactor existing tests to use it**

**Step 3: Run tests**

**Step 4: Commit**

```bash
git commit -am "test: add validBaseProject helper and refactor test setup"
```

---

### Task 24: Add integration test that validates all YAML fixtures

**Files:**
- Create: `utilities/dot-project/integration_test.go`

**Rationale:** Ensures the example YAML files in `yaml/` actually pass (or fail, for `bad-project.yaml`) validation. Catches issues like the `example-project.yaml` governance misplacement.

**Step 1: Write test**

```go
func TestYAMLFixtures(t *testing.T) {
    tests := []struct {
        file      string
        expectValid bool
    }{
        {"yaml/test-project.yaml", true},
        {"yaml/example-project.yaml", true},
        {"yaml/bad-project.yaml", false},
    }
    for _, tt := range tests {
        t.Run(tt.file, func(t *testing.T) {
            data, err := os.ReadFile(tt.file)
            require.NoError(t, err)
            var project Project
            err = yaml.Unmarshal(data, &project)
            require.NoError(t, err)
            errs := validateProjectStruct(project)
            if tt.expectValid {
                assert.Empty(t, errs, "expected %s to be valid", tt.file)
            } else {
                assert.NotEmpty(t, errs, "expected %s to have errors", tt.file)
            }
        })
    }
}
```

Note: This uses standard library testing only (no external assert libs needed -- use manual checks matching existing test patterns).

**Step 2: Run tests**

**Step 3: Commit**

```bash
git commit -am "test: add integration test for YAML fixture files"
```

---

## Summary: Execution Order

The recommended execution order respects dependencies:

| Order | Task | Phase | Dependency |
|-------|------|-------|------------|
| 1 | Task 1: Fix `.gitignore` | Housekeeping | None |
| 2 | Task 2: Consolidate types | Housekeeping | None |
| 3 | Task 3: Fix example YAML | Housekeeping | None |
| 4 | Task 4: Fix Makefile | Housekeeping | None |
| 5 | Task 23: Test helper | Tests | None |
| 6 | Task 5: Add `--output` flag | Housekeeping | Task 4 |
| 7 | Task 6: Schema versioning | Schema | Task 23 |
| 8 | Task 7: Phase validation | Schema | Task 23 |
| 9 | Task 8: Maturity ordering | Schema | Task 7 |
| 10 | Task 9: URL validation | Schema | Task 23 |
| 11 | Task 10: Add `slug` field | Schema | Task 6 |
| 12 | Task 14: Strict parsing | Schema | Task 3 |
| 13 | Task 11: Add `project_lead` | Schema | Task 10 |
| 14 | Task 12: Add Slack channel | Schema | None |
| 15 | Task 13: Add landscape section | Schema | None |
| 16 | Task 24: Integration test | Tests | Tasks 3, 6-14 |
| 17 | Task 15: Fix CI Go version | CI | None |
| 18 | Task 16: Add CI coverage | CI | Task 15 |
| 19 | Task 17: Maintainer action | CI | None |
| 20 | Task 18: Project action | CI | None |
| 21 | Task 19: Template repo | CI | Tasks 17, 18 |
| 22 | Task 20: Rewrite README | Docs | All schema tasks |
| 23 | Task 22: SCHEMA.md | Docs | All schema tasks |
| 24 | Task 21: Update AGENT.md | Docs | All tasks |

---

## Out of Scope (Future Work)

These are important for the overall `.project` ecosystem but are separate efforts:

1. **Landscape Updater** (`cmd/landscape-updater/`) -- Watches for maturity changes in `project.yaml` and opens PRs to `cncf/landscape`. Mentioned in AGENT.md but not started.
2. **6-Month Maintainer Staleness Ping** -- Automation that checks last-modified dates on maintainer files and pings `project_lead` or `#cncf_slack_channel`.
3. **Governance/Security Audit Automation** -- Periodic checks that security policies, CODEOWNERS, etc. referenced in the project YAML actually exist at the URLs.
4. **Dockerized Validator** -- The `Dockerfile` referenced in AGENT.md.
5. **JSON Schema Generation** -- Auto-generate a JSON Schema from the Go types for use by editors (VS Code YAML extension, etc.).
6. **Migration Tooling** -- Scripts to help existing CNCF projects create their initial `.project` repo from landscape data.
