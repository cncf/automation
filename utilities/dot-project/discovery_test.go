package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRepoFile writes a file inside dir, creating parent directories.
func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", rel, err)
	}
}

const minimalProjectYAML = `schema_version: "1.0.0"
slug: "%s"
name: "%s"
description: "A project used in tests"
maturity_log:
  - phase: "sandbox"
    date: "2024-01-15T00:00:00Z"
    issue: "https://github.com/cncf/sandbox/issues/1"
repositories:
  - url: "https://github.com/example-org/%s"
    primary: true
`

func projectYAML(slug, name string) string {
	return fmt.Sprintf(minimalProjectYAML, slug, name, slug)
}

func maintainersYAML(projectID, org, team string, members ...string) string {
	var b strings.Builder
	b.WriteString("maintainers:\n")
	b.WriteString("  - project_id: \"" + projectID + "\"\n")
	if org != "" {
		b.WriteString("    org: \"" + org + "\"\n")
	}
	b.WriteString("    teams:\n")
	b.WriteString("      - name: \"" + team + "\"\n")
	b.WriteString("        members:\n")
	for _, m := range members {
		b.WriteString("          - " + m + "\n")
	}
	return b.String()
}

// newMultiRepo builds a valid two-project repository in a temp dir.
func newMultiRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeRepoFile(t, dir, "org.yaml", `schema_version: "1.0.0"
org: "example-org"
projects:
  - id: "alpha"
  - id: "beta"
    path: "beta-project"
`)
	writeRepoFile(t, dir, "alpha/project.yaml", projectYAML("alpha", "Alpha"))
	writeRepoFile(t, dir, "alpha/maintainers.yaml", maintainersYAML("alpha", "example-org", "alpha-maintainers", "alice"))
	writeRepoFile(t, dir, "beta-project/project.yaml", projectYAML("beta", "Beta"))
	writeRepoFile(t, dir, "beta-project/maintainers.yaml", maintainersYAML("beta", "example-org", "beta-maintainers", "bob"))
	return dir
}

func hasErrorContaining(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

func TestDiscoverSingleProject(t *testing.T) {
	t.Run("project.yaml and maintainers.yaml at root", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoFile(t, dir, "project.yaml", projectYAML("solo", "Solo"))
		writeRepoFile(t, dir, "maintainers.yaml", maintainersYAML("solo", "", "maintainers", "alice"))

		d, err := Discover(dir)
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}
		if d.Mode != LayoutSingle {
			t.Errorf("mode = %q, want %q", d.Mode, LayoutSingle)
		}
		if d.Org != nil {
			t.Error("expected no org config for a single-project repo")
		}
		if len(d.Projects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(d.Projects))
		}
		if d.Projects[0].Dir != "" {
			t.Errorf("expected root project to have an empty dir, got %q", d.Projects[0].Dir)
		}
		if d.Projects[0].MaintainersPath == "" {
			t.Error("expected maintainers path to be set when the file exists")
		}
	})

	t.Run("maintainers.yaml is optional at root", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoFile(t, dir, "project.yaml", projectYAML("solo", "Solo"))

		d, err := Discover(dir)
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}
		if d.Projects[0].MaintainersPath != "" {
			t.Errorf("expected empty maintainers path, got %q", d.Projects[0].MaintainersPath)
		}
		if len(d.MaintainersPaths()) != 0 {
			t.Errorf("expected no maintainers paths, got %v", d.MaintainersPaths())
		}
	})

	t.Run("empty repo root is an error", func(t *testing.T) {
		if _, err := Discover(t.TempDir()); err == nil {
			t.Fatal("expected an error for a repo with neither org.yaml nor project.yaml")
		}
	})
}

func TestDiscoverMultiProject(t *testing.T) {
	dir := newMultiRepo(t)

	d, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if d.Mode != LayoutMulti {
		t.Fatalf("mode = %q, want %q", d.Mode, LayoutMulti)
	}
	if d.Org == nil || d.Org.Org != "example-org" {
		t.Fatalf("expected org example-org, got %+v", d.Org)
	}
	if len(d.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(d.Projects))
	}

	// path defaults to id, and an explicit path overrides it
	if d.Projects[0].Dir != "alpha" {
		t.Errorf("alpha dir = %q, want %q", d.Projects[0].Dir, "alpha")
	}
	if d.Projects[1].Dir != "beta-project" {
		t.Errorf("beta dir = %q, want %q", d.Projects[1].Dir, "beta-project")
	}

	if got := len(d.ProjectPaths()); got != 2 {
		t.Errorf("ProjectPaths returned %d entries, want 2", got)
	}
	if got := len(d.MaintainersPaths()); got != 2 {
		t.Errorf("MaintainersPaths returned %d entries, want 2", got)
	}
}

