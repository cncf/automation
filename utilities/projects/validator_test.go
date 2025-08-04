package projects

import (
	"os"
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
