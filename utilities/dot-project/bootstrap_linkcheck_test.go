package projects

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPruneBrokenLinks(t *testing.T) {
	doc := `maturity_log:
  - phase: "sandbox"
    date: "2024-01-01T00:00:00Z"
    issue: "https://github.com/cncf/toc/issues/1"

repositories:
  - url: "https://github.com/acme/core"
    primary: true
  - url: "https://github.com/acme/gone"
    # tags: [x]

website: "https://acme.io"

security:
  policy:
    path: "https://github.com/acme/core/blob/main/SECURITY.md"
  contact:
    advisory_url: "https://github.com/acme/core/security/advisories/new"
`
	broken := map[string]string{
		"https://github.com/cncf/toc/issues/1":               "404 Not Found",
		"https://github.com/acme/gone":                       "404 Not Found",
		"https://acme.io":                                    "410 Gone",
		"https://github.com/acme/core/blob/main/SECURITY.md": "404 Not Found",
	}
	got := PruneBrokenLinks(doc, broken)

	for _, want := range []string{
		`    issue: "https://github.com/cncf/toc/issues/XXX" # TODO: https://github.com/cncf/toc/issues/1 returned 404 Not Found — set the TOC issue URL`,
		"  # TODO: https://github.com/acme/gone returned 404 Not Found — fix or replace this repository URL\n  - url: \"https://github.com/acme/gone\"\n    # tags: [x]",
		"# TODO: https://acme.io returned 410 Gone — add the correct link\n# website: \"https://acme.io\"",
		"  # TODO: https://github.com/acme/core/blob/main/SECURITY.md returned 404 Not Found — add the correct link\n  # policy:\n  #   path: \"https://github.com/acme/core/blob/main/SECURITY.md\"\n  contact:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing:\n%s\n\ngot:\n%s", want, got)
		}
	}

	var p Project
	if err := yaml.Unmarshal([]byte(got), &p); err != nil {
		t.Fatalf("pruned YAML does not parse: %v\n%s", err, got)
	}
	if len(p.Repositories) != 2 || p.Website != "" || p.Security.Policy != nil || p.Security.Contact == nil {
		t.Errorf("unexpected parse result: %+v", p)
	}
	if errs := validateProjectStruct(p); containsSubstring(errs, "security.policy") || containsSubstring(errs, "maturity_log") {
		t.Errorf("pruned YAML introduced validation errors: %v", errs)
	}
}

// A broken repository must never be removed: repositories is required, so
// pruning the only entry would produce a project.yaml that fails validation.
func TestPruneBrokenLinksKeepsOnlyRepository(t *testing.T) {
	r := &BootstrapResult{
		Name: "Acme", Slug: "acme", Description: "d", GitHubOrg: "acme", DefaultBranch: "main",
		Repositories: []string{"https://github.com/acme/acme"},
		PrimaryRepo:  "https://github.com/acme/acme",
		BrokenLinks:  map[string]string{"https://github.com/acme/acme": "404 Not Found"},
	}
	out, err := GenerateProjectYAML(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "# TODO: https://github.com/acme/acme returned 404 Not Found — fix or replace this repository URL") {
		t.Errorf("broken repository not flagged:\n%s", out)
	}
	var p Project
	if err := yaml.Unmarshal(out, &p); err != nil {
		t.Fatalf("pruned YAML does not parse: %v", err)
	}
	if errs := validateProjectStruct(p); containsSubstring(errs, "repositor") {
		t.Errorf("pruned YAML introduced repository errors: %v", errs)
	}
}

func containsSubstring(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func TestCheckBootstrapLinksPrunesAndDropsUnknownHandles(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.Contains(p, "SECURITY.md"),
			strings.HasSuffix(p, "/users/ghost"):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	c := NewLinkChecker("")
	c.Client = srv.Client()
	c.APIBase = srv.URL + "/api"
	c.WebBase = srv.URL + "/web"
	c.RecheckDelay = 0

	result := &BootstrapResult{
		Slug:          "acme",
		Name:          "Acme",
		GitHubOrg:     "acme",
		GitHubRepo:    "core",
		DefaultBranch: "main",
		Repositories:  []string{"https://github.com/acme/core"},
		Maintainers:   []string{"ghost", "alice"},
		ProjectLead:   "ghost",
		Sources:       map[string]string{"project_lead": "foundation-csv"},
	}
	if _, err := CheckBootstrapLinks(context.Background(), result, c); err != nil {
		t.Fatal(err)
	}

	if _, ok := result.BrokenLinks["https://github.com/acme/core/blob/main/SECURITY.md"]; !ok {
		t.Fatalf("guessed SECURITY.md not flagged: %v", result.BrokenLinks)
	}
	if len(result.Maintainers) != 1 || result.Maintainers[0] != "alice" || result.ProjectLead != "alice" {
		t.Errorf("unknown handle not removed: maintainers=%v lead=%q", result.Maintainers, result.ProjectLead)
	}

	py, err := GenerateProjectYAML(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(py), "# TODO: https://github.com/acme/core/blob/main/SECURITY.md returned 404 Not Found") {
		t.Errorf("project.yaml not pruned:\n%s", py)
	}
	var p Project
	if err := yaml.Unmarshal(py, &p); err != nil {
		t.Fatalf("pruned project.yaml does not parse: %v", err)
	}

	my, err := GenerateMaintainersYAML(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(my), "# - ghost") || strings.Contains(string(my), "\n          - ghost") {
		t.Errorf("maintainers.yaml should list ghost only as a comment:\n%s", my)
	}

	sec, err := tmplGen("security", securityMDTemplate, result)()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sec), "SECURITY.md") {
		t.Errorf("SECURITY.md should not link to the missing policy:\n%s", sec)
	}
}
