package projects

import (
	"os"
	"path/filepath"
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
		Name:        "Test Project",
		Description: "A valid test project",
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
	project := Project{
		Name:        "Test",
		Description: "Test",
		MaturityLog: []MaturityEntry{
			{
				Phase: "",          // Missing phase
				Date:  time.Time{}, // Zero date
				Issue: "",          // Missing issue
			},
		},
		Repositories: []string{"https://github.com/test/repo"},
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
	project := Project{
		Name:        "Test",
		Description: "Test",
		MaturityLog: []MaturityEntry{
			{
				Phase: "incubating",
				Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Issue: "https://github.com/cncf/toc/issues/123",
			},
		},
		Repositories: []string{"https://github.com/test/repo"},
		Audits: []Audit{
			{
				Date: time.Time{},   // Zero date
				Type: "",            // Missing type
				URL:  "invalid-url", // Invalid URL
			},
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
	project := Project{
		Name:        "Test",
		Description: "Test",
		MaturityLog: []MaturityEntry{
			{
				Phase: "incubating",
				Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Issue: "https://github.com/cncf/toc/issues/123",
			},
		},
		Repositories: []string{
			"https://github.com/test/repo", // Valid
			"invalid-url",                  // Invalid
			"",                             // Empty
		},
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
	project := Project{
		Name:        "Test",
		Description: "Test",
		MaturityLog: []MaturityEntry{
			{
				Phase: "incubating",
				Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Issue: "https://github.com/cncf/toc/issues/123",
			},
		},
		Repositories: []string{"https://github.com/test/repo"},
		Security: &SecurityConfig{
			Policy: &PathRef{Path: ""}, // Empty path should fail
		},
		Governance: &GovernanceConfig{
			Contributing: &PathRef{Path: "CONTRIBUTING.md"}, // Valid
			Codeowners:   &PathRef{Path: ""},                // Empty path should fail
		},
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
