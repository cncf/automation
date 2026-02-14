package projects

import "time"

// BootstrapConfig holds configuration for the bootstrap command.
type BootstrapConfig struct {
	ProjectName string // Display name to search for (e.g., "Kubernetes")
	GitHubOrg   string // GitHub organization (e.g., "kubernetes")
	GitHubRepo  string // Primary repository name (e.g., "kubernetes")
	GitHubToken string // GitHub personal access token (optional but recommended)
	OutputDir   string // Directory to write scaffold output
}

// BootstrapResult is the merged, normalized output from all data sources.
// The scaffold generator consumes this to produce project.yaml and maintainers.yaml.
type BootstrapResult struct {
	// Identification
	Slug        string `json:"slug" yaml:"slug"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`

	// URLs
	Website      string            `json:"website,omitempty" yaml:"website,omitempty"`
	Repositories []string          `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Artwork      string            `json:"artwork,omitempty" yaml:"artwork,omitempty"`
	Social       map[string]string `json:"social,omitempty" yaml:"social,omitempty"`

	// Maturity
	MaturityPhase string    `json:"maturity_phase,omitempty" yaml:"maturity_phase,omitempty"`
	AcceptedDate  time.Time `json:"accepted_date,omitempty" yaml:"accepted_date,omitempty"`

	// Landscape
	LandscapeCategory    string `json:"landscape_category,omitempty" yaml:"landscape_category,omitempty"`
	LandscapeSubcategory string `json:"landscape_subcategory,omitempty" yaml:"landscape_subcategory,omitempty"`

	// Contacts
	ProjectLead      string `json:"project_lead,omitempty" yaml:"project_lead,omitempty"`
	CNCFSlackChannel string `json:"cncf_slack_channel,omitempty" yaml:"cncf_slack_channel,omitempty"`

	// Maintainers discovered from CODEOWNERS/OWNERS/MAINTAINERS files
	Maintainers []string `json:"maintainers,omitempty" yaml:"maintainers,omitempty"`
	Reviewers   []string `json:"reviewers,omitempty" yaml:"reviewers,omitempty"`

	// CLOMonitor scores (informational, included as YAML comments)
	CLOMonitorScore *CLOMonitorScore `json:"clomonitor_score,omitempty" yaml:"clomonitor_score,omitempty"`

	// Community health files discovered via GitHub community profile
	HasCodeOfConduct   bool   `json:"has_code_of_conduct,omitempty" yaml:"has_code_of_conduct,omitempty"`
	HasContributing    bool   `json:"has_contributing,omitempty" yaml:"has_contributing,omitempty"`
	HasLicense         bool   `json:"has_license,omitempty" yaml:"has_license,omitempty"`
	HasReadme          bool   `json:"has_readme,omitempty" yaml:"has_readme,omitempty"`
	HasSecurityPolicy  bool   `json:"has_security_policy,omitempty" yaml:"has_security_policy,omitempty"`
	SecurityContactURL string `json:"security_contact_url,omitempty" yaml:"security_contact_url,omitempty"`
	HasAdopters        bool   `json:"has_adopters,omitempty" yaml:"has_adopters,omitempty"`
	IdentityTypeHint   string `json:"identity_type_hint,omitempty" yaml:"identity_type_hint,omitempty"` // "dco", "dco+cla", or ""

	// Source tracking: which fields came from which source
	Sources map[string]string `json:"sources,omitempty" yaml:"sources,omitempty"`

	// TODOs: fields the user must manually fill in
	TODOs []string `json:"todos,omitempty" yaml:"todos,omitempty"`
}

// CLOMonitorProject represents a project entry from the CLOMonitor API search results.
type CLOMonitorProject struct {
	ID           string           `json:"project_id"`
	Name         string           `json:"name"`
	DisplayName  string           `json:"display_name"`
	Description  string           `json:"description"`
	HomeURL      string           `json:"home_url"`
	LogoURL      string           `json:"logo_url"`
	DevStatsURL  string           `json:"devstats_url"`
	AcceptedAt   string           `json:"accepted_at"`
	Maturity     string           `json:"maturity"`
	Foundation   string           `json:"foundation"`
	Category     string           `json:"category"`
	Subcategory  string           `json:"subcategory"`
	Score        *CLOMonitorScore `json:"score"`
	Repositories []CLOMonitorRepo `json:"repositories"`
	Rating       string           `json:"rating"`
	UpdatedAt    string           `json:"updated_at"`
}

// CLOMonitorRepo represents a repository within a CLOMonitor project.
type CLOMonitorRepo struct {
	Name   string            `json:"name"`
	URL    string            `json:"url"`
	Report *CLOMonitorReport `json:"report"`
}

// CLOMonitorReport holds the per-repository check report from CLOMonitor.
type CLOMonitorReport struct {
	Documentation interface{} `json:"documentation"`
	License       interface{} `json:"license"`
	BestPractices interface{} `json:"best_practices"`
	Security      interface{} `json:"security"`
	Legal         interface{} `json:"legal"`
}

// CLOMonitorScore holds the global and per-category scores.
type CLOMonitorScore struct {
	Global        float64 `json:"global"`
	Documentation float64 `json:"documentation"`
	License       float64 `json:"license"`
	BestPractices float64 `json:"best_practices"`
	Security      float64 `json:"security"`
	Legal         float64 `json:"legal"`
}

// GitHubRepoData holds data fetched from the GitHub repos API.
type GitHubRepoData struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	HTMLURL         string `json:"html_url"`
	Homepage        string `json:"homepage"`
	Language        string `json:"language"`
	DefaultBranch   string `json:"default_branch"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	License         *struct {
		Key    string `json:"key"`
		Name   string `json:"name"`
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
	Topics []string `json:"topics"`
}

// GitHubOrgData holds data fetched from the GitHub orgs API.
type GitHubOrgData struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	Blog        string `json:"blog"`
	TwitterUser string `json:"twitter_username"`
	Email       string `json:"email"`
}

// GitHubCommunityProfile holds the community health profile from GitHub.
type GitHubCommunityProfile struct {
	HealthPercentage int `json:"health_percentage"`
	Files            struct {
		CodeOfConduct *struct {
			URL string `json:"url"`
		} `json:"code_of_conduct"`
		Contributing *struct {
			URL string `json:"url"`
		} `json:"contributing"`
		License *struct {
			Key    string `json:"key"`
			Name   string `json:"name"`
			SPDXID string `json:"spdx_id"`
			URL    string `json:"url"`
		} `json:"license"`
		Readme *struct {
			URL string `json:"url"`
		} `json:"readme"`
	} `json:"files"`
	Description           string `json:"description"`
	ContentReportsEnabled bool   `json:"content_reports_enabled"`
}

// GitHubContentEntry represents a file entry from the GitHub contents API.
type GitHubContentEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
}
