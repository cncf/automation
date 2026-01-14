package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

const (
	landscapeURL = "https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml"
	sandboxOwner = "cncf"
	sandboxRepo  = "sandbox"
	gitVoteLabel = "gitvote/passed"
)

// SandboxIssue represents a project that passed GitVote
type SandboxIssue struct {
	Number      int
	Title       string
	ProjectName string
	RepoURL     string
	WebsiteURL  string
	Description string
	State       string
	Labels      []string
	CreatedAt   time.Time
	ClosedAt    *time.Time
}

// LandscapeProject represents a project in the CNCF Landscape
type LandscapeProject struct {
	Name        string `yaml:"name"`
	HomepageURL string `yaml:"homepage_url"`
	RepoURL     string `yaml:"repo_url"`
	Logo        string `yaml:"logo"`
	Project     string `yaml:"project"`
	Crunchbase  string `yaml:"crunchbase"`
}

// LandscapeCategory represents a category in the landscape
type LandscapeCategory struct {
	Category      string                 `yaml:"category"`
	Subcategories []LandscapeSubcategory `yaml:"subcategories"`
}

// LandscapeSubcategory represents a subcategory
type LandscapeSubcategory struct {
	Subcategory string             `yaml:"subcategory"`
	Items       []LandscapeProject `yaml:"items"`
}

// Landscape represents the full landscape.yml structure
type Landscape struct {
	Landscape []LandscapeCategory `yaml:"landscape"`
}

// MissingProject contains info about a project missing from the landscape
type MissingProject struct {
	Issue       SandboxIssue
	SuggestedEntry LandscapeProject
}

