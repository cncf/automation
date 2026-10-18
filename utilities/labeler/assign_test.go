package main

import (
	"context"
	"errors"
	"testing"
)

func createTestConfigWithAssignRules() *LabelsYAML {
	return &LabelsYAML{
		AutoCreate:         false,
		DefinitionRequired: true,
		Debug:              true,
		Ruleset: []Rule{
			{
				Name: "assign",
				Kind: "match",
				Spec: RuleSpec{Command: "/assign"},
				Actions: []Action{
					{Kind: "assign"},
				},
			},
			{
				Name: "unassign",
				Kind: "match",
				Spec: RuleSpec{Command: "/unassign"},
				Actions: []Action{
					{Kind: "unassign"},
				},
			},
		},
	}
}

func newAssignRequest(body, author string) *LabelRequest {
	return &LabelRequest{
		Owner:         "test-owner",
		Repo:          "test-repo",
		IssueNumber:   1,
		CommentBody:   body,
		CommentAuthor: author,
	}
}

func TestLabeler_Assign_ExplicitUsers(t *testing.T) {
	client := NewMockGitHubClient()
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/assign @alice bob", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	want := []string{"alice", "bob"}
	if !slicesEqual(client.AddedAssignees[1], want) {
		t.Errorf("expected assignees %v to be added, got %v", want, client.AddedAssignees[1])
	}
	if len(client.RemovedAssignees[1]) != 0 {
		t.Errorf("expected no assignees to be removed, got %v", client.RemovedAssignees[1])
	}
}

func TestLabeler_Assign_NoArgsAssignsCommentAuthor(t *testing.T) {
	client := NewMockGitHubClient()
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/assign", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	if !slicesEqual(client.AddedAssignees[1], []string{"carol"}) {
		t.Errorf("expected comment author to be assigned, got %v", client.AddedAssignees[1])
	}
}

func TestLabeler_Assign_NoArgsNoAuthorIsNoop(t *testing.T) {
	client := NewMockGitHubClient()
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	// Without a comment author there is nobody to self-assign; this must not error.
	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/assign", "")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	if len(client.AddedAssignees[1]) != 0 {
		t.Errorf("expected no assignees to be added, got %v", client.AddedAssignees[1])
	}
}

func TestLabeler_Assign_SkipsAlreadyAssignedAndDuplicates(t *testing.T) {
	client := NewMockGitHubClient()
	client.Assignees[1] = []string{"alice"}
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/assign @alice @bob @bob", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	if !slicesEqual(client.AddedAssignees[1], []string{"bob"}) {
		t.Errorf("expected only 'bob' to be added, got %v", client.AddedAssignees[1])
	}
}

func TestLabeler_Assign_SkipsUnassignableUsers(t *testing.T) {
	client := NewMockGitHubClient()
	client.Assignable = []string{"alice"}
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/assign @alice @stranger", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	if !slicesEqual(client.AddedAssignees[1], []string{"alice"}) {
		t.Errorf("expected only assignable user 'alice' to be added, got %v", client.AddedAssignees[1])
	}
}

func TestLabeler_Assign_IsAssigneeErrorIsReported(t *testing.T) {
	client := NewMockGitHubClient()
	client.IsAssigneeError = errors.New("boom")
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	// ProcessRequest logs per-rule errors rather than returning them, so assert
	// on the observable side effect: nothing is assigned on an API error.
	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/assign @alice", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}
	if len(client.AddedAssignees[1]) != 0 {
		t.Errorf("expected no assignees to be added on IsAssignee error, got %v", client.AddedAssignees[1])
	}
}

func TestLabeler_Unassign_ExplicitUsers(t *testing.T) {
	client := NewMockGitHubClient()
	client.Assignees[1] = []string{"alice", "bob", "carol"}
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/unassign @alice bob", "dave")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	if !slicesEqual(client.RemovedAssignees[1], []string{"alice", "bob"}) {
		t.Errorf("expected alice and bob to be removed, got %v", client.RemovedAssignees[1])
	}
	if !slicesEqual(client.Assignees[1], []string{"carol"}) {
		t.Errorf("expected carol to remain assigned, got %v", client.Assignees[1])
	}
}

func TestLabeler_Unassign_NoArgsRemovesCommentAuthor(t *testing.T) {
	client := NewMockGitHubClient()
	client.Assignees[1] = []string{"alice", "carol"}
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/unassign", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	if !slicesEqual(client.RemovedAssignees[1], []string{"carol"}) {
		t.Errorf("expected comment author to be unassigned, got %v", client.RemovedAssignees[1])
	}
}

func TestLabeler_Unassign_SkipsUsersNotAssigned(t *testing.T) {
	client := NewMockGitHubClient()
	client.Assignees[1] = []string{"alice"}
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/unassign @bob", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	if len(client.RemovedAssignees[1]) != 0 {
		t.Errorf("expected no removal request for a user who is not assigned, got %v", client.RemovedAssignees[1])
	}
}

func TestLabeler_Assign_CommandMustBeOwnToken(t *testing.T) {
	client := NewMockGitHubClient()
	labeler := NewLabeler(client, createTestConfigWithAssignRules())

	// "/assignee" must not trigger the "/assign" rule.
	if err := labeler.ProcessRequest(context.Background(), newAssignRequest("/assignee @alice", "carol")); err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}
	if len(client.AddedAssignees[1]) != 0 {
		t.Errorf("expected '/assignee' not to match '/assign', got %v", client.AddedAssignees[1])
	}
}

func TestNormalizeLogins(t *testing.T) {
	got := normalizeLogins([]string{"@Alice", "bob", "", "@", "@alice", "Bob"})
	want := []string{"Alice", "bob"}
	if !slicesEqual(got, want) {
		t.Errorf("normalizeLogins = %v, want %v", got, want)
	}
}
