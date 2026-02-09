package projects

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYAMLFixtures(t *testing.T) {
	tests := []struct {
		file        string
		expectValid bool
	}{
		{"yaml/test-project.yaml", true},
		{"yaml/example-project.yaml", true},
		{"yaml/bad-project.yaml", false},
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
