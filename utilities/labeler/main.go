package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

type Action struct {
	Kind  string                 `yaml:"kind"`
	Label string                 `yaml:"label,omitempty"`
	Spec  map[string]interface{} `yaml:"spec,omitempty"`
}

type RuleSpec struct {
	Command        string        `yaml:"command,omitempty"`
	Rules          []interface{} `yaml:"rules,omitempty"`
	Match          string        `yaml:"match,omitempty"`
	MatchCondition string        `yaml:"matchCondition,omitempty"`
	MatchPath      string        `yaml:"matchPath,omitempty"`
}

type Rule struct {
	Name    string   `yaml:"name"`
	Kind    string   `yaml:"kind"`
	Spec    RuleSpec `yaml:"spec"`
	Actions []Action `yaml:"actions"`
}

type LabelsYAML struct {
	Labels     []Label `yaml:"labels"`
	Ruleset    []Rule  `yaml:"ruleset"`
	AutoCreate bool    `yaml:"autoCreateLabels"`
	AutoDelete bool    `yaml:"autoDeleteLabels"`
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

	// Parse changed files if provided
	var files []string
	if *changedFiles != "" {
		files = strings.Split(*changedFiles, ",")
	}

	// Process filePath rules
	if len(files) > 0 {
		for _, rule := range cfg.Ruleset {
			if rule.Kind == "filePath" {
				for _, file := range files {
					matched, err := filepath.Match(rule.Spec.MatchPath, file)
					if err != nil {
						log.Printf("error matching file path: %v", err)
						continue
					}
					if matched {
						for _, action := range rule.Actions {
							switch action.Kind {
							case "apply-label":
								label := renderLabel(action.Label, nil)
								applyLabel(ctx, client, owner, repo, toInt(issueNum), label, cfg)
							case "remove-label":
								match, _ := action.Spec["match"].(string)
								if match != "" {
									removeLabel(ctx, client, owner, repo, toInt(issueNum), match)
								}
							}
						}
					}
				}
			}
		}
	}

	issue, _, err := client.Issues.Get(ctx, owner, repo, toInt(issueNum))
	if err != nil {
		log.Fatalf("failed to fetch issue: %v", err)
	}
	fmt.Printf("Issue #%d: %s\n", *issue.Number, *issue.Title)

	lines := strings.Split(commentBody, "\n")
	for _, rule := range cfg.Ruleset {
		if rule.Spec.Command != "" {
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, rule.Spec.Command) {
					parts := strings.Fields(line)
					argv := []string{}
					if len(parts) > 1 {
						argv = parts[1:]
					}
					for _, action := range rule.Actions {
						switch action.Kind {
						case "apply-label":
							label := renderLabel(action.Label, argv)
							applyLabel(ctx, client, owner, repo, toInt(issueNum), label, cfg)
						case "remove-label":
							match, _ := action.Spec["match"].(string)
							if match != "" {
								removeLabel(ctx, client, owner, repo, toInt(issueNum), match)
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

	// Fetch existing labels
	existingLabels, _, err := client.Issues.ListLabels(ctx, owner, repo, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch labels: %v", err)
	}

	// Check if the label already exists
	for _, lbl := range existingLabels {
		if lbl.GetName() == labelName {
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
			return nil
		}
	}

	// Create the label if it doesn't exist
	_, _, err = client.Issues.CreateLabel(ctx, owner, repo, &github.Label{
		Name:        &labelName,
		Color:       &color,
		Description: &description,
	})
	if err != nil {
		return fmt.Errorf("failed to create label %s: %v", labelName, err)
	}
	return nil
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
	return "000000", "Automatically applied label", labelName // Default values
}

func applyLabel(ctx context.Context, client *github.Client, owner, repo string, issueNum int, label string, cfg *LabelsYAML) {
	fmt.Printf("Applying label: %s\n", label)

	// Get label definition from config
	color, description, resolvedLabel := getLabelDefinition(cfg, label)

	// Ensure the label exists with the defined color and description
	if err := ensureLabelExists(ctx, client, owner, repo, resolvedLabel, color, description, cfg); err != nil {
		log.Printf("skipping label %s due to error: %v", resolvedLabel, err)
		return
	}

	_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, issueNum, []string{resolvedLabel})
	if err != nil {
		log.Printf("failed to apply label %s: %v", resolvedLabel, err)
	}
}

func removeLabel(ctx context.Context, client *github.Client, owner, repo string, issueNum int, label string) {
	fmt.Printf("Removing label: %s\n", label)
	_, err := client.Issues.RemoveLabelForIssue(ctx, owner, repo, issueNum, label)
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
