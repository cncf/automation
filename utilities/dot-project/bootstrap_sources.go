package projects

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	cloMonitorSearchPath    = "/api/projects/search"
	defaultCLOMonitorURL    = "https://clomonitor.io"
	defaultGitHubAPIURL     = "https://api.github.com"
	defaultLandscapeYAMLURL = "https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml"
	landscapeLogoBaseURL    = "https://landscape.cncf.io/logos/"
)

// fetchFromCLOMonitor queries the CLOMonitor search API for a CNCF project by display name.
// It passes text and foundation=cncf as query parameters so the server filters results,
// then fuzzy-matches the display_name among the returned candidates.
// baseURL overrides the CLOMonitor API base (use "" for default).
// Returns nil if no match is found.
func fetchFromCLOMonitor(name string, client *http.Client, baseURL string) (*CLOMonitorProject, error) {
	if baseURL == "" {
		baseURL = defaultCLOMonitorURL
	}

	params := url.Values{}
	params.Set("text", name)
	params.Set("foundation", "cncf")
	fullURL := baseURL + cloMonitorSearchPath + "?" + params.Encode()

	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("CLOMonitor request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CLOMonitor returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading CLOMonitor response: %w", err)
	}

	var results []CLOMonitorProject
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("parsing CLOMonitor response: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	// Fuzzy match by display name among the server-filtered results
	var candidates []string
	for _, p := range results {
		candidates = append(candidates, p.DisplayName)
	}

	best, score := fuzzyMatch(name, candidates)
	if score == 0 {
		return nil, nil
	}

	for i, p := range results {
		if p.DisplayName == best {
			return &results[i], nil
		}
	}

	return nil, nil
}

// landscapeYAMLRoot is the top-level structure of the CNCF landscape.yml file.
type landscapeYAMLRoot struct {
	Landscape []landscapeYAMLCategory `yaml:"landscape"`
}

// landscapeYAMLCategory represents a category in the landscape.yml.
type landscapeYAMLCategory struct {
	Name          string                     `yaml:"name"`
	Subcategories []landscapeYAMLSubcategory `yaml:"subcategories"`
}

// landscapeYAMLSubcategory represents a subcategory in the landscape.yml.
type landscapeYAMLSubcategory struct {
	Name  string              `yaml:"name"`
	Items []landscapeYAMLItem `yaml:"items"`
}

// landscapeYAMLItem represents an individual project/product item in the landscape.yml.
type landscapeYAMLItem struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	HomepageURL string                 `yaml:"homepage_url"`
	RepoURL     string                 `yaml:"repo_url"`
	Logo        string                 `yaml:"logo"`
	Twitter     string                 `yaml:"twitter"`
	Project     string                 `yaml:"project"` // maturity: sandbox, incubating, graduated, archived
	Extra       map[string]interface{} `yaml:"extra,omitempty"`
}

