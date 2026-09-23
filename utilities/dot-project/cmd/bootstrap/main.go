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
		name           = flag.String("name", "", "Project display name to search for (e.g., 'Kubernetes')")
		githubOrg      = flag.String("github-org", "", "GitHub organization (e.g., 'kubernetes')")
		githubRepo     = flag.String("github-repo", "", "Primary GitHub repository name (e.g., 'kubernetes')")
		githubToken    = flag.String("github-token", "", "GitHub personal access token (or set GITHUB_TOKEN env)")
		outputDir      = flag.String("output-dir", ".", "Directory to write scaffold output")
		layout         = flag.String("layout", "auto", "Repository layout: auto (detect from the landscape), single, or multi")
		skipLandscape  = flag.Bool("skip-landscape", false, "Skip CNCF landscape YAML lookup")
		skipCLO        = flag.Bool("skip-clomonitor", false, "Skip CLOMonitor API lookup")
		skipGH         = flag.Bool("skip-github", false, "Skip GitHub API lookup")
		maintainersCSV = flag.String("maintainers-csv", "", "Optional path to a local project-maintainers.csv (default: fetch from cncf/foundation)")
		dryRun         = flag.Bool("dry-run", false, "Print generated YAML to stdout without writing files")
		force          = flag.Bool("force", false, "Overwrite auxiliary files (never overwrites project.yaml or maintainers.yaml)")
		envFile        = flag.String("env-file", ".env", "Path to a .env file to load (e.g. GITHUB_TOKEN=...); real env vars take precedence")
	)
	flag.Parse()

	switch *layout {
	case "auto", "single", "multi":
	default:
		fmt.Fprintf(os.Stderr, "Error: -layout must be auto, single, or multi (got %q)\n", *layout)
		os.Exit(1)
	}

	// Load a .env file (if present) before resolving the token below. Real
	// environment variables always take precedence over file values.
	if applied, err := projects.LoadDotEnv(*envFile); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: could not read %s: %v\n", *envFile, err)
	} else if len(applied) > 0 {
		fmt.Fprintf(os.Stderr, "  Loaded %d variable(s) from %s\n", len(applied), *envFile)
	}

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
	slug := projects.Slugify(projectName)

	// GitHub token from env if not provided via flag (GITHUB_TOKEN, then GH_TOKEN). The value
	// may come from the shell or from the .env file loaded above.
	token := *githubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		fmt.Fprintf(os.Stderr, "  Note: no GitHub token set (-github-token / GITHUB_TOKEN / GH_TOKEN / %s).\n", *envFile)
		fmt.Fprintln(os.Stderr, "        Unauthenticated GitHub API requests are rate-limited (HTTP 403 once exceeded).")
		fmt.Fprintln(os.Stderr, "        Provide a token via any of:")
		fmt.Fprintf(os.Stderr, "          - an env file (%s) containing:  GITHUB_TOKEN=ghp_xxx\n", *envFile)
		fmt.Fprintln(os.Stderr, "          - the environment:         GITHUB_TOKEN=ghp_xxx go run ./cmd/bootstrap ...")
		fmt.Fprintln(os.Stderr, "          - the flag:                -github-token ghp_xxx")
	}

	client := &http.Client{Timeout: projects.DefaultHTTPTimeout}


	entries, multi := detectOrgProjects(*layout, org, projectName, slug, repo, client)

	// A repository that holds several CNCF projects needs one metadata
	// directory per project, so run the whole pipeline once per project.
	if multi {
		if err := runMultiProject(entries, org, *outputDir, *dryRun, *force, pipelineInputs{
			token:          token,
			client:         client,
			skipLandscape:  *skipLandscape,
			skipCLO:        *skipCLO,
			skipGH:         *skipGH,
			maintainersCSV: *maintainersCSV,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	result, suggestions := buildResult(pipelineInputs{
		projectName:    projectName,
		org:            org,
		repo:           repo,
		slug:           slug,
		token:          token,
		client:         client,
		skipLandscape:  *skipLandscape,
		skipCLO:        *skipCLO,
		skipGH:         *skipGH,
		maintainersCSV: *maintainersCSV,
	})

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

	if section := projects.BuildSuggestionsSection(suggestions); section != "" {
		fmt.Fprintf(os.Stderr, "\nMaintainer suggestions (from org governance files, not yet in the CSV):\n\n%s\n", section)
		if !*dryRun {
			if path, err := projects.WriteSuggestionsFile(*outputDir, suggestions); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write suggestions file: %v\n", err)
			} else if path != "" {
				fmt.Fprintf(os.Stderr, "  Wrote maintainer suggestions to %s\n", path)
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

// removeTODO returns todos without any entries equal to target.
func removeTODO(todos []string, target string) []string {
	var out []string
	for _, t := range todos {
		if t != target {
			out = append(out, t)
		}
	}
	return out
}

// pipelineInputs carries everything the per-project bootstrap pipeline needs.
// A multi-project organization runs the same pipeline once per project.
type pipelineInputs struct {
	projectName    string
	org            string
	repo           string
	slug           string
	token          string
	client         *http.Client
	skipLandscape  bool
	skipCLO        bool
	skipGH         bool
	maintainersCSV string
}

// buildResult runs the full discovery pipeline for one project.
func buildResult(in pipelineInputs) (*projects.BootstrapResult, []projects.MaintainerSuggestion) {
	projectName, org, repo, slug := in.projectName, in.org, in.repo, in.slug
	token, client := in.token, in.client
	skipLandscape, skipCLO, skipGH := &in.skipLandscape, &in.skipCLO, &in.skipGH
	maintainersCSV := &in.maintainersCSV

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
		}
	}

	// Phase 3.5: Search for TOC/sandbox onboarding issue (if no URL from landscape)
	// Note: Works without a token (unauthenticated, lower rate limit) but
	// a GITHUB_TOKEN is recommended to avoid hitting rate limits.
	var tocURL string
	if !*skipGH && (landscapeData == nil || landscapeData.AnnualReviewURL == "") {
		fmt.Fprintf(os.Stderr, "  Searching for TOC/sandbox onboarding issue...\n")

		// Build a de-duplicated list of name variants to try, most specific first:
		//   1. Landscape display name
		//   2. User-supplied project name
		//   3. GitHub org name
		seen := map[string]bool{}
		var tocSearchNames []string
		for _, n := range []string{
			func() string {
				if landscapeData != nil {
					return landscapeData.Name
				}
				return ""
			}(),
			projectName,
			org,
		} {
			if n != "" && !seen[n] {
				seen[n] = true
				tocSearchNames = append(tocSearchNames, n)
			}
		}

		var tocErr error
		for _, sn := range tocSearchNames {
			tocURL, tocErr = projects.SearchTOCIssues(sn, org, token, client, "")
			if tocErr != nil {
				log.Printf("  Warning: TOC issue search failed for %q: %v", sn, tocErr)
				continue
			}
			if tocURL != "" {
				break
			}
		}
		if tocURL != "" {
			fmt.Fprintf(os.Stderr, "  Found TOC/onboarding issue: %s\n", tocURL)
		} else {
			fmt.Fprintf(os.Stderr, "  No TOC/onboarding issue found\n")
		}
	}

	// Phase 3.7: Discover maintainers from the CNCF foundation maintainers CSV.
	var csvMaintainers []string
	{
		seen := map[string]bool{}
		var searchNames []string
		add := func(n string) {
			n = strings.TrimSpace(n)
			if n != "" && !seen[strings.ToLower(n)] {
				seen[strings.ToLower(n)] = true
				searchNames = append(searchNames, n)
			}
		}
		if landscapeData != nil {
			add(landscapeData.Name)
		}
		add(projectName)
		add(org)
		add(repo)
		if cloProject != nil {
			add(cloProject.DisplayName)
		}

		// Report the CSV source (local path or remote URL) for visibility; any
		// fetch/parse issue with it is surfaced in the warning logged below.
		csvSource := *maintainersCSV
		if csvSource == "" {
			csvSource = projects.DefaultFoundationMaintainersCSVURL
		}
		fmt.Fprintf(os.Stderr, "  Discovering maintainers from foundation CSV (%s)...\n", csvSource)
		blocks, err := projects.FetchFoundationMaintainers(*maintainersCSV, client)
		if err != nil {
			log.Printf("  Warning: maintainers CSV lookup failed: %v", err)
		} else {
			csvMaintainers = projects.MatchProjectMaintainers(blocks, searchNames...)
			if len(csvMaintainers) > 0 {
				fmt.Fprintf(os.Stderr, "  Discovered %d maintainer(s) from foundation CSV\n", len(csvMaintainers))
			} else {
				fmt.Fprintf(os.Stderr, "  No maintainers matched in foundation CSV for: %s\n", strings.Join(searchNames, ", "))
			}
		}
	}

	// Phase 3.8: Discover maintainer suggestions and Slack channels from org repos.
	// Maintainer suggestions are advisory only — the foundation CSV remains the
	// source of truth. Slack channels are merged in as candidates to verify.
	var suggestions []projects.MaintainerSuggestion
	var orgSlackChannels []string
	if !*skipGH && org != "" {
		csvSet := map[string]bool{}
		for _, h := range csvMaintainers {
			csvSet[strings.ToLower(h)] = true
		}
		fmt.Fprintf(os.Stderr, "  Discovering maintainer suggestions and Slack channels from org repos...\n")
		suggestions, orgSlackChannels = projects.DiscoverGovernanceSuggestions(org, repo, token, client, "", csvSet)
		if len(suggestions) > 0 {
			fmt.Fprintf(os.Stderr, "  Found %d maintainer suggestion(s) not yet in the CSV\n", len(suggestions))
		} else {
			fmt.Fprintf(os.Stderr, "  No additional maintainer suggestions found\n")
		}
		if len(orgSlackChannels) > 0 {
			fmt.Fprintf(os.Stderr, "  Found %d Slack channel(s) across org repos\n", len(orgSlackChannels))
		}
	}

	// Phase 4: Merge data
	fmt.Fprintf(os.Stderr, "  Merging data sources...\n")
	result := projects.MergeBootstrapData(slug, landscapeData, cloProject, ghData)

	// Merge org-wide discovered Slack channels as additional candidates to verify.
	if len(orgSlackChannels) > 0 {
		projects.AddDiscoveredSlackChannels(result, orgSlackChannels)
		result.TODOs = removeTODO(result.TODOs, "Set slack_channels")
	}

	// Maintainers come exclusively from the foundation CSV. On no match, leave
	// the roster empty and flag a TODO.
	result.Maintainers = csvMaintainers
	result.TODOs = removeTODO(result.TODOs, "Add maintainer GitHub handles")
	if len(csvMaintainers) > 0 {
		result.Sources["maintainers"] = "foundation-csv"
		if result.ProjectLead == "" {
			result.ProjectLead = csvMaintainers[0]
			result.Sources["project_lead"] = "foundation-csv"
			result.TODOs = removeTODO(result.TODOs, "Set project_lead GitHub handle")
		}
	} else {
		result.TODOs = append(result.TODOs,
			"No maintainers found in cncf/foundation project-maintainers.csv — add maintainer handles manually")
	}

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

	return result, suggestions
}

// detectOrgProjects asks the landscape whether org hosts more than one CNCF
// project. The landscape is the only authority we have: project maintainers
// know their own structure, but CNCF tooling cannot infer it from a repository.
//
// A project the community has split but CNCF has not yet ratified has no
// landscape entry of its own, so it correctly stays part of the existing
// project until that entry exists.
// detectOrgProjects decides whether the repository should hold several CNCF
// projects. The CNCF landscape is the only reliable source for this: nothing in
// a GitHub organization itself says "these two repositories are separately
// accepted projects". The returned bool reports whether the multi-project
// layout should be generated, which is not simply len(entries) > 1 because
// -layout multi is the escape hatch for a project whose split the CNCF has not
// recorded in the landscape yet.
func detectOrgProjects(layout, org, projectName, slug, repo string, client *http.Client) ([]projects.OrgProject, bool) {
	if layout == "single" {
		return nil, false
	}

	var entries []projects.OrgProject
	if org != "" {
		fmt.Fprintf(os.Stderr, "  Checking whether %s hosts multiple CNCF projects...\n", org)
		found, err := projects.FindOrgProjects(org, client, "")
		if err != nil {
			log.Printf("  Warning: could not scan the landscape for %s: %v", org, err)
			if layout != "multi" {
				return nil, false
			}
		}
		entries = found
		for _, e := range entries {
			fmt.Fprintf(os.Stderr, "    - %s (%s, %s)\n", e.Name, e.Slug, e.Maturity)
		}
	}

	if layout == "multi" {
		// The landscape may not list the split yet, so fall back to the
		// project given on the command line and let the maintainers add the
		// remaining directories by hand.
		if len(entries) == 0 {
			log.Printf("  Warning: -layout multi was requested but the landscape lists no CNCF project for %q;", org)
			log.Printf("           scaffolding %q only — add the remaining projects to %s by hand", slug, projects.OrgFileName)
			entries = []projects.OrgProject{{Name: projectName, Slug: slug, Repo: repo}}
		}
		return entries, true
	}

	if len(entries) < 2 {
		return nil, false
	}
	fmt.Fprintf(os.Stderr, "  %s hosts %d CNCF projects; generating the multi-project layout\n", org, len(entries))
	return entries, true
}

// runMultiProject bootstraps every CNCF project in the organization into its
// own directory, alongside a single org.yaml index and one shared set of
// repository-level files.
func runMultiProject(entries []projects.OrgProject, org, outputDir string, dryRun, force bool, base pipelineInputs) error {
	results := make([]*projects.BootstrapResult, 0, len(entries))
	allSuggestions := make(map[string][]projects.MaintainerSuggestion)

	for _, entry := range entries {
		in := base
		in.projectName = entry.Name
		in.org = org
		in.repo = entry.Repo
		in.slug = entry.Slug
		if in.repo == "" {
			in.repo = entry.Slug
		}

		fmt.Fprintln(os.Stderr)
		result, suggestions := buildResult(in)

		// The directory name is the routing key every tool uses, so it has to
		// agree with the slug inside project.yaml.
		result.Slug = entry.Slug

		results = append(results, result)
		if len(suggestions) > 0 {
			allSuggestions[entry.Slug] = suggestions
		}
	}

	if dryRun {
		orgYAML, err := projects.GenerateOrgYAML(org, entries)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "\n--- %s ---\n", projects.OrgFileName)
		fmt.Println(string(orgYAML))

		for i, entry := range entries {
			projectYAML, err := projects.GenerateProjectYAML(results[i])
			if err != nil {
				return fmt.Errorf("generating %s/%s: %w", entry.Slug, projects.ProjectFileName, err)
			}
			fmt.Fprintf(os.Stderr, "--- %s/%s ---\n", entry.Slug, projects.ProjectFileName)
			fmt.Println(string(projectYAML))

			maintainersYAML, err := projects.GenerateMaintainersYAML(results[i])
			if err != nil {
				return fmt.Errorf("generating %s/%s: %w", entry.Slug, projects.MaintainersFileName, err)
			}
			fmt.Fprintf(os.Stderr, "--- %s/%s ---\n", entry.Slug, projects.MaintainersFileName)
			fmt.Println(string(maintainersYAML))
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n  Writing multi-project scaffold to %s...\n", outputDir)
	var opts []projects.WriteScaffoldOption
	if force {
		opts = append(opts, projects.WithForce())
	}
	if err := projects.WriteMultiScaffold(outputDir, org, entries, results, opts...); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nScaffold written to %s:\n", outputDir)
	fmt.Fprintf(os.Stderr, "  - %s\n", projects.OrgFileName)
	for _, entry := range entries {
		fmt.Fprintf(os.Stderr, "  - %s/%s\n", entry.Slug, projects.ProjectFileName)
		fmt.Fprintf(os.Stderr, "  - %s/%s\n", entry.Slug, projects.MaintainersFileName)
	}
	fmt.Fprintf(os.Stderr, "  - README.md\n")
	fmt.Fprintf(os.Stderr, "  - SECURITY.md\n")
	fmt.Fprintf(os.Stderr, "  - CODEOWNERS\n")
	fmt.Fprintf(os.Stderr, "  - .gitignore\n")
	fmt.Fprintf(os.Stderr, "  - .github/workflows/validate.yaml\n")
	fmt.Fprintf(os.Stderr, "  - .github/workflows/update-landscape.yml\n")

	for slug, suggestions := range allSuggestions {
		if section := projects.BuildSuggestionsSection(suggestions); section != "" {
			fmt.Fprintf(os.Stderr, "\nMaintainer suggestions for %s (from org governance files, not yet in the CSV):\n\n%s\n", slug, section)
		}
	}

	fmt.Fprintf(os.Stderr, "\nNext steps:\n")
	fmt.Fprintf(os.Stderr, "  - Review each project's maintainers.yaml: the same organization often has\n")
	fmt.Fprintf(os.Stderr, "    shared governance (a steering committee) and project-specific teams.\n")
	fmt.Fprintf(os.Stderr, "  - A team name may appear in several projects. GitHub teams are org-scoped,\n")
	fmt.Fprintf(os.Stderr, "    so its members are the union of those definitions.\n")

	return nil
}
