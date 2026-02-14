package projects

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectProjectURLs(t *testing.T) {
	project := validBaseProject()
	project.Website = "https://test-project.io"
	project.Artwork = "https://github.com/cncf/artwork/tree/master/projects/test"
	project.Security = &SecurityConfig{
		Policy: &PathRef{Path: "https://github.com/test/repo/blob/main/SECURITY.md"},
	}
	project.Governance = &GovernanceConfig{
		Contributing: &PathRef{Path: "CONTRIBUTING.md"}, // relative path, should NOT be checked
	}

	checks := collectProjectURLs(project)

	// Should include: website, artwork, 1 repo, security.policy
	// Should NOT include governance.contributing (relative path)
	hasSecurityPolicy := false
	hasContributing := false
	for _, c := range checks {
		if c.Field == "security.policy.path" {
			hasSecurityPolicy = true
		}
		if c.Field == "governance.contributing.path" {
			hasContributing = true
		}
	}
	if !hasSecurityPolicy {
		t.Error("expected security.policy.path in checks")
	}
	if hasContributing {
		t.Error("did not expect governance.contributing.path (relative path) in checks")
	}
}

func TestAuditProject(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/not-found" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	project := validBaseProject()
	project.Website = server.URL + "/ok"
	project.Repositories = []string{server.URL + "/repo"}
	project.Artwork = server.URL + "/not-found"

	result := AuditProject(project, server.Client())
	if result.PassCount == 0 {
		t.Error("expected at least one pass")
	}
	if result.FailCount == 0 {
		t.Error("expected at least one fail (artwork returns 404)")
	}
}

func TestFormatAuditResult(t *testing.T) {
	result := AuditResult{
		ProjectSlug: "test",
		Checks: []AuditCheck{
			{Field: "website", URL: "https://example.com", Status: "pass", StatusCode: 200},
			{Field: "artwork", URL: "https://example.com/missing", Status: "fail", Error: "HTTP 404"},
		},
		PassCount: 1,
		FailCount: 1,
	}

	output := FormatAuditResult(result)
	if !strings.Contains(output, "[OK]") {
		t.Error("expected [OK] in output")
	}
	if !strings.Contains(output, "[FAIL]") {
		t.Error("expected [FAIL] in output")
	}
	if !strings.Contains(output, "1 passed, 1 failed") {
		t.Error("expected summary counts in output")
	}
}
