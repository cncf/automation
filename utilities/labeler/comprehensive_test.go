package main

import (
	"context"
	"testing"

	"github.com/google/go-github/v55/github"
)

// Additional comprehensive tests for all rule types

func TestLabeler_ProcessMatchRule_RemoveCommand(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfigWithRemoveRules()
	labeler := NewLabeler(client, config)

	// Add some existing labels
	triageLabel := &github.Label{Name: stringPtr("triage/valid")}
	tagLabel := &github.Label{Name: stringPtr("tag/infrastructure")}
	client.IssueLabels[1] = []*github.Label{triageLabel, tagLabel}

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "/remove-triage valid",
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that triage/valid was removed
	removedLabels := client.RemovedLabels[1]
	if !sliceContains(removedLabels, "triage/valid") {
		t.Errorf("Expected 'triage/valid' to be removed, got: %v", removedLabels)
	}
}

func TestLabeler_ProcessFilePathRule_MultipleFiles(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfigWithFilePathRules()
	labeler := NewLabeler(client, config)

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "",
		ChangedFiles: []string{
			"tags/tag-infrastructure/charter.md",
			"tags/tag-developer-experience/README.md",
		},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that toc label was applied (for charter.md)
	appliedLabels := client.AppliedLabels[1]
	if !sliceContains(appliedLabels, "toc") {
		t.Errorf("Expected 'toc' label to be applied, got: %v", appliedLabels)
	}
	if !sliceContains(appliedLabels, "tag/developer-experience") {
		t.Errorf("Expected 'tag/developer-experience' label to be applied, got: %v", appliedLabels)
	}
}

func TestLabeler_ProcessMatchRule_WildcardRemoval(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfigWithWildcardRules()
	labeler := NewLabeler(client, config)

	// Add multiple triage labels
	triageValid := &github.Label{Name: stringPtr("triage/valid")}
	triageDuplicate := &github.Label{Name: stringPtr("triage/duplicate")}
	client.IssueLabels[1] = []*github.Label{triageValid, triageDuplicate}

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "/clear-triage",
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that all triage/* labels were removed
	removedLabels := client.RemovedLabels[1]
	if !sliceContains(removedLabels, "triage/valid") {
		t.Errorf("Expected 'triage/valid' to be removed, got: %v", removedLabels)
	}
	if !sliceContains(removedLabels, "triage/duplicate") {
		t.Errorf("Expected 'triage/duplicate' to be removed, got: %v", removedLabels)
	}
}

func TestLabeler_ProcessComplexScenario(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	// Start with needs-triage label
	needsTriageLabel := &github.Label{Name: stringPtr("needs-triage")}
	client.IssueLabels[1] = []*github.Label{needsTriageLabel}

	req := &LabelRequest{
		Owner:       "test-owner",
		Repo:        "test-repo",
		IssueNumber: 1,
		CommentBody: `This is a complex scenario.
/triage valid
/tag developer-experience
Some more comments here.`,
		ChangedFiles: []string{"tags/tag-developer-experience/some-file.md"},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	appliedLabels := client.AppliedLabels[1]
	removedLabels := client.RemovedLabels[1]

	// Should remove needs-triage
	if !sliceContains(removedLabels, "needs-triage") {
		t.Errorf("Expected 'needs-triage' to be removed, got: %v", removedLabels)
	}

	// Should apply triage/valid and tag/developer-experience
	if !sliceContains(appliedLabels, "triage/valid") {
		t.Errorf("Expected 'triage/valid' to be applied, got: %v", appliedLabels)
	}
	if !sliceContains(appliedLabels, "tag/developer-experience") {
		t.Errorf("Expected 'tag/developer-experience' to be applied, got: %v", appliedLabels)
	}
}

// Helper functions for additional test configs

func createTestConfigWithRemoveRules() *LabelsYAML {
	config := createTestConfig()
	config.Ruleset = append(config.Ruleset, Rule{
		Name: "remove-triage",
		Kind: "match",
		Spec: RuleSpec{
			Command: "/remove-triage",
		},
		Actions: []Action{
			{
				Kind: "remove-label",
				Spec: ActionSpec{Match: "triage/{{ argv.0 }}"},
			},
		},
	})
	return config
}

func createTestConfigWithFilePathRules() *LabelsYAML {
	config := createTestConfig()
	config.Ruleset = append(config.Ruleset, Rule{
		Name: "tag-developer-experience-dir",
		Kind: "filePath",
		Spec: RuleSpec{
			MatchPath: "tags/tag-developer-experience/*",
		},
		Actions: []Action{
			{
				Kind: "apply-label",
				Spec: ActionSpec{Label: "tag/developer-experience"},
			},
		},
	})
	return config
}

func createTestConfigWithWildcardRules() *LabelsYAML {
	config := createTestConfig()
	config.Ruleset = append(config.Ruleset, Rule{
		Name: "clear-triage",
		Kind: "match",
		Spec: RuleSpec{
			Command: "/clear-triage",
		},
		Actions: []Action{
			{
				Kind: "remove-label",
				Spec: ActionSpec{Match: "triage/*"},
			},
		},
	})
	return config
}
