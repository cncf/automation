package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v55/github"
)

// MockGitHubClient implements GitHubClient for testing
type MockGitHubClient struct {
	Issues       map[int]*github.Issue
	Labels       []*github.Label
	IssueLabels  map[int][]*github.Label
	CreatedLabels map[string]*github.Label
	DeletedLabels []string
	AppliedLabels map[int][]string
	RemovedLabels map[int][]string
	GetLabelError error
}

func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		Issues:        make(map[int]*github.Issue),
		IssueLabels:   make(map[int][]*github.Label),
		CreatedLabels: make(map[string]*github.Label),
		DeletedLabels: []string{},
		AppliedLabels: make(map[int][]string),
		RemovedLabels: make(map[int][]string),
	}
}

func (m *MockGitHubClient) GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, *github.Response, error) {
	if issue, exists := m.Issues[number]; exists {
		return issue, nil, nil
	}
	title := "Test Issue"
	return &github.Issue{Number: &number, Title: &title}, nil, nil
}

func (m *MockGitHubClient) ListLabelsByIssue(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.Label, *github.Response, error) {
	return m.IssueLabels[number], nil, nil
}

func (m *MockGitHubClient) AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error) {
	if m.AppliedLabels[number] == nil {
		m.AppliedLabels[number] = []string{}
	}
	m.AppliedLabels[number] = append(m.AppliedLabels[number], labels...)
	
	// Add to issue labels
	for _, label := range labels {
		labelObj := &github.Label{Name: &label}
		m.IssueLabels[number] = append(m.IssueLabels[number], labelObj)
	}
	return nil, nil, nil
}

func (m *MockGitHubClient) RemoveLabelForIssue(ctx context.Context, owner, repo string, number int, label string) (*github.Response, error) {
	if m.RemovedLabels[number] == nil {
		m.RemovedLabels[number] = []string{}
	}
	m.RemovedLabels[number] = append(m.RemovedLabels[number], label)
	
	// Remove from issue labels
	for i, lbl := range m.IssueLabels[number] {
		if lbl.GetName() == label {
			m.IssueLabels[number] = append(m.IssueLabels[number][:i], m.IssueLabels[number][i+1:]...)
			break
		}
	}
	return nil, nil
}

func (m *MockGitHubClient) ListLabels(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.Label, *github.Response, error) {
	return m.Labels, nil, nil
}

func (m *MockGitHubClient) CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error) {
	m.CreatedLabels[label.GetName()] = label
	m.Labels = append(m.Labels, label)
	return label, nil, nil
}

func (m *MockGitHubClient) EditLabel(ctx context.Context, owner, repo, name string, label *github.Label) (*github.Label, *github.Response, error) {
	for _, lbl := range m.Labels {
		if lbl.GetName() == name {
			*lbl = *label
			break
		}
	}
	return label, nil, nil
}

func (m *MockGitHubClient) DeleteLabel(ctx context.Context, owner, repo, name string) (*github.Response, error) {
	m.DeletedLabels = append(m.DeletedLabels, name)
	
	// Remove from labels
	for i, lbl := range m.Labels {
		if lbl.GetName() == name {
			m.Labels = append(m.Labels[:i], m.Labels[i+1:]...)
			break
		}
	}
	return nil, nil
}

func (m *MockGitHubClient) GetLabel(ctx context.Context, owner, repo, name string) (*github.Label, *github.Response, error) {
	if m.GetLabelError != nil {
		return nil, nil, m.GetLabelError
	}
	for _, lbl := range m.Labels {
		if lbl.GetName() == name {
			return lbl, nil, nil
		}
	}
	// Return error if label doesn't exist
	return nil, nil, &github.ErrorResponse{Message: "Not Found"}
}

