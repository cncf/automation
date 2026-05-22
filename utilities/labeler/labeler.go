package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v3"
)

// GitHubClient interface to allow mocking
type GitHubClient interface {
	GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, *github.Response, error)
	ListLabelsByIssue(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.Label, *github.Response, error)
	AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
	RemoveLabelForIssue(ctx context.Context, owner, repo string, number int, label string) (*github.Response, error)
	ListLabels(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.Label, *github.Response, error)
	CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error)
	EditLabel(ctx context.Context, owner, repo, name string, label *github.Label) (*github.Label, *github.Response, error)
	DeleteLabel(ctx context.Context, owner, repo, name string) (*github.Response, error)
	GetLabel(ctx context.Context, owner, repo, name string) (*github.Label, *github.Response, error)
}

// GitHubClientWrapper wraps the actual GitHub client
type GitHubClientWrapper struct {
	client *github.Client
}

func (g *GitHubClientWrapper) GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, *github.Response, error) {
	return g.client.Issues.Get(ctx, owner, repo, number)
}

func (g *GitHubClientWrapper) ListLabelsByIssue(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.Label, *github.Response, error) {
	return g.client.Issues.ListLabelsByIssue(ctx, owner, repo, number, opts)
}

func (g *GitHubClientWrapper) AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error) {
	return g.client.Issues.AddLabelsToIssue(ctx, owner, repo, number, labels)
}

func (g *GitHubClientWrapper) RemoveLabelForIssue(ctx context.Context, owner, repo string, number int, label string) (*github.Response, error) {
	return g.client.Issues.RemoveLabelForIssue(ctx, owner, repo, number, label)
}

func (g *GitHubClientWrapper) ListLabels(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.Label, *github.Response, error) {
	return g.client.Issues.ListLabels(ctx, owner, repo, opts)
}

func (g *GitHubClientWrapper) CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error) {
	return g.client.Issues.CreateLabel(ctx, owner, repo, label)
}

func (g *GitHubClientWrapper) EditLabel(ctx context.Context, owner, repo, name string, label *github.Label) (*github.Label, *github.Response, error) {
	return g.client.Issues.EditLabel(ctx, owner, repo, name, label)
}

func (g *GitHubClientWrapper) DeleteLabel(ctx context.Context, owner, repo, name string) (*github.Response, error) {
	return g.client.Issues.DeleteLabel(ctx, owner, repo, name)
}

func (g *GitHubClientWrapper) GetLabel(ctx context.Context, owner, repo, name string) (*github.Label, *github.Response, error) {
	return g.client.Issues.GetLabel(ctx, owner, repo, name)
}

// Labeler handles the core labeling logic
type Labeler struct {
	client GitHubClient
	config *LabelsYAML
}

// NewLabeler creates a new Labeler instance
func NewLabeler(client GitHubClient, config *LabelsYAML) *Labeler {
	return &Labeler{
		client: client,
		config: config,
	}
}

// ProcessRequest processes a labeling request
func (l *Labeler) ProcessRequest(ctx context.Context, req *LabelRequest) error {
	if l.config.AutoDelete {
		if err := l.deleteUndefinedLabels(ctx, req.Owner, req.Repo); err != nil {
			log.Printf("failed to delete undefined labels: %v", err)
		}
	}

	if l.config.AutoCreate {
		if err := l.ensureDefinedLabelsExist(ctx, req.Owner, req.Repo); err != nil {
			log.Printf("failed to ensure defined labels exist: %v", err)
		}
	}

	issue, _, err := l.client.GetIssue(ctx, req.Owner, req.Repo, req.IssueNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch issue: %v", err)
	}

	if l.config.Debug {
		log.Printf("Processing issue #%d: %s", *issue.Number, *issue.Title)
	}

	return l.processRules(ctx, req, issue)
}

// LabelRequest represents a labeling request
type LabelRequest struct {
	Owner        string
	Repo         string
	IssueNumber  int
	CommentBody  string
	ChangedFiles []string
}

