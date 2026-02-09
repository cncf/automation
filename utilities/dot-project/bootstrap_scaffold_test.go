package projects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateProjectYAML(t *testing.T) {
	t.Run("generates valid YAML with all fields", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:                 "test-project",
			Name:                 "Test Project",
			Description:          "A test project description",
			Website:              "https://test-project.io",
			Repositories:         []string{"https://github.com/test-org/test-project"},
			Artwork:              "https://github.com/cncf/artwork/tree/master/projects/test-project",
			Social:               map[string]string{"twitter": "https://twitter.com/testproject"},
			MaturityPhase:        "incubating",
			LandscapeCategory:    "Observability",
			LandscapeSubcategory: "Monitoring",
			ProjectLead:          "alice",
			CNCFSlackChannel:     "#test-project",
			HasSecurityPolicy:    true,
			HasContributing:      true,
			HasLicense:           true,
			HasReadme:            true,
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("GenerateProjectYAML() error = %v", err)
		}

		yamlStr := string(output)

		// Should contain key fields
		if !strings.Contains(yamlStr, "slug: \"test-project\"") {
			t.Error("output should contain slug")
		}
		if !strings.Contains(yamlStr, "name: \"Test Project\"") {
			t.Error("output should contain name")
		}
		if !strings.Contains(yamlStr, "schema_version: \"1.0.0\"") {
			t.Error("output should contain schema_version")
		}

		// Should be parseable YAML (after stripping comments)
		var project Project
		decoder := yaml.NewDecoder(strings.NewReader(yamlStr))
		if err := decoder.Decode(&project); err != nil {
			t.Fatalf("generated YAML is not parseable: %v\n---\n%s", err, yamlStr)
		}

		if project.Slug != "test-project" {
			t.Errorf("parsed Slug = %q, want %q", project.Slug, "test-project")
		}
		if project.Name != "Test Project" {
			t.Errorf("parsed Name = %q, want %q", project.Name, "Test Project")
		}
	})

	t.Run("includes TODO comments for missing fields", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:  "minimal-project",
			Name:  "Minimal Project",
			TODOs: []string{"Add project description", "Set maturity phase"},
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("GenerateProjectYAML() error = %v", err)
		}

		yamlStr := string(output)
		if !strings.Contains(yamlStr, "# TODO:") {
			t.Error("output should contain TODO comments")
		}
	})

	t.Run("round-trip: generate and validate", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:          "roundtrip-test",
			Name:          "Roundtrip Test",
			Description:   "Testing round trip",
			MaturityPhase: "sandbox",
			Repositories:  []string{"https://github.com/test/roundtrip"},
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("GenerateProjectYAML() error = %v", err)
		}

		var project Project
		if err := yaml.Unmarshal(output, &project); err != nil {
			t.Fatalf("cannot parse generated YAML: %v", err)
		}

		errors := ValidateProjectStruct(project)
		// Should have no errors for the fields that are set
		for _, e := range errors {
			// Maturity log issue and date are TODOs, so expect those errors
			if !strings.Contains(e, "maturity_log") {
				t.Errorf("unexpected validation error: %s", e)
			}
		}
	})
}

func TestGenerateMaintainersYAML(t *testing.T) {
	t.Run("generates valid maintainers YAML", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:        "test-project",
			Maintainers: []string{"alice", "bob", "carol"},
		}

		output, err := GenerateMaintainersYAML(result)
		if err != nil {
			t.Fatalf("GenerateMaintainersYAML() error = %v", err)
		}

		yamlStr := string(output)
		if !strings.Contains(yamlStr, "project_id: \"test-project\"") {
			t.Error("output should contain project_id")
		}
		if !strings.Contains(yamlStr, "alice") {
			t.Error("output should contain maintainer alice")
		}

		// Should be parseable
		var config MaintainersConfig
		if err := yaml.Unmarshal(output, &config); err != nil {
			t.Fatalf("generated YAML is not parseable: %v\n---\n%s", err, yamlStr)
		}

		if len(config.Maintainers) != 1 {
			t.Fatalf("expected 1 maintainer entry, got %d", len(config.Maintainers))
		}
		if config.Maintainers[0].ProjectID != "test-project" {
			t.Errorf("ProjectID = %q, want %q", config.Maintainers[0].ProjectID, "test-project")
		}
	})

	t.Run("handles empty maintainers list", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:        "test-project",
			Maintainers: nil,
		}

		output, err := GenerateMaintainersYAML(result)
		if err != nil {
			t.Fatalf("GenerateMaintainersYAML() error = %v", err)
		}

		yamlStr := string(output)
		if !strings.Contains(yamlStr, "# TODO:") {
			t.Error("output should contain TODO for missing maintainers")
		}
	})
}

func TestWriteScaffold(t *testing.T) {
	t.Run("writes all scaffold files", func(t *testing.T) {
		dir := t.TempDir()
		result := &BootstrapResult{
			Slug:          "test-project",
			Name:          "Test Project",
			Description:   "A test project",
			MaturityPhase: "sandbox",
			Repositories:  []string{"https://github.com/test/test-project"},
			Maintainers:   []string{"alice"},
		}

		err := WriteScaffold(dir, result)
		if err != nil {
			t.Fatalf("WriteScaffold() error = %v", err)
		}

		// Check project.yaml exists
		if _, err := os.Stat(filepath.Join(dir, "project.yaml")); os.IsNotExist(err) {
			t.Error("project.yaml not created")
		}

		// Check maintainers.yaml exists
		if _, err := os.Stat(filepath.Join(dir, "maintainers.yaml")); os.IsNotExist(err) {
			t.Error("maintainers.yaml not created")
		}

		// Check workflow file exists
		workflowPath := filepath.Join(dir, ".github", "workflows", "validate.yaml")
		if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
			t.Error("validate.yaml workflow not created")
		}
	})

	t.Run("does not overwrite existing files", func(t *testing.T) {
		dir := t.TempDir()
		existingContent := "existing content"
		os.WriteFile(filepath.Join(dir, "project.yaml"), []byte(existingContent), 0644)

		result := &BootstrapResult{
			Slug: "test-project",
			Name: "Test Project",
		}

		err := WriteScaffold(dir, result)
		if err == nil {
			t.Fatal("expected error when file exists")
		}

		// Verify original content is preserved
		data, _ := os.ReadFile(filepath.Join(dir, "project.yaml"))
		if string(data) != existingContent {
			t.Error("existing file was overwritten")
		}
	})
}
