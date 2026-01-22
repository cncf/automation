package projects

import (
	"net/http"
	"time"
)

// SchemaVersionWithExtensions is the minimum version that supports extensions
const SchemaVersionWithExtensions = "1.1.0"

type ProjectList []string

type Project struct {
	Name         string            `json:"name" yaml:"name"`
	Description  string            `json:"description" yaml:"description"`
	MaturityLog  []MaturityEntry   `json:"maturity_log" yaml:"maturity_log"`
	Repositories []string          `json:"repositories" yaml:"repositories"`
	Social       map[string]string `json:"social" yaml:"social"`
	Artwork      string            `json:"artwork" yaml:"artwork"`             // Artwork URL
	Website      string            `json:"website" yaml:"website"`             // Project website URL
	MailingLists []string          `json:"mailing_lists" yaml:"mailing_lists"` // Mailing lists for project
	Audits       []Audit           `json:"audits" yaml:"audits"`               // Security audits for project

	// New fields merged from project.toml
	SchemaVersion string               `json:"schema_version,omitempty" yaml:"schema_version,omitempty"`
	Type          string               `json:"type,omitempty" yaml:"type,omitempty"`
	Security      *SecurityConfig      `json:"security" yaml:"security"`
	Governance    *GovernanceConfig    `json:"governance,omitempty" yaml:"governance,omitempty"`
	Legal         *LegalConfig         `json:"legal,omitempty" yaml:"legal,omitempty"`
	Documentation *DocumentationConfig `json:"documentation,omitempty" yaml:"documentation,omitempty"`

	// Extensions for third-party tools (requires schema_version >= 1.1.0)
	Extensions map[string]Extension `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// Extension represents a third-party tool extension configuration
type Extension struct {
	// Metadata about the extension
	Metadata *ExtensionMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	// Tool-specific configuration (arbitrary key-value pairs)
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// ExtensionMetadata contains information about the extension provider
type ExtensionMetadata struct {
	Author     string `json:"author,omitempty" yaml:"author,omitempty"`
	Homepage   string `json:"homepage,omitempty" yaml:"homepage,omitempty"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	License    string `json:"license,omitempty" yaml:"license,omitempty"`
	Version    string `json:"version,omitempty" yaml:"version,omitempty"`
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
}

type LegalConfig struct {
	License *PathRef `json:"license,omitempty" yaml:"license,omitempty"`
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
	Maintainers []MaintainerEntry `yaml:"maintainers"`
}

// MaintainerEntry represents maintainers for a single project
type MaintainerEntry struct {
	ProjectID string `yaml:"project_id"`
	Org       string `yaml:"org,omitempty"`
	Teams     []Team `yaml:"teams"`
}

// Team represents a GitHub team and its members
type Team struct {
	Name    string   `yaml:"name"`
	Members []string `yaml:"members"`
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