func (l *Labeler) processRules(ctx context.Context, req *LabelRequest, issue *github.Issue) error {
	for _, rule := range l.config.Ruleset {
		if err := l.processRule(ctx, req, rule); err != nil {
			log.Printf("error processing rule %s: %v", rule.Name, err)
		}
	}
	return nil
}

func (l *Labeler) processRule(ctx context.Context, req *LabelRequest, rule Rule) error {
	switch rule.Kind {
	case "filePath":
		return l.processFilePathRule(ctx, req, rule)
	case "match":
		return l.processMatchRule(ctx, req, rule)
	case "label":
		return l.processLabelRule(ctx, req, rule)
	default:
		return fmt.Errorf("unknown rule kind: %s", rule.Kind)
	}
}

func (l *Labeler) processFilePathRule(ctx context.Context, req *LabelRequest, rule Rule) error {
	if len(req.ChangedFiles) == 0 {
		if l.config.Debug {
			log.Printf("No changed files to process for rule %s", rule.Name)
		}
		return nil
	}

	for _, file := range req.ChangedFiles {
		matched, err := filepath.Match(rule.Spec.MatchPath, file)
		if err != nil {
			return fmt.Errorf("error matching file path: %v", err)
		}

		shouldApply := matched
		if rule.Spec.MatchCondition == "NOT" {
			shouldApply = !matched
		}

		if shouldApply {
			for _, action := range rule.Actions {
				if err := l.executeAction(ctx, req, action, nil); err != nil {
					log.Printf("error executing action: %v", err)
				}
			}
		}
	}
	return nil
}

func (l *Labeler) processMatchRule(ctx context.Context, req *LabelRequest, rule Rule) error {
	if rule.Spec.Command == "" {
		return fmt.Errorf("match rule missing command")
	}

	if !strings.HasPrefix(rule.Spec.Command, "/") {
		if l.config.Debug {
			log.Printf("Command `%s` does not start with a forward slash, skipping", rule.Spec.Command)
		}
		return nil
	}

	lines := strings.Split(req.CommentBody, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, rule.Spec.Command) {
			parts := strings.Fields(line)
			argv := []string{}
			if len(parts) > 1 {
				argv = parts[1:]
			}

			if len(rule.Spec.MatchList) > 0 && len(argv) > 0 {
				if !slices.Contains(rule.Spec.MatchList, argv[0]) {
					if l.config.Debug {
						log.Printf("Invalid argument `%s` for command %s", argv[0], rule.Spec.Command)
					}
					continue
				}
			}

			for _, action := range rule.Actions {
				if err := l.executeAction(ctx, req, action, argv); err != nil {
					log.Printf("error executing action: %v", err)
				}
			}
		}
	}
	return nil
}

func (l *Labeler) processLabelRule(ctx context.Context, req *LabelRequest, rule Rule) error {
	// Compile and validate the rule's match pattern once, before iterating
	// existing labels. Doing this only inside the loop would skip validation
	// on issues that have zero labels yet (e.g. a freshly-opened PR), which
	// would let a malformed pattern silently fall through to
	// foundNamespace=false and fire the rule in the wrong direction.
	patterns, err := compilePatterns(rule.Spec.Match)
	if err != nil {
		return fmt.Errorf("rule %q: invalid match pattern %q: %v",
			rule.Name, rule.Spec.Match, err)
	}

	existingLabels, _, err := l.client.ListLabelsByIssue(ctx, req.Owner, req.Repo, req.IssueNumber, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch labels for issue: %v", err)
	}

	foundNamespace := false
	for _, lbl := range existingLabels {
		if matchAny(patterns, lbl.GetName()) {
			foundNamespace = true
			break
		}
	}

	// Decide whether to fire the rule's actions based on matchCondition:
	//   "AND" → fire when the namespace IS present on the issue.
	//   "NOT" (or unset, the default) → fire when the namespace is NOT present.
	// Without honoring "AND" here, a rule like
	//     match: triage/*
	//     matchCondition: AND
	//     actions: [remove-label: needs-triage]
	// behaves identically to its "NOT" sibling and ends up firing in exactly
	// the wrong situations (e.g. removing needs-triage when no triage/* is set
	// while a fresh PR is being opened, and never removing it once one is).
	shouldApply := !foundNamespace
	if strings.EqualFold(rule.Spec.MatchCondition, "AND") {
		shouldApply = foundNamespace
	}

	if l.config.Debug {
		log.Printf("Label rule %s: foundNamespace=%v, matchCondition=%s, shouldApply=%v",
			rule.Name, foundNamespace, rule.Spec.MatchCondition, shouldApply)
	}

	if shouldApply {
		for _, action := range rule.Actions {
			if err := l.executeAction(ctx, req, action, nil); err != nil {
				log.Printf("error executing action: %v", err)
			}
		}
	}
	return nil
}

