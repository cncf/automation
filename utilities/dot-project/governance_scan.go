package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// suggestionsFileName is the gitignored artifact (under .cache/) that holds the
// rendered "Suggested maintainers" section. It is consumed by provision.sh to
// populate the onboarding issue.
const suggestionsFileName = "maintainer-suggestions.md"

// DiscoverGovernanceSuggestions scans an org's repositories in a single shared
// pass and returns two things:
//  1. maintainer handle suggestions found in governance files
//     (CODEOWNERS / OWNERS / MAINTAINERS), excluding handles already in csvHandles;
//  2. CNCF Slack channel names referenced in each repo's README / CONTRIBUTING /
//     COMMUNITY files.
func DiscoverGovernanceSuggestions(org, primaryRepo, token string, client *http.Client, baseURL string, csvHandles map[string]bool) ([]MaintainerSuggestion, []string) {
	if baseURL == "" {
		baseURL = defaultGitHubAPIURL
	}
	doGet := func(path string) (*http.Response, error) {
		req, err := http.NewRequest("GET", baseURL+path, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "token "+token)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		return client.Do(req)
	}

	c := newOrgScanCollector()

	ctx := context.Background()
	scanContentsForSuggestions(ctx, c, doGet, client,
		fmt.Sprintf("/repos/%s/%s/contents/", org, primaryRepo), fmt.Sprintf("%s/%s", org, primaryRepo))
	scanContentsForSuggestions(ctx, c, doGet, client,
		fmt.Sprintf("/repos/%s/%s/contents/.github", org, primaryRepo), fmt.Sprintf("%s/%s", org, primaryRepo))
	scanContentsForSuggestions(ctx, c, doGet, client,
		fmt.Sprintf("/repos/%s/.github/contents/", org), fmt.Sprintf("%s/.github", org))

	scanOrgReposForSuggestions(c, org, primaryRepo, doGet, client)

	return c.suggestions.suggestions(csvHandles), c.slack
}

// defaultOrgScanWorkers is the default number of concurrent goroutines used to
// scan repositories in an org. This keeps API usage well within GitHub's
// secondary rate limits while providing a significant speedup over sequential
// scanning.
const defaultOrgScanWorkers = 10

// orgScanCollector accumulates the results of a single shared org-wide scan
// pass: maintainer suggestions (from governance files) and Slack channels (from
// community documentation files). All methods are safe for concurrent use.
type orgScanCollector struct {
	mu          sync.Mutex
	suggestions *suggestionCollector
	slackSeen   map[string]bool
	slack       []string
}

func newOrgScanCollector() *orgScanCollector {
	return &orgScanCollector{
		suggestions: newSuggestionCollector(),
		slackSeen:   make(map[string]bool),
	}
}

// addSlack records discovered Slack channel names, deduplicating across repos
// while preserving discovery order.
func (c *orgScanCollector) addSlack(channels []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ch := range channels {
		if ch == "" || c.slackSeen[ch] {
			continue
		}
		c.slackSeen[ch] = true
		c.slack = append(c.slack, ch)
	}
}

// addGovernance records governance file handles into the suggestion collector.
func (c *orgScanCollector) addGovernance(filename, content, sourceRepo string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	addGovernanceFileToCollector(c.suggestions, filename, content, sourceRepo)
}

// isCommunityDocFile reports whether name is a README / CONTRIBUTING / COMMUNITY
// document (any extension), matched case-insensitively. These files are scanned
// for CNCF Slack channel references.
func isCommunityDocFile(name string) bool {
	base := strings.ToUpper(name)
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	switch base {
	case "README", "CONTRIBUTING", "COMMUNITY":
		return true
	}
	return false
}

func isGovernanceMaintainerFile(name string) bool {
	switch strings.ToUpper(name) {
	case "CODEOWNERS", "OWNERS", "MAINTAINERS", "MAINTAINERS.MD":
		return true
	}
	return false
}

