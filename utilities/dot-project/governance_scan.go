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

func DiscoverGovernanceSuggestions(org, primaryRepo, token string, client *http.Client, baseURL string, csvHandles map[string]bool) []MaintainerSuggestion {
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

	c := newSuggestionCollector()

	scanContentsForSuggestions(c, doGet, client,
		fmt.Sprintf("/repos/%s/%s/contents/", org, primaryRepo), fmt.Sprintf("%s/%s", org, primaryRepo))
	scanContentsForSuggestions(c, doGet, client,
		fmt.Sprintf("/repos/%s/%s/contents/.github", org, primaryRepo), fmt.Sprintf("%s/%s", org, primaryRepo))
	scanContentsForSuggestions(c, doGet, client,
		fmt.Sprintf("/repos/%s/.github/contents/", org), fmt.Sprintf("%s/.github", org))

	scanOrgReposForSuggestions(c, org, primaryRepo, doGet, client)

	return c.suggestions(csvHandles)
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

// scanContentsForSuggestions lists a contents path and feeds any governance
// files found there into the collector, labelled with sourceRepo.
func scanContentsForSuggestions(c *suggestionCollector, doGet func(string) (*http.Response, error), client *http.Client, contentPath, sourceRepo string) {
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
		if entry.Type == "file" && entry.DownloadURL != "" && isGovernanceMaintainerFile(entry.Name) {
			content, err := fetchFileContent(client, entry.DownloadURL)
			if err == nil && content != "" {
				addGovernanceFileToCollector(c, entry.Name, content, sourceRepo)
			}
		}
	}
}

// scanOrgReposForSuggestions lists every repo in the org and scans each repo
// root for governance files, skipping the primary repo and the org .github repo
// (already scanned by the caller).
func scanOrgReposForSuggestions(c *suggestionCollector, org, primaryRepo string, doGet func(string) (*http.Response, error), client *http.Client) {
	alreadyChecked := map[string]bool{
		strings.ToLower(primaryRepo): true,
		".github":                    true,
	}

	fmt.Fprintf(os.Stderr, "  Scanning all repos in %s org for maintainer suggestions...\n", org)

	page := 1
	for {
		path := fmt.Sprintf("/orgs/%s/repos?per_page=100&page=%d", org, page)
		resp, err := doGet(path)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			fmt.Fprintf(os.Stderr, "  Warning: could not list repos for org %s\n", org)
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