// Discover reports layout, not correctness: a declared project whose files are
// missing must still be discovered so ValidateRepo can name the missing file.
func TestDiscoverReportsLayoutNotCorrectness(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, "org.yaml", `schema_version: "1.0.0"
org: "example-org"
projects:
  - id: "ghost"
`)

	d, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if len(d.Projects) != 1 {
		t.Fatalf("expected the declared project to be discovered, got %d", len(d.Projects))
	}
	if d.Projects[0].ProjectPath == "" {
		t.Error("expected ProjectPath to be set even though the file is missing")
	}
}

func TestDiscoverOrgTakesPrecedence(t *testing.T) {
	dir := newMultiRepo(t)
	writeRepoFile(t, dir, "project.yaml", projectYAML("alpha", "Alpha"))

	d, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if d.Mode != LayoutMulti {
		t.Errorf("org.yaml must win over a root project.yaml, got mode %q", d.Mode)
	}
}

func TestLoadOrgFromFile(t *testing.T) {
	t.Run("rejects unknown fields", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoFile(t, dir, "org.yaml", `schema_version: "1.0.0"
org: "example-org"
defaults:
  governance:
    contributing:
      path: "CONTRIBUTING.md"
projects:
  - id: "alpha"
`)
		if _, err := LoadOrgFromFile(filepath.Join(dir, "org.yaml")); err == nil {
			t.Fatal("expected unknown field 'defaults' to be rejected")
		}
	})

	t.Run("parses the committed fixture", func(t *testing.T) {
		org, err := LoadOrgFromFile(filepath.Join("testdata", "multiproject", "org.yaml"))
		if err != nil {
			t.Fatalf("LoadOrgFromFile failed: %v", err)
		}
		if len(org.Projects) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(org.Projects))
		}
		if got := org.Projects[0].ResolvedPath(); got != "alpha" {
			t.Errorf("ResolvedPath with no path = %q, want %q", got, "alpha")
		}
		if got := org.Projects[1].ResolvedPath(); got != "beta-project" {
			t.Errorf("ResolvedPath with explicit path = %q, want %q", got, "beta-project")
		}
	})
}

func TestValidateOrgStruct(t *testing.T) {
	valid := OrgConfig{
		SchemaVersion: "1.0.0",
		Org:           "example-org",
		Projects:      []OrgProjectEntry{{ID: "alpha"}, {ID: "beta", Path: "beta-project"}},
	}

	if errs := ValidateOrgStruct(valid); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}

	tests := []struct {
		name    string
		mutate  func(*OrgConfig)
		wantErr string
	}{
		{"missing schema_version", func(o *OrgConfig) { o.SchemaVersion = "" }, "schema_version is required"},
		{"unsupported schema_version", func(o *OrgConfig) { o.SchemaVersion = "9.9.9" }, "unsupported org schema_version"},
		{"missing org", func(o *OrgConfig) { o.Org = "" }, "org is required"},
		{"org with slash", func(o *OrgConfig) { o.Org = "example/org" }, "without slashes or spaces"},
		{"no projects", func(o *OrgConfig) { o.Projects = nil }, "at least one project"},
		{"missing id", func(o *OrgConfig) { o.Projects[0].ID = "" }, "id is required"},
		{"invalid id", func(o *OrgConfig) { o.Projects[0].ID = "Alpha_One" }, "lowercase alphanumeric"},
		{"duplicate id", func(o *OrgConfig) { o.Projects[1].ID = "alpha" }, "duplicate project id"},
		{"duplicate path", func(o *OrgConfig) { o.Projects[1].Path = "alpha" }, "duplicate project path"},
		{"nested path", func(o *OrgConfig) { o.Projects[0].Path = "projects/alpha" }, "single directory name"},
		{"dot path", func(o *OrgConfig) { o.Projects[0].Path = ".github" }, "must not start with a dot"},
		{"parent path", func(o *OrgConfig) { o.Projects[0].Path = ".." }, "must be a directory name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := OrgConfig{
				SchemaVersion: valid.SchemaVersion,
				Org:           valid.Org,
				Projects:      []OrgProjectEntry{{ID: "alpha"}, {ID: "beta", Path: "beta-project"}},
			}
			tt.mutate(&org)

			errs := ValidateOrgStruct(org)
			if !hasErrorContaining(errs, tt.wantErr) {
				t.Errorf("expected an error containing %q, got %v", tt.wantErr, errs)
			}
		})
	}
}

