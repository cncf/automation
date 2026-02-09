package projects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidator(t *testing.T) {
	// Create a test cache directory
	cacheDir := ".test-cache"
	defer os.RemoveAll(cacheDir)

	_ = NewValidator(cacheDir) // Test that constructor works

	// Test validation of a valid project
	validProject := Project{
		SchemaVersion: "1.0.0",
		Slug:          "test-project",
		Name:          "Test Project",
		Description:   "A valid test project",
		MaturityLog: []MaturityEntry{
			{
				Phase: "incubating",
				Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Issue: "https://github.com/cncf/toc/issues/123",
			},
		},
		Repositories: []string{"https://github.com/test/repo"},
		Website:      "https://test.io",
		Artwork:      "https://test.io/artwork",
		Audits: []Audit{
			{
				Date: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
				Type: "security",
				URL:  "https://test.io/audit.pdf",
			},
		},
	}

	errors := validateProjectStruct(validProject)
	if len(errors) != 0 {
		t.Errorf("Expected valid project to have no errors, got: %v", errors)
	}

	// Test validation of an invalid project
	invalidProject := Project{
		Name:         "",                // Missing required field
		Description:  "",                // Missing required field
		MaturityLog:  []MaturityEntry{}, // Empty required field
		Repositories: []string{},        // Empty required field
		Website:      "invalid-url",     // Invalid URL
		Artwork:      "also-invalid",    // Invalid URL
	}

	errors = validateProjectStruct(invalidProject)
	expectedErrors := []string{
		"name is required",
		"description is required",
		"slug is required",
		"schema_version is required",
		"maturity_log is required and cannot be empty",
		"repositories is required and cannot be empty",
		"website is not a valid URL: invalid-url",
		"artwork is not a valid URL: also-invalid",
	}

	if len(errors) != len(expectedErrors) {
		t.Errorf("Expected %d errors, got %d: %v", len(expectedErrors), len(errors), errors)
	}

	for i, expectedError := range expectedErrors {
		if i >= len(errors) || errors[i] != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, errors[i])
		}
	}
}

