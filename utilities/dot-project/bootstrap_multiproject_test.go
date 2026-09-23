package projects

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const orgLandscapeYAML = `landscape:
  - category:
    name: Provisioning
    subcategories:
      - subcategory:
        name: Key Management
        items:
          - item:
            name: SPIFFE
            repo_url: https://github.com/spiffe/spiffe
            project: graduated
          - item:
            name: SPIRE
            repo_url: https://github.com/spiffe/spire
            project: graduated
          - item:
            name: Solo Project
            repo_url: https://github.com/solo/solo
            project: sandbox
          - item:
            name: Old Thing
            repo_url: https://github.com/spiffe/old-thing
            project: archived
          - item:
            name: Not A CNCF Project
            repo_url: https://github.com/spiffe/vendor-tool
          - item:
            name: Elsewhere
            repo_url: https://gitlab.com/spiffe/mirror
            project: sandbox
`

func landscapeServer(t *testing.T, body string) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write failed: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

func TestFindOrgProjects(t *testing.T) {
	client, url := landscapeServer(t, orgLandscapeYAML)

	t.Run("finds every active CNCF project in the org", func(t *testing.T) {
		got, err := FindOrgProjects("spiffe", client, url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var names []string
		for _, p := range got {
			names = append(names, p.Name)
		}
		// Sorted by name; the archived item, the non-CNCF item and the
		// non-GitHub item are all excluded.
		want := []string{"SPIFFE", "SPIRE"}
		if strings.Join(names, ",") != strings.Join(want, ",") {
			t.Fatalf("got %v, want %v", names, want)
		}

		if got[1].Slug != "spire" || got[1].Repo != "spire" || got[1].Maturity != "graduated" {
			t.Errorf("unexpected entry: %+v", got[1])
		}
	})

	t.Run("case-insensitive org match", func(t *testing.T) {
		got, err := FindOrgProjects("SPIFFE", client, url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(got))
		}
	})

	t.Run("single-project org yields one entry", func(t *testing.T) {
		got, err := FindOrgProjects("solo", client, url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// One entry means the caller keeps the legacy single-project layout.
		if len(got) != 1 {
			t.Fatalf("expected 1 project, got %d: %+v", len(got), got)
		}
	})

	t.Run("unknown org yields nothing", func(t *testing.T) {
		got, err := FindOrgProjects("nobody", client, url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no projects, got %+v", got)
		}
	})

	t.Run("empty org yields nothing without a request", func(t *testing.T) {
		got, err := FindOrgProjects("", nil, url)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})
}

func TestSplitGitHubRepoURL(t *testing.T) {
	cases := []struct {
		in       string
		org      string
		repo     string
		whyNotOK string
	}{
		{in: "https://github.com/spiffe/spire", org: "spiffe", repo: "spire"},
		{in: "http://github.com/spiffe/spire", org: "spiffe", repo: "spire"},
		{in: "github.com/spiffe/spire", org: "spiffe", repo: "spire"},
		{in: "git@github.com:spiffe/spire", org: "spiffe", repo: "spire"},
		{in: "https://github.com/spiffe/spire.git", org: "spiffe", repo: "spire"},
		{in: "https://github.com/spiffe/spire/", org: "spiffe", repo: "spire"},
		{in: "https://gitlab.com/spiffe/spire", whyNotOK: "not GitHub"},
		{in: "https://github.com/spiffe", whyNotOK: "org only, no repo"},
		{in: "", whyNotOK: "empty"},
	}

	for _, c := range cases {
		org, repo := splitGitHubRepoURL(c.in)
		if org != c.org || repo != c.repo {
			t.Errorf("%q: got (%q, %q), want (%q, %q) [%s]", c.in, org, repo, c.org, c.repo, c.whyNotOK)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"SPIFFE":              "spiffe",
		"Fluent Bit":          "fluent-bit",
		"OpenTelemetry":       "opentelemetry",
		"in-toto":             "in-toto",
		"Cloud  Custodian":    "cloud-custodian",
		"K8s (Kubernetes)!":   "k8s-kubernetes",
		"-leading-and-trail-": "leading-and-trail",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenerateOrgYAML(t *testing.T) {
	entries := []OrgProject{
		{Name: "SPIFFE", Slug: "spiffe", Maturity: "graduated"},
		{Name: "SPIRE", Slug: "spire", Maturity: "graduated"},
	}

	data, err := GenerateOrgYAML("spiffe", entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The generated index must be loadable by the very parser that decides
	// the repository layout, otherwise bootstrap would emit a repo that its
	// own validator rejects.
	dir := t.TempDir()
	path := filepath.Join(dir, OrgFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	cfg, err := LoadOrgFromFile(path)
	if err != nil {
		t.Fatalf("generated %s does not load: %v\n%s", OrgFileName, err, data)
	}
	if errs := ValidateOrgStruct(cfg); len(errs) > 0 {
		t.Fatalf("generated %s does not validate: %v\n%s", OrgFileName, errs, data)
	}
	if cfg.Org != "spiffe" || len(cfg.Projects) != 2 {
		t.Fatalf("unexpected parse: %+v", cfg)
	}
	if cfg.Projects[0].ID != "spiffe" || cfg.Projects[1].ID != "spire" {
		t.Errorf("unexpected project ids: %+v", cfg.Projects)
	}

	t.Run("refuses to generate an empty index", func(t *testing.T) {
		if _, err := GenerateOrgYAML("spiffe", nil); err == nil {
			t.Fatal("expected an error for an index with no projects")
		}
	})
}

func TestWriteMultiScaffold(t *testing.T) {
	entries := []OrgProject{
		{Name: "SPIFFE", Slug: "spiffe", Maturity: "graduated", Repo: "spiffe"},
		{Name: "SPIRE", Slug: "spire", Maturity: "graduated", Repo: "spire"},
	}
	newResults := func() []*BootstrapResult {
		return []*BootstrapResult{
			{Slug: "spiffe", Name: "SPIFFE", GitHubOrg: "spiffe", GitHubRepo: "spiffe", Sources: map[string]string{}},
			{Slug: "spire", Name: "SPIRE", GitHubOrg: "spiffe", GitHubRepo: "spire", Sources: map[string]string{}},
		}
	}

	t.Run("writes an index, one directory per project, and shared files", func(t *testing.T) {
		dir := t.TempDir()
		if err := WriteMultiScaffold(dir, "spiffe", entries, newResults()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, rel := range []string{
			OrgFileName,
			filepath.Join("spiffe", ProjectFileName),
			filepath.Join("spiffe", MaintainersFileName),
			filepath.Join("spire", ProjectFileName),
			filepath.Join("spire", MaintainersFileName),
			"README.md",
			".gitignore",
			filepath.Join(".github", "workflows", "validate.yaml"),
			filepath.Join(".github", "workflows", "update-landscape.yml"),
		} {
			if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
				t.Errorf("expected %s to exist: %v", rel, err)
			}
		}

		// Neither metadata file may exist at the root: that is exactly what
		// ValidateRepo rejects in a multi-project repository.
		for _, rel := range []string{ProjectFileName, MaintainersFileName} {
			if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
				t.Errorf("%s must not be written at the repository root", rel)
			}
		}
	})

	t.Run("the result validates end to end", func(t *testing.T) {
		dir := t.TempDir()
		if err := WriteMultiScaffold(dir, "spiffe", entries, newResults()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		d, err := Discover(dir)
		if err != nil {
			t.Fatalf("generated repository is not discoverable: %v", err)
		}
		if d.Mode != LayoutMulti {
			t.Fatalf("expected %v, got %v", LayoutMulti, d.Mode)
		}
		if len(d.Projects) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(d.Projects))
		}

		result, err := ValidateRepo(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Valid {
			t.Fatalf("generated repository does not validate: %v", result.Errors)
		}
	})

	t.Run("team names are namespaced per project", func(t *testing.T) {
		dir := t.TempDir()
		if err := WriteMultiScaffold(dir, "spiffe", entries, newResults()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// GitHub teams are org-scoped, so two projects in one org that both
		// scaffolded a team called "maintainers" would silently share it.
		for _, slug := range []string{"spiffe", "spire"} {
			data, err := os.ReadFile(filepath.Join(dir, slug, MaintainersFileName))
			if err != nil {
				t.Fatalf("read failed: %v", err)
			}
			want := "name: \"" + slug + "-maintainers\""
			if !strings.Contains(string(data), want) {
				t.Errorf("%s: expected %s, got:\n%s", slug, want, data)
			}
		}
	})

	t.Run("refuses to overwrite existing metadata", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "spire"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "spire", ProjectFileName), []byte("hand written\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}

		err := WriteMultiScaffold(dir, "spiffe", entries, newResults())
		if err == nil {
			t.Fatal("expected an error when a project.yaml already exists")
		}
		if !strings.Contains(err.Error(), "refusing to overwrite") {
			t.Errorf("unexpected error: %v", err)
		}

		// The existing file must be untouched.
		data, readErr := os.ReadFile(filepath.Join(dir, "spire", ProjectFileName))
		if readErr != nil || string(data) != "hand written\n" {
			t.Errorf("existing file was modified: %q (%v)", data, readErr)
		}
	})

	t.Run("mismatched inputs are rejected", func(t *testing.T) {
		if err := WriteMultiScaffold(t.TempDir(), "spiffe", entries, newResults()[:1]); err == nil {
			t.Fatal("expected an error when results do not match projects")
		}
		if err := WriteMultiScaffold(t.TempDir(), "spiffe", nil, nil); err == nil {
			t.Fatal("expected an error for a repository with no projects")
		}
	})

	t.Run("an unusable directory name is rejected", func(t *testing.T) {
		bad := []OrgProject{{Name: "Dot", Slug: ".hidden"}}
		results := []*BootstrapResult{{Slug: ".hidden", Name: "Dot", Sources: map[string]string{}}}
		if err := WriteMultiScaffold(t.TempDir(), "spiffe", bad, results); err == nil {
			t.Fatal("expected an error for a dot-prefixed project directory")
		}
	})
}