// Helper function to create a test config
func createTestConfig() *LabelsYAML {
	return &LabelsYAML{
		AutoCreate:         true,
		AutoDelete:         false,
		DefinitionRequired: true,
		Debug:              true,
		Labels: []Label{
			{Name: "needs-triage", Color: "ededed", Description: "Needs triage"},
			{Name: "triage/valid", Color: "0e8a16", Description: "Valid issue"},
			{Name: "triage/duplicate", Color: "ebf84a", Description: "Duplicate issue"},
			{Name: "kind/enhancement", Color: "61D6C3", Description: "Enhancement"},
			{Name: "tag/developer-experience", Color: "c2e0c6", Description: "TAG Developer Experience"},
			{Name: "toc", Color: "CF0CBE", Description: "TOC specific issue"},
			{Name: "help wanted", Color: "159818", Description: "Help wanted"},
		},
		Ruleset: []Rule{
			// needs-triage rule
			{
				Name: "needs-triage",
				Kind: "label",
				Spec: RuleSpec{
					Match:          "triage/*",
					MatchCondition: "NOT",
				},
				Actions: []Action{
					{
						Kind: "apply-label",
						Spec: ActionSpec{Label: "needs-triage"},
					},
				},
			},
			// triage command rule
			{
				Name: "apply-triage",
				Kind: "match",
				Spec: RuleSpec{
					Command: "/triage",
					MatchList: []string{
						"valid",
						"duplicate",
						"needs-information",
						"not-planned",
					},
				},
				Actions: []Action{
					{
						Kind: "remove-label",
						Spec: ActionSpec{Match: "needs-triage"},
					},
					{
						Kind: "remove-label",
						Spec: ActionSpec{Match: "triage/*"},
					},
					{
						Kind: "apply-label",
						Spec: ActionSpec{Label: "triage/{{ argv.0 }}"},
					},
				},
			},
			// tag command rule
			{
				Name: "apply-tag",
				Kind: "match",
				Spec: RuleSpec{
					Command: "/tag",
					MatchList: []string{
						"developer-experience",
						"infrastructure",
						"operational-resilience",
						"security-compliance",
						"workloads-foundation",
					},
				},
				Actions: []Action{
					{
						Kind: "apply-label",
						Spec: ActionSpec{Label: "tag/{{ argv.0 }}"},
					},
				},
			},
			// file path rule
			{
				Name: "charter",
				Kind: "filePath",
				Spec: RuleSpec{
					MatchPath: "tags/*/charter.md",
				},
				Actions: []Action{
					{
						Kind: "apply-label",
						Spec: ActionSpec{Label: "toc"},
					},
				},
			},
		},
	}
}

func TestLabeler_ProcessFilePathRule(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "",
		ChangedFiles: []string{"tags/tag-infrastructure/charter.md"},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that toc label was applied (along with needs-triage)
	appliedLabels := client.AppliedLabels[1]
	if !sliceContains(appliedLabels, "toc") {
		t.Errorf("Expected 'toc' label to be applied, got: %v", appliedLabels)
	}
	if !sliceContains(appliedLabels, "needs-triage") {
		t.Errorf("Expected 'needs-triage' label to be applied, got: %v", appliedLabels)
	}
}

func TestLabeler_ProcessMatchRule_TriageCommand(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	// Add needs-triage label to the issue initially
	needsTriageLabel := &github.Label{Name: stringPtr("needs-triage")}
	client.IssueLabels[1] = []*github.Label{needsTriageLabel}

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "/triage valid",
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that needs-triage was removed and triage/valid was applied
	removedLabels := client.RemovedLabels[1]
	appliedLabels := client.AppliedLabels[1]

	// Should remove needs-triage and apply triage/valid (wildcard removal may not trigger)
	if !sliceContains(removedLabels, "needs-triage") {
		t.Errorf("Expected 'needs-triage' to be removed, got: %v", removedLabels)
	}

	expectedApplied := []string{"triage/valid"}
	if !slicesEqual(appliedLabels, expectedApplied) {
		t.Errorf("Expected applied labels %v, got: %v", expectedApplied, appliedLabels)
	}
}

