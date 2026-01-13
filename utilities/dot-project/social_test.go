package projects

import (
	"testing"
	"time"
)

func TestSocialValidation(t *testing.T) {
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
		Social: map[string]string{
			"twitter": "invalid-url",
			"slack":   "https://slack.com", // Valid
			"github":  "also-invalid",
		},
	}

	errors := validateProjectStruct(project)
	expectedErrors := []string{
		"social.twitter is not a valid URL: invalid-url",
		"social.github is not a valid URL: also-invalid",
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
