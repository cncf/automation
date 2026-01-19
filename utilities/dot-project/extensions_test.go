package projects

import (
	"testing"
	"time"
)

func TestExtensionValidation(t *testing.T) {
	baseProject := Project{
		Name:          "Test Project",
		Description:   "A test project",
		SchemaVersion: "1.1.0",
		MaturityLog: []MaturityEntry{
			{
				Phase: "incubating",
				Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Issue: "https://github.com/cncf/toc/issues/123",
			},
		},
		Repositories: []string{"https://github.com/test/repo"},
	}

	t.Run("valid extension", func(t *testing.T) {
		project := baseProject
		project.Extensions = map[string]Extension{
			"my-tool": {
				Metadata: &ExtensionMetadata{
					Author:     "Test Author",
					Homepage:   "https://example.com",
					Repository: "https://github.com/test/tool",
					License:    "Apache-2.0",
					Version:    "1.0.0",
				},
				Config: map[string]interface{}{
					"enabled": true,
					"setting": "value",
				},
			},
		}

		errors := validateProjectStruct(project)
		if len(errors) != 0 {
			t.Errorf("Expected no errors for valid extension, got: %v", errors)
		}
	})

	t.Run("extension without schema version", func(t *testing.T) {
		project := baseProject
		project.SchemaVersion = ""
		project.Extensions = map[string]Extension{
			"my-tool": {Config: map[string]interface{}{"key": "value"}},
		}

		errors := validateProjectStruct(project)
		found := false
		for _, err := range errors {
			if err == "extensions require schema_version >= 1.1.0" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected schema version error, got: %v", errors)
		}
	})

	t.Run("extension with old schema version", func(t *testing.T) {
		project := baseProject
		project.SchemaVersion = "1.0.0"
		project.Extensions = map[string]Extension{
			"my-tool": {Config: map[string]interface{}{"key": "value"}},
		}

		errors := validateProjectStruct(project)
		found := false
		for _, err := range errors {
			if err == "extensions require schema_version >= 1.1.0" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected schema version error, got: %v", errors)
		}
	})

	t.Run("reserved extension name", func(t *testing.T) {
		project := baseProject
		project.Extensions = map[string]Extension{
			"name": {Config: map[string]interface{}{"key": "value"}},
		}

		errors := validateProjectStruct(project)
		found := false
		for _, err := range errors {
			if err == "extensions.name: 'name' is a reserved name" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected reserved name error, got: %v", errors)
		}
	})

	t.Run("invalid extension name format", func(t *testing.T) {
		project := baseProject
		project.Extensions = map[string]Extension{
			"my tool!": {Config: map[string]interface{}{"key": "value"}},
		}

		errors := validateProjectStruct(project)
		found := false
		for _, err := range errors {
			if err == "extensions.my tool!: invalid name format (use alphanumeric, hyphens, underscores, dots)" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid name format error, got: %v", errors)
		}
	})

	t.Run("invalid metadata URL", func(t *testing.T) {
		project := baseProject
		project.Extensions = map[string]Extension{
			"my-tool": {
				Metadata: &ExtensionMetadata{
					Homepage: "not-a-url",
				},
			},
		}

		errors := validateProjectStruct(project)
		found := false
		for _, err := range errors {
			if err == "extensions.my-tool.metadata.homepage is not a valid URL" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid URL error, got: %v", errors)
		}
	})
}

func TestIsValidExtensionName(t *testing.T) {
	testCases := []struct {
		name  string
		valid bool
	}{
		{"my-tool", true},
		{"my_tool", true},
		{"my.tool", true},
		{"MyTool123", true},
		{"tool", true},
		{"my tool", false},
		{"my-tool!", false},
		{"", false},
		{"tool@name", false},
		{"tool#name", false},
	}

	for _, tc := range testCases {
		result := isValidExtensionName(tc.name)
		if result != tc.valid {
			t.Errorf("isValidExtensionName(%q) = %v, expected %v", tc.name, result, tc.valid)
		}
	}
}

func TestIsVersionAtLeast(t *testing.T) {
	testCases := []struct {
		version    string
		minVersion string
		expected   bool
	}{
		{"1.1.0", "1.1.0", true},
		{"1.2.0", "1.1.0", true},
		{"2.0.0", "1.1.0", true},
		{"1.0.0", "1.1.0", false},
		{"0.9.0", "1.1.0", false},
	}

	for _, tc := range testCases {
		result := isVersionAtLeast(tc.version, tc.minVersion)
		if result != tc.expected {
			t.Errorf("isVersionAtLeast(%q, %q) = %v, expected %v",
				tc.version, tc.minVersion, result, tc.expected)
		}
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that projects without extensions still validate correctly
	project := Project{
		Name:          "Legacy Project",
		Description:   "A project without extensions",
		SchemaVersion: "1.0.0",
		MaturityLog: []MaturityEntry{
			{
				Phase: "incubating",
				Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Issue: "https://github.com/cncf/toc/issues/123",
			},
		},
		Repositories: []string{"https://github.com/test/repo"},
	}

	errors := validateProjectStruct(project)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for legacy project, got: %v", errors)
	}
}