func (l *Labeler) executeAction(ctx context.Context, req *LabelRequest, action Action, argv []string) error {
	var label string
	if action.Spec.Label != "" {
		label = l.renderLabel(action.Spec.Label, argv)
	}
	if action.Spec.Match != "" {
		label = l.renderLabel(action.Spec.Match, argv)
		if !l.isValidLabel(label) && !strings.Contains(label, "/*") {
			if l.config.Debug {
				log.Printf("Label `%s` is not defined in labels.yaml", label)
			}
			return nil
		}
	}

	switch action.Kind {
	case "apply-label":
		return l.applyLabel(ctx, req.Owner, req.Repo, req.IssueNumber, label)
	case "remove-label":
		if label != "" {
			return l.removeLabel(ctx, req.Owner, req.Repo, req.IssueNumber, label)
		}
	}
	return nil
}

// compilePatterns expands and validates a rule's match pattern, returning
// the slice of concrete path.Match patterns that should be tried for each
// label. Supports a single level of comma-separated brace alternation
// (e.g. "{toc,tag/*,sub/*}") in addition to the wildcards understood by
// path.Match.
//
// Validation happens here rather than at match time so callers can surface
// a malformed pattern even on issues that have no labels yet — otherwise
// the match loop would never execute and the rule would silently fire
// based on foundNamespace=false.
//
// path.Match (not filepath.Match) is used so that "/" is always treated as
// the separator regardless of the host OS — labels are not filesystem
// paths and "*" must not cross "/" on any platform.
func compilePatterns(pattern string) ([]string, error) {
	patterns, err := expandBraces(pattern)
	if err != nil {
		return nil, err
	}
	// path.Match reports ErrBadPattern on malformed inputs (unclosed
	// character classes, etc.) regardless of the name argument, so a
	// single call per expanded pattern is enough to validate it.
	for _, p := range patterns {
		if _, err := path.Match(p, ""); err != nil {
			return nil, fmt.Errorf("malformed pattern %q: %v", p, err)
		}
	}
	return patterns, nil
}

// matchAny reports whether name matches any of the pre-compiled patterns.
// The patterns must already have been validated by compilePatterns; any
// path.Match error is therefore unexpected and treated as "no match".
func matchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if matched, _ := path.Match(p, name); matched {
			return true
		}
	}
	return false
}

