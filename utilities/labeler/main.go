package main

import (
	"context"
	"fmt"
	"log"
	"os"
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
	Labels  []Label `yaml:"labels"`
	Ruleset []Rule  `yaml:"ruleset"`
}

func loadConfig(path string) (*LabelsYAML, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg LabelsYAML
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func githubClient() *github.Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable not set")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return github.NewClient(oauth2.NewClient(context.Background(), ts))
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: labeler <owner> <repo> <issue_number> <comment_body>")
		os.Exit(1)
	}
	owner := os.Args[1]
	repo := os.Args[2]
	issueNum := os.Args[3]
	commentBody := os.Args[4]

	cfg, err := loadConfig("labels.yaml")
	if err != nil {
		log.Fatalf("failed to load labels.yaml: %v", err)
	}

	client := githubClient()
	ctx := context.Background()

	issue, _, err := client.Issues.Get(ctx, owner, repo, toInt(issueNum))
	if err != nil {
		log.Fatalf("failed to fetch issue: %v", err)
	}
	fmt.Printf("Issue #%d: %s\n", *issue.Number, *issue.Title)

	// Scan for all supported commands anywhere in the comment body
	lines := strings.Split(commentBody, "\n")
	for _, rule := range cfg.Ruleset {
		if rule.Spec.Command != "" {
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, rule.Spec.Command) {
					// Split command and args
					parts := strings.Fields(line)
					argv := []string{}
					if len(parts) > 1 {
						argv = parts[1:]
					}
					for _, action := range rule.Actions {
						switch action.Kind {
						case "apply-label":
							label := renderLabel(action.Label, argv)
							applyLabel(ctx, client, owner, repo, toInt(issueNum), label)
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

// renderLabel replaces {{ argv.0 }} etc. in label templates
func renderLabel(template string, argv []string) string {
	label := template
	for i, v := range argv {
		label = replaceAll(label, fmt.Sprintf("{{ argv.%d }}", i), v)
	}
	return label
}

func replaceAll(s, old, new string) string {
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			break
		}
		s = s[:idx] + new + s[idx+len(old):]
	}
	return s
}

func indexOf(s, substr string) int {
	return strings.Index(s, substr)
}

func applyLabel(ctx context.Context, client *github.Client, owner, repo string, issueNum int, label string) {
	fmt.Printf("Applying label: %s\n", label)
	_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, issueNum, []string{label})
	if err != nil {
		log.Printf("failed to apply label %s: %v", label, err)
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