func TestLabeler_ProcessMatchRule_TagCommand(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "/tag developer-experience",
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that both needs-triage and tag/developer-experience were applied
	appliedLabels := client.AppliedLabels[1]
	
	if !sliceContains(appliedLabels, "tag/developer-experience") {
		t.Errorf("Expected 'tag/developer-experience' label to be applied, got: %v", appliedLabels)
	}
	if !sliceContains(appliedLabels, "needs-triage") {
		t.Errorf("Expected 'needs-triage' label to be applied, got: %v", appliedLabels)
	}
}

func TestLabeler_ProcessLabelRule_NeedsTriage(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	// Issue has no triage labels
	client.IssueLabels[1] = []*github.Label{} // Explicitly set to empty

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "",
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that needs-triage was applied
	appliedLabels := client.AppliedLabels[1]
	if !sliceContains(appliedLabels, "needs-triage") {
		t.Errorf("Expected 'needs-triage' label to be applied, got: %v", appliedLabels)
	}
}

func TestLabeler_ProcessLabelRule_HasTriageLabel(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	// Issue already has a triage label
	triageLabel := &github.Label{Name: stringPtr("triage/valid")}
	client.IssueLabels[1] = []*github.Label{triageLabel}

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "",
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that needs-triage was NOT applied
	appliedLabels := client.AppliedLabels[1]
	if sliceContains(appliedLabels, "needs-triage") {
		t.Errorf("needs-triage should not be applied when triage label exists, but was applied: %v", appliedLabels)
	}
}

// TestLabeler_ProcessLabelRule_RemoveWithMatchConditionAND exercises the
// regression where a label rule with `matchCondition: AND` was treated like
// `NOT` and fired in exactly the wrong situations. See cncf/toc#2164 for the
// user-visible symptom: a freshly-opened PR would have `needs-triage`,
// `needs-kind`, and `needs-group` added and immediately removed in the same
// workflow run, and a manually-applied `triage/*`/`kind/*`/group label would
// never auto-clear its paired `needs-*` label.
func TestLabeler_ProcessLabelRule_RemoveWithMatchConditionAND(t *testing.T) {
	// Minimal config that pairs a NOT rule (apply needs-triage when no
	// triage/* exists) with an AND rule (remove needs-triage when one does).
	cfg := &LabelsYAML{
		AutoCreate:         true,
		AutoDelete:         false,
		DefinitionRequired: true,
		Labels: []Label{
			{Name: "needs-triage", Color: "ededed", Description: "Needs triage"},
			{Name: "triage/valid", Color: "0e8a16", Description: "Valid issue"},
		},
		Ruleset: []Rule{
			{
				Name: "needs-triage",
				Kind: "label",
				Spec: RuleSpec{Match: "triage/*", MatchCondition: "NOT"},
				Actions: []Action{
					{Kind: "apply-label", Spec: ActionSpec{Label: "needs-triage"}},
				},
			},
			{
				Name: "remove-needs-triage",
				Kind: "label",
				Spec: RuleSpec{Match: "triage/*", MatchCondition: "AND"},
				Actions: []Action{
					{Kind: "remove-label", Spec: ActionSpec{Match: "needs-triage"}},
				},
			},
		},
	}

	t.Run("no triage/* present: only NOT rule fires", func(t *testing.T) {
		client := NewMockGitHubClient()
		labeler := NewLabeler(client, cfg)
		client.IssueLabels[1] = []*github.Label{}

		err := labeler.ProcessRequest(context.Background(), &LabelRequest{
			Owner: "o", Repo: "r", IssueNumber: 1,
		})
		if err != nil {
			t.Fatalf("ProcessRequest failed: %v", err)
		}

		if !sliceContains(client.AppliedLabels[1], "needs-triage") {
			t.Errorf("expected needs-triage to be applied, got applied=%v", client.AppliedLabels[1])
		}
		if sliceContains(client.RemovedLabels[1], "needs-triage") {
			t.Errorf("AND rule must not fire when no triage/* is present; removed=%v", client.RemovedLabels[1])
		}
	})

	t.Run("triage/* present: only AND rule fires", func(t *testing.T) {
		client := NewMockGitHubClient()
		labeler := NewLabeler(client, cfg)
		client.IssueLabels[2] = []*github.Label{
			{Name: stringPtr("needs-triage")},
			{Name: stringPtr("triage/valid")},
		}

		err := labeler.ProcessRequest(context.Background(), &LabelRequest{
			Owner: "o", Repo: "r", IssueNumber: 2,
		})
		if err != nil {
			t.Fatalf("ProcessRequest failed: %v", err)
		}

		if sliceContains(client.AppliedLabels[2], "needs-triage") {
			t.Errorf("NOT rule must not fire when triage/* is present; applied=%v", client.AppliedLabels[2])
		}
		if !sliceContains(client.RemovedLabels[2], "needs-triage") {
			t.Errorf("expected AND rule to remove needs-triage, got removed=%v", client.RemovedLabels[2])
		}
	})
}