// ownersAliasSuffixes are common suffixes of OWNERS_ALIASES group names (which
// are referenced in OWNERS approvers/reviewers lists but are not real GitHub
// usernames). Handles ending in these are skipped.
var ownersAliasSuffixes = []string{"-approvers", "-reviewers", "-admins", "-maintainers", "-owners", "-leads"}

// isOwnersAlias reports whether name looks like an OWNERS alias group rather
// than an individual GitHub handle.
func isOwnersAlias(name string) bool {
	n := strings.ToLower(name)
	for _, suf := range ownersAliasSuffixes {
		if strings.HasSuffix(n, suf) {
			return true
		}
	}
	return false
}

// addGovernanceFileToCollector parses a governance file's content and records
// its handles with the appropriate role and source (e.g. "org/repo:MAINTAINERS").
func addGovernanceFileToCollector(c *suggestionCollector, filename, content, sourceRepo string) {
	source := sourceRepo + ":" + filename
	switch {
	case strings.EqualFold(filename, "CODEOWNERS"):
		for _, h := range parseCodeowners(content) {
			c.add(h, roleCodeowner, source)
		}
	case strings.EqualFold(filename, "OWNERS"):
		approvers, reviewers := parseOwnersFile(content)
		for _, h := range approvers {
			if isOwnersAlias(h) {
				continue
			}
			c.add(h, roleApprover, source)
		}
		for _, h := range reviewers {
			if isOwnersAlias(h) {
				continue
			}
			c.add(h, roleReviewer, source)
		}
	case strings.EqualFold(filename, "MAINTAINERS"), strings.EqualFold(filename, "MAINTAINERS.md"):
		for _, h := range parseMaintainersFile(content) {
			c.add(h, roleMaintainer, source)
		}
	}
}

// scanContentsForSuggestions lists a contents path and, in a single pass, feeds
// any governance files into the maintainer collector and extracts Slack channel
// references from README / CONTRIBUTING / COMMUNITY files.
//
// It respects ctx cancellation and returns true if a rate-limit (403) was hit,
// signalling the caller to stop further requests.
func scanContentsForSuggestions(ctx context.Context, c *orgScanCollector, doGet func(string) (*http.Response, error), client *http.Client, contentPath, sourceRepo string) (rateLimited bool) {
	if ctx.Err() != nil {
		return false
	}
	resp, err := doGet(contentPath)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			rateLimited = resp.StatusCode == http.StatusForbidden
			resp.Body.Close()
		}
		return rateLimited
	}
	var entries []GitHubContentEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		resp.Body.Close()
		return false
	}
	resp.Body.Close()

	for _, entry := range entries {
		if ctx.Err() != nil {
			return false
		}
		if entry.Type != "file" || entry.DownloadURL == "" {
			continue
		}
		switch {
		case isGovernanceMaintainerFile(entry.Name):
			content, err := fetchFileContent(client, entry.DownloadURL)
			if err == nil && content != "" {
				c.addGovernance(entry.Name, content, sourceRepo)
			}
		case isCommunityDocFile(entry.Name):
			content, err := fetchFileContent(client, entry.DownloadURL)
			if err == nil && content != "" {
				c.addSlack(extractSlackChannels(content))
			}
		}
	}
	return false
}

// repoEntry is a lightweight struct for repos queued for scanning.
type repoEntry struct {
	Name string
}

// isGHSARepo matches GitHub Security Advisory temporary fork repos.
// These have names like "project-ghsa-xxxx-xxxx-xxxx" and are auto-created
// private forks that mirror the parent's CODEOWNERS but aren't real project repos.
func isGHSARepo(name string) bool {
	return strings.Contains(strings.ToLower(name), "-ghsa-")
}

// isGitHubPagesRepo matches GitHub Pages repos (e.g. "org.github.io").
// These are static documentation/blog sites and won't contain meaningful
// governance files distinct from the main project.
func isGitHubPagesRepo(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".github.io")
}

