package projects

import "time"

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
	Security      *SecurityConfig      `json:"security,omitempty" yaml:"security,omitempty"`
	Governance    *GovernanceConfig    `json:"governance,omitempty" yaml:"governance,omitempty"`
	Legal         *LegalConfig         `json:"legal,omitempty" yaml:"legal,omitempty"`
	Documentation *DocumentationConfig `json:"documentation,omitempty" yaml:"documentation,omitempty"`
}

type PathRef struct {
	Path string `json:"path" yaml:"path"`
}

type SecurityConfig struct {
	Policy      *PathRef `json:"policy,omitempty" yaml:"policy,omitempty"`
	ThreatModel *PathRef `json:"threat_model,omitempty" yaml:"threat_model,omitempty"`
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