func TestMaturityLogValidation(t *testing.T) {
	project := validBaseProject()
	project.MaturityLog = []MaturityEntry{
		{
			Phase: "",          // Missing phase
			Date:  time.Time{}, // Zero date
			Issue: "",          // Missing issue
		},
	}

	errors := validateProjectStruct(project)
	expectedErrors := []string{
		"maturity_log[0].phase is required",
		"maturity_log[0].date is required",
		"maturity_log[0].issue is required",
	}

	for _, expectedError := range expectedErrors {
		found := false
		for _, error := range errors {
			if error == expectedError {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error '%s' not found in: %v", expectedError, errors)
		}
	}
}

func TestAuditsValidation(t *testing.T) {
	project := validBaseProject()
	project.Audits = []Audit{
		{
			Date: time.Time{},   // Zero date
			Type: "",            // Missing type
			URL:  "invalid-url", // Invalid URL
		},
	}

	errors := validateProjectStruct(project)
	expectedErrors := []string{
		"audits[0].date is required",
		"audits[0].type is required",
		"audits[0].url is not a valid URL: invalid-url",
	}

	for _, expectedError := range expectedErrors {
		found := false
		for _, error := range errors {
			if error == expectedError {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error '%s' not found in: %v", expectedError, errors)
		}
	}
}

func TestRepositoriesValidation(t *testing.T) {
	project := validBaseProject()
	project.Repositories = []string{
		"https://github.com/test/repo", // Valid
		"invalid-url",                  // Invalid
		"",                             // Empty
	}

	errors := validateProjectStruct(project)
	expectedErrors := []string{
		"repositories[1] is not a valid URL: invalid-url",
		"repositories[2] is not a valid URL: ",
	}

	for _, expectedError := range expectedErrors {
		found := false
		for _, error := range errors {
			if error == expectedError {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error '%s' not found in: %v", expectedError, errors)
		}
	}
}

func TestHashCalculation(t *testing.T) {
	content1 := "test content"
	content2 := "different content"
	content3 := "test content" // Same as content1

	hash1 := calculateHash(content1)
	hash2 := calculateHash(content2)
	hash3 := calculateHash(content3)

	if hash1 == hash2 {
		t.Error("Different content should produce different hashes")
	}

	if hash1 != hash3 {
		t.Error("Same content should produce same hashes")
	}

	// Check hash format (should be hex string)
	if len(hash1) != 64 { // SHA256 produces 64 character hex string
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}

func TestIsValidURL(t *testing.T) {
	testCases := []struct {
		url   string
		valid bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://github.com/user/repo", true},
		{"ftp://example.com", false},
		{"example.com", false},
		{"", false},
		{"not-a-url", false},
		{"https://", false},
		{"https://x", false},               // No domain (no dot)
		{"https://.", false},               // Invalid domain
		{"https://example.com/path", true}, // URL with path
	}

	for _, tc := range testCases {
		result := isValidURL(tc.url)
		if result != tc.valid {
			t.Errorf("isValidURL(%q) = %v, expected %v", tc.url, result, tc.valid)
		}
	}
}

func TestNormalizeHandles(t *testing.T) {
	input := []string{" Alice ", "bob", "@CarOl", "bob", ""}
	cleaned, errors := normalizeHandles(input)

	if len(cleaned) != 3 {
		t.Fatalf("expected 3 cleaned handles, got %d (%v)", len(cleaned), cleaned)
	}

	expected := map[string]struct{}{"Alice": {}, "bob": {}, "CarOl": {}}
	for _, handle := range cleaned {
		if _, ok := expected[handle]; !ok {
			t.Fatalf("unexpected handle in cleaned slice: %s (values: %v)", handle, cleaned)
		}
	}

	if len(errors) == 0 {
		t.Fatalf("expected duplicate/empty errors, got none")
	}
}

func TestValidateMaintainersFile(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	validator := NewValidator(cacheDir)

	maintainersPath := filepath.Join(tempDir, "maintainers.yaml")
	maintainersContent := `maintainers:
- project_id: "test-project"
  teams:
    - name: "project-maintainers"
      members:
        - alice
        - bob
`
	if err := os.WriteFile(maintainersPath, []byte(maintainersContent), 0644); err != nil {
		t.Fatalf("failed to write maintainers file: %v", err)
	}

	results, err := validator.ValidateMaintainersFile(maintainersPath, false)
	if err != nil {
		t.Fatalf("unexpected error validating maintainers: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Valid {
		t.Fatalf("expected maintainers to be valid, got errors: %v", results[0].Errors)
	}

	if results[0].VerificationAttempted {
		t.Fatalf("verification should not have been attempted when disabled")
	}
}

func TestValidateMaintainersFile_Verification(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	validator := NewValidator(cacheDir)

	maintainersPath := filepath.Join(tempDir, "maintainers.yaml")
	maintainersContent := `maintainers:
- project_id: "test-project"
  teams:
    - name: "project-maintainers"
      members:
        - alice
        - bob
`
	if err := os.WriteFile(maintainersPath, []byte(maintainersContent), 0644); err != nil {
		t.Fatalf("failed to write maintainers file: %v", err)
	}

	t.Setenv("MAINTAINER_API_ENDPOINT", "https://api-gw.platform.linuxfoundation.org/")
	t.Setenv("MAINTAINER_API_STUB", "")
	t.Setenv("LFX_AUTH_TOKEN", "") // Ensure LFX token is unset for stub test

	results, err := validator.ValidateMaintainersFile(maintainersPath, true)
	if err != nil {
		t.Fatalf("unexpected error validating maintainers: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if !res.Valid {
		t.Fatalf("expected valid result, got errors: %v", res.Errors)
	}

	if !res.VerificationAttempted {
		t.Fatalf("expected verification to be attempted")
	}

	if !res.VerificationPassed {
		t.Fatalf("expected verification to pass")
	}

	if len(res.VerifiedHandles) != 2 {
		t.Fatalf("expected 2 verified handles, got %d", len(res.VerifiedHandles))
	}
}

func TestNewFieldsValidation(t *testing.T) {
	project := validBaseProject()
	project.Security = &SecurityConfig{
		Policy: &PathRef{Path: ""}, // Empty path should fail
	}
	project.Governance = &GovernanceConfig{
		Contributing: &PathRef{Path: "CONTRIBUTING.md"}, // Valid
		Codeowners:   &PathRef{Path: ""},                // Empty path should fail
	}

	errors := validateProjectStruct(project)
	expectedErrors := []string{
		"security.policy.path is required",
		"governance.codeowners.path is required",
	}

	for _, expectedError := range expectedErrors {
		found := false
		for _, error := range errors {
			if error == expectedError {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error '%s' not found in: %v", expectedError, errors)
		}
	}
}

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
			project := validBaseProject()
			project.SchemaVersion = tt.version
			errs := validateProjectStruct(project)
			if tt.expectError {
				found := false
				for _, e := range errs {
					if strings.Contains(e, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got: %v", tt.errorContains, errs)
				}
			} else {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got: %v", errs)
				}
			}
		})
	}
}

func TestMaturityLogOrdering(t *testing.T) {
	tests := []struct {
		name        string
		entries     []MaturityEntry
		expectError bool
	}{
		{
			"correct order",
			[]MaturityEntry{
				{Phase: "sandbox", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/1"},
				{Phase: "incubating", Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/2"},
			},
			false,
		},
		{
			"wrong order",
			[]MaturityEntry{
				{Phase: "incubating", Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/2"},
				{Phase: "sandbox", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/1"},
			},
			true,
		},
		{
			"same date is ok",
			[]MaturityEntry{
				{Phase: "sandbox", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/1"},
				{Phase: "incubating", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/2"},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := validBaseProject()
			project.MaturityLog = tt.entries
			errs := validateProjectStruct(project)
			hasOrderError := false
			for _, e := range errs {
				if strings.Contains(e, "chronological order") {
					hasOrderError = true
					break
				}
			}
			if tt.expectError && !hasOrderError {
				t.Errorf("expected ordering error, got: %v", errs)
			}
			if !tt.expectError && hasOrderError {
				t.Errorf("did not expect ordering error, got: %v", errs)
			}
		})
	}
}

func TestSlugValidation(t *testing.T) {
	tests := []struct {
		name        string
		slug        string
		expectError bool
	}{
		{"valid slug", "kubernetes", false},
		{"valid with hyphen", "cert-manager", false},
		{"valid with numbers", "k3s", false},
		{"empty slug", "", true},
		{"uppercase", "Kubernetes", true},
		{"spaces", "my project", true},
		{"underscores", "my_project", true},
		{"leading hyphen", "-kubernetes", true},
		{"trailing hyphen", "kubernetes-", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := validBaseProject()
			project.Slug = tt.slug
			errs := validateProjectStruct(project)
			hasSlugError := false
			for _, e := range errs {
				if strings.Contains(e, "slug") {
					hasSlugError = true
					break
				}
			}
			if tt.expectError && !hasSlugError {
				t.Errorf("expected slug error for %q, got: %v", tt.slug, errs)
			}
			if !tt.expectError && hasSlugError {
				t.Errorf("did not expect slug error for %q, got: %v", tt.slug, errs)
			}
		})
	}
}

func TestProjectLeadValidation(t *testing.T) {
	project := validBaseProject()
	project.ProjectLead = "jdoe"
	errs := validateProjectStruct(project)
	if len(errs) != 0 {
		t.Errorf("expected no errors with valid project_lead, got: %v", errs)
	}

	// With @ prefix should also work (stripped internally)
	project.ProjectLead = "@jdoe"
	errs = validateProjectStruct(project)
	if len(errs) != 0 {
		t.Errorf("expected no errors with @-prefixed project_lead, got: %v", errs)
	}
}

func TestSlackChannelValidation(t *testing.T) {
	tests := []struct {
		name        string
		channel     string
		expectError bool
	}{
		{"valid channel", "#kubernetes", false},
		{"empty is ok", "", false},
		{"missing hash", "kubernetes", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := validBaseProject()
			project.CNCFSlackChannel = tt.channel
			errs := validateProjectStruct(project)
			hasChannelError := false
			for _, e := range errs {
				if strings.Contains(e, "cncf_slack_channel") {
					hasChannelError = true
					break
				}
			}
			if tt.expectError && !hasChannelError {
				t.Errorf("expected slack channel error for %q, got: %v", tt.channel, errs)
			}
			if !tt.expectError && hasChannelError {
				t.Errorf("did not expect slack channel error for %q, got: %v", tt.channel, errs)
			}
		})
	}
}

func TestLandscapeValidation(t *testing.T) {
	// No landscape section is fine
	project := validBaseProject()
	errs := validateProjectStruct(project)
	if len(errs) != 0 {
		t.Errorf("expected no errors without landscape, got: %v", errs)
	}

	// Valid landscape
	project.Landscape = &LandscapeConfig{
		Category:    "Orchestration & Management",
		Subcategory: "Scheduling & Orchestration",
	}
	errs = validateProjectStruct(project)
	if len(errs) != 0 {
		t.Errorf("expected no errors with valid landscape, got: %v", errs)
	}

	// Missing subcategory
	project.Landscape = &LandscapeConfig{
		Category: "Orchestration & Management",
	}
	errs = validateProjectStruct(project)
	hasSubcatError := false
	for _, e := range errs {
		if strings.Contains(e, "landscape.subcategory") {
			hasSubcatError = true
		}
	}
	if !hasSubcatError {
		t.Errorf("expected landscape.subcategory error, got: %v", errs)
	}
}

func TestMaturityPhaseValues(t *testing.T) {
	tests := []struct {
		name        string
		phase       string
		expectError bool
	}{
		{"sandbox", "sandbox", false},
		{"incubating", "incubating", false},
		{"graduated", "graduated", false},
		{"archived", "archived", false},
		{"invalid phase", "invalid-phase", true},
		{"typo", "graduating", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := validBaseProject()
			project.MaturityLog = []MaturityEntry{
				{Phase: tt.phase, Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/1"},
			}
			errs := validateProjectStruct(project)
			hasPhaseError := false
			for _, e := range errs {
				if strings.Contains(e, "invalid value") {
					hasPhaseError = true
					break
				}
			}
			if tt.expectError && !hasPhaseError {
				t.Errorf("expected phase validation error for %q, got: %v", tt.phase, errs)
			}
			if !tt.expectError && hasPhaseError {
				t.Errorf("did not expect phase validation error for %q, got: %v", tt.phase, errs)
			}
		})
	}
}
