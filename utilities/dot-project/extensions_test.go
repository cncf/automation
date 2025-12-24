package projects

import (
	"testing"
)

func TestValidateExtensions(t *testing.T) {
	tests := []struct {
		name     string
		project  Project
		wantErrs int
	}{
		{
			name: "valid extensions",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Extensions: map[string]Extension{
					"security-scanner": {
						Version:     "1.0.0",
						Description: "Security scanning tool",
						Config: map[string]interface{}{
							"enabled": true,
						},
						Metadata: &ExtensionMetadata{
							Author:     "Security Team",
							Homepage:   "https://security.example.com",
							Repository: "https://github.com/security/scanner",
							License:    "Apache-2.0",
						},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "missing version",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Extensions: map[string]Extension{
					"tool": {
						Description: "A tool without version",
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "invalid extension name",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Extensions: map[string]Extension{
					"invalid@name": {
						Version: "1.0.0",
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "reserved extension name",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Extensions: map[string]Extension{
					"cncf": {
						Version: "1.0.0",
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "invalid metadata URLs",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Extensions: map[string]Extension{
					"tool": {
						Version: "1.0.0",
						Metadata: &ExtensionMetadata{
							Homepage:   "invalid-url",
							Repository: "also-invalid",
						},
					},
				},
			},
			wantErrs: 2,
		},
		{
			name: "valid experimental fields",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Experimental: map[string]interface{}{
					"custom_feature": map[string]interface{}{
						"enabled": true,
					},
					"beta_api": "v2",
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid experimental field name",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Experimental: map[string]interface{}{
					"invalid@field": "value",
				},
			},
			wantErrs: 1,
		},
		{
			name: "reserved experimental field name",
			project: Project{
				Name:        "Test Project",
				Description: "Test description",
				Experimental: map[string]interface{}{
					"core": "value",
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateExtensions(tt.project)
			if len(errors) != tt.wantErrs {
				t.Errorf("validateExtensions() got %d errors, want %d. Errors: %v", len(errors), tt.wantErrs, errors)
			}
		})
	}
}

func TestIsValidExtensionName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"valid-name", true},
		{"valid_name", true},
		{"valid.name", true},
		{"ValidName123", true},
		{"123valid", true},
		{"", false},
		{"invalid@name", false},
		{"invalid name", false},
		{"invalid-name-that-is-way-too-long-and-exceeds-the-maximum-allowed-length", false},
		{"-invalid", false},
		{"_invalid", false},
		{".invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidExtensionName(tt.name); got != tt.want {
				t.Errorf("isValidExtensionName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsReservedExtensionName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"name", true},
		{"description", true},
		{"cncf", true},
		{"kubernetes", true},
		{"core", true},
		{"system", true},
		{"extensions", true},
		{"experimental", true},
		{"custom-tool", false},
		{"my_extension", false},
		{"valid.extension", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReservedExtensionName(tt.name); got != tt.want {
				t.Errorf("isReservedExtensionName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}