package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v3"
)

// LabelConfig represents the structure of labels.yaml
// Only the fields needed for logic are included here

type Label struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
	Previously  []struct {
		Name string `yaml:"name"`
	} `yaml:"previously"`
}

type ActionSpec struct {
	Match string `yaml:"match,omitempty"`
}

type Action struct {
	Kind  string     `yaml:"kind"`
	Label string     `yaml:"label,omitempty"`
	Spec  ActionSpec `yaml:"spec,omitempty"`
}

type RuleSpec struct {
	Command        string        `yaml:"command,omitempty"`
	Rules          []interface{} `yaml:"rules,omitempty"`
	Match          string        `yaml:"match,omitempty"`
	MatchCondition string        `yaml:"matchCondition,omitempty"`
	MatchPath      string        `yaml:"matchPath,omitempty"`
	MatchList      []string      `yaml:"matchList,omitempty"`
}

type Rule struct {
	Name    string   `yaml:"name"`
	Kind    string   `yaml:"kind"`
	Spec    RuleSpec `yaml:"spec"`
	Actions []Action `yaml:"actions"`
}

type LabelsYAML struct {
	Labels             []Label `yaml:"labels"`
	Ruleset            []Rule  `yaml:"ruleset"`
	AutoCreate         bool    `yaml:"autoCreateLabels"`
	AutoDelete         bool    `yaml:"autoDeleteLabels"`
	DefinitionRequired bool    `yaml:"definitionRequired"`
	Debug              bool    `yaml:"debug"`
}

func loadConfigFromURL(url string) (*LabelsYAML, error) {
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

func githubClient() (*github.Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return github.NewClient(oauth2.NewClient(context.Background(), ts)), nil
}

var (
	changedFiles = flag.String("changed-files", "", "Comma-separated list of changed files")
)

func main() {
	flag.Parse()

	if len(flag.Args()) < 5 {
		fmt.Println("Usage: labeler [flags] <labels_url> <owner> <repo> <issue_number> <comment_body>")
		os.Exit(1)
	}
	labelsURL := flag.Arg(0)
	owner := flag.Arg(1)
	repo := flag.Arg(2)
	issueNum := flag.Arg(3)
	commentBody := flag.Arg(4)

	if labelsURL == "" {
		log.Fatal("labels URL not set")
	}

	cfg, err := loadConfigFromURL(labelsURL)
	if err != nil {
		log.Fatalf("failed to load labels.yaml: %v", err)
	}

	client, err := githubClient()
	if err != nil {
		log.Fatalf("failed to create GitHub client: %v", err)
	}
	ctx := context.Background()

	if cfg.AutoDelete {
		deleteUndefinedLabels(ctx, client, owner, repo, cfg)
	}

	if cfg.AutoCreate {
		for _, label := range cfg.Labels {
			color, description, labelName := getLabelDefinition(cfg, label.Name)
			if err := ensureLabelExists(ctx, client, owner, repo, labelName, color, description, cfg); err != nil {
				log.Printf("skipping label %s due to error: %v", labelName, err)
				continue
			}
		}
	}

	// Parse changed files if provided
	var files []string
	if *changedFiles != "" {
		files = strings.Split(*changedFiles, ",")
	}

	issue, _, err := client.Issues.Get(ctx, owner, repo, toInt(issueNum))
	if err != nil {
		log.Fatalf("failed to fetch issue: %v", err)
	}
	fmt.Printf("Issue #%d: %s\n", *issue.Number, *issue.Title)
	lines := strings.Split(commentBody, "\n")

	for _, rule := range cfg.Ruleset {
		if rule.Kind == "filePath" {
			if len(files) == 0 {
				if cfg.Debug {
					log.Printf("No changed files to process for rule %s", rule.Name)
				}
				continue
			}
			for _, file := range files {
				matched, err := filepath.Match(rule.Spec.MatchPath, file)
				if err != nil {
					log.Printf("error matching file path: %v", err)
					continue
				}
				if matched {
					for _, action := range rule.Actions {
						label := renderLabel(action.Label, nil)
						switch action.Kind {
						case "apply-label":
							applyLabel(ctx, client, owner, repo, toInt(issueNum), label, cfg)
						case "remove-label":
							if label != "" {
								removeLabel(ctx, client, owner, repo, toInt(issueNum), label)
							}
						}
					}
				}
			}
		}

		if rule.Spec.Command != "" {
			if !strings.HasPrefix(rule.Spec.Command, "/") {
				log.Printf("Command `%s` does not start with a forward slash, skipping", rule.Spec.Command)
				continue
			}
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, rule.Spec.Command) {
					parts := strings.Fields(line)
					log.Printf("%v", parts)
					argv := []string{}
					if len(parts) > 1 {
						argv = parts[1:]
					}
					/*
						for key, arg := range argv {
							// Handle namespace prefix
							slashIndex := strings.Index(arg, "/")
							if slashIndex == -1 {
								log.Printf("Argument `%s` does not contain a namespace, skipping", arg)
								continue
							}
							namespace := arg[:slashIndex]
							fullArg := fmt.Sprintf("%s/%s", namespace, arg[slashIndex+1:])
							log.Printf("Checking full argument `%s` against command %s", fullArg, rule.Spec.Command)

							argv[key] = fullArg // Replace argv[0] with full argument
						}
					*/
					if len(rule.Spec.MatchList) > 0 {
						// Validate argv.0 against matchList
						valid := slices.Contains(rule.Spec.MatchList, argv[0])
						if !valid {
							log.Printf("Invalid argument `%s` for command %s", argv[0], rule.Spec.Command)
							continue
						}
					}

					for _, action := range rule.Actions {
						label := ""
						if action.Label != "" {
							label = renderLabel(action.Label, argv)
						}
						if action.Spec.Match != "" {
							// Handle match condition for action
							if cfg.Debug {
								log.Printf("Checking action label `%s` against Spec.Match `%s` and Argv `%v`", action.Label, action.Spec.Match, argv)
							}
							label = renderLabel(action.Spec.Match, argv)
							if !cfg.isValidLabel(label) && !strings.Contains(label, "/*") {
								log.Printf("Label `%s` is not defined in labels.yaml", label)
								continue
							}
						}

						switch action.Kind {
						case "apply-label":
							applyLabel(ctx, client, owner, repo, toInt(issueNum), label, cfg)
						case "remove-label":
							if label != "" {
								removeLabel(ctx, client, owner, repo, toInt(issueNum), label)
							}
						}
					}
				}
			}
		} else if rule.Kind == "label" {
			// Handle default namespaced label logic dynamically
			if rule.Spec.MatchCondition == "NOT" {
				namespacePattern := rule.Spec.Match
				existingLabels, _, err := client.Issues.ListLabelsByIssue(ctx, owner, repo, toInt(issueNum), nil)
				if err != nil {
					log.Printf("failed to fetch labels for issue: %v", err)
					continue
				}

				foundNamespace := false
				for _, lbl := range existingLabels {
					matched, _ := filepath.Match(namespacePattern, lbl.GetName())
					if matched {
						foundNamespace = true
						break
					}
				}

				if !foundNamespace {
					for _, action := range rule.Actions {
						label := renderLabel(action.Label, nil)
						switch action.Kind {
						case "apply-label":
							applyLabel(ctx, client, owner, repo, toInt(issueNum), label, cfg)
						case "remove-label":
							if label != "" {
								removeLabel(ctx, client, owner, repo, toInt(issueNum), label)
							}
						}
					}
				}
			}
		}
	}
}

func deleteUndefinedLabels(ctx context.Context, client *github.Client, owner, repo string, cfg *LabelsYAML) {
	existingLabels, _, err := client.Issues.ListLabels(ctx, owner, repo, nil)
	if err != nil {
		log.Printf("failed to fetch existing labels: %v", err)
		return
	}

	definedLabels := map[string]bool{}
	for _, label := range cfg.Labels {
		definedLabels[label.Name] = true
		for _, prev := range label.Previously {
			definedLabels[prev.Name] = true
		}
	}

	for _, lbl := range existingLabels {
		if !definedLabels[lbl.GetName()] {
			log.Printf("deleting undefined label: %s", lbl.GetName())
			_, err := client.Issues.DeleteLabel(ctx, owner, repo, lbl.GetName())
			if err != nil {
				log.Printf("failed to delete label %s: %v", lbl.GetName(), err)
			}
		}
	}
}

// renderLabel replaces {{ argv.0 }} etc. in label templates
func renderLabel(template string, argv []string) string {
	label := template
	for i, v := range argv {
		label = strings.ReplaceAll(label, fmt.Sprintf("{{ argv.%d }}", i), v)
	}
	return label
}

