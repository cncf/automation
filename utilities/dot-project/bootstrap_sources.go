package projects

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	cloMonitorSearchPath    = "/api/projects/search"
	defaultCLOMonitorURL    = "https://clomonitor.io"
	defaultGitHubAPIURL     = "https://api.github.com"
	defaultLandscapeYAMLURL = "https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml"
	landscapeLogoBaseURL    = "https://landscape.cncf.io/logos/"
)

// fetchFromCLOMonitor queries the CLOMonitor API for a CNCF project by display name.
// It fuzzy-matches the display_name, filtering for foundation=cncf.
// baseURL overrides the CLOMonitor API base (use "" for default).
// Returns nil if no match is found.
func fetchFromCLOMonitor(name string, client *http.Client, baseURL string) (*CLOMonitorProject, error) {
	if baseURL == "" {
		baseURL = defaultCLOMonitorURL
	}

	url := baseURL + cloMonitorSearchPath
	resp, err := client.Get(url)
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

	// Filter to CNCF foundation only
	var cncfProjects []CLOMonitorProject
	for _, p := range results {
		if strings.EqualFold(p.Foundation, "cncf") {
			cncfProjects = append(cncfProjects, p)
		}
	}

	if len(cncfProjects) == 0 {
		return nil, nil
	}

	// Fuzzy match by display name
	var candidates []string
	for _, p := range cncfProjects {
		candidates = append(candidates, p.DisplayName)
	}

	best, score := fuzzyMatch(name, candidates)
	if score == 0 {
		return nil, nil
	}

	for i, p := range cncfProjects {
		if p.DisplayName == best {
			return &cncfProjects[i], nil
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
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	HomepageURL string `yaml:"homepage_url"`
	RepoURL     string `yaml:"repo_url"`
	Logo        string `yaml:"logo"`
	Twitter     string `yaml:"twitter"`
	Project     string `yaml:"project"` // maturity: sandbox, incubating, graduated, archived
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
			return &LandscapeData{
				Name:        iwc.item.Name,
				Description: iwc.item.Description,
				HomepageURL: iwc.item.HomepageURL,
				RepoURL:     iwc.item.RepoURL,
				LogoURL:     logoURL,
				Twitter:     iwc.item.Twitter,
				Maturity:    iwc.item.Project,
				Category:    iwc.category,
				Subcategory: iwc.subcategory,
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

	// Discover governance files from repo root, .github/ dir, and org .github repo
	discoverGovernanceFiles(result, org, repo, doGet, client)

	return result, nil
}

// governanceFile describes a file to look for and how to parse it.
type governanceFile struct {
	name      string
	parseFunc func(data *GitHubData, content string)
}

var governanceFiles = []governanceFile{
	{
		name: "CODEOWNERS",
		parseFunc: func(data *GitHubData, content string) {
			handles := parseCodeowners(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, handles)
		},
	},
	{
		name: "OWNERS",
		parseFunc: func(data *GitHubData, content string) {
			approvers, reviewers := parseOwnersFile(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, approvers)
			data.Reviewers = mergeStringSlices(data.Reviewers, reviewers)
		},
	},
	{
		name: "MAINTAINERS",
		parseFunc: func(data *GitHubData, content string) {
			handles := parseMaintainersFile(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, handles)
		},
	},
	{
		name: "MAINTAINERS.md",
		parseFunc: func(data *GitHubData, content string) {
			handles := parseMaintainersFile(content)
			data.Maintainers = mergeStringSlices(data.Maintainers, handles)
		},
	},
	{
		name: "ADOPTERS.md",
		parseFunc: func(data *GitHubData, content string) {
			data.HasAdopters = true
		},
	},
}

// discoverGovernanceFiles looks for CODEOWNERS, OWNERS, MAINTAINERS in the repo root,
// .github/ subdirectory, and the org-level .github repo.
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
						gf.parseFunc(result, content)
					}
				}
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
				s = float64(matchCount) / float64(total) * 0.5
			}
		}

		if s > bestScore {
			bestScore = s
			bestCandidate = candidate
		}
	}

	return bestCandidate, bestScore
}

// mergeBootstrapData combines data from landscape, CLOMonitor, and GitHub
// into a single BootstrapResult. Priority order: landscape > CLOMonitor > GitHub.
func mergeBootstrapData(slug string, landscape *LandscapeData, clomonitor *CLOMonitorProject, github *GitHubData) *BootstrapResult {
	result := &BootstrapResult{
		Slug:    slug,
		Social:  make(map[string]string),
		Sources: make(map[string]string),
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

	// CLOMonitor score
	if clomonitor != nil && clomonitor.Score != nil {
		result.CLOMonitorScore = clomonitor.Score
	}

	// Maintainers and reviewers from GitHub
	if github != nil {
		result.Maintainers = github.Maintainers
		result.Reviewers = github.Reviewers

		// Community health signals
		if github.Community != nil {
			result.HasCodeOfConduct = github.Community.Files.CodeOfConduct != nil
			result.HasContributing = github.Community.Files.Contributing != nil
			result.HasLicense = github.Community.Files.License != nil
			result.HasReadme = github.Community.Files.Readme != nil
		}

		result.HasAdopters = github.HasAdopters
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
	result.TODOs = append(result.TODOs, "Add maturity_log entry with TOC issue URL")
	result.TODOs = append(result.TODOs, "Set project_lead GitHub handle")
	result.TODOs = append(result.TODOs, "Set cncf_slack_channel")
	result.TODOs = append(result.TODOs, "Set identity_type under legal (dco, cla, or none)")
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
