package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"projects"
)

func main() {
	var (
		name          = flag.String("name", "", "Project display name to search for (e.g., 'Kubernetes')")
		githubOrg     = flag.String("github-org", "", "GitHub organization (e.g., 'kubernetes')")
		githubRepo    = flag.String("github-repo", "", "Primary GitHub repository name (e.g., 'kubernetes')")
		githubToken   = flag.String("github-token", "", "GitHub personal access token (or set GITHUB_TOKEN env)")
		outputDir     = flag.String("output-dir", ".", "Directory to write scaffold output")
		skipLandscape = flag.Bool("skip-landscape", false, "Skip CNCF landscape YAML lookup")
		skipCLO       = flag.Bool("skip-clomonitor", false, "Skip CLOMonitor API lookup")
		skipGH        = flag.Bool("skip-github", false, "Skip GitHub API lookup")
		dryRun        = flag.Bool("dry-run", false, "Print generated YAML to stdout without writing files")
		force         = flag.Bool("force", false, "Overwrite auxiliary files (never overwrites project.yaml or maintainers.yaml)")
	)
	flag.Parse()

	// Validate required inputs
	if *name == "" && *githubOrg == "" {
		fmt.Fprintln(os.Stderr, "Error: at least one of -name or -github-org is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	// Normalize --github-org and --github-repo: accept full GitHub URLs or plain slugs.
	// e.g. https://github.com/meshery/meshery → org="meshery", repo="meshery"
	var org, repo string
	if *githubOrg != "" {
		var impliedRepo string
		var err error
		org, impliedRepo, err = projects.ParseGitHubURL(*githubOrg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: --github-org: %v\n", err)
			os.Exit(1)
		}
		// If the org URL contained a repo segment and --github-repo wasn't set, use it.
		if impliedRepo != "" && *githubRepo == "" {
			repo = impliedRepo
		}
	}
	if *githubRepo != "" {
		var repoOrg string
		var err error
		repoOrg, repo, err = projects.ParseGitHubURL(*githubRepo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: --github-repo: %v\n", err)
			os.Exit(1)
		}
		// URL had only one segment (github.com/repo) — the org segment is the repo name.
		if repo == "" {
			repo = repoOrg
		}
	}

	// Derive defaults
	projectName := *name
	if projectName == "" {
		projectName = org
	}
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

	client := &http.Client{Timeout: projects.DefaultHTTPTimeout}

	fmt.Fprintf(os.Stderr, "Bootstrapping project: %s (slug: %s)\n", projectName, slug)

	// Phase 1: Fetch from CNCF Landscape
	var landscapeData *projects.LandscapeData
	if !*skipLandscape {
		fmt.Fprintf(os.Stderr, "  Fetching from CNCF landscape...\n")
		var err error
		landscapeData, err = projects.FetchFromLandscape(projectName, client, "")
		if err != nil {
			log.Printf("  Warning: Landscape fetch failed: %v", err)
		} else if landscapeData != nil {
			fmt.Fprintf(os.Stderr, "  Found in landscape: %s (maturity: %s, category: %s / %s)\n",
				landscapeData.Name, landscapeData.Maturity, landscapeData.Category, landscapeData.Subcategory)
		} else {
			fmt.Fprintf(os.Stderr, "  Not found in landscape\n")
		}
	}

	// Phase 2: Fetch from CLOMonitor
	// Build a de-duplicated list of name variants to try, most specific first:
	//   1. Landscape display name
	//   2. User-supplied project name
	//   3. GitHub org name
	var cloProject *projects.CLOMonitorProject
	if !*skipCLO {
		fmt.Fprintf(os.Stderr, "  Fetching from CLOMonitor...\n")

		seen := map[string]bool{}
		var cloSearchNames []string
		landscapeName := ""
		if landscapeData != nil {
			landscapeName = landscapeData.Name
		}
		for _, n := range []string{landscapeName, projectName, org} {
			if n != "" && !seen[n] {
				seen[n] = true
				cloSearchNames = append(cloSearchNames, n)
			}
		}

		for _, sn := range cloSearchNames {
			var err error
			cloProject, err = projects.FetchFromCLOMonitor(sn, client, "")
			if err != nil {
				log.Printf("  Warning: CLOMonitor fetch failed for %q: %v", sn, err)
				continue
			}
			if cloProject != nil {
				break
			}
		}

		if cloProject != nil {
			fmt.Fprintf(os.Stderr, "  Found on CLOMonitor: %s (maturity: %s, score: %.0f)\n",
				cloProject.DisplayName, cloProject.Maturity, cloProject.Score.Global)
		} else {
			fmt.Fprintf(os.Stderr, "  Not found on CLOMonitor\n")
		}
	}

	// Phase 3: Fetch from GitHub
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

	// Phase 3.5: Search for TOC/sandbox onboarding issue (if no URL from landscape)
	// Note: Works without a token (unauthenticated, lower rate limit) but
	// a GITHUB_TOKEN is recommended to avoid hitting rate limits.
	var tocURL string
	if !*skipGH && (landscapeData == nil || landscapeData.AnnualReviewURL == "") {
		fmt.Fprintf(os.Stderr, "  Searching for TOC/sandbox onboarding issue...\n")
		var err error
		tocURL, err = projects.SearchTOCIssues(projectName, org, token, client, "")
		if err != nil {
			log.Printf("  Warning: TOC issue search failed: %v", err)
		} else if tocURL != "" {
			fmt.Fprintf(os.Stderr, "  Found TOC/onboarding issue: %s\n", tocURL)
		} else {
			fmt.Fprintf(os.Stderr, "  No TOC/onboarding issue found\n")
		}
	}

	// Phase 4: Merge data
	fmt.Fprintf(os.Stderr, "  Merging data sources...\n")
	result := projects.MergeBootstrapData(slug, landscapeData, cloProject, ghData)

	// Apply TOC issue URL from search if not already set by landscape
	if result.TOCIssueURL == "" && tocURL != "" {
		result.TOCIssueURL = tocURL
		result.Sources["toc_issue_url"] = "github_search"
		// Remove the TOC issue TODO since we found one
		var filteredTODOs []string
		for _, todo := range result.TODOs {
			if todo != "Add maturity_log entry with TOC issue URL" {
				filteredTODOs = append(filteredTODOs, todo)
			}
		}
		result.TODOs = filteredTODOs
	}

	// Ensure org/repo are set even if GitHub fetch was skipped
	if result.GitHubOrg == "" && org != "" {
		result.GitHubOrg = org
	}
	if result.GitHubRepo == "" && repo != "" {
		result.GitHubRepo = repo
	}

	// Phase 5: Generate output
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
		var opts []projects.WriteScaffoldOption
		if *force {
			opts = append(opts, projects.WithForce())
		}
		if err := projects.WriteScaffold(*outputDir, result, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\nScaffold written to %s:\n", *outputDir)
		fmt.Fprintf(os.Stderr, "  - project.yaml\n")
		fmt.Fprintf(os.Stderr, "  - maintainers.yaml\n")
		fmt.Fprintf(os.Stderr, "  - README.md\n")
		if result.SecurityPolicyURL == "" {
			fmt.Fprintf(os.Stderr, "  - SECURITY.md\n")
		} else {
			fmt.Fprintf(os.Stderr, "  - SECURITY.md (skipped: using %s)\n", result.SecurityPolicyURL)
		}
		fmt.Fprintf(os.Stderr, "  - CODEOWNERS\n")
		fmt.Fprintf(os.Stderr, "  - .gitignore\n")
		fmt.Fprintf(os.Stderr, "  - .github/workflows/validate.yaml\n")
		fmt.Fprintf(os.Stderr, "  - .github/workflows/update-landscape.yml\n")

		// Report discovered file URLs
		if result.SecurityPolicyURL != "" || result.ContributingURL != "" || result.CodeOfConductURL != "" || result.LicenseURL != "" {
			fmt.Fprintln(os.Stderr, "\nDiscovered existing files:")
			if result.SecurityPolicyURL != "" {
				fmt.Fprintf(os.Stderr, "  SECURITY.md: %s\n", result.SecurityPolicyURL)
			}
			if result.ContributingURL != "" {
				fmt.Fprintf(os.Stderr, "  CONTRIBUTING.md: %s\n", result.ContributingURL)
			}
			if result.CodeOfConductURL != "" {
				fmt.Fprintf(os.Stderr, "  CODE_OF_CONDUCT: %s\n", result.CodeOfConductURL)
			}
			if result.LicenseURL != "" {
				fmt.Fprintf(os.Stderr, "  LICENSE: %s\n", result.LicenseURL)
			}
		}
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
