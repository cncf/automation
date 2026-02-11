package projects

import (
	"net/http"
	"time"
)

type Project struct {
	Name         string            `json:"name" yaml:"name"`
	Description  string            `json:"description" yaml:"description"`
	MaturityLog  []MaturityEntry   `json:"maturity_log" yaml:"maturity_log"`
	Repositories []string          `json:"repositories" yaml:"repositories"`
	Social       map[string]string `json:"social" yaml:"social"`
	Artwork      string            `json:"artwork" yaml:"artwork"`                       // Artwork URL
	Website      string            `json:"website" yaml:"website"`                       // Project website URL
	MailingLists []string          `json:"mailing_lists" yaml:"mailing_lists"`           // Mailing lists for project
	Audits       []Audit           `json:"audits" yaml:"audits"`                         // Security audits for project
	Adopters     *PathRef          `json:"adopters,omitempty" yaml:"adopters,omitempty"` // Link to ADOPTERS.md or similar

	// Distribution
	PackageManagers map[string]string `json:"package_managers,omitempty" yaml:"package_managers,omitempty"` // Registry identifiers (e.g., npm, pypi, docker)

	// Schema and identification
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Type          string `json:"type,omitempty" yaml:"type,omitempty"`
	Slug          string `json:"slug" yaml:"slug"` // Unique project identifier (lowercase, alphanumeric + hyphens)

	// Contacts and channels
	ProjectLead      string `json:"project_lead,omitempty" yaml:"project_lead,omitempty"`             // GitHub handle of primary contact
	CNCFSlackChannel string `json:"cncf_slack_channel,omitempty" yaml:"cncf_slack_channel,omitempty"` // CNCF Slack channel (e.g., "#kubernetes")

	// Governance, security, legal, documentation
	Security      *SecurityConfig      `json:"security,omitempty" yaml:"security,omitempty"`
	Governance    *GovernanceConfig    `json:"governance,omitempty" yaml:"governance,omitempty"`
	Legal         *LegalConfig         `json:"legal,omitempty" yaml:"legal,omitempty"`
	Documentation *DocumentationConfig `json:"documentation,omitempty" yaml:"documentation,omitempty"`

	// CNCF Landscape integration
	Landscape *LandscapeConfig `json:"landscape,omitempty" yaml:"landscape,omitempty"`
}

// LandscapeConfig maps the project to its CNCF Landscape location
type LandscapeConfig struct {
	Category    string `json:"category" yaml:"category"`
	Subcategory string `json:"subcategory" yaml:"subcategory"`
}

type PathRef struct {
	Path string `json:"path" yaml:"path"`
}

type SecurityConfig struct {
	Policy      *PathRef `json:"policy" yaml:"policy"`
	ThreatModel *PathRef `json:"threat_model,omitempty" yaml:"threat_model,omitempty"`
	Contact     string   `json:"contact" yaml:"contact"`
}

type GovernanceConfig struct {
	Contributing  *PathRef `json:"contributing,omitempty" yaml:"contributing,omitempty"`
	Codeowners    *PathRef `json:"codeowners,omitempty" yaml:"codeowners,omitempty"`
	GovernanceDoc *PathRef `json:"governance_doc,omitempty" yaml:"governance_doc,omitempty"`
	GitVoteConfig *PathRef `json:"gitvote_config,omitempty" yaml:"gitvote_config,omitempty"`
	// Governance DD items - link to relevant documentation for each item.
	VendorNeutralityStatement *PathRef            `json:"vendor_neutrality_statement,omitempty" yaml:"vendor_neutrality_statement,omitempty"` // Accuracy and Clarity - Incubating: Suggested | Graduated: Required
	DecisionMakingProcess     *PathRef            `json:"decision_making_process,omitempty" yaml:"decision_making_process,omitempty"`         // Decisions and Role Assignments - Incubating: Suggested | Graduated: Required
	RolesAndTeams             *PathRef            `json:"roles_and_teams,omitempty" yaml:"roles_and_teams,omitempty"`                         // Decisions and Role Assignments - Incubating: Suggested | Graduated: Required
	CodeOfConduct             *PathRef            `json:"code_of_conduct,omitempty" yaml:"code_of_conduct,omitempty"`                         // Code of Conduct - Incubating: Required | Graduated: Required
	SubProjectList            *PathRef            `json:"sub_project_list,omitempty" yaml:"sub_project_list,omitempty"`                       // (If used) Subprojects - Incubating: Required | Graduated: Required
	SubProjectDocs            *PathRef            `json:"sub_project_docs,omitempty" yaml:"sub_project_docs,omitempty"`                       // (If used) Subprojects - Incubating: Suggested | Graduated: Required
	ContributorLadder         *PathRef            `json:"contributor_ladder,omitempty" yaml:"contributor_ladder,omitempty"`                   // Contributors and Community - Incubating: Suggested | Graduated: Suggested
	ChangeProcess             *PathRef            `json:"change_process,omitempty" yaml:"change_process,omitempty"`                           // Contributors and Community - Incubating: Required | Graduated: Required
	CommsChannels             *PathRef            `json:"comms_channels,omitempty" yaml:"comms_channels,omitempty"`                           // Contributors and Community - Incubating: Required | Graduated: Required
	CommunityCalendar         *PathRef            `json:"community_calendar,omitempty" yaml:"community_calendar,omitempty"`                   // Contributors and Community - Incubating: Required | Graduated: Required
	ContributorGuide          *PathRef            `json:"contributor_guide,omitempty" yaml:"contributor_guide,omitempty"`                     // Contributors and Community - Incubating: Required | Graduated: Required
	MaintainerLifecycle       MaintainerLifecycle `json:"maintainer_lifecycle,omitempty" yaml:"maintainer_lifecycle,omitempty"`
}