func ensureLabelExists(ctx context.Context, client *github.Client, owner, repo, labelName, color, description string, cfg *LabelsYAML) error {
	if !cfg.AutoCreate {
		return fmt.Errorf("label %s does not exist and auto-create-labels is disabled", labelName)
	}

	lbl, _, err := client.Issues.GetLabel(ctx, owner, repo, labelName)
	if err != nil {
		// Create the label if it doesn't exist
		lbl, _, err = client.Issues.CreateLabel(ctx, owner, repo, &github.Label{
			Name:        &labelName,
			Color:       &color,
			Description: &description,
		})
		if err != nil {
			return fmt.Errorf("failed to create label %s: %v", labelName, err)
		}
	}

	// Update label if color or description differs
	if lbl.GetColor() != color || lbl.GetDescription() != description {
		_, _, err := client.Issues.EditLabel(ctx, owner, repo, labelName, &github.Label{
			Name:        &labelName,
			Color:       &color,
			Description: &description,
		})
		if err != nil {
			return fmt.Errorf("failed to update label %s: %v", labelName, err)
		}
	}
	if cfg.Debug {
		log.Printf("label %s exists", labelName)
	}
	return nil
}

func (cfg *LabelsYAML) isValidLabel(label string) bool {
	for _, lbl := range cfg.Labels {
		if lbl.Name == label {
			return true
		}
	}
	return false
}

func getLabelDefinition(cfg *LabelsYAML, labelName string) (string, string, string) {
	for _, label := range cfg.Labels {
		if label.Name == labelName {
			return label.Color, label.Description, label.Name
		}
		for _, prev := range label.Previously {
			if prev.Name == labelName {
				return label.Color, label.Description, label.Name
			}
		}
	}
	if cfg.DefinitionRequired {
		log.Printf("label %s is not defined in labels.yaml, but auto-create is disabled", labelName)
		return "", "", "" // Return empty if not required
	}
	return "000000", "Automatically applied label", labelName // Default values
}

func applyLabel(ctx context.Context, client *github.Client, owner, repo string, issueNum int, label string, cfg *LabelsYAML) {
	fmt.Printf("Applying label: %s\n", label)

	// Get current labels for the issue
	existingLabels, _, err := client.Issues.ListLabelsByIssue(ctx, owner, repo, issueNum, nil)
	if err != nil {
		log.Printf("failed to fetch labels for issue: %v", err)
		return
	}

	// Check if the label is already applied
	for _, lbl := range existingLabels {
		if lbl.GetName() == label {
			log.Printf("label %s is already applied, skipping", label)
			return
		}
	}

	// Get label definition from config
	color, description, resolvedLabel := getLabelDefinition(cfg, label)

	if resolvedLabel == "" {
		log.Printf("label %s is not defined in labels.yaml and auto-create is disabled",
			label)
		return
	}

	// Ensure the label exists with the defined color and description
	if err := ensureLabelExists(ctx, client, owner, repo, resolvedLabel, color, description, cfg); err != nil {
		log.Printf("skipping label %s due to error: %v", resolvedLabel, err)
		return
	}

	_, _, err = client.Issues.AddLabelsToIssue(ctx, owner, repo, issueNum, []string{resolvedLabel})
	if err != nil {
		log.Printf("failed to apply label %s: %v", resolvedLabel, err)
	}
}

func removeLabel(ctx context.Context, client *github.Client, owner, repo string, issueNum int, label string) {
	fmt.Printf("Removing label: %s\n", label)

	// Get current labels for the issue
	existingLabels, _, err := client.Issues.ListLabelsByIssue(ctx, owner, repo, issueNum, nil)
	if err != nil {
		log.Printf("failed to fetch labels for issue: %v", err)
		return
	}

	// Check if the label is not applied
	labelFound := false
	for _, lbl := range existingLabels {
		if strings.Contains(label, "/*") {
			// Handle wildcard removal
			if strings.HasPrefix(lbl.GetName(), strings.TrimSuffix(label, "*")) {
				removeLabel(ctx, client, owner, repo, issueNum, lbl.GetName())
			}
		}
		if lbl.GetName() == label {
			labelFound = true
			break
		}
	}

	if !labelFound {
		log.Printf("label %s is not applied, skipping removal", label)
		return
	}

	_, err = client.Issues.RemoveLabelForIssue(ctx, owner, repo, issueNum, label)
	if err != nil {
		log.Printf("failed to remove label %s: %v", label, err)
	}
}

func toInt(s string) int {
	n, err := fmt.Sscanf(s, "%d", new(int))
	if err != nil || n != 1 {
		log.Fatalf("invalid issue number: %s", s)
	}
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
