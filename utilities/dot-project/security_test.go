package projects

import (
	"strings"
	"testing"
	"time"
)

func TestSecurityContactValidation(t *testing.T) {
	tests := []struct {
		name          string
		contact       *SecurityContact
		expectedError string
	}{
		{
			name:          "Valid Email Only",
			contact:       &SecurityContact{Email: "security@example.com"},
			expectedError: "",
		},
		{
			name:          "Valid Email with Name",
			contact:       &SecurityContact{Email: "Security Team <security@example.com>"},
			expectedError: "",
		},
		{
			name:          "Invalid Email",
			contact:       &SecurityContact{Email: "not-an-email"},
			expectedError: "security.contact.email is not a valid email: not-an-email",
		},
		{
			name:          "Valid Advisory URL",
			contact:       &SecurityContact{AdvisoryURL: "https://github.com/my-org/my-repo/security/advisories/new"},
			expectedError: "",
		},
		{
			name:          "Invalid Advisory URL - Wrong Domain",
			contact:       &SecurityContact{AdvisoryURL: "https://evil.com/org/repo/security/advisories/new"},
			expectedError: "security.contact.advisory_url must be a valid GitHub Security Advisory URL",
		},
		{
			name:          "Invalid Advisory URL - Missing /new",
			contact:       &SecurityContact{AdvisoryURL: "https://github.com/org/repo/security/advisories"},
			expectedError: "security.contact.advisory_url must be a valid GitHub Security Advisory URL",
		},
		{
			name:          "Invalid Advisory URL - Not advisories path",
			contact:       &SecurityContact{AdvisoryURL: "https://github.com/org/repo/issues/new"},
			expectedError: "security.contact.advisory_url must be a valid GitHub Security Advisory URL",
		},
		{
			name: "Both Email and Advisory URL Valid",
			contact: &SecurityContact{
				Email:       "security@example.com",
				AdvisoryURL: "https://github.com/org/repo/security/advisories/new",
			},
			expectedError: "",
		},
		{
			name:          "Neither Set - Empty Object",
			contact:       &SecurityContact{},
			expectedError: "security.contact must have at least one of email or advisory_url",
		},
		{
			name:          "Nil Contact - Optional",
			contact:       nil,
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

			if tt.expectedError == "" {
				for _, err := range errors {
					if strings.Contains(err, "security.contact") {
						t.Errorf("Unexpected security contact error: %s", err)
					}
				}
			} else {
				found := false
				for _, err := range errors {
					if strings.Contains(err, tt.expectedError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing %q not found in: %v", tt.expectedError, errors)
				}
			}
		})
	}
}