// fetchFromLandscape fetches the CNCF landscape.yml from GitHub and searches for
// a CNCF project by name. Only items with a "project" field (indicating CNCF membership)
// are considered. baseURL overrides the landscape YAML URL (use "" for default).
// Returns nil if no match is found.
func fetchFromLandscape(name string, client *http.Client, baseURL string) (*LandscapeData, error) {
	if baseURL == "" {
		baseURL = defaultLandscapeYAMLURL
	}

	resp, err := client.Get(baseURL)
	if err != nil {
		return nil, fmt.Errorf("landscape YAML request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("landscape YAML returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading landscape YAML: %w", err)
	}

	var root landscapeYAMLRoot
	if err := yaml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parsing landscape YAML: %w", err)
	}

	// Collect all CNCF project items (those with a "project" field set)
	type itemWithContext struct {
		item        landscapeYAMLItem
		category    string
		subcategory string
	}
	var cncfItems []itemWithContext
	var names []string

	for _, cat := range root.Landscape {
		for _, subcat := range cat.Subcategories {
			for _, item := range subcat.Items {
				if item.Project != "" {
					cncfItems = append(cncfItems, itemWithContext{
						item:        item,
						category:    cat.Name,
						subcategory: subcat.Name,
					})
					names = append(names, item.Name)
				}
			}
		}
	}

	if len(cncfItems) == 0 {
		return nil, nil
	}

	// Fuzzy match by name
	best, score := fuzzyMatch(name, names)
	if score == 0 {
		return nil, nil
	}

	// Find the matching item
	for _, iwc := range cncfItems {
		if iwc.item.Name == best {
			logoURL := ""
			if iwc.item.Logo != "" {
				logoURL = landscapeLogoBaseURL + iwc.item.Logo
			}

			// Extract extra fields
			slackURL, _ := getExtraString(iwc.item.Extra, "slack_url")
			chatChannel, _ := getExtraString(iwc.item.Extra, "chat_channel")
			acceptedDate, _ := getExtraString(iwc.item.Extra, "accepted")
			annualReviewURL, _ := getExtraString(iwc.item.Extra, "annual_review_url")
			packageManagerURL, _ := getExtraString(iwc.item.Extra, "package_manager_url")

			return &LandscapeData{
				Name:              iwc.item.Name,
				Description:       iwc.item.Description,
				HomepageURL:       iwc.item.HomepageURL,
				RepoURL:           iwc.item.RepoURL,
				LogoURL:           logoURL,
				Twitter:           iwc.item.Twitter,
				Maturity:          iwc.item.Project,
				Category:          iwc.category,
				Subcategory:       iwc.subcategory,
				SlackURL:          slackURL,
				ChatChannel:       chatChannel,
				AcceptedDate:      acceptedDate,
				AnnualReviewURL:   annualReviewURL,
				PackageManagerURL: packageManagerURL,
			}, nil
		}
	}

	return nil, nil
}

// FetchFromLandscape is the exported wrapper for fetchFromLandscape.
func FetchFromLandscape(name string, client *http.Client, baseURL string) (*LandscapeData, error) {
	return fetchFromLandscape(name, client, baseURL)
}

// GitHubData holds all data fetched from the GitHub API for a single repo.
type GitHubData struct {
	Repo        *GitHubRepoData         `json:"repo"`
	Org         *GitHubOrgData          `json:"org"`
	Community   *GitHubCommunityProfile `json:"community"`
	Maintainers []string                `json:"maintainers,omitempty"`
	Reviewers   []string                `json:"reviewers,omitempty"`
	HasAdopters bool                    `json:"has_adopters,omitempty"`

	// Discovered file URLs (from Community Profile or governance file scan)
	SecurityPolicyURL string `json:"security_policy_url,omitempty"`
	ContributingURL   string `json:"contributing_url,omitempty"`
	CodeOfConductURL  string `json:"code_of_conduct_url,omitempty"`
	LicenseURL        string `json:"license_url,omitempty"`

	// Auto-detected identity type signals
	HasDCO bool `json:"has_dco,omitempty"`
	HasCLA bool `json:"has_cla,omitempty"`

	// Slack channels discovered across the org
	SlackChannels []string `json:"slack_channels,omitempty"`
}