// expandBraces expands a single, non-nested brace alternation in pattern.
// "{a,b/*,c}" → ["a", "b/*", "c"], "foo-{a,b}" → ["foo-a", "foo-b"].
// Patterns without braces are returned unchanged as a single-element slice.
//
// Multiple brace pairs (e.g. "foo-{a,b}-{c,d}") and nested braces
// (e.g. "{a,{b,c}}") are not supported and produce an error rather than a
// partial expansion that would silently mis-match (e.g. "foo-{a,b}-{c,d}"
// would expand to "foo-a-{c,d}" / "foo-b-{c,d}", and path.Match would then
// treat "{c,d}" as literal characters).
//
// A mismatched '}' before any '{' is treated the same way — better to
// surface the malformed pattern than to silently match unexpectedly.
func expandBraces(pattern string) ([]string, error) {
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")
	if start == -1 && end == -1 {
		return []string{pattern}, nil
	}
	if start == -1 || end < start {
		return nil, fmt.Errorf("unbalanced braces in pattern %q", pattern)
	}
	inner := pattern[start+1 : end]
	suffix := pattern[end+1:]
	if strings.ContainsAny(inner, "{}") || strings.ContainsAny(suffix, "{}") {
		return nil, fmt.Errorf(
			"pattern %q has nested or multiple brace groups; only a single, non-nested {a,b,c} is supported",
			pattern,
		)
	}
	prefix := pattern[:start]
	parts := strings.Split(inner, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, prefix+strings.TrimSpace(p)+suffix)
	}
	return out, nil
}

func (l *Labeler) renderLabel(template string, argv []string) string {
	label := template
	for i, v := range argv {
		label = strings.ReplaceAll(label, fmt.Sprintf("{{ argv.%d }}", i), v)
	}
	return label
}

func (l *Labeler) isValidLabel(label string) bool {
	for _, lbl := range l.config.Labels {
		if lbl.Name == label {
			return true
		}
	}
	return false
}

func (l *Labeler) applyLabel(ctx context.Context, owner, repo string, issueNum int, label string) error {
	if l.config.Debug {
		log.Printf("Applying label: %s", label)
	}

	// Get current labels for the issue
	existingLabels, _, err := l.client.ListLabelsByIssue(ctx, owner, repo, issueNum, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch labels for issue: %v", err)
	}

	// Check if the label is already applied
	for _, lbl := range existingLabels {
		if lbl.GetName() == label {
			if l.config.Debug {
				log.Printf("label %s is already applied, skipping", label)
			}
			return nil
		}
	}

	// Get label definition from config
	color, description, resolvedLabel := l.getLabelDefinition(label)
	if resolvedLabel == "" {
		return fmt.Errorf("label %s is not defined in labels.yaml and auto-create is disabled", label)
	}

	// Ensure the label exists with the defined color and description
	if err := l.ensureLabelExists(ctx, owner, repo, resolvedLabel, color, description); err != nil {
		return fmt.Errorf("failed to ensure label exists: %v", err)
	}

	_, _, err = l.client.AddLabelsToIssue(ctx, owner, repo, issueNum, []string{resolvedLabel})
	if err != nil {
		return fmt.Errorf("failed to apply label %s: %v", resolvedLabel, err)
	}
	return nil
}

func (l *Labeler) removeLabel(ctx context.Context, owner, repo string, issueNum int, label string) error {
	if l.config.Debug {
		log.Printf("Removing label: %s", label)
	}

	// Get current labels for the issue
	existingLabels, _, err := l.client.ListLabelsByIssue(ctx, owner, repo, issueNum, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch labels for issue: %v", err)
	}

	// Handle wildcard removal
	if strings.Contains(label, "/*") {
		prefix := strings.TrimSuffix(label, "*")
		removed := false
		for _, lbl := range existingLabels {
			if strings.HasPrefix(lbl.GetName(), prefix) {
				if err := l.removeLabel(ctx, owner, repo, issueNum, lbl.GetName()); err != nil {
					log.Printf("error removing label %s: %v", lbl.GetName(), err)
				} else {
					removed = true
				}
			}
		}
		if !removed && l.config.Debug {
			log.Printf("no labels matching pattern %s found to remove", label)
		}
		return nil
	}

	// Check if the label is applied
	labelFound := false
	for _, lbl := range existingLabels {
		if lbl.GetName() == label {
			labelFound = true
			break
		}
	}

	if !labelFound {
		if l.config.Debug {
			log.Printf("label %s is not applied, skipping removal", label)
		}
		return nil
	}

	_, err = l.client.RemoveLabelForIssue(ctx, owner, repo, issueNum, label)
	if err != nil {
		return fmt.Errorf("failed to remove label %s: %v", label, err)
	}
	return nil
}

