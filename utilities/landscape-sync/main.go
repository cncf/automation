package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

const (
	landscapeURL               = "https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml"
	sandboxOwner               = "cncf"
	sandboxRepo                = "sandbox"
	gitVoteLabel               = "gitvote/passed"
	contributionAgreementLabel = "contribution-agreement/signed"
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
	Issue          SandboxIssue
	SuggestedEntry LandscapeProject
}

type LandscapeProjectIndex struct {
	ByName map[string]LandscapeProject
	ByRepo map[string]LandscapeProject
}

type repoResolver func(string) (string, error)

func main() {
	var (
		outputFile   string
		jsonOutput   bool
		generateYAML bool
		dryRun       bool
		verbose      bool
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

	fmt.Println("🔍 Fetching issues with gitvote/passed and contribution-agreement/signed labels from cncf/sandbox...")
	issues, err := fetchGitVotePassedIssues(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Found %d issues with required labels\n", len(issues))

	fmt.Println("📥 Fetching CNCF Landscape YAML...")
	landscape, err := fetchLandscape()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching landscape: %v\n", err)
		os.Exit(1)
	}

	// Extract all project names from landscape
	landscapeProjects := extractLandscapeProjects(landscape)
	fmt.Printf("   Found %d projects in landscape\n", len(landscapeProjects.ByName))

	fmt.Println("🔄 Comparing projects...")
	resolveRepo := func(repoURL string) (string, error) {
		return resolveGitHubRepo(ctx, client, repoURL)
	}
	missing := findMissingProjects(issues, landscapeProjects, resolveRepo, verbose)
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
		Labels:      []string{gitVoteLabel, contributionAgreementLabel},
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

func extractLandscapeProjects(landscape *Landscape) LandscapeProjectIndex {
	projects := LandscapeProjectIndex{
		ByName: make(map[string]LandscapeProject),
		ByRepo: make(map[string]LandscapeProject),
	}

	for _, category := range landscape.Landscape {
		for _, subcategory := range category.Subcategories {
			for _, item := range subcategory.Items {
				projects.ByName[canonicalProjectName(item.Name)] = item
				if repo, ok := parseGitHubRepo(item.RepoURL); ok {
					projects.ByRepo[repo] = item
				}
			}
		}
	}

	return projects
}

func canonicalProjectName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.NewReplacer("-", "", "_", "", " ", "").Replace(name)
}

func parseGitHubRepo(repoURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(repoURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}

	repo := strings.TrimSuffix(parts[1], ".git")
	return strings.ToLower(parts[0] + "/" + repo), true
}

func resolveGitHubRepo(ctx context.Context, client *github.Client, repoURL string) (string, error) {
	repoPath, ok := parseGitHubRepo(repoURL)
	if !ok {
		return "", fmt.Errorf("invalid GitHub repository URL %q", repoURL)
	}

	parts := strings.SplitN(repoPath, "/", 2)
	repository, _, err := client.Repositories.Get(ctx, parts[0], parts[1])
	if err != nil {
		return "", fmt.Errorf("resolve GitHub repository %s: %w", repoPath, err)
	}

	return strings.ToLower(repository.GetFullName()), nil
}

func findMissingProjects(issues []SandboxIssue, landscapeProjects LandscapeProjectIndex, resolveRepo repoResolver, verbose bool) []MissingProject {
	var missing []MissingProject
	logf := func(format string, args ...any) {
		if verbose {
			fmt.Printf(format, args...)
		}
	}

	for _, issue := range issues {
		projectName := issue.ProjectName
		_, found := landscapeProjects.ByName[canonicalProjectName(projectName)]

		if !found {
			if repo, ok := parseGitHubRepo(issue.RepoURL); ok {
				_, found = landscapeProjects.ByRepo[repo]
			}
		}

		if !found && resolveRepo != nil && issue.RepoURL != "" {
			repo, err := resolveRepo(issue.RepoURL)
			if err != nil {
				logf("   ! Could not resolve repository for %s: %v\n", projectName, err)
			}
			if err == nil {
				_, found = landscapeProjects.ByRepo[strings.ToLower(repo)]
			}
		}

		if found {
			logf("   ✓ %s found in landscape\n", projectName)
			continue
		}

		logf("   ✗ %s NOT found in landscape\n", projectName)

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
	fmt.Fprintf(w, "Found **%d** projects with `gitvote/passed` and `contribution-agreement/signed` labels that may be missing from the CNCF Landscape.\n\n", len(missing))

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