// fetchFromGitHub fetches repository, organization, community profile, and
// discovers CODEOWNERS/OWNERS/MAINTAINERS files from the GitHub API.
// baseURL overrides the GitHub API URL (use "" for default).
func fetchFromGitHub(org, repo, token string, client *http.Client, baseURL string) (*GitHubData, error) {
	if baseURL == "" {
		baseURL = defaultGitHubAPIURL
	}

	result := &GitHubData{}

	// Helper to make authenticated requests
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

	// Fetch repo data
	resp, err := doGet(fmt.Sprintf("/repos/%s/%s", org, repo))
	if err != nil {
		return nil, fmt.Errorf("GitHub repo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub repo returned HTTP %d for %s/%s", resp.StatusCode, org, repo)
	}

	var repoData GitHubRepoData
	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return nil, fmt.Errorf("parsing GitHub repo response: %w", err)
	}
	result.Repo = &repoData

	// Fetch org data (non-fatal if fails)
	if resp, err := doGet(fmt.Sprintf("/orgs/%s", org)); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var orgData GitHubOrgData
			if err := json.NewDecoder(resp.Body).Decode(&orgData); err == nil {
				result.Org = &orgData
			}
		}
	}

	// Fetch community profile (non-fatal if fails)
	if resp, err := doGet(fmt.Sprintf("/repos/%s/%s/community/profile", org, repo)); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var community GitHubCommunityProfile
			if err := json.NewDecoder(resp.Body).Decode(&community); err == nil {
				result.Community = &community
			}
		}
	}

	// Extract file URLs from community profile
	if result.Community != nil {
		if f := result.Community.Files.Contributing; f != nil && f.HTMLURL != "" {
			result.ContributingURL = f.HTMLURL
		}
		if f := result.Community.Files.CodeOfConductFile; f != nil && f.HTMLURL != "" {
			result.CodeOfConductURL = f.HTMLURL
		}
		if f := result.Community.Files.License; f != nil && f.HTMLURL != "" {
			result.LicenseURL = f.HTMLURL
		}
	}

	// Discover governance files from repo root, .github/ dir, and org .github repo
	discoverGovernanceFiles(result, org, repo, doGet, client)

	// Detect DCO/CLA from commit messages and .github config files
	hasDCO, hasCLA, _ := detectDCOCLA(org, repo, token, client, baseURL)
	result.HasDCO = hasDCO
	result.HasCLA = hasCLA

	// Fetch README and extract Slack channels
	if resp, err := doGet(fmt.Sprintf("/repos/%s/%s/readme", org, repo)); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var readmeData struct {
				DownloadURL string `json:"download_url"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&readmeData); err == nil && readmeData.DownloadURL != "" {
				if content, err := fetchFileContent(client, readmeData.DownloadURL); err == nil {
					channels := extractSlackChannels(content)
					result.SlackChannels = mergeStringSlices(result.SlackChannels, channels)
				}
			}
		}
	}

	return result, nil
}

// governanceFile describes a file to look for and how to parse it.
type governanceFile struct {
	name      string
	parseFunc func(data *GitHubData, content string, htmlURL string)
}

var governanceFiles = []governanceFile{
	{
		name: "CODEOWNERS",
		parseFunc: func(data *GitHubData, content string, _ string) {
			handles := parseCodeowners(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, handles)
		},
	},
	{
		name: "OWNERS",
		parseFunc: func(data *GitHubData, content string, _ string) {
			approvers, reviewers := parseOwnersFile(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, approvers)
			data.Reviewers = mergeStringSlices(data.Reviewers, reviewers)
		},
	},
	{
		name: "MAINTAINERS",
		parseFunc: func(data *GitHubData, content string, _ string) {
			handles := parseMaintainersFile(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, handles)
		},
	},
	{
		name: "MAINTAINERS.md",
		parseFunc: func(data *GitHubData, content string, _ string) {
			handles := parseMaintainersFile(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, handles)
		},
	},
	{
		name: "ADOPTERS.md",
		parseFunc: func(data *GitHubData, _ string, _ string) {
			data.HasAdopters = true
		},
	},
	{
		name: "SECURITY.md",
		parseFunc: func(data *GitHubData, _ string, htmlURL string) {
			if data.SecurityPolicyURL == "" {
				data.SecurityPolicyURL = htmlURL
			}
		},
	},
	{
		name: "CONTRIBUTING.md",
		parseFunc: func(data *GitHubData, content string, _ string) {
			channels := extractSlackChannels(content)
			data.SlackChannels = mergeStringSlices(data.SlackChannels, channels)
		},
	},
	{
		name: "COMMUNITY.md",
		parseFunc: func(data *GitHubData, content string, _ string) {
			channels := extractSlackChannels(content)
			data.SlackChannels = mergeStringSlices(data.SlackChannels, channels)
		},
	},
}

// discoverGovernanceFiles looks for CODEOWNERS, OWNERS, MAINTAINERS in the repo root,
// .github/ subdirectory, and the org-level .github repo, then scans all repos in the
// org to collect maintainer handles across the entire org.
func discoverGovernanceFiles(result *GitHubData, org, repo string, doGet func(string) (*http.Response, error), client *http.Client) {
	// Locations to search, in order of priority
	contentPaths := []string{
		fmt.Sprintf("/repos/%s/%s/contents/", org, repo),
		fmt.Sprintf("/repos/%s/%s/contents/.github", org, repo),
		fmt.Sprintf("/repos/%s/.github/contents/", org),
	}

	for _, contentPath := range contentPaths {
		resp, err := doGet(contentPath)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		var entries []GitHubContentEntry
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, entry := range entries {
			for _, gf := range governanceFiles {
				if strings.EqualFold(entry.Name, gf.name) && entry.Type == "file" && entry.DownloadURL != "" {
					content, err := fetchFileContent(client, entry.DownloadURL)
					if err == nil && content != "" {
						gf.parseFunc(result, content, entry.HTMLURL)
					}
				}
			}
		}
	}

	// Scan all org repos to collect maintainer handles across the entire org.
	scanOrgReposForGovernance(result, org, repo, doGet, client)
}

// scanOrgReposForGovernance lists all repositories in the org and searches each one
// for governance files, skipping the primary repo root already checked by discoverGovernanceFiles.
// Handles found across all repos are merged and deduplicated into result.Maintainers.
func scanOrgReposForGovernance(result *GitHubData, org, primaryRepo string, doGet func(string) (*http.Response, error), client *http.Client) {
	// Already-checked repos — skip them to avoid duplicate work
	alreadyChecked := map[string]bool{
		strings.ToLower(primaryRepo): true,
		".github":                    true,
	}

	fmt.Fprintf(os.Stderr, "  Scanning all repos in %s org for maintainer handles and Slack channels...\n", org)
	before := len(result.Maintainers)

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

			fmt.Fprintf(os.Stderr, "    Checking %s/%s...\n", org, r.Name)
			scanRepoRootForGovernance(result, org, r.Name, doGet, client)
		}

		// No more pages
		if len(repos) < 100 {
			break
		}
		page++
	}

	found := len(result.Maintainers) - before
	if found > 0 {
		fmt.Fprintf(os.Stderr, "  Found %d maintainer(s) across org repos\n", found)
	}
}

// scanRepoRootForGovernance lists the root contents of a single repo and parses
// any governance files found there. It also extracts Slack channels from the
// repo README to discover channels across the entire org.
func scanRepoRootForGovernance(result *GitHubData, org, repo string, doGet func(string) (*http.Response, error), client *http.Client) {
	resp, err := doGet(fmt.Sprintf("/repos/%s/%s/contents/", org, repo))
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
		for _, gf := range governanceFiles {
			if strings.EqualFold(entry.Name, gf.name) && entry.Type == "file" && entry.DownloadURL != "" {
				content, err := fetchFileContent(client, entry.DownloadURL)
				if err == nil && content != "" {
					gf.parseFunc(result, content, entry.HTMLURL)
				}
			}
		}
		// Also check README for Slack channels
		if entry.Type == "file" && entry.DownloadURL != "" &&
			strings.EqualFold(strings.TrimSuffix(entry.Name, ".md"), "README") {
			content, err := fetchFileContent(client, entry.DownloadURL)
			if err == nil && content != "" {
				channels := extractSlackChannels(content)
				result.SlackChannels = mergeStringSlices(result.SlackChannels, channels)
			}
		}
	}
}

// fetchFileContent fetches the raw content of a file.
func fetchFileContent(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// mergeStringSlices merges two string slices, deduplicating entries.
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// getExtraString safely extracts a string value from a landscape item's extra map.
func getExtraString(extra map[string]interface{}, key string) (string, bool) {
	if extra == nil {
		return "", false
	}
	v, ok := extra[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// extractChannelFromSlackURL extracts a channel name from a Slack URL.
// Handles URLs like:
//   - https://cloud-native.slack.com/messages/my-project
//   - https://cloud-native.slack.com/channels/my-project
//   - https://cloud-native.slack.com/archives/my-project
//
// Returns the channel name without "#" prefix, or "" if not found.
func extractChannelFromSlackURL(slackURL string) string {
	for _, segment := range []string{"/messages/", "/channels/", "/archives/"} {
		parts := strings.SplitN(slackURL, segment, 2)
		if len(parts) == 2 && parts[1] != "" {
			channel := strings.TrimRight(parts[1], "/")
			if channel != "" {
				return channel
			}
		}
	}
	return ""
}

// LandscapeData represents data fetched from the CNCF landscape.
type LandscapeData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	HomepageURL string `json:"homepage_url"`
	RepoURL     string `json:"repo_url"`
	LogoURL     string `json:"logo_url"`
	Twitter     string `json:"twitter"`
	Maturity    string `json:"maturity"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`

	// Extra fields from landscape YAML
	SlackURL          string `json:"slack_url,omitempty"`
	ChatChannel       string `json:"chat_channel,omitempty"`
	AcceptedDate      string `json:"accepted_date,omitempty"`
	AnnualReviewURL   string `json:"annual_review_url,omitempty"`
	PackageManagerURL string `json:"package_manager_url,omitempty"`
}

// FetchFromCLOMonitor is the exported wrapper for fetchFromCLOMonitor.
func FetchFromCLOMonitor(name string, client *http.Client, baseURL string) (*CLOMonitorProject, error) {
	return fetchFromCLOMonitor(name, client, baseURL)
}

// FetchFromGitHub is the exported wrapper for fetchFromGitHub.
func FetchFromGitHub(org, repo, token string, client *http.Client, baseURL string) (*GitHubData, error) {
	return fetchFromGitHub(org, repo, token, client, baseURL)
}

// MergeBootstrapData is the exported wrapper for mergeBootstrapData.
func MergeBootstrapData(slug string, landscape *LandscapeData, clomonitor *CLOMonitorProject, github *GitHubData) *BootstrapResult {
	return mergeBootstrapData(slug, landscape, clomonitor, github)
}

// fuzzyMatch performs case-insensitive substring matching to find the best
// candidate matching the query. Returns the best match and a score (0 = no match).
func fuzzyMatch(query string, candidates []string) (best string, score float64) {
	if len(candidates) == 0 {
		return "", 0
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))

	var bestCandidate string
	var bestScore float64

	for _, candidate := range candidates {
		candidateLower := strings.ToLower(strings.TrimSpace(candidate))

		var s float64
		if candidateLower == queryLower {
			s = 1.0 // Exact match
		} else if strings.Contains(candidateLower, queryLower) {
			// Query is a substring of candidate
			s = float64(len(queryLower)) / float64(len(candidateLower))
		} else if strings.Contains(queryLower, candidateLower) {
			// Candidate is a substring of query
			s = float64(len(candidateLower)) / float64(len(queryLower))
		} else {
			// Check word overlap
			queryWords := strings.Fields(queryLower)
			candidateWords := strings.Fields(candidateLower)
			matchCount := 0
			for _, qw := range queryWords {
				for _, cw := range candidateWords {
					if qw == cw {
						matchCount++
						break
					}
				}
			}
			if matchCount > 0 {
				total := len(queryWords)
				if len(candidateWords) > total {
					total = len(candidateWords)
				}
				s = float64(matchCount) / float64(total) * DefaultFuzzyMatchWeight
			}
		}

		if s > bestScore {
			bestScore = s
			bestCandidate = candidate
		}
	}

	return bestCandidate, bestScore
}

// detectDCOCLA checks recent commits for Signed-off-by lines (DCO) and
// .github/ directory for CLA config files.
// Returns (hasDCO, hasCLA, error). Errors are non-fatal; returns (false, false, nil) on failure.
func detectDCOCLA(org, repo, token string, client *http.Client, baseURL string) (bool, bool, error) {
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

	var hasDCO, hasCLA bool

	// Check recent commits for DCO (Signed-off-by)
	resp, err := doGet(fmt.Sprintf("/repos/%s/%s/commits?per_page=%d", org, repo, DefaultDCOCommitSampleSize))
	if err == nil && resp.StatusCode == http.StatusOK {
		var commits []struct {
			Commit struct {
				Message string `json:"message"`
			} `json:"commit"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&commits); err == nil {
			signedCount := 0
			for _, c := range commits {
				if strings.Contains(c.Commit.Message, "Signed-off-by:") {
					signedCount++
				}
			}
			// If more than DefaultDCOSignedRatio of commits have Signed-off-by, likely using DCO
			if len(commits) > 0 && float64(signedCount)/float64(len(commits)) > DefaultDCOSignedRatio {
				hasDCO = true
			}
		}
		resp.Body.Close()
	} else if resp != nil {
		resp.Body.Close()
	}

	// Check for CLA config files in .github/
	claFiles := []string{"cla.yml", "cla.yaml", ".clabot", "clabot.config"}
	resp, err = doGet(fmt.Sprintf("/repos/%s/%s/contents/.github", org, repo))
	if err == nil && resp.StatusCode == http.StatusOK {
		var entries []GitHubContentEntry
		if err := json.NewDecoder(resp.Body).Decode(&entries); err == nil {
			for _, entry := range entries {
				for _, claFile := range claFiles {
					if strings.EqualFold(entry.Name, claFile) {
						hasCLA = true
						break
					}
				}
			}
		}
		resp.Body.Close()
	} else if resp != nil {
		resp.Body.Close()
	}

	return hasDCO, hasCLA, nil
}