// TestLabeler_ProcessLabelRule_BracePattern verifies that label-rule
// `match` values can use brace alternation like `{toc,tag/*,sub/*}`.
// filepath.Match alone does not understand braces, so before this fix a
// pattern with braces only matched a literal label starting with "{".
func TestLabeler_ProcessLabelRule_BracePattern(t *testing.T) {
	cfg := &LabelsYAML{
		AutoCreate:         true,
		AutoDelete:         false,
		DefinitionRequired: true,
		Labels: []Label{
			{Name: "needs-group", Color: "FBCA04", Description: "Needs a group label"},
			{Name: "toc", Color: "C13B8A", Description: "TOC specific issue"},
			{Name: "tag/infrastructure", Color: "0E8A8A", Description: "TAG Infrastructure"},
			{Name: "sub/mentoring", Color: "1D76DB", Description: "TOC Mentoring Subproject"},
		},
		Ruleset: []Rule{
			{
				Name: "needs-group",
				Kind: "label",
				Spec: RuleSpec{Match: "{toc,tag/*,sub/*}", MatchCondition: "NOT"},
				Actions: []Action{
					{Kind: "apply-label", Spec: ActionSpec{Label: "needs-group"}},
				},
			},
			{
				Name: "remove-needs-group",
				Kind: "label",
				Spec: RuleSpec{Match: "{toc,tag/*,sub/*}", MatchCondition: "AND"},
				Actions: []Action{
					{Kind: "remove-label", Spec: ActionSpec{Match: "needs-group"}},
				},
			},
		},
	}

	cases := []struct {
		name        string
		existing    []*github.Label
		wantApply   bool
		wantRemove  bool
	}{
		{
			name:       "no group label → needs-group applied",
			existing:   []*github.Label{},
			wantApply:  true,
			wantRemove: false,
		},
		{
			name:       "toc present → needs-group removed",
			existing:   []*github.Label{{Name: stringPtr("needs-group")}, {Name: stringPtr("toc")}},
			wantApply:  false,
			wantRemove: true,
		},
		{
			name:       "tag/* present → needs-group removed",
			existing:   []*github.Label{{Name: stringPtr("needs-group")}, {Name: stringPtr("tag/infrastructure")}},
			wantApply:  false,
			wantRemove: true,
		},
		{
			name:       "sub/* present → needs-group removed",
			existing:   []*github.Label{{Name: stringPtr("needs-group")}, {Name: stringPtr("sub/mentoring")}},
			wantApply:  false,
			wantRemove: true,
		},
	}

	for i, tc := range cases {
		issueNum := i + 1
		t.Run(tc.name, func(t *testing.T) {
			client := NewMockGitHubClient()
			labeler := NewLabeler(client, cfg)
			client.IssueLabels[issueNum] = tc.existing

			err := labeler.ProcessRequest(context.Background(), &LabelRequest{
				Owner: "o", Repo: "r", IssueNumber: issueNum,
			})
			if err != nil {
				t.Fatalf("ProcessRequest failed: %v", err)
			}

			gotApply := sliceContains(client.AppliedLabels[issueNum], "needs-group")
			gotRemove := sliceContains(client.RemovedLabels[issueNum], "needs-group")
			if gotApply != tc.wantApply {
				t.Errorf("apply needs-group: got %v want %v (applied=%v)", gotApply, tc.wantApply, client.AppliedLabels[issueNum])
			}
			if gotRemove != tc.wantRemove {
				t.Errorf("remove needs-group: got %v want %v (removed=%v)", gotRemove, tc.wantRemove, client.RemovedLabels[issueNum])
			}
		})
	}
}

