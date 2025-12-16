package projects

import (
	"testing"
	"time"
)

func TestSecurityContactValidation(t *testing.T) {
	tests := []struct {
		name          string
		contact       string
		expectedError string
	}{
		{
			name:          "Valid Email",
			contact:       "security@example.com",
			expectedError: "",
		},
		{
			name:          "Valid Email with Name",
			contact:       "Security Team <security@example.com>",
			expectedError: "",
		},
		{
			name:          "Invalid Email",
			contact:       "not-an-email",
			expectedError: "security.contact is not a valid email: not-an-email",
		},
		{
			name:          "Empty Email",
			contact:       "",
			expectedError: "", // Optional field
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := Project{
				Name:        "Test",
				Description: "Test",
				MaturityLog: []MaturityEntry{
					{
						Phase: "incubating",
						Date:  time.Now(),
						Issue: "https://github.com/cncf/toc/issues/123",
					},
				},
				Repositories: []string{"https://github.com/test/repo"},
				Security: &SecurityConfig{
					Contact: tt.contact,
				},
			}

			errors := validateProjectStruct(project)

			found := false
			if tt.expectedError == "" {
				for _, err := range errors {
					if err == "security.contact is not a valid email: "+tt.contact {
						t.Errorf("Unexpected error: %s", err)
					}
				}
			} else {
				for _, err := range errors {
					if err == tt.expectedError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error '%s' not found in: %v", tt.expectedError, errors)
				}
			}
		})
	}
}