// SearchTOCIssues searches cncf/toc and cncf/sandbox for onboarding or
// maturity-change issues related to the given project.
// Returns the best-match issue URL, or "" if none found.
func SearchTOCIssues(projectName, orgName, token string, client *http.Client, baseURL string) (string, error) {
	if baseURL == "" {
		baseURL = defaultGitHubAPIURL
	}

	// Search queries to try, in order of specificity
	queries := []string{
		fmt.Sprintf(`"%s" repo:cncf/sandbox in:title`, projectName),
		fmt.Sprintf(`"%s" repo:cncf/toc in:title`, projectName),
		fmt.Sprintf(`"%s" repo:cncf/sandbox in:title`, orgName),
		fmt.Sprintf(`"%s" repo:cncf/toc in:title`, orgName),
	}

	for _, q := range queries {
		req, err := http.NewRequest("GET", baseURL+"/search/issues?q="+url.QueryEscape(q)+"&per_page=5", nil)
		if err != nil {
			return "", err
		}
		if token != "" {
			req.Header.Set("Authorization", "token "+token)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		// Ensure the response body is closed after we're done reading it.
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var searchResult struct {
			TotalCount int `json:"total_count"`
			Items      []struct {
				HTMLURL string `json:"html_url"`
				Title   string `json:"title"`
				Number  int    `json:"number"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &searchResult); err != nil {
			continue
		}

		if searchResult.TotalCount > 0 && len(searchResult.Items) > 0 {
			return searchResult.Items[0].HTMLURL, nil
		}
	}

	return "", nil
}

// mergeBootstrapData combines data from landscape, CLOMonitor, and GitHub
// into a single BootstrapResult. Priority order: landscape > CLOMonitor > GitHub.
func mergeBootstrapData(slug string, landscape *LandscapeData, clomonitor *CLOMonitorProject, github *GitHubData) *BootstrapResult {
	result := &BootstrapResult{
		Slug:    slug,
		Social:  make(map[string]string),
		Sources: make(map[string]string),
	}

	// GitHub context for scaffold generation (org/repo)
	if github != nil && github.Org != nil {
		result.GitHubOrg = github.Org.Login
	}
	if github != nil && github.Repo != nil {
		result.GitHubRepo = github.Repo.Name
	}

	// Name: landscape > clomonitor > github > slug
	if landscape != nil && landscape.Name != "" {
		result.Name = landscape.Name
		result.Sources["name"] = "landscape"
	} else if clomonitor != nil && clomonitor.DisplayName != "" {
		result.Name = clomonitor.DisplayName
		result.Sources["name"] = "clomonitor"
	} else if github != nil && github.Repo != nil && github.Repo.Name != "" {
		result.Name = github.Repo.Name
		result.Sources["name"] = "github"
	} else {
		result.Name = slug
		result.Sources["name"] = "slug"
	}

	// Description: landscape > clomonitor > github
	if landscape != nil && landscape.Description != "" {
		result.Description = landscape.Description
		result.Sources["description"] = "landscape"
	} else if clomonitor != nil && clomonitor.Description != "" {
		result.Description = clomonitor.Description
		result.Sources["description"] = "clomonitor"
	} else if github != nil && github.Repo != nil && github.Repo.Description != "" {
		result.Description = github.Repo.Description
		result.Sources["description"] = "github"
	}

	// Website: landscape > clomonitor > github repo homepage
	if landscape != nil && landscape.HomepageURL != "" {
		result.Website = landscape.HomepageURL
		result.Sources["website"] = "landscape"
	} else if clomonitor != nil && clomonitor.HomeURL != "" {
		result.Website = clomonitor.HomeURL
		result.Sources["website"] = "clomonitor"
	} else if github != nil && github.Repo != nil && github.Repo.Homepage != "" {
		result.Website = github.Repo.Homepage
		result.Sources["website"] = "github"
	}

	// Repositories: landscape repo > clomonitor repos > github repo URL
	if landscape != nil && landscape.RepoURL != "" {
		result.Repositories = []string{landscape.RepoURL}
		result.Sources["repositories"] = "landscape"
	} else if clomonitor != nil && len(clomonitor.Repositories) > 0 {
		for _, r := range clomonitor.Repositories {
			if r.URL != "" {
				result.Repositories = append(result.Repositories, r.URL)
			}
		}
		result.Sources["repositories"] = "clomonitor"
	} else if github != nil && github.Repo != nil && github.Repo.HTMLURL != "" {
		result.Repositories = []string{github.Repo.HTMLURL}
		result.Sources["repositories"] = "github"
	}

	// Maturity: landscape > clomonitor
	if landscape != nil && landscape.Maturity != "" {
		result.MaturityPhase = landscape.Maturity
		result.Sources["maturity"] = "landscape"
	} else if clomonitor != nil && clomonitor.Maturity != "" {
		result.MaturityPhase = clomonitor.Maturity
		result.Sources["maturity"] = "clomonitor"
	}

	// Landscape category/subcategory
	if landscape != nil {
		result.LandscapeCategory = landscape.Category
		result.LandscapeSubcategory = landscape.Subcategory
		result.Sources["landscape_category"] = "landscape"
	} else if clomonitor != nil {
		result.LandscapeCategory = clomonitor.Category
		result.LandscapeSubcategory = clomonitor.Subcategory
		result.Sources["landscape_category"] = "clomonitor"
	}

	// Artwork: landscape logo
	if landscape != nil && landscape.LogoURL != "" {
		result.Artwork = landscape.LogoURL
		result.Sources["artwork"] = "landscape"
	} else if clomonitor != nil && clomonitor.LogoURL != "" {
		result.Artwork = clomonitor.LogoURL
		result.Sources["artwork"] = "clomonitor"
	}

	// Social: twitter from landscape or github org
	if landscape != nil && landscape.Twitter != "" {
		result.Social["twitter"] = landscape.Twitter
		result.Sources["social.twitter"] = "landscape"
	} else if github != nil && github.Org != nil && github.Org.TwitterUser != "" {
		result.Social["twitter"] = "https://twitter.com/" + github.Org.TwitterUser
		result.Sources["social.twitter"] = "github"
	}

	// Slack channel: landscape extra.chat_channel > derived from extra.slack_url > GitHub org-wide scan
	// Landscape values are authoritative; GitHub-discovered channels become candidates.
	if landscape != nil && landscape.ChatChannel != "" {
		result.CNCFSlackChannel = landscape.ChatChannel
		result.Sources["cncf_slack_channel"] = "landscape"
	} else if landscape != nil && landscape.SlackURL != "" {
		// Derive channel name from slack URL: .../messages/<channel-name> or .../channels/<channel-name> or .../archives/<channel-name>
		channel := extractChannelFromSlackURL(landscape.SlackURL)
		if channel != "" {
			result.CNCFSlackChannel = "#" + channel
			result.Sources["cncf_slack_channel"] = "landscape"
		}
	}

	// Collect all GitHub-discovered Slack channels
	if github != nil && len(github.SlackChannels) > 0 {
		if result.CNCFSlackChannel == "" {
			// No landscape value — promote the first GitHub channel as primary
			result.CNCFSlackChannel = github.SlackChannels[0]
			result.Sources["cncf_slack_channel"] = "github_readme"
			// Remaining channels become candidates
			if len(github.SlackChannels) > 1 {
				result.CNCFSlackCandidates = append([]string{}, github.SlackChannels[1:]...)
			}
		} else {
			// Landscape provided a primary — all GitHub channels are candidates
			// (excluding duplicates of the landscape value)
			for _, ch := range github.SlackChannels {
				if ch != result.CNCFSlackChannel {
					result.CNCFSlackCandidates = append(result.CNCFSlackCandidates, ch)
				}
			}
		}
	}

	// Accepted date from landscape extra
	if landscape != nil && landscape.AcceptedDate != "" {
		if t, err := time.Parse("2006-01-02", landscape.AcceptedDate); err == nil {
			result.AcceptedDate = t
			result.Sources["accepted_date"] = "landscape"
		}
	}

	// TOC issue URL from landscape annual_review_url (best available automated source)
	if landscape != nil && landscape.AnnualReviewURL != "" {
		result.TOCIssueURL = landscape.AnnualReviewURL
		result.Sources["toc_issue_url"] = "landscape"
	}

	// CLOMonitor score
	if clomonitor != nil && clomonitor.Score != nil {
		result.CLOMonitorScore = clomonitor.Score
	}

	// Maintainers and reviewers from GitHub
	if github != nil {
		result.Maintainers = github.Maintainers
		result.Reviewers = github.Reviewers

		// Infer project_lead from first discovered maintainer
		if len(github.Maintainers) > 0 && result.ProjectLead == "" {
			result.ProjectLead = github.Maintainers[0]
			result.Sources["project_lead"] = "github"
		}

		// Community health signals
		if github.Community != nil {
			result.HasCodeOfConduct = github.Community.Files.CodeOfConduct != nil
			result.HasContributing = github.Community.Files.Contributing != nil
			result.HasLicense = github.Community.Files.License != nil
			result.HasReadme = github.Community.Files.Readme != nil
		}

		result.HasAdopters = github.HasAdopters

		// DCO/CLA detection from GitHub
		result.HasDCO = github.HasDCO
		result.HasCLA = github.HasCLA
		if github.HasDCO || github.HasCLA {
			result.Sources["identity_type"] = "github"
		}

		// Discovered file URLs
		if github.SecurityPolicyURL != "" {
			result.SecurityPolicyURL = github.SecurityPolicyURL
			result.HasSecurityPolicy = true
			result.Sources["security_policy"] = "github"
		}
		if github.ContributingURL != "" {
			result.ContributingURL = github.ContributingURL
			result.Sources["contributing"] = "github"
		}
		if github.CodeOfConductURL != "" {
			result.CodeOfConductURL = github.CodeOfConductURL
			result.Sources["code_of_conduct"] = "github"
		}
		if github.LicenseURL != "" {
			result.LicenseURL = github.LicenseURL
			result.Sources["license"] = "github"
		}
	}

	// Generate TODOs for missing required fields
	if result.Description == "" {
		result.TODOs = append(result.TODOs, "Add project description")
	}
	if result.Website == "" {
		result.TODOs = append(result.TODOs, "Add project website URL")
	}
	if len(result.Repositories) == 0 {
		result.TODOs = append(result.TODOs, "Add at least one repository URL")
	}
	if result.MaturityPhase == "" {
		result.TODOs = append(result.TODOs, "Set maturity phase (sandbox, incubating, graduated)")
	}
	if len(result.Maintainers) == 0 {
		result.TODOs = append(result.TODOs, "Add maintainer GitHub handles")
	}
	if result.TOCIssueURL == "" {
		result.TODOs = append(result.TODOs, "Add maturity_log entry with TOC issue URL")
	}
	if result.ProjectLead == "" {
		result.TODOs = append(result.TODOs, "Set project_lead GitHub handle")
	}
	if result.CNCFSlackChannel == "" {
		result.TODOs = append(result.TODOs, "Set cncf_slack_channel")
	}
	if !result.HasDCO && !result.HasCLA {
		result.TODOs = append(result.TODOs, "Set identity_type under legal (has_dco, has_cla)")
	}
	if !result.HasAdopters {
		result.TODOs = append(result.TODOs, "Add adopters list (ADOPTERS.md)")
	}
	result.TODOs = append(result.TODOs, "Add package_managers if distributed via registries")

	// Clean up empty social map
	if len(result.Social) == 0 {
		result.Social = nil
	}

	return result
}