// TestLabeler_ProcessLabelRule_InvalidPattern verifies that a malformed
// `match` pattern surfaces as an error instead of silently leaving
// foundNamespace=false and causing the rule to fire in the wrong direction.
// `[` opens a character class that is never closed, so filepath.Match
// returns ErrBadPattern.
func TestLabeler_ProcessLabelRule_InvalidPattern(t *testing.T) {
	cfg := &LabelsYAML{
		AutoCreate:         true,
		AutoDelete:         false,
		DefinitionRequired: true,
		Labels: []Label{
			{Name: "needs-triage", Color: "ededed", Description: "Needs triage"},
		},
		Ruleset: []Rule{
			{
				Name: "bad-pattern",
				Kind: "label",
				Spec: RuleSpec{Match: "triage/[", MatchCondition: "NOT"},
				Actions: []Action{
					{Kind: "apply-label", Spec: ActionSpec{Label: "needs-triage"}},
				},
			},
		},
	}

	client := NewMockGitHubClient()
	labeler := NewLabeler(client, cfg)
	client.IssueLabels[1] = []*github.Label{{Name: stringPtr("some-label")}}

	// processRules swallows per-rule errors (it only logs them), so calling
	// processLabelRule directly is the cleanest way to assert on the error.
	err := labeler.processLabelRule(context.Background(), &LabelRequest{
		Owner: "o", Repo: "r", IssueNumber: 1,
	}, cfg.Ruleset[0])
	if err == nil {
		t.Fatalf("expected an error for invalid match pattern, got nil")
	}
	if !strings.Contains(err.Error(), "bad-pattern") || !strings.Contains(err.Error(), "triage/[") {
		t.Errorf("error should mention rule name and bad pattern; got: %v", err)
	}
	if sliceContains(client.AppliedLabels[1], "needs-triage") {
		t.Errorf("rule with invalid pattern must not apply labels; applied=%v", client.AppliedLabels[1])
	}
}

