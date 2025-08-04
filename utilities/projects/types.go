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
