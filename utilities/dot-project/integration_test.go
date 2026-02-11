package projects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Project YAML fixture tests (testdata/ and example/)
// ---------------------------------------------------------------------------

func TestYAMLFixtures(t *testing.T) {
	tests := []struct {
		file        string
		expectValid bool
	}{
		{"testdata/test-project.yaml", true},
		{"example/project.yaml", true},
		{"testdata/bad-project.yaml", false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tt.file, err)
			}

			var project Project
			decoder := yaml.NewDecoder(strings.NewReader(string(data)))
			decoder.KnownFields(true)
			if err := decoder.Decode(&project); err != nil {
				if tt.expectValid {
					t.Fatalf("expected %s to parse without error, got: %v", tt.file, err)
				}
				return // Parse error on invalid file is expected
			}

			errs := validateProjectStruct(project)
			if tt.expectValid && len(errs) > 0 {
				t.Errorf("expected %s to be valid, got errors: %v", tt.file, errs)
			}
			if !tt.expectValid && len(errs) == 0 {
				t.Errorf("expected %s to have validation errors", tt.file)
			}
		})
	}
}

func TestStrictYAMLParsing(t *testing.T) {
	yamlContent := `
name: "Test"
description: "Test"
slug: "test"
schema_version: "1.0.0"
unknown_field: "should fail"
maturity_log:
  - phase: "sandbox"
    date: "2024-01-01T00:00:00Z"
    issue: "https://github.com/cncf/toc/issues/1"
repositories:
  - "https://github.com/test/repo"
`
	var project Project
	decoder := yaml.NewDecoder(strings.NewReader(yamlContent))
	decoder.KnownFields(true)
	err := decoder.Decode(&project)
	if err == nil {
		t.Error("expected error for unknown field, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected unknown field error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Maintainer YAML fixture tests (testdata/ and example/)
// ---------------------------------------------------------------------------

func TestMaintainerFixtures(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	pv := NewValidator(cacheDir)

	// Disable external verification for fixture tests
	t.Setenv("LFX_AUTH_TOKEN", "")
	t.Setenv("MAINTAINER_API_ENDPOINT", "")

	tests := []struct {
		file        string
		expectValid bool
		expectCount int // expected number of maintainer entries
	}{
		{"testdata/maintainers.yaml", true, 1},
		{"example/maintainers.yaml", true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			results, err := pv.ValidateMaintainersFile(tt.file, false)
			if err != nil {
				t.Fatalf("failed to validate %s: %v", tt.file, err)
			}
			if len(results) != tt.expectCount {
				t.Fatalf("expected %d maintainer entries, got %d", tt.expectCount, len(results))
			}
			for _, r := range results {
				if tt.expectValid && !r.Valid {
					t.Errorf("expected %s entry %q to be valid, got errors: %v", tt.file, r.ProjectID, r.Errors)
				}
				if !tt.expectValid && r.Valid {
					t.Errorf("expected %s entry %q to have validation errors", tt.file, r.ProjectID)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Project list fixture test (testdata/)
// ---------------------------------------------------------------------------

func TestProjectListFixture(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	config := &Config{
		ProjectListURL: "testdata/projectlist.yaml",
		CacheDir:       cacheDir,
		OutputFormat:   "text",
	}
	pv := &ProjectValidator{config: config}

	urls, err := pv.loadProjectList()
	if err != nil {
		t.Fatalf("failed to load testdata/projectlist.yaml: %v", err)
	}
	if len(urls) == 0 {
		t.Fatal("expected at least one project URL in testdata/projectlist.yaml")
	}
	if !strings.Contains(urls[0], "test-project.yaml") {
		t.Errorf("expected first URL to reference test-project.yaml, got %q", urls[0])
	}
}

// ---------------------------------------------------------------------------
// End-to-end: example/ folder validates as a complete .project repo
// ---------------------------------------------------------------------------

func TestExampleFolderE2E(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	pv := NewValidator(cacheDir)

	// Disable external verification
	t.Setenv("LFX_AUTH_TOKEN", "")
	t.Setenv("MAINTAINER_API_ENDPOINT", "")

	// 1. Load and validate the project via LoadProjectFromFile
	t.Run("project loads and validates via LoadProjectFromFile", func(t *testing.T) {
		project, err := LoadProjectFromFile("example/project.yaml")
		if err != nil {
			t.Fatalf("LoadProjectFromFile failed: %v", err)
		}

		// Verify key fields are populated
		if project.Name != "Kubernetes" {
			t.Errorf("expected name 'Kubernetes', got %q", project.Name)
		}
		if project.Slug != "kubernetes" {
			t.Errorf("expected slug 'kubernetes', got %q", project.Slug)
		}
		if project.SchemaVersion != "1.0.0" {
			t.Errorf("expected schema_version '1.0.0', got %q", project.SchemaVersion)
		}
		if len(project.MaturityLog) < 3 {
			t.Errorf("expected at least 3 maturity entries, got %d", len(project.MaturityLog))
		}
		if len(project.Repositories) < 1 {
			t.Errorf("expected at least 1 repository, got %d", len(project.Repositories))
		}

		// Struct validation should produce zero errors
		errs := validateProjectStruct(project)
		if len(errs) > 0 {
			t.Errorf("example project should be valid, got errors: %v", errs)
		}
	})

	// 2. Validate maintainers
	t.Run("maintainers validate successfully", func(t *testing.T) {
		results, err := pv.ValidateMaintainersFile("example/maintainers.yaml", false)
		if err != nil {
			t.Fatalf("maintainer validation failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 maintainer entry, got %d", len(results))
		}
		r := results[0]
		if !r.Valid {
			t.Errorf("expected maintainers to be valid, got errors: %v", r.Errors)
		}
		if r.ProjectID != "kubernetes" {
			t.Errorf("expected project_id 'kubernetes', got %q", r.ProjectID)
		}
	})

	// 3. Landscape conversion produces expected output
	t.Run("landscape conversion produces valid entry", func(t *testing.T) {
		project, err := LoadProjectFromFile("example/project.yaml")
		if err != nil {
			t.Fatalf("LoadProjectFromFile failed: %v", err)
		}

		entry := ProjectToLandscapeEntry(project)
		if entry.Name != "Kubernetes" {
			t.Errorf("expected landscape name 'Kubernetes', got %q", entry.Name)
		}
		if entry.HomepageURL != "https://kubernetes.io" {
			t.Errorf("expected homepage 'https://kubernetes.io', got %q", entry.HomepageURL)
		}
		if entry.RepoURL != "https://github.com/kubernetes/kubernetes" {
			t.Errorf("expected repo URL 'https://github.com/kubernetes/kubernetes', got %q", entry.RepoURL)
		}
		if entry.Twitter != "https://twitter.com/kubernetesio" {
			t.Errorf("expected twitter 'https://twitter.com/kubernetesio', got %q", entry.Twitter)
		}
		if entry.Project != "graduated" {
			t.Errorf("expected project maturity 'graduated', got %q", entry.Project)
		}
	})

	// 4. Maintainer slug matches project slug (cross-file consistency)
	t.Run("maintainer project_id matches project slug", func(t *testing.T) {
		project, err := LoadProjectFromFile("example/project.yaml")
		if err != nil {
			t.Fatalf("LoadProjectFromFile failed: %v", err)
		}

		results, err := pv.ValidateMaintainersFile("example/maintainers.yaml", false)
		if err != nil {
			t.Fatalf("maintainer validation failed: %v", err)
		}

		if results[0].ProjectID != project.Slug {
			t.Errorf("maintainer project_id %q does not match project slug %q",
				results[0].ProjectID, project.Slug)
		}
	})

	// 5. Extract handles from example maintainers
	t.Run("handle extraction works on example maintainers", func(t *testing.T) {
		handles, err := pv.ExtractHandles("example/maintainers.yaml")
		if err != nil {
			t.Fatalf("ExtractHandles failed: %v", err)
		}
		if len(handles) == 0 {
			t.Fatal("expected at least one handle extracted from example maintainers")
		}
		// thockin should be present (project lead in example)
		if !handles["thockin"] {
			t.Error("expected handle 'thockin' to be present in example maintainers")
		}
	})

	// 6. CI workflow file exists and is valid YAML
	t.Run("workflow file exists and parses as valid YAML", func(t *testing.T) {
		data, err := os.ReadFile("example/.github/workflows/validate.yaml")
		if err != nil {
			t.Fatalf("failed to read example workflow: %v", err)
		}
		var workflow map[string]interface{}
		if err := yaml.Unmarshal(data, &workflow); err != nil {
			t.Fatalf("example workflow is not valid YAML: %v", err)
		}
		if _, ok := workflow["name"]; !ok {
			t.Error("workflow missing 'name' field")
		}
		if _, ok := workflow["jobs"]; !ok {
			t.Error("workflow missing 'jobs' field")
		}
	})
}

// ---------------------------------------------------------------------------
// Testdata fixtures: ensure test-project round-trips through LoadProjectFromFile
// ---------------------------------------------------------------------------

func TestTestdataProjectLoadAndValidate(t *testing.T) {
	project, err := LoadProjectFromFile("testdata/test-project.yaml")
	if err != nil {
		t.Fatalf("LoadProjectFromFile failed for testdata/test-project.yaml: %v", err)
	}

	if project.Slug != "test-project" {
		t.Errorf("expected slug 'test-project', got %q", project.Slug)
	}

	errs := validateProjectStruct(project)
	if len(errs) > 0 {
		t.Errorf("testdata/test-project.yaml should be valid, got errors: %v", errs)
	}

	// Also check landscape conversion
	entry := ProjectToLandscapeEntry(project)
	if entry.Name != "Test Project - Updated" {
		t.Errorf("expected landscape name 'Test Project - Updated', got %q", entry.Name)
	}
}