func (l *Labeler) getLabelDefinition(labelName string) (string, string, string) {
	for _, label := range l.config.Labels {
		if label.Name == labelName {
			return label.Color, label.Description, label.Name
		}
		for _, prev := range label.Previously {
			if prev.Name == labelName {
				return label.Color, label.Description, label.Name
			}
		}
	}
	if l.config.DefinitionRequired {
		return "", "", ""
	}
	return "000000", "Automatically applied label", labelName
}

func (l *Labeler) ensureLabelExists(ctx context.Context, owner, repo, labelName, color, description string) error {
	lbl, _, err := l.client.GetLabel(ctx, owner, repo, labelName)
	if err != nil {
		if !isLabelNotFoundError(err) {
			return fmt.Errorf("failed to check label %s: %v", labelName, err)
		}
		if !l.config.AutoCreate {
			return fmt.Errorf("label %s does not exist and auto-create-labels is disabled", labelName)
		}
		// Create the label when missing and auto-create is enabled.
		_, _, err = l.client.CreateLabel(ctx, owner, repo, &github.Label{
			Name:        &labelName,
			Color:       &color,
			Description: &description,
		})
		if err != nil {
			return fmt.Errorf("failed to create label %s: %v", labelName, err)
		}
		return nil
	}

	if !l.config.AutoCreate {
		return nil
	}

	// Update label if color or description differs
	if lbl.GetColor() != color || lbl.GetDescription() != description {
		_, _, err := l.client.EditLabel(ctx, owner, repo, labelName, &github.Label{
			Name:        &labelName,
			Color:       &color,
			Description: &description,
		})
		if err != nil {
			return fmt.Errorf("failed to update label %s: %v", labelName, err)
		}
	}
	return nil
}

func isLabelNotFoundError(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		if ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
			return true
		}
		return strings.EqualFold(ghErr.Message, "Not Found")
	}
	return false
}

func (l *Labeler) ensureDefinedLabelsExist(ctx context.Context, owner, repo string) error {
	for _, label := range l.config.Labels {
		color, description, labelName := l.getLabelDefinition(label.Name)
		if err := l.ensureLabelExists(ctx, owner, repo, labelName, color, description); err != nil {
			log.Printf("skipping label %s due to error: %v", labelName, err)
		}
	}
	return nil
}

func (l *Labeler) deleteUndefinedLabels(ctx context.Context, owner, repo string) error {
	existingLabels, _, err := l.client.ListLabels(ctx, owner, repo, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch existing labels: %v", err)
	}

	definedLabels := map[string]bool{}
	for _, label := range l.config.Labels {
		definedLabels[label.Name] = true
		for _, prev := range label.Previously {
			definedLabels[prev.Name] = true
		}
	}

	for _, lbl := range existingLabels {
		if !definedLabels[lbl.GetName()] {
			if l.config.Debug {
				log.Printf("deleting undefined label: %s", lbl.GetName())
			}
			_, err := l.client.DeleteLabel(ctx, owner, repo, lbl.GetName())
			if err != nil {
				log.Printf("failed to delete label %s: %v", lbl.GetName(), err)
			}
		}
	}
	return nil
}

// LoadConfigFromURL loads configuration from a URL
func LoadConfigFromURL(url string) (*LabelsYAML, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch labels.yaml from URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch labels.yaml: HTTP %d", resp.StatusCode)
	}

	var cfg LabelsYAML
	dec := yaml.NewDecoder(resp.Body)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode labels.yaml: %v", err)
	}
	return &cfg, nil
}

// CreateGitHubClient creates a new GitHub client
func CreateGitHubClient(token string) (*GitHubClientWrapper, error) {
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not provided")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return &GitHubClientWrapper{
		client: github.NewClient(oauth2.NewClient(context.Background(), ts)),
	}, nil
}