func TestExpandBraces(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"triage/*", []string{"triage/*"}},
		{"{toc,tag/*,sub/*}", []string{"toc", "tag/*", "sub/*"}},
		{"foo-{a,b}", []string{"foo-a", "foo-b"}},
		{"{a, b ,c}", []string{"a", "b", "c"}}, // surrounding whitespace trimmed
	}
	for _, tc := range cases {
		got := expandBraces(tc.in)
		if !slicesEqual(got, tc.want) {
			t.Errorf("expandBraces(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestLabeler_ProcessMatchRule_InvalidArgument(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	req := &LabelRequest{
		Owner:        "test-owner",
		Repo:         "test-repo",
		IssueNumber:  1,
		CommentBody:  "/tag invalid-tag", // Not in matchList
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that only needs-triage was applied (no tag label due to invalid argument)
	appliedLabels := client.AppliedLabels[1]
	if !sliceContains(appliedLabels, "needs-triage") {
		t.Errorf("Expected 'needs-triage' to be applied, got: %v", appliedLabels)
	}
	
	// Should not contain any tag labels
	for _, label := range appliedLabels {
		if strings.HasPrefix(label, "tag/") {
			t.Errorf("No tag labels should be applied for invalid argument, but found: %s", label)
		}
	}
}

func TestLabeler_ProcessMatchRule_MultilineComment(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	labeler := NewLabeler(client, config)

	req := &LabelRequest{
		Owner:       "test-owner",
		Repo:        "test-repo",
		IssueNumber: 1,
		CommentBody: `This is a comment
/triage valid
And some more text`,
		ChangedFiles: []string{},
	}

	ctx := context.Background()
	err := labeler.ProcessRequest(ctx, req)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check that triage/valid was applied from multiline comment
	appliedLabels := client.AppliedLabels[1]
	found := false
	for _, label := range appliedLabels {
		if label == "triage/valid" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'triage/valid' label to be applied from multiline comment, got: %v", appliedLabels)
	}
}

func TestEnsureLabelExists_ExistingLabel_AutoCreateDisabled(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	config.AutoCreate = false
	labeler := NewLabeler(client, config)

	existingName := "needs-triage"
	existingColor := "ededed"
	existingDescription := "Needs triage"
	client.Labels = []*github.Label{
		{
			Name:        &existingName,
			Color:       &existingColor,
			Description: &existingDescription,
		},
	}

	err := labeler.ensureLabelExists(context.Background(), "test-owner", "test-repo", existingName, existingColor, existingDescription)
	if err != nil {
		t.Fatalf("expected no error for existing label with auto-create disabled, got: %v", err)
	}
	if len(client.CreatedLabels) != 0 {
		t.Fatalf("expected no label creation, got: %v", client.CreatedLabels)
	}
}

func TestEnsureLabelExists_MissingLabel_AutoCreateDisabled(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	config.AutoCreate = false
	labeler := NewLabeler(client, config)

	err := labeler.ensureLabelExists(context.Background(), "test-owner", "test-repo", "missing-label", "ffffff", "Missing label")
	if err == nil {
		t.Fatal("expected error for missing label with auto-create disabled")
	}
	if !strings.Contains(err.Error(), "auto-create-labels is disabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureLabelExists_MissingLabel_AutoCreateEnabled(t *testing.T) {
	client := NewMockGitHubClient()
	config := createTestConfig()
	config.AutoCreate = true
	labeler := NewLabeler(client, config)

	labelName := "missing-label"
	labelColor := "ffffff"
	labelDescription := "Missing label"

	err := labeler.ensureLabelExists(context.Background(), "test-owner", "test-repo", labelName, labelColor, labelDescription)
	if err != nil {
		t.Fatalf("expected no error creating missing label with auto-create enabled, got: %v", err)
	}

	createdLabel, ok := client.CreatedLabels[labelName]
	if !ok {
		t.Fatalf("expected label %s to be created", labelName)
	}
	if createdLabel.GetColor() != labelColor {
		t.Fatalf("expected created label color %s, got %s", labelColor, createdLabel.GetColor())
	}
	if createdLabel.GetDescription() != labelDescription {
		t.Fatalf("expected created label description %s, got %s", labelDescription, createdLabel.GetDescription())
	}
}

func TestEnsureLabelExists_GetLabelNon404Error(t *testing.T) {
	client := NewMockGitHubClient()
	client.GetLabelError = &github.ErrorResponse{
		Message:  "Internal Server Error",
		Response: &http.Response{StatusCode: http.StatusInternalServerError},
	}

	config := createTestConfig()
	config.AutoCreate = true
	labeler := NewLabeler(client, config)

	err := labeler.ensureLabelExists(context.Background(), "test-owner", "test-repo", "needs-triage", "ededed", "Needs triage")
	if err == nil {
		t.Fatal("expected error when GetLabel returns non-404 API error")
	}
	if !strings.Contains(err.Error(), "failed to check label") {
		t.Fatalf("expected non-404 error to fail label check, got: %v", err)
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
