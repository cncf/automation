package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"projects"
)

func main() {
	var (
		name        = flag.String("name", "", "Project display name to search for (e.g., 'Kubernetes')")
		githubOrg   = flag.String("github-org", "", "GitHub organization (e.g., 'kubernetes')")
		githubRepo  = flag.String("github-repo", "", "Primary GitHub repository name (e.g., 'kubernetes')")
		githubToken = flag.String("github-token", "", "GitHub personal access token (or set GITHUB_TOKEN env)")
		outputDir   = flag.String("output-dir", ".", "Directory to write scaffold output")
		skipCLO     = flag.Bool("skip-clomonitor", false, "Skip CLOMonitor API lookup")
		skipGH      = flag.Bool("skip-github", false, "Skip GitHub API lookup")
		dryRun      = flag.Bool("dry-run", false, "Print generated YAML to stdout without writing files")
	)
	flag.Parse()

	// Validate required inputs
	if *name == "" && *githubOrg == "" {
		fmt.Fprintln(os.Stderr, "Error: at least one of -name or -github-org is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	// Derive defaults
	projectName := *name
	if projectName == "" {
		projectName = *githubOrg
	}

	org := *githubOrg
	repo := *githubRepo
	if repo == "" && org != "" {
		repo = org // Common pattern: org name == primary repo name
	}

	// Slug: lowercase, hyphenated
	slug := strings.ToLower(strings.ReplaceAll(projectName, " ", "-"))
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	// Clean up multiple consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	// GitHub token from env if not provided via flag
	token := *githubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Fprintf(os.Stderr, "Bootstrapping project: %s (slug: %s)\n", projectName, slug)

	// Phase 1: Fetch from CLOMonitor
	var cloProject *projects.CLOMonitorProject
	if !*skipCLO {
		fmt.Fprintf(os.Stderr, "  Fetching from CLOMonitor...\n")
		var err error
		cloProject, err = projects.FetchFromCLOMonitor(projectName, client, "")
		if err != nil {
			log.Printf("  Warning: CLOMonitor fetch failed: %v", err)
		} else if cloProject != nil {
			fmt.Fprintf(os.Stderr, "  Found on CLOMonitor: %s (maturity: %s, score: %.0f)\n",
				cloProject.DisplayName, cloProject.Maturity, cloProject.Score.Global)
		} else {
			fmt.Fprintf(os.Stderr, "  Not found on CLOMonitor\n")
		}
	}

	// Phase 2: Fetch from GitHub
	var ghData *projects.GitHubData
	if !*skipGH && org != "" {
		fmt.Fprintf(os.Stderr, "  Fetching from GitHub: %s/%s...\n", org, repo)
		var err error
		ghData, err = projects.FetchFromGitHub(org, repo, token, client, "")
		if err != nil {
			log.Printf("  Warning: GitHub fetch failed: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "  Found on GitHub: %s\n", ghData.Repo.FullName)
			if len(ghData.Maintainers) > 0 {
				fmt.Fprintf(os.Stderr, "  Discovered %d maintainer(s) from governance files\n", len(ghData.Maintainers))
			}
		}
	}

	// Phase 3: Merge data
	fmt.Fprintf(os.Stderr, "  Merging data sources...\n")
	result := projects.MergeBootstrapData(slug, nil, cloProject, ghData)

	// Phase 4: Generate output
	if *dryRun {
		fmt.Fprintln(os.Stderr, "\n--- project.yaml ---")
		projectYAML, err := projects.GenerateProjectYAML(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating project.yaml: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(projectYAML))

		fmt.Fprintln(os.Stderr, "--- maintainers.yaml ---")
		maintainersYAML, err := projects.GenerateMaintainersYAML(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating maintainers.yaml: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(maintainersYAML))
	} else {
		fmt.Fprintf(os.Stderr, "  Writing scaffold to %s...\n", *outputDir)
		if err := projects.WriteScaffold(*outputDir, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\nScaffold written to %s:\n", *outputDir)
		fmt.Fprintf(os.Stderr, "  - project.yaml\n")
		fmt.Fprintf(os.Stderr, "  - maintainers.yaml\n")
		fmt.Fprintf(os.Stderr, "  - .github/workflows/validate.yaml\n")
	}

	// Show TODOs
	if len(result.TODOs) > 0 {
		fmt.Fprintln(os.Stderr, "\nRemaining TODOs:")
		for _, todo := range result.TODOs {
			fmt.Fprintf(os.Stderr, "  - %s\n", todo)
		}
	}

	// Show data sources
	if len(result.Sources) > 0 {
		fmt.Fprintln(os.Stderr, "\nData sources used:")
		for field, source := range result.Sources {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", field, source)
		}
	}
}