// scanOrgReposForSuggestions lists every repo in the org and scans each repo
// root for governance files and community docs, skipping the primary repo and
// the org .github repo (already scanned by the caller).
//
// Repos are scanned concurrently using a bounded worker pool
// (defaultOrgScanWorkers goroutines) to keep wall-clock time low for large orgs.
func scanOrgReposForSuggestions(c *orgScanCollector, org, primaryRepo string, doGet func(string) (*http.Response, error), client *http.Client) {
	alreadyChecked := map[string]bool{
		strings.ToLower(primaryRepo): true,
		".github":                    true,
		".github-private":            true,
	}

	fmt.Fprintf(os.Stderr, "  Scanning all repos in %s org for maintainer suggestions and Slack channels...\n", org)

	// Phase 1: Collect the full list of repos to scan (sequential — single
	// pagination pass, cheap).
	var toScan []repoEntry

	page := 1
	for {
		path := fmt.Sprintf("/orgs/%s/repos?per_page=100&page=%d", org, page)
		resp, err := doGet(path)
		if err != nil || resp.StatusCode != http.StatusOK {
			hint := ""
			if resp != nil {
				hint = rateLimitHint(resp)
				resp.Body.Close()
			}
			fmt.Fprintf(os.Stderr, "  Warning: could not list repos for org %s%s\n", org, hint)
			return
		}

		var repos []struct {
			Name       string `json:"name"`
			Fork       bool   `json:"fork"`
			Archived   bool   `json:"archived"`
			Disabled   bool   `json:"disabled"`
			IsTemplate bool   `json:"is_template"`
			Size       int    `json:"size"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return
		}
		resp.Body.Close()

		if len(repos) == 0 {
			break
		}

		for _, r := range repos {
			if alreadyChecked[strings.ToLower(r.Name)] {
				continue
			}
			// Skip repos that won't have relevant active governance data:
			// - Forks: governance files belong to the upstream project
			// - Archived: no longer maintained, stale data
			// - Disabled: inaccessible repos
			// - Templates: boilerplate, not project-specific
			// - GHSA repos: temporary private forks for security advisories
			// - GitHub Pages (*.github.io): static sites, no governance
			// - Empty repos (size 0): nothing to scan
			if r.Fork || r.Archived || r.Disabled || r.IsTemplate || r.Size == 0 {
				continue
			}
			if isGHSARepo(r.Name) || isGitHubPagesRepo(r.Name) {
				continue
			}
			toScan = append(toScan, repoEntry{Name: r.Name})
		}

		if len(repos) < 100 {
			break
		}
		page++
	}

	if len(toScan) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "  Found %d repos to scan (using %d workers)...\n", len(toScan), defaultOrgScanWorkers)

	// Phase 2: Fan out scanning across a bounded worker pool.
	// A shared context is cancelled on the first rate-limit (403) so remaining
	// workers drain quickly without burning API quota.
	ctx, cancel := context.WithTimeout(context.Background(), DefaultHTTPTimeout*time.Duration(len(toScan)))
	defer cancel()

	work := make(chan repoEntry, len(toScan))
	for _, r := range toScan {
		work <- r
	}
	close(work)

	var wg sync.WaitGroup
	workers := defaultOrgScanWorkers
	if len(toScan) < workers {
		workers = len(toScan)
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range work {
				if ctx.Err() != nil {
					return
				}
				if rateLimited := scanContentsForSuggestions(ctx, c, doGet, client,
					fmt.Sprintf("/repos/%s/%s/contents/", org, r.Name),
					fmt.Sprintf("%s/%s", org, r.Name)); rateLimited {
					fmt.Fprintf(os.Stderr, "  Rate-limited by GitHub API; stopping org scan early.\n")
					cancel()
					return
				}
			}
		}()
	}

	wg.Wait()
}

func WriteSuggestionsFile(outputDir string, suggestions []MaintainerSuggestion) (string, error) {
	section := BuildSuggestionsSection(suggestions)
	if section == "" {
		return "", nil
	}
	dir := filepath.Join(outputDir, ".cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, suggestionsFileName)
	if err := os.WriteFile(path, []byte(section), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