type LegalConfig struct {
	License      *PathRef      `json:"license,omitempty" yaml:"license,omitempty"`
	IdentityType *IdentityType `json:"identity_type,omitempty" yaml:"identity_type,omitempty"`
}

// IdentityType represents a project's contributor identity agreement (DCO, CLA, or none)
type IdentityType struct {
	Type string   `json:"type" yaml:"type"`                   // Required: "dco", "cla", or "none"
	URL  *PathRef `json:"url,omitempty" yaml:"url,omitempty"` // Optional link to the DCO/CLA document
}

// ValidIdentityTypes lists the allowed identity type values
var ValidIdentityTypes = map[string]bool{
	"dco":  true,
	"cla":  true,
	"none": true,
}

type DocumentationConfig struct {
	Readme       *PathRef `json:"readme,omitempty" yaml:"readme,omitempty"`
	Support      *PathRef `json:"support,omitempty" yaml:"support,omitempty"`
	Architecture *PathRef `json:"architecture,omitempty" yaml:"architecture,omitempty"`
	API          *PathRef `json:"api,omitempty" yaml:"api,omitempty"`
}

type MaturityEntry struct {
	Phase string    `json:"phase" yaml:"phase"`
	Date  time.Time `json:"date" yaml:"date"`
	Issue string    `json:"issue" yaml:"issue"`
}

type Audit struct {
	Date time.Time `json:"date" yaml:"date"` // Date of the audit
	Type string    `json:"type" yaml:"type"` // Type of audit (e.g., "security", "performance")
	URL  string    `json:"url" yaml:"url"`   // URL to the audit report
}

// MaintainersConfig represents the maintainers configuration file
type MaintainersConfig struct {
	Maintainers []MaintainerEntry `json:"maintainers" yaml:"maintainers"`
}

// MaintainerEntry represents maintainers for a single project
type MaintainerEntry struct {
	ProjectID string `json:"project_id" yaml:"project_id"`
	Org       string `json:"org,omitempty" yaml:"org,omitempty"`
	Teams     []Team `json:"teams" yaml:"teams"`
}

// Team represents a GitHub team and its members
type Team struct {
	Name    string   `json:"name" yaml:"name"`
	Members []string `json:"members" yaml:"members"`
}

// MaintainerLifecycle represents maintainer lifecycle documentation
type MaintainerLifecycle struct {
	OnboardingDoc     *PathRef `json:"onboarding_doc,omitempty" yaml:"onboarding_doc,omitempty"`
	ProgressionLadder *PathRef `json:"progression_ladder,omitempty" yaml:"progression_ladder,omitempty"`
	MentoringProgram  []string `json:"mentoring_program,omitempty" yaml:"mentoring_program,omitempty"` // Array of URLs
	OffboardingPolicy *PathRef `json:"offboarding_policy,omitempty" yaml:"offboarding_policy,omitempty"`
}

// MaintainerValidationResult captures validation results for maintainers
type MaintainerValidationResult struct {
	ProjectID             string   `json:"project_id" yaml:"project_id"`
	Org                   string   `json:"org,omitempty" yaml:"org,omitempty"`
	Valid                 bool     `json:"valid" yaml:"valid"`
	Errors                []string `json:"errors,omitempty" yaml:"errors,omitempty"`
	VerificationAttempted bool     `json:"verification_attempted" yaml:"verification_attempted"`
	VerificationPassed    bool     `json:"verification_passed" yaml:"verification_passed"`
	VerifiedHandles       []string `json:"verified_handles,omitempty" yaml:"verified_handles,omitempty"`
}

// Config represents the validator configuration
type Config struct {
	ProjectListURL string `yaml:"project_list_url"`
	CacheDir       string `yaml:"cache_dir"`
	OutputFormat   string `yaml:"output_format"` // json, yaml, text
}

// ValidationResult represents the result of validating a project
type ValidationResult struct {
	URL          string    `json:"url"`
	ProjectName  string    `json:"project_name,omitempty"`
	Valid        bool      `json:"valid"`
	Errors       []string  `json:"errors,omitempty"`
	Changed      bool      `json:"changed"`
	LastChecked  time.Time `json:"last_checked"`
	PreviousHash string    `json:"previous_hash,omitempty"`
	CurrentHash  string    `json:"current_hash"`
}

// CacheEntry represents cached project data
type CacheEntry struct {
	URL         string    `json:"url"`
	Hash        string    `json:"hash"`
	LastChecked time.Time `json:"last_checked"`
	Content     string    `json:"content"`
}

// Cache manages cached project data
type Cache struct {
	Entries map[string]CacheEntry `json:"entries"`
	dir     string
}

// ProjectValidator validates remote project YAML files
type ProjectValidator struct {
	config *Config
	cache  *Cache
	client *http.Client
}

// ProjectListEntry represents a single entry in the project list
type ProjectListEntry struct {
	URL string `json:"url" yaml:"url"`
	ID  string `json:"id,omitempty" yaml:"id,omitempty"`
}

// ProjectListConfig represents the structure of the project list file
type ProjectListConfig struct {
	Projects []ProjectListEntry `json:"projects" yaml:"projects"`
}
