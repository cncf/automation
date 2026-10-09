package projects

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func newTestChecker(srv *httptest.Server) *LinkChecker {
	c := NewLinkChecker("tok")
	c.Client = srv.Client()
	c.APIBase = srv.URL + "/api"
	c.WebBase = srv.URL + "/web"
	c.RecheckDelay = 0
	return c
}

func TestCollectProjectLinksResolvesRelativePaths(t *testing.T) {
	p := Project{
		Website:      "https://example.io",
		Repositories: []RepositoryEntry{{URL: "https://github.com/acme/other"}, {URL: "https://github.com/acme/core", Primary: true}},
		MailingLists: []string{"dev@lists.acme.io", "https://lists.acme.io/g/dev"},
		Security:     &SecurityConfig{Policy: &PathRef{Path: "SECURITY.md"}},
		Governance:   &GovernanceConfig{Contributing: &PathRef{Path: "https://github.com/acme/core/blob/main/CONTRIBUTING.md"}},
		MaturityLog:  []MaturityEntry{{Issue: "https://github.com/cncf/toc/issues/XXX"}},
	}
	links := CollectProjectLinks(p, "project.yaml")

	byField := map[string]Link{}
	for _, l := range links {
		byField[l.Field] = l
	}
	if got := byField["security.policy"].Resolved; got != "https://github.com/acme/core/blob/HEAD/SECURITY.md" {
		t.Errorf("relative path resolved to %q", got)
	}
	if _, ok := byField["mailing_lists[0]"]; ok {
		t.Error("bare mailing-list address should not be collected")
	}
	if _, ok := byField["mailing_lists[1]"]; !ok {
		t.Error("mailing-list URL should be collected")
	}
	if byField["maturity_log[0].issue"].Note == "" {
		t.Error("XXX placeholder should carry a note")
	}
	if byField["governance.contributing"].Resolved != p.Governance.Contributing.Path {
		t.Error("absolute PathRef should be checked as-is")
	}
}

func TestCollectProjectLinksRelativeWithoutGitHubRepo(t *testing.T) {
	p := Project{
		Repositories: []RepositoryEntry{{URL: "https://gitlab.com/acme/core"}},
		Adopters:     &PathRef{Path: "ADOPTERS.md"},
	}
	links := CollectProjectLinks(p, "project.yaml")
	if len(links) != 2 || links[1].Note == "" {
		t.Fatalf("expected an unresolvable-path note, got %+v", links)
	}
}

func TestCollectMaintainerLinks(t *testing.T) {
	cfg := MaintainersConfig{Maintainers: []MaintainerEntry{{
		ProjectID: "core",
		Teams:     []Team{{Name: "maintainers", Members: []string{"@alice", "github-handle", "bad handle"}}},
	}}}
	links := CollectMaintainerLinks(cfg, "maintainers.yaml")
	if len(links) != 3 {
		t.Fatalf("got %d links", len(links))
	}
	if links[0].Resolved != "https://github.com/alice" {
		t.Errorf("handle resolved to %q", links[0].Resolved)
	}
	if links[1].Note == "" {
		t.Error("placeholder handle should carry a note")
	}
	if links[2].Resolved != "bad handle" {
		t.Errorf("invalid handle should be checked raw, got %q", links[2].Resolved)
	}
}