func TestValidateRepoSingleProject(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoFile(t, dir, "project.yaml", projectYAML("solo", "Solo"))
		writeRepoFile(t, dir, "maintainers.yaml", maintainersYAML("solo", "", "maintainers", "alice"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !result.Valid {
			t.Fatalf("expected a valid repo, got errors %v", result.Errors)
		}
		if result.Mode != LayoutSingle {
			t.Errorf("mode = %q, want %q", result.Mode, LayoutSingle)
		}
	})

	t.Run("project_id must match slug", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoFile(t, dir, "project.yaml", projectYAML("solo", "Solo"))
		writeRepoFile(t, dir, "maintainers.yaml", maintainersYAML("typo", "", "maintainers", "alice"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if result.Valid {
			t.Fatal("expected a project_id/slug mismatch to be invalid")
		}
		if !hasErrorContaining(result.Errors, "does not match the slug") {
			t.Errorf("expected a slug mismatch error, got %v", result.Errors)
		}
	})
}

func TestValidateRepoMultiProject(t *testing.T) {
	t.Run("valid fixture on disk", func(t *testing.T) {
		result, err := ValidateRepo(filepath.Join("testdata", "multiproject"))
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !result.Valid {
			t.Fatalf("expected the committed fixture to be valid, got errors %v", result.Errors)
		}
		if len(result.Projects) != 2 {
			t.Errorf("expected 2 projects, got %v", result.Projects)
		}
		// steering-committee is shared with identical members: no warning.
		if len(result.Warnings) != 0 {
			t.Errorf("expected no warnings, got %v", result.Warnings)
		}
	})

	t.Run("root project.yaml is rejected", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "project.yaml", projectYAML("alpha", "Alpha"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !hasErrorContaining(result.Errors, "must not exist at the repository root") {
			t.Errorf("expected a root project.yaml error, got %v", result.Errors)
		}
	})

	t.Run("root maintainers.yaml is rejected", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "maintainers.yaml", maintainersYAML("alpha", "example-org", "maintainers", "alice"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !hasErrorContaining(result.Errors, "must not exist at the repository root") {
			t.Errorf("expected a root maintainers.yaml error, got %v", result.Errors)
		}
	})

	t.Run("missing per-project files are reported", func(t *testing.T) {
		dir := newMultiRepo(t)
		if err := os.Remove(filepath.Join(dir, "alpha", "maintainers.yaml")); err != nil {
			t.Fatalf("failed to remove file: %v", err)
		}
		if err := os.Remove(filepath.Join(dir, "beta-project", "project.yaml")); err != nil {
			t.Fatalf("failed to remove file: %v", err)
		}

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !hasErrorContaining(result.Errors, "alpha/maintainers.yaml") {
			t.Errorf("expected a missing maintainers error, got %v", result.Errors)
		}
		if !hasErrorContaining(result.Errors, "beta-project/project.yaml") {
			t.Errorf("expected a missing project error, got %v", result.Errors)
		}
	})

	t.Run("undeclared project directory is reported", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "gamma/project.yaml", projectYAML("gamma", "Gamma"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !hasErrorContaining(result.Errors, "is not declared in org.yaml") {
			t.Errorf("expected an undeclared directory error, got %v", result.Errors)
		}
	})

	t.Run("dot directories are not treated as projects", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, ".github/project.yaml", projectYAML("gamma", "Gamma"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected dot directories to be ignored, got errors %v", result.Errors)
		}
	})

	t.Run("slug must match the declared id", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "alpha/project.yaml", projectYAML("not-alpha", "Alpha"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !hasErrorContaining(result.Errors, "declares id") {
			t.Errorf("expected an id/slug mismatch error, got %v", result.Errors)
		}
	})

	t.Run("duplicate slugs across projects are rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoFile(t, dir, "org.yaml", `schema_version: "1.0.0"
org: "example-org"
projects:
  - id: "alpha"
  - id: "beta"
`)
		writeRepoFile(t, dir, "alpha/project.yaml", projectYAML("dup", "Alpha"))
		writeRepoFile(t, dir, "alpha/maintainers.yaml", maintainersYAML("dup", "example-org", "alpha-maintainers", "alice"))
		writeRepoFile(t, dir, "beta/project.yaml", projectYAML("dup", "Beta"))
		writeRepoFile(t, dir, "beta/maintainers.yaml", maintainersYAML("dup", "example-org", "beta-maintainers", "bob"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !hasErrorContaining(result.Errors, "duplicate slug") {
			t.Errorf("expected a duplicate slug error, got %v", result.Errors)
		}
	})

	t.Run("org mismatch is rejected", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "alpha/maintainers.yaml", maintainersYAML("alpha", "other-org", "alpha-maintainers", "alice"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !hasErrorContaining(result.Errors, "does not match the org") {
			t.Errorf("expected an org mismatch error, got %v", result.Errors)
		}
	})
}

// A team name shared across projects is legitimate: GitHub teams are
// org-scoped and a team's remit may span projects. Identical definitions are
// silent; differing definitions warn so a human confirms the sharing is
// deliberate, but never fail the build.
func TestValidateRepoSharedTeams(t *testing.T) {
	t.Run("identical members are silent", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "alpha/maintainers.yaml", maintainersYAML("alpha", "example-org", "security", "alice", "bob"))
		writeRepoFile(t, dir, "beta-project/maintainers.yaml", maintainersYAML("beta", "example-org", "security", "bob", "alice"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected no warnings for identical team definitions, got %v", result.Warnings)
		}
	})

	t.Run("differing members warn but stay valid", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "alpha/maintainers.yaml", maintainersYAML("alpha", "example-org", "authentication", "alice", "bob"))
		writeRepoFile(t, dir, "beta-project/maintainers.yaml", maintainersYAML("beta", "example-org", "authentication", "carol", "dave"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !result.Valid {
			t.Fatalf("a shared team must not fail validation, got errors %v", result.Errors)
		}
		if !hasErrorContaining(result.Warnings, "authentication") {
			t.Errorf("expected a shared team warning, got %v", result.Warnings)
		}
		if !hasErrorContaining(result.Warnings, "union") {
			t.Errorf("expected the warning to explain union semantics, got %v", result.Warnings)
		}
	})

	// GitHub matches teams by slug, so two spellings of one name are one team
	// and a differing roster is still a collision worth reporting.
	t.Run("differing spellings are still one team", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "alpha/maintainers.yaml", maintainersYAML("alpha", "example-org", "Release Managers", "alice"))
		writeRepoFile(t, dir, "beta-project/maintainers.yaml", maintainersYAML("beta", "example-org", "release-managers", "bob"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if !result.Valid {
			t.Fatalf("a shared team must not fail validation, got errors %v", result.Errors)
		}
		if !hasErrorContaining(result.Warnings, "release-managers") {
			t.Errorf("expected a shared team warning naming the slug, got %v", result.Warnings)
		}
		for _, spelling := range []string{"Release Managers", "release-managers"} {
			if !hasErrorContaining(result.Warnings, spelling) {
				t.Errorf("expected the warning to list the spelling %q, got %v", spelling, result.Warnings)
			}
		}
	})

	t.Run("differing spellings with identical members are silent", func(t *testing.T) {
		dir := newMultiRepo(t)
		writeRepoFile(t, dir, "alpha/maintainers.yaml", maintainersYAML("alpha", "example-org", "Release Managers", "alice"))
		writeRepoFile(t, dir, "beta-project/maintainers.yaml", maintainersYAML("beta", "example-org", "release-managers", "alice"))

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("ValidateRepo failed: %v", err)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected no warnings for identical team definitions, got %v", result.Warnings)
		}
	})
}

func TestFormatRepoValidationResult(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		out := FormatRepoValidationResult(&RepoValidationResult{
			Mode: LayoutMulti, Projects: []string{"alpha", "beta"}, Valid: true,
		})
		if !strings.Contains(out, "multi") || !strings.Contains(out, "alpha, beta") {
			t.Errorf("unexpected output: %s", out)
		}
		if !strings.Contains(out, "valid") {
			t.Errorf("expected a validity line, got: %s", out)
		}
	})

	t.Run("invalid with warnings", func(t *testing.T) {
		out := FormatRepoValidationResult(&RepoValidationResult{
			Mode:     LayoutMulti,
			Valid:    false,
			Errors:   []string{"something is wrong"},
			Warnings: []string{"heads up"},
		})
		if !strings.Contains(out, "WARNING: heads up") {
			t.Errorf("expected the warning to be rendered, got: %s", out)
		}
		if !strings.Contains(out, "something is wrong") {
			t.Errorf("expected the error to be rendered, got: %s", out)
		}
	})
}
