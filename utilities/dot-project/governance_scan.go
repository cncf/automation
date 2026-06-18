package projects

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	scanContentsForSuggestions(c, doGet, client,
		fmt.Sprintf("/repos/%s/%s/contents/", org, primaryRepo), fmt.Sprintf("%s/%s", org, primaryRepo))
	scanContentsForSuggestions(c, doGet, client,
		fmt.Sprintf("/repos/%s/%s/contents/.github", org, primaryRepo), fmt.Sprintf("%s/%s", org, primaryRepo))
	scanContentsForSuggestions(c, doGet, client,
		fmt.Sprintf("/repos/%s/.github/contents/", org), fmt.Sprintf("%s/.github", org))

	scanOrgReposForSuggestions(c, org, primaryRepo, doGet, client)

	return c.suggestions.suggestions(csvHandles), c.slack
}

// orgScanCollector accumulates the results of a single shared org-wide scan
// pass: maintainer suggestions (from governance files) and Slack channels (from
// community documentation files).
type orgScanCollector struct {
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
	for _, ch := range channels {
		if ch == "" || c.slackSeen[ch] {
			continue
		}
		c.slackSeen[ch] = true
		c.slack = append(c.slack, ch)
	}
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
func scanContentsForSuggestions(c *orgScanCollector, doGet func(string) (*http.Response, error), client *http.Client, contentPath, sourceRepo string) {
	resp, err := doGet(contentPath)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	var entries []GitHubContentEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		resp.Body.Close()
		return
	}
	resp.Body.Close()

	for _, entry := range entries {
		if entry.Type != "file" || entry.DownloadURL == "" {
			continue
		}
		switch {
		case isGovernanceMaintainerFile(entry.Name):
			content, err := fetchFileContent(client, entry.DownloadURL)
			if err == nil && content != "" {
				addGovernanceFileToCollector(c.suggestions, entry.Name, content, sourceRepo)
			}
		case isCommunityDocFile(entry.Name):
			content, err := fetchFileContent(client, entry.DownloadURL)
			if err == nil && content != "" {
				c.addSlack(extractSlackChannels(content))
			}
		}
	}
}

// scanOrgReposForSuggestions lists every repo in the org and scans each repo
// root for governance files and community docs, skipping the primary repo and
// the org .github repo (already scanned by the caller).
func scanOrgReposForSuggestions(c *orgScanCollector, org, primaryRepo string, doGet func(string) (*http.Response, error), client *http.Client) {
	alreadyChecked := map[string]bool{
		strings.ToLower(primaryRepo): true,
		".github":                    true,
	}

	fmt.Fprintf(os.Stderr, "  Scanning all repos in %s org for maintainer suggestions and Slack channels...\n", org)

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
			Name string `json:"name"`
			Fork bool   `json:"fork"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return
		}
		resp.Body.Close()

		if len(repos) == 0 {
			return
		}

		for _, r := range repos {
			if alreadyChecked[strings.ToLower(r.Name)] {
				continue
			}
			// Skip forks: their governance files belong to the upstream project,
			// not this org, and would produce misleading suggestions.
			if r.Fork {
				continue
			}
			fmt.Fprintf(os.Stderr, "    Checking %s/%s...\n", org, r.Name)
			scanContentsForSuggestions(c, doGet, client,
				fmt.Sprintf("/repos/%s/%s/contents/", org, r.Name),
				fmt.Sprintf("%s/%s", org, r.Name))
		}

		if len(repos) < 100 {
			break
		}
		page++
	}
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