func TestIsPlaceholder(t *testing.T) {
	for v, want := range map[string]bool{
		"https://github.com/cncf/toc/issues/XXX":      true,
		"TODO: add link":                              true,
		"https://example.com/x":                       true,
		"github-handle":                               true,
		"https://github.com/acme/todomvc":             false,
		"https://github.com/acme/todo-app":            false,
		"https://acme.io/docs/todo":                   false,
		"https://github.com/acme/core/blob/main/x.md": false,
	} {
		if got := isPlaceholder(v); got != want {
			t.Errorf("isPlaceholder(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestLinkCheckerGitHubMapping(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" && strings.HasPrefix(r.URL.Path, "/api") {
			t.Errorf("missing token on %s", r.URL)
		}
		switch r.URL.RequestURI() {
		case "/api/repos/acme/core",
			"/api/repos/acme/core/contents/docs/SECURITY.md?ref=main",
			"/api/repos/acme/core/issues/12",
			"/api/users/alice",
			"/web/acme/core/blob/master/README.md":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := newTestChecker(srv)

	cases := map[string]LinkStatus{
		"https://github.com/acme/core":                            LinkOK,
		"https://github.com/acme/core/blob/main/docs/SECURITY.md": LinkOK,
		"https://github.com/acme/core/pull/12":                    LinkOK,
		"https://github.com/acme/core/security/advisories/new":    LinkOK,
		"https://github.com/alice":                                LinkOK,
		"https://github.com/acme/core/blob/master/README.md":      LinkOK, // renamed branch: web fallback
		"https://github.com/acme/core/blob/main/MISSING.md":       LinkBroken,
		"https://github.com/acme/gone":                            LinkBroken,
		"http://github.com/acme/core":                             LinkWarning,
	}
	var links []Link
	for u := range cases {
		links = append(links, Link{URL: u, Resolved: u})
	}
	for _, r := range c.Check(context.Background(), links) {
		if r.Status != cases[r.URL] {
			t.Errorf("%s: got %v (%s), want %v", r.URL, r.Status, r.Detail, cases[r.URL])
		}
	}
}

func TestLinkCheckerWebStatuses(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/head-unsupported":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/blocked":
			w.WriteHeader(http.StatusForbidden)
		case "/gone":
			w.WriteHeader(http.StatusGone)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := newTestChecker(srv)
	// The test server is 127.0.0.1, which has dots, so it is checked as a web link.
	cases := map[string]LinkStatus{
		"/ok":               LinkOK,
		"/head-unsupported": LinkOK,
		"/blocked":          LinkWarning,
		"/error":            LinkWarning,
		"/gone":             LinkBroken,
		"/missing":          LinkBroken,
	}
	var links []Link
	for p := range cases {
		links = append(links, Link{URL: p, Resolved: srv.URL + p})
	}
	for _, r := range c.Check(context.Background(), links) {
		if want := cases[r.URL]; r.Status != want {
			t.Errorf("%s: got %v (%s), want %v", r.URL, r.Status, r.Detail, want)
		}
	}
}

func TestLinkCheckerRechecksBeforeReportingBroken(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && calls.Add(1) == 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestChecker(srv)
	c.Client.Transport = http.DefaultTransport

	res := c.Check(context.Background(), []Link{{URL: "x", Resolved: srv.URL + "/flaky"}})
	// http:// is downgraded to a warning, but it must not be broken.
	if res[0].Status == LinkBroken {
		t.Fatalf("transient 404 reported as broken: %+v", res[0])
	}
}

func TestLinkCheckerInvalidValues(t *testing.T) {
	c := NewLinkChecker("")
	res := c.Check(context.Background(), []Link{
		{URL: "htps://github.com/acme", Resolved: "htps://github.com/acme"},
		{URL: "bad handle", Resolved: "bad handle"},
		{URL: "https://github.com/cncf/toc/issues/XXX", Note: "placeholder has not been replaced"},
	})
	if res[0].Status != LinkBroken || res[1].Status != LinkBroken {
		t.Errorf("invalid values should be broken: %+v", res[:2])
	}
	if res[2].Status != LinkWarning {
		t.Errorf("placeholder should be a warning, got %v", res[2].Status)
	}
}

func TestLinkCheckerDedupes(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := newTestChecker(srv)
	u := srv.URL + "/same"
	c.Check(context.Background(), []Link{{URL: u, Resolved: u}, {URL: u, Resolved: u}, {URL: u, Resolved: u}})
	if n := calls.Load(); n != 1 {
		t.Errorf("expected 1 request, got %d", n)
	}
}

func TestLocalProjectFiles(t *testing.T) {
	for _, p := range []string{"", "/dev/null"} {
		if got, err := LocalProjectFiles(p); err != nil || got != nil {
			t.Errorf("LocalProjectFiles(%q) = %v, %v; want nil, nil", p, got, err)
		}
	}

	t.Setenv("LC_ROOT", "/repo")
	list := filepath.Join(t.TempDir(), "list.yaml")
	data := "projects:\n" +
		"  - url: \"file://${LC_ROOT}/a/project.yaml\"\n" +
		"  - url: \"b/project.yaml\"\n" +
		"  - url: \"https://example.com/project.yaml\"\n"
	if err := os.WriteFile(list, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LocalProjectFiles(list)
	if err != nil {
		t.Fatal(err)
	}
	// Remote entries are skipped; plain paths count as files, as in the validator.
	want := []string{"/repo/a/project.yaml", "b/project.yaml"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := LocalProjectFiles(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected an error for a missing list")
	}
	if err := os.WriteFile(list, []byte("projects: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LocalProjectFiles(list); err == nil {
		t.Error("expected an error for an unparsable list")
	}
}

func TestExcludeKnownLinks(t *testing.T) {
	cur := []Link{{URL: "a"}, {URL: "b"}, {URL: "c"}}
	base := []Link{{URL: "a"}, {URL: "c"}}
	got := ExcludeKnownLinks(cur, base)
	if len(got) != 1 || got[0].URL != "b" {
		t.Errorf("got %+v", got)
	}
}

func TestCollectRepoLinksMultiProject(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("org.yaml", "projects:\n  - id: alpha\n")
	write("alpha/project.yaml", "name: Alpha\nwebsite: https://alpha.io\nrepositories:\n  - https://github.com/acme/alpha\n")
	write("alpha/maintainers.yaml", "maintainers:\n  - project_id: alpha\n    teams:\n      - name: m\n        members: [bob]\n")

	links, err := CollectRepoLinks(root)
	if err != nil {
		t.Fatal(err)
	}
	sources := map[string]bool{}
	for _, l := range links {
		sources[l.Source] = true
	}
	if !sources["alpha/project.yaml"] || !sources["alpha/maintainers.yaml"] {
		t.Errorf("unexpected sources %v", sources)
	}
}

func TestFormatLinkOutputs(t *testing.T) {
	results := []LinkResult{
		{Link: Link{Source: "a/project.yaml", Field: "website", URL: "https://x.io|y"}, Status: LinkBroken, Detail: "404 Not Found"},
		{Link: Link{Source: "a/project.yaml", Field: "artwork", URL: "https://z.io"}, Status: LinkWarning, Detail: "could not be confirmed (403 Forbidden)"},
		{Link: Link{Source: "a/project.yaml", Field: "repositories[0]", URL: "https://github.com/a/b"}, Status: LinkOK},
	}
	md := FormatLinkReportMarkdown(results)
	if !strings.Contains(md, `https://x.io\|y`) || !strings.Contains(md, "### Broken") || !strings.Contains(md, "### Warnings") {
		t.Errorf("markdown missing content:\n%s", md)
	}
	ann := FormatLinkAnnotations(results)
	if !strings.Contains(ann, "::error file=a/project.yaml,title=website::") || strings.Contains(ann, "repositories[0]") {
		t.Errorf("annotations wrong:\n%s", ann)
	}
	if !strings.Contains(FormatLinkResultsText(results), "1 broken, 1 warnings") {
		t.Error("text summary wrong")
	}
}