func main() {
	var (
		outputFile    string
		jsonOutput    bool
		generateYAML  bool
		dryRun        bool
		verbose       bool
	)

	flag.StringVar(&outputFile, "output", "", "Output file for missing projects report")
	flag.BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	flag.BoolVar(&generateYAML, "yaml", false, "Generate YAML entries for missing projects")
	flag.BoolVar(&dryRun, "dry-run", true, "Dry run mode (don't create PRs)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()

	ctx := context.Background()

	// Initialize GitHub client
	var client *github.Client
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	fmt.Println("🔍 Fetching GitVote passed issues from cncf/sandbox...")
	issues, err := fetchGitVotePassedIssues(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d issues with gitvote/passed label\n", len(issues))

	fmt.Println("📥 Fetching CNCF Landscape YAML...")
	landscape, err := fetchLandscape()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching landscape: %v\n", err)
		os.Exit(1)
	}

	// Extract all project names from landscape
	landscapeProjects := extractLandscapeProjects(landscape)
	fmt.Printf("   Found %d projects in landscape\n", len(landscapeProjects))

	fmt.Println("🔄 Comparing projects...")
	missing := findMissingProjects(issues, landscapeProjects, verbose)
	fmt.Printf("   Found %d projects potentially missing from landscape\n", len(missing))

	if len(missing) == 0 {
		fmt.Println("✅ All GitVote passed projects are in the landscape!")
		return
	}

	// Output results
	if jsonOutput {
		outputJSON(missing, outputFile)
	} else if generateYAML {
		outputYAMLEntries(missing, outputFile)
	} else {
		outputReport(missing, outputFile)
	}

	if !dryRun && token != "" {
		fmt.Println("🚀 Creating pull request...")
		// PR creation logic would go here
		fmt.Println("   PR creation not yet implemented - use --dry-run=false when ready")
	}
}

func fetchGitVotePassedIssues(ctx context.Context, client *github.Client) ([]SandboxIssue, error) {
	var allIssues []SandboxIssue

	opts := &github.IssueListByRepoOptions{
		Labels:      []string{gitVoteLabel},
		State:       "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, sandboxOwner, sandboxRepo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list issues: %w", err)
		}

		for _, issue := range issues {
			if issue.IsPullRequest() {
				continue
			}

			si := SandboxIssue{
				Number:      issue.GetNumber(),
				Title:       issue.GetTitle(),
				ProjectName: extractProjectName(issue.GetTitle()),
				State:       issue.GetState(),
				CreatedAt:   issue.GetCreatedAt().Time,
			}

			if issue.ClosedAt != nil {
				t := issue.ClosedAt.Time
				si.ClosedAt = &t
			}

			for _, label := range issue.Labels {
				si.Labels = append(si.Labels, label.GetName())
			}

			// Extract URLs from issue body
			body := issue.GetBody()
			si.RepoURL = extractURL(body, "repo")
			si.WebsiteURL = extractURL(body, "website")
			si.Description = extractDescription(body)

			allIssues = append(allIssues, si)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

func extractProjectName(title string) string {
	// Title format: "[Sandbox] ProjectName"
	re := regexp.MustCompile(`\[Sandbox\]\s*(.+)`)
	matches := re.FindStringSubmatch(title)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return strings.TrimSpace(title)
}

func extractURL(body, urlType string) string {
	var patterns []string

	switch urlType {
	case "repo":
		patterns = []string{
			`(?i)Project repo URL[^\n]*\n+([^\n]+github\.com[^\n\s]+)`,
			`(?i)Org repo URL[^\n]*\n+([^\n]+github\.com[^\n\s]+)`,
			`https://github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+`,
		}
	case "website":
		patterns = []string{
			`(?i)Website URL[^\n]*\n+([^\n]+)`,
			`(?i)Website[^\n]*\n+(https?://[^\n\s]+)`,
		}
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			url := strings.TrimSpace(matches[1])
			// Clean up URL
			url = strings.TrimSuffix(url, ")")
			url = strings.TrimPrefix(url, "(")
			if url != "" && url != "N/A" && url != "_No response_" {
				return url
			}
		}
	}
	return ""
}

func extractDescription(body string) string {
	// Try to extract project summary or description
	patterns := []string{
		`(?i)### Project summary\s*\n+([^\n#]+)`,
		`(?i)### Project description\s*\n+([^\n#]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			desc := strings.TrimSpace(matches[1])
			if desc != "" && desc != "_No response_" {
				// Truncate if too long
				if len(desc) > 200 {
					desc = desc[:197] + "..."
				}
				return desc
			}
		}
	}
	return ""
}

func fetchLandscape() (*Landscape, error) {
	resp, err := http.Get(landscapeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch landscape: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var landscape Landscape
	if err := yaml.Unmarshal(body, &landscape); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &landscape, nil
}

func extractLandscapeProjects(landscape *Landscape) map[string]LandscapeProject {
	projects := make(map[string]LandscapeProject)

	for _, category := range landscape.Landscape {
		for _, subcategory := range category.Subcategories {
			for _, item := range subcategory.Items {
				// Normalize name for comparison
				normalizedName := strings.ToLower(strings.TrimSpace(item.Name))
				projects[normalizedName] = item
			}
		}
	}

	return projects
}

func findMissingProjects(issues []SandboxIssue, landscapeProjects map[string]LandscapeProject, verbose bool) []MissingProject {
	var missing []MissingProject

	for _, issue := range issues {
		projectName := issue.ProjectName
		normalizedName := strings.ToLower(strings.TrimSpace(projectName))

		// Check various name variations
		found := false
		variations := []string{
			normalizedName,
			strings.ReplaceAll(normalizedName, "-", ""),
			strings.ReplaceAll(normalizedName, " ", ""),
			strings.ReplaceAll(normalizedName, "_", ""),
		}

		for _, variation := range variations {
			if _, exists := landscapeProjects[variation]; exists {
				found = true
				if verbose {
					fmt.Printf("   ✓ %s found in landscape\n", projectName)
				}
				break
			}
		}

		if !found {
			if verbose {
				fmt.Printf("   ✗ %s NOT found in landscape\n", projectName)
			}

			mp := MissingProject{
				Issue: issue,
				SuggestedEntry: LandscapeProject{
					Name:        projectName,
					HomepageURL: issue.WebsiteURL,
					RepoURL:     issue.RepoURL,
					Project:     "sandbox",
				},
			}
			missing = append(missing, mp)
		}
	}

	return missing
}

func outputReport(missing []MissingProject, outputFile string) {
	var w io.Writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return
		}
		defer f.Close()
		w = f
	}

	fmt.Fprintln(w, "# Missing Projects Report")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Generated: %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, "Found **%d** projects with `gitvote/passed` label that may be missing from the CNCF Landscape.\n\n", len(missing))

	fmt.Fprintln(w, "| # | Project | Issue | State | Repo URL | Website |")
	fmt.Fprintln(w, "|---|---------|-------|-------|----------|---------|")

	for i, mp := range missing {
		repoURL := mp.Issue.RepoURL
		if repoURL == "" {
			repoURL = "N/A"
		}
		websiteURL := mp.Issue.WebsiteURL
		if websiteURL == "" {
			websiteURL = "N/A"
		}

		fmt.Fprintf(w, "| %d | **%s** | [#%d](https://github.com/cncf/sandbox/issues/%d) | %s | %s | %s |\n",
			i+1,
			mp.Issue.ProjectName,
			mp.Issue.Number,
			mp.Issue.Number,
			mp.Issue.State,
			repoURL,
			websiteURL,
		)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "## Notes")
	fmt.Fprintln(w, "- Projects with OPEN state may still be in the onboarding process")
	fmt.Fprintln(w, "- Some projects may have different names in the landscape")
	fmt.Fprintln(w, "- Manual verification is recommended before adding entries")
}

func outputJSON(missing []MissingProject, outputFile string) {
	var w io.Writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return
		}
		defer f.Close()
		w = f
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(missing); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
	}
}

func outputYAMLEntries(missing []MissingProject, outputFile string) {
	var w io.Writer = os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return
		}
		defer f.Close()
		w = f
	}

	fmt.Fprintln(w, "# Suggested landscape.yml entries for missing projects")
	fmt.Fprintln(w, "# These entries need manual review and completion (logo, crunchbase, etc.)")
	fmt.Fprintln(w, "")

	for _, mp := range missing {
		fmt.Fprintf(w, "# %s (Issue #%d)\n", mp.Issue.ProjectName, mp.Issue.Number)
		fmt.Fprintln(w, "- item:")

		entry := mp.SuggestedEntry
		fmt.Fprintf(w, "    name: %s\n", entry.Name)

		if entry.HomepageURL != "" {
			fmt.Fprintf(w, "    homepage_url: %s\n", entry.HomepageURL)
		} else {
			fmt.Fprintln(w, "    homepage_url: # TODO: Add homepage URL")
		}

		if entry.RepoURL != "" {
			fmt.Fprintf(w, "    repo_url: %s\n", entry.RepoURL)
		} else {
			fmt.Fprintln(w, "    repo_url: # TODO: Add repo URL")
		}

		fmt.Fprintln(w, "    logo: # TODO: Add logo filename")
		fmt.Fprintln(w, "    project: sandbox")
		fmt.Fprintln(w, "    crunchbase: # TODO: Add Crunchbase URL or use https://www.crunchbase.com/organization/cloud-native-computing-foundation")

		if mp.Issue.Description != "" {
			fmt.Fprintf(w, "    # Description: %s\n", mp.Issue.Description)
		}

		fmt.Fprintln(w, "")
	}
}
