package projects

import (
	"fmt"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

// StringOrSlice is a YAML/JSON type that accepts both a plain string scalar
// and a sequence of strings. Single-element values round-trip as a plain
// string for backward-compatible YAML output; multi-element values serialize
// as a YAML list.
//
// Used for:
//   - project_lead  — supports multiple leads
//   - package_managers values — supports multiple images/packages per registry
type StringOrSlice []string

// UnmarshalYAML allows project.yaml files to use either form:
//
//	project_lead: "jdoe"          # scalar (backward-compatible)
//	project_lead:                 # list
//	  - "jdoe"
//	  - "jsmith"
func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*s = StringOrSlice{value.Value}
		return nil
	case yaml.SequenceNode:
		var ss []string
		if err := value.Decode(&ss); err != nil {
			return err
		}
		*s = ss
		return nil
	default:
		return fmt.Errorf("expected string or sequence, got YAML node type %v", value.Tag)
	}
}

// MarshalYAML serializes a single-element StringOrSlice as a plain string
// scalar (readable, backward-compatible). Multi-element values serialize as a
// YAML sequence. An empty slice serializes as nil (omitempty will drop it).
func (s StringOrSlice) MarshalYAML() (interface{}, error) {
	switch len(s) {
	case 0:
		return nil, nil
	case 1:
		return s[0], nil
	default:
		return []string(s), nil
	}
}

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
	// PackageManagers maps registry names to one or more identifiers.
	// A single identifier may be given as a plain string; multiple identifiers
	// (e.g., multi-arch Docker images) may be given as a list.
	PackageManagers map[string]StringOrSlice `json:"package_managers,omitempty" yaml:"package_managers,omitempty"`

	// Schema and identification
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`
	Type          string `json:"type,omitempty" yaml:"type,omitempty"`
	Slug          string `json:"slug" yaml:"slug"` // Unique project identifier (lowercase, alphanumeric + hyphens)

	// Contacts and channels
	// ProjectLeads holds one or more GitHub handles or GitHub team references
	// (org/team-name) for the project lead(s). Accepts a plain string (single
	// lead, backward-compatible) or a YAML list (multiple leads).
	ProjectLeads StringOrSlice `json:"project_lead,omitempty" yaml:"project_lead,omitempty"`

	// SlackChannels lists one or more Slack channels for the project.
	// Mark the channel most end-users should join with primary: true.
	SlackChannels []SlackChannel `json:"slack_channels,omitempty" yaml:"slack_channels,omitempty"`

	// Governance, security, legal, documentation
	Security      *SecurityConfig      `json:"security,omitempty" yaml:"security,omitempty"`
	Governance    *GovernanceConfig    `json:"governance,omitempty" yaml:"governance,omitempty"`
	Legal         *LegalConfig         `json:"legal,omitempty" yaml:"legal,omitempty"`
	Documentation *DocumentationConfig `json:"documentation,omitempty" yaml:"documentation,omitempty"`

	// CNCF Landscape integration
	Landscape *LandscapeConfig `json:"landscape,omitempty" yaml:"landscape,omitempty"`
}

// SlackChannel represents a single Slack channel for a project.
// Projects may list one or more channels; the channel that most end-users
// should join is marked with Primary: true.
type SlackChannel struct {
	Workspace string `json:"workspace,omitempty" yaml:"workspace,omitempty"` // Slack workspace identifier (e.g., "cncf")
	Link      string `json:"link,omitempty" yaml:"link,omitempty"`           // Invite or channel URL
	Name      string `json:"name" yaml:"name"`                               // Channel name (e.g., "#kubernetes-dev")
	Primary   bool   `json:"primary,omitempty" yaml:"primary,omitempty"`     // Whether this is the primary channel for end-users
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
	Policy      *PathRef         `json:"policy" yaml:"policy"`
	ThreatModel *PathRef         `json:"threat_model,omitempty" yaml:"threat_model,omitempty"`
	Contact     *SecurityContact `json:"contact,omitempty" yaml:"contact,omitempty"`
}

// SecurityContact represents security contact information.
// At least one of Email or AdvisoryURL must be set when the section is present.
type SecurityContact struct {
	Email       string `json:"email,omitempty" yaml:"email,omitempty"`
	AdvisoryURL string `json:"advisory_url,omitempty" yaml:"advisory_url,omitempty"`
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

// IdentityType represents a project's contributor identity agreements.
// DCO can be used alone, or DCO + CLA together. By default, CLA requires DCO.
// Some projects have an exception to use CLA without DCO; set cla_only: true for those.
type IdentityType struct {
	HasDCO  bool     `json:"has_dco" yaml:"has_dco"`                       // Whether the project uses DCO
	HasCLA  bool     `json:"has_cla" yaml:"has_cla"`                       // Whether the project uses CLA (requires DCO unless cla_only is true)
	CLAOnly bool     `json:"cla_only,omitempty" yaml:"cla_only,omitempty"` // Exception: allows CLA without DCO
	DCOURL  *PathRef `json:"dco_url,omitempty" yaml:"dco_url,omitempty"`   // Optional link to DCO document
	CLAURL  *PathRef `json:"cla_url,omitempty" yaml:"cla_url,omitempty"`   // Optional link to CLA document
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
