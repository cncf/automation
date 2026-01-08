package main

import (
	"strings"
	"testing"

	"projects"

	"gopkg.in/yaml.v3"
)

func TestUpdateLandscape(t *testing.T) {
	// Setup mock landscape
	landscapeYAML := `
landscape:
  - category: Orchestration & Management
    subcategories:
      - items:
          - name: Kubernetes
            repo_url: https://github.com/kubernetes/kubernetes
            homepage_url: https://old.kubernetes.io
            description: Old description
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(landscapeYAML), &root); err != nil {
		t.Fatalf("Failed to parse mock landscape: %v", err)
	}

	// Setup mock project
	project := &projects.Project{
		Name:        "Kubernetes",
		Description: "New description",
		Website:     "https://kubernetes.io",
		Repositories: []string{
			"https://github.com/kubernetes/kubernetes",
		},
		Social: map[string]string{
			"twitter":  "https://twitter.com/kubernetesio",
			"slack":    "https://kubernetes.slack.com",
			"linkedin": "https://linkedin.com/company/kubernetes",
		},
	}

	// Run update
	updated := updateLandscape(&root, project)
	if !updated {
		t.Fatal("Expected update to return true")
	}

	// Verify update
	out, err := yaml.Marshal(&root)
	if err != nil {
		t.Fatalf("Failed to marshal updated landscape: %v", err)
	}

	outStr := string(out)
	if !strings.Contains(outStr, "homepage_url: https://kubernetes.io") {
		t.Errorf("Homepage URL not updated. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "description: New description") {
		t.Errorf("Description not updated. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "twitter: https://twitter.com/kubernetesio") {
		t.Errorf("Twitter not added. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "slack_url: https://kubernetes.slack.com") {
		t.Errorf("Slack URL not added/mapped. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "linkedin_url: https://linkedin.com/company/kubernetes") {
		t.Errorf("LinkedIn URL not added/mapped. Got:\n%s", outStr)
	}
}
