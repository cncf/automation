package projects

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AuditResult represents the result of checking a project's governance/security references
type AuditResult struct {
	ProjectSlug string       `json:"project_slug"`
	Checks      []AuditCheck `json:"checks"`
	PassCount   int          `json:"pass_count"`
	FailCount   int          `json:"fail_count"`
	SkipCount   int          `json:"skip_count"`
}

// AuditCheck represents a single URL accessibility check
type AuditCheck struct {
	Field      string `json:"field"`
	URL        string `json:"url"`
	Status     string `json:"status"` // "pass", "fail", "skip"
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AuditProject checks that all referenced URLs in a project are accessible
func AuditProject(project Project, client *http.Client) AuditResult {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	result := AuditResult{
		ProjectSlug: project.Slug,
	}

	// Collect all URLs to check
	urlChecks := collectProjectURLs(project)

	for _, check := range urlChecks {
		if check.URL == "" {
			check.Status = "skip"
			result.SkipCount++
		} else {
			resp, err := client.Head(check.URL)
			if err != nil {
				check.Status = "fail"
				check.Error = err.Error()
				result.FailCount++
			} else {
				resp.Body.Close()
				check.StatusCode = resp.StatusCode
				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					check.Status = "pass"
					result.PassCount++
				} else {
					check.Status = "fail"
					check.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
					result.FailCount++
				}
			}
		}
		result.Checks = append(result.Checks, check)
	}

	return result
}

// collectProjectURLs gathers all URL references from a project for checking
func collectProjectURLs(project Project) []AuditCheck {
	var checks []AuditCheck

	// Website
	if project.Website != "" {
		checks = append(checks, AuditCheck{Field: "website", URL: project.Website})
	}

	// Artwork
	if project.Artwork != "" {
		checks = append(checks, AuditCheck{Field: "artwork", URL: project.Artwork})
	}

	// Repositories
	for i, repo := range project.Repositories {
		checks = append(checks, AuditCheck{Field: fmt.Sprintf("repositories[%d]", i), URL: repo})
	}

	// Audit report URLs
	for i, audit := range project.Audits {
		checks = append(checks, AuditCheck{Field: fmt.Sprintf("audits[%d].url", i), URL: audit.URL})
	}

	// Security paths (if they look like URLs)
	if project.Security != nil {
		if project.Security.Policy != nil && isHTTPURL(project.Security.Policy.Path) {
			checks = append(checks, AuditCheck{Field: "security.policy.path", URL: project.Security.Policy.Path})
		}
		if project.Security.ThreatModel != nil && isHTTPURL(project.Security.ThreatModel.Path) {
			checks = append(checks, AuditCheck{Field: "security.threat_model.path", URL: project.Security.ThreatModel.Path})
		}
	}

	// Governance paths
	if project.Governance != nil {
		if project.Governance.Contributing != nil && isHTTPURL(project.Governance.Contributing.Path) {
			checks = append(checks, AuditCheck{Field: "governance.contributing.path", URL: project.Governance.Contributing.Path})
		}
		if project.Governance.GovernanceDoc != nil && isHTTPURL(project.Governance.GovernanceDoc.Path) {
			checks = append(checks, AuditCheck{Field: "governance.governance_doc.path", URL: project.Governance.GovernanceDoc.Path})
		}
	}

	// Documentation paths
	if project.Documentation != nil {
		if project.Documentation.Readme != nil && isHTTPURL(project.Documentation.Readme.Path) {
			checks = append(checks, AuditCheck{Field: "documentation.readme.path", URL: project.Documentation.Readme.Path})
		}
	}

	return checks
}

// isHTTPURL checks if a string looks like an HTTP URL (as opposed to a relative path)
func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// FormatAuditResult formats an audit result as human-readable text
func FormatAuditResult(result AuditResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Governance & Security Audit: %s\n", result.ProjectSlug))
	b.WriteString(strings.Repeat("=", 40))
	b.WriteString("\n\n")

	for _, check := range result.Checks {
		var icon string
		switch check.Status {
		case "pass":
			icon = "OK"
		case "fail":
			icon = "FAIL"
		case "skip":
			icon = "SKIP"
		}
		b.WriteString(fmt.Sprintf("  [%s] %s: %s", icon, check.Field, check.URL))
		if check.Error != "" {
			b.WriteString(fmt.Sprintf(" (%s)", check.Error))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\nSummary: %d passed, %d failed, %d skipped\n",
		result.PassCount, result.FailCount, result.SkipCount))
	return b.String()
}
