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
			GitHubOrg:            "test-org",
			GitHubRepo:           "test-project",
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

		// Should always have type: "project"
		if !strings.Contains(yamlStr, `type: "project"`) {
			t.Error("output should always contain type: project")
		}

		// Should use full GitHub URLs for paths
		if !strings.Contains(yamlStr, "https://github.com/test-org/test-project/blob/main/") {
			t.Error("output should contain full GitHub URLs for paths")
		}

		// Should reference CNCF Code of Conduct
		if !strings.Contains(yamlStr, "cncf/foundation/blob/main/code-of-conduct.md") {
			t.Error("output should reference CNCF Code of Conduct")
		}

		// Should always include identity_type defaults
		if !strings.Contains(yamlStr, "has_dco: true") {
			t.Error("output should include has_dco: true default")
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

	t.Run("uses discovered URLs instead of defaults", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:              "test-project",
			Name:              "Test Project",
			Description:       "A test project",
			GitHubOrg:         "test-org",
			GitHubRepo:        "test-project",
			MaturityPhase:     "sandbox",
			Repositories:      []string{"https://github.com/test-org/test-project"},
			SecurityPolicyURL: "https://github.com/test-org/.github/blob/main/SECURITY.md",
			ContributingURL:   "https://github.com/test-org/.github/blob/main/CONTRIBUTING.md",
			CodeOfConductURL:  "https://github.com/test-org/.github/blob/main/CODE_OF_CONDUCT.md",
			LicenseURL:        "https://github.com/test-org/test-project/blob/main/LICENSE",
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("GenerateProjectYAML() error = %v", err)
		}

		yamlStr := string(output)

		// Security policy should use discovered org-level URL
		if !strings.Contains(yamlStr, "https://github.com/test-org/.github/blob/main/SECURITY.md") {
			t.Error("security.policy.path should use discovered org-level SECURITY.md URL")
		}
		// Contributing should use discovered URL
		if !strings.Contains(yamlStr, "https://github.com/test-org/.github/blob/main/CONTRIBUTING.md") {
			t.Error("governance.contributing.path should use discovered CONTRIBUTING.md URL")
		}
		// Code of conduct should use discovered URL instead of CNCF default
		if !strings.Contains(yamlStr, "https://github.com/test-org/.github/blob/main/CODE_OF_CONDUCT.md") {
			t.Error("governance.code_of_conduct.path should use discovered CODE_OF_CONDUCT.md URL")
		}
		if strings.Contains(yamlStr, "cncf/foundation/blob/main/code-of-conduct.md") {
			t.Error("governance.code_of_conduct.path should NOT contain CNCF default when discovered URL exists")
		}
		// License should use discovered URL
		if !strings.Contains(yamlStr, "https://github.com/test-org/test-project/blob/main/LICENSE") {
			t.Error("legal.license.path should use discovered LICENSE URL")
		}
	})

	t.Run("round-trip: generate and validate", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:          "roundtrip-test",
			Name:          "Roundtrip Test",
			Description:   "Testing round trip",
			GitHubOrg:     "test-org",
			GitHubRepo:    "roundtrip",
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
			GitHubOrg:   "test-org",
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
		if !strings.Contains(yamlStr, `org: "test-org"`) {
			t.Error("output should contain org field from GitHubOrg")
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
	t.Run("writes all 8 scaffold files", func(t *testing.T) {
		dir := t.TempDir()
		result := &BootstrapResult{
			Slug:          "test-project",
			Name:          "Test Project",
			Description:   "A test project",
			GitHubOrg:     "test-org",
			GitHubRepo:    "test-project",
			MaturityPhase: "sandbox",
			Repositories:  []string{"https://github.com/test-org/test-project"},
			Maintainers:   []string{"alice"},
		}

		err := WriteScaffold(dir, result)
		if err != nil {
			t.Fatalf("WriteScaffold() error = %v", err)
		}

		// All 8 files must exist
		expectedFiles := []string{
			"project.yaml",
			"maintainers.yaml",
			"README.md",
			"SECURITY.md",
			"CODEOWNERS",
			".gitignore",
			filepath.Join(".github", "workflows", "validate.yaml"),
			filepath.Join(".github", "workflows", "update-landscape.yml"),
		}
		for _, f := range expectedFiles {
			if _, err := os.Stat(filepath.Join(dir, f)); os.IsNotExist(err) {
				t.Errorf("%s not created", f)
			}
		}

		// Spot-check README content
		readmeData, _ := os.ReadFile(filepath.Join(dir, "README.md"))
		if !strings.Contains(string(readmeData), "Test Project") {
			t.Error("README.md should contain project name")
		}

		// Spot-check SECURITY.md uses advisory URL
		secData, _ := os.ReadFile(filepath.Join(dir, "SECURITY.md"))
		if !strings.Contains(string(secData), "test-org/test-project/security/advisories/new") {
			t.Error("SECURITY.md should contain advisory URL")
		}

		// Spot-check CODEOWNERS contains maintainer
		coData, _ := os.ReadFile(filepath.Join(dir, "CODEOWNERS"))
		if !strings.Contains(string(coData), "@alice") {
			t.Error("CODEOWNERS should contain @alice")
		}

		// Spot-check validate.yaml uses SHA-pinned refs
		valData, _ := os.ReadFile(filepath.Join(dir, ".github", "workflows", "validate.yaml"))
		valStr := string(valData)
		if !strings.Contains(valStr, "@de0fac2e4500dabe0009e67214ff5f5447ce83dd") {
			t.Error("validate.yaml should SHA-pin actions/checkout")
		}
		if !strings.Contains(valStr, "@95d25b12337a14e4a74f690c856f6903584e839e") {
			t.Error("validate.yaml should SHA-pin cncf/automation actions")
		}
		if strings.Contains(valStr, "@main") {
			t.Error("validate.yaml should not reference @main for cncf/automation actions")
		}

		// Spot-check update-landscape.yml uses correct secret and SHA-pinned refs
		lsData, _ := os.ReadFile(filepath.Join(dir, ".github", "workflows", "update-landscape.yml"))
		lsStr := string(lsData)
		if !strings.Contains(lsStr, "LANDSCAPE_REPO_TOKEN") {
			t.Error("update-landscape.yml should use LANDSCAPE_REPO_TOKEN secret")
		}
		if !strings.Contains(lsStr, "landscape-update@95d25b12337a14e4a74f690c856f6903584e839e") {
			t.Error("update-landscape.yml should SHA-pin landscape-update action")
		}
		if strings.Contains(lsStr, "uses: cncf/automation/.github/workflows/") {
			t.Error("update-landscape.yml should use composite action pattern, not reusable workflow")
		}

		// Spot-check .gitignore content
		giData, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if !strings.Contains(string(giData), ".DS_Store") {
			t.Error(".gitignore should contain .DS_Store")
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

func TestWriteScaffold_SkipsExistingFiles(t *testing.T) {
	t.Run("skips SECURITY.md when SecurityPolicyURL is set", func(t *testing.T) {
		dir := t.TempDir()
		result := &BootstrapResult{
			Slug:              "test-project",
			Name:              "Test Project",
			Description:       "A test project",
			GitHubOrg:         "test-org",
			GitHubRepo:        "test-project",
			MaturityPhase:     "sandbox",
			Repositories:      []string{"https://github.com/test-org/test-project"},
			Maintainers:       []string{"alice"},
			SecurityPolicyURL: "https://github.com/test-org/.github/blob/main/SECURITY.md",
		}

		err := WriteScaffold(dir, result)
		if err != nil {
			t.Fatalf("WriteScaffold() error = %v", err)
		}

		// SECURITY.md should NOT be created
		if _, err := os.Stat(filepath.Join(dir, "SECURITY.md")); !os.IsNotExist(err) {
			t.Error("SECURITY.md should NOT be created when SecurityPolicyURL is set")
		}

		// Other files should still exist
		for _, f := range []string{"project.yaml", "maintainers.yaml", "README.md", "CODEOWNERS"} {
			if _, err := os.Stat(filepath.Join(dir, f)); os.IsNotExist(err) {
				t.Errorf("%s should still be created", f)
			}
		}
	})

	t.Run("skips CODEOWNERS when it already exists on disk", func(t *testing.T) {
		dir := t.TempDir()
		existingContent := "* @existing-owner\n"
		os.WriteFile(filepath.Join(dir, "CODEOWNERS"), []byte(existingContent), 0644)

		result := &BootstrapResult{
			Slug:        "test-project",
			Name:        "Test Project",
			GitHubOrg:   "test-org",
			GitHubRepo:  "test-project",
			Maintainers: []string{"alice"},
		}

		err := WriteScaffold(dir, result)
		if err != nil {
			t.Fatalf("WriteScaffold() error = %v", err)
		}

		// CODEOWNERS should be preserved (not overwritten)
		data, _ := os.ReadFile(filepath.Join(dir, "CODEOWNERS"))
		if string(data) != existingContent {
			t.Errorf("CODEOWNERS was overwritten; got %q, want %q", string(data), existingContent)
		}
	})
}

func TestGenerateProjectYAML_AutoDetected(t *testing.T) {
	t.Run("uses auto-detected slack channel with verification comment", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:             "test-project",
			Name:             "Test Project",
			Description:      "A test",
			GitHubOrg:        "test-org",
			GitHubRepo:       "test-project",
			CNCFSlackChannel: "#test-project",
			Sources:          map[string]string{"cncf_slack_channel": "landscape"},
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		yamlStr := string(output)

		if !strings.Contains(yamlStr, `cncf_slack_channel: "#test-project"`) {
			t.Error("should contain cncf_slack_channel value")
		}
		if !strings.Contains(yamlStr, "AUTO-DETECTED") {
			t.Error("should contain AUTO-DETECTED verification comment")
		}
	})

	t.Run("uses auto-detected TOC issue URL", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:          "test-project",
			Name:          "Test Project",
			Description:   "A test",
			GitHubOrg:     "test-org",
			GitHubRepo:    "test-project",
			MaturityPhase: "sandbox",
			TOCIssueURL:   "https://github.com/cncf/toc/pull/1143",
			Sources:       map[string]string{"toc_issue_url": "landscape"},
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		yamlStr := string(output)

		if !strings.Contains(yamlStr, "https://github.com/cncf/toc/pull/1143") {
			t.Error("should contain auto-detected TOC URL")
		}
		if strings.Contains(yamlStr, "cncf/toc/issues/XXX") {
			t.Error("should NOT contain placeholder XXX when URL was auto-detected")
		}
	})

	t.Run("uses auto-detected DCO/CLA values", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:        "test-project",
			Name:        "Test Project",
			Description: "A test",
			GitHubOrg:   "test-org",
			GitHubRepo:  "test-project",
			HasDCO:      true,
			HasCLA:      false,
			Sources:     map[string]string{"identity_type": "github"},
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		yamlStr := string(output)

		if !strings.Contains(yamlStr, "has_dco: true") {
			t.Error("should contain has_dco: true")
		}
		if !strings.Contains(yamlStr, "has_cla: false") {
			t.Error("should contain has_cla: false")
		}
		if !strings.Contains(yamlStr, "AUTO-DETECTED") {
			t.Error("should contain AUTO-DETECTED verification comment for identity_type")
		}
	})

	t.Run("keeps TOC placeholder when not auto-detected", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:          "test-project",
			Name:          "Test Project",
			Description:   "A test",
			GitHubOrg:     "test-org",
			GitHubRepo:    "test-project",
			MaturityPhase: "sandbox",
			Sources:       map[string]string{},
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		yamlStr := string(output)

		if !strings.Contains(yamlStr, "cncf/toc/issues/XXX") {
			t.Error("should contain placeholder XXX when TOC URL not auto-detected")
		}
	})

	t.Run("uses default identity_type when not auto-detected", func(t *testing.T) {
		result := &BootstrapResult{
			Slug:        "test-project",
			Name:        "Test Project",
			Description: "A test",
			GitHubOrg:   "test-org",
			GitHubRepo:  "test-project",
			Sources:     map[string]string{},
		}

		output, err := GenerateProjectYAML(result)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		yamlStr := string(output)

		if !strings.Contains(yamlStr, "has_dco: true") {
			t.Error("should contain default has_dco: true")
		}
		if !strings.Contains(yamlStr, "has_cla: false") {
			t.Error("should contain default has_cla: false")
		}
		if strings.Contains(yamlStr, "AUTO-DETECTED") {
			t.Error("should NOT contain AUTO-DETECTED when identity_type is not auto-detected")
		}
	})
}
