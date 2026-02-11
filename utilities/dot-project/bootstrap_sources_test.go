package projects

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchFromCLOMonitor(t *testing.T) {
	t.Run("finds project by display name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/projects/search" {
				results := []CLOMonitorProject{
					{
						Name:        "kubernetes",
						DisplayName: "Kubernetes",
						Description: "Production-Grade Container Orchestration",
						HomeURL:     "https://kubernetes.io",
						Maturity:    "graduated",
						Foundation:  "cncf",
						Category:    "Orchestration & Management",
						Subcategory: "Scheduling & Orchestration",
						Score: &CLOMonitorScore{
							Global:        90.0,
							Documentation: 85.0,
							License:       100.0,
							BestPractices: 80.0,
							Security:      95.0,
							Legal:         90.0,
						},
						Repositories: []CLOMonitorRepo{
							{Name: "kubernetes", URL: "https://github.com/kubernetes/kubernetes"},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(results)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		result, err := fetchFromCLOMonitor("Kubernetes", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromCLOMonitor() error = %v", err)
		}
		if result == nil {
			t.Fatal("fetchFromCLOMonitor() returned nil")
		}
		if result.DisplayName != "Kubernetes" {
			t.Errorf("DisplayName = %q, want %q", result.DisplayName, "Kubernetes")
		}
		if result.Maturity != "graduated" {
			t.Errorf("Maturity = %q, want %q", result.Maturity, "graduated")
		}
		if result.Score == nil || result.Score.Global != 90.0 {
			t.Errorf("Score.Global = %v, want 90.0", result.Score)
		}
	})

	t.Run("case insensitive fuzzy match", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := []CLOMonitorProject{
				{Name: "argo", DisplayName: "Argo", Foundation: "cncf"},
				{Name: "argo-cd", DisplayName: "Argo CD", Foundation: "cncf"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		}))
		defer server.Close()

		result, err := fetchFromCLOMonitor("argo cd", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromCLOMonitor() error = %v", err)
		}
		if result == nil {
			t.Fatal("expected a match for 'argo cd'")
		}
		if result.DisplayName != "Argo CD" {
			t.Errorf("DisplayName = %q, want %q", result.DisplayName, "Argo CD")
		}
	})

	t.Run("no match returns nil", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]CLOMonitorProject{})
		}))
		defer server.Close()

		result, err := fetchFromCLOMonitor("nonexistent", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromCLOMonitor() error = %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})

	t.Run("filters by CNCF foundation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := []CLOMonitorProject{
				{Name: "test", DisplayName: "Test", Foundation: "lfai"},
				{Name: "test-cncf", DisplayName: "Test CNCF", Foundation: "cncf"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		}))
		defer server.Close()

		result, err := fetchFromCLOMonitor("test", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromCLOMonitor() error = %v", err)
		}
		if result == nil {
			t.Fatal("expected a CNCF match")
		}
		if result.Foundation != "cncf" {
			t.Errorf("Foundation = %q, want %q", result.Foundation, "cncf")
		}
	})

	t.Run("server error returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := fetchFromCLOMonitor("test", server.Client(), server.URL)
		if err == nil {
			t.Fatal("expected error for server error")
		}
	})
}

func TestFetchFromGitHub(t *testing.T) {
	t.Run("fetches repo, org, and community profile", func(t *testing.T) {
		var serverURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/repos/test-org/test-repo":
				json.NewEncoder(w).Encode(GitHubRepoData{
					Name:          "test-repo",
					FullName:      "test-org/test-repo",
					Description:   "A test repository",
					HTMLURL:       "https://github.com/test-org/test-repo",
					Homepage:      "https://test-project.io",
					DefaultBranch: "main",
					Topics:        []string{"cloud-native", "cncf"},
				})
			case "/orgs/test-org":
				json.NewEncoder(w).Encode(GitHubOrgData{
					Login:       "test-org",
					Name:        "Test Org",
					Description: "A test org",
					Blog:        "https://test-org.io",
					TwitterUser: "testorg",
				})
			case "/repos/test-org/test-repo/community/profile":
				json.NewEncoder(w).Encode(GitHubCommunityProfile{
					HealthPercentage: 80,
				})
			case "/repos/test-org/test-repo/contents/":
				json.NewEncoder(w).Encode([]GitHubContentEntry{
					{Name: "README.md", Path: "README.md", Type: "file"},
					{Name: "CODEOWNERS", Path: "CODEOWNERS", Type: "file", DownloadURL: serverURL + "/raw/CODEOWNERS"},
				})
			case "/raw/CODEOWNERS":
				w.Write([]byte("* @alice @bob\n"))
			case "/repos/test-org/test-repo/contents/.github":
				w.WriteHeader(http.StatusNotFound)
			case "/repos/test-org/.github/contents/":
				w.WriteHeader(http.StatusNotFound)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		serverURL = server.URL

		result, err := fetchFromGitHub("test-org", "test-repo", "", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromGitHub() error = %v", err)
		}
		if result == nil {
			t.Fatal("fetchFromGitHub() returned nil")
		}
		if result.Repo == nil {
			t.Fatal("Repo is nil")
		}
		if result.Repo.Description != "A test repository" {
			t.Errorf("Repo.Description = %q, want %q", result.Repo.Description, "A test repository")
		}
		if result.Org == nil {
			t.Fatal("Org is nil")
		}
		if result.Org.TwitterUser != "testorg" {
			t.Errorf("Org.TwitterUser = %q, want %q", result.Org.TwitterUser, "testorg")
		}
		// CODEOWNERS should be parsed
		if len(result.Maintainers) == 0 {
			t.Error("expected maintainers from CODEOWNERS")
		}
	})

	t.Run("discovers OWNERS file", func(t *testing.T) {
		var serverURL2 string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/repos/test-org/test-repo":
				json.NewEncoder(w).Encode(GitHubRepoData{Name: "test-repo", FullName: "test-org/test-repo", DefaultBranch: "main"})
			case "/orgs/test-org":
				json.NewEncoder(w).Encode(GitHubOrgData{Login: "test-org"})
			case "/repos/test-org/test-repo/community/profile":
				json.NewEncoder(w).Encode(GitHubCommunityProfile{})
			case "/repos/test-org/test-repo/contents/":
				json.NewEncoder(w).Encode([]GitHubContentEntry{
					{Name: "OWNERS", Path: "OWNERS", Type: "file", DownloadURL: serverURL2 + "/raw/OWNERS"},
				})
			case "/raw/OWNERS":
				w.Write([]byte("approvers:\n  - alice\n  - bob\nreviewers:\n  - carol\n"))
			case "/repos/test-org/test-repo/contents/.github":
				w.WriteHeader(http.StatusNotFound)
			case "/repos/test-org/.github/contents/":
				w.WriteHeader(http.StatusNotFound)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		serverURL2 = server.URL

		result, err := fetchFromGitHub("test-org", "test-repo", "", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromGitHub() error = %v", err)
		}
		if len(result.Maintainers) != 2 {
			t.Errorf("expected 2 maintainers (approvers), got %d: %v", len(result.Maintainers), result.Maintainers)
		}
		if len(result.Reviewers) != 1 {
			t.Errorf("expected 1 reviewer, got %d: %v", len(result.Reviewers), result.Reviewers)
		}
	})

	t.Run("repo not found returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := fetchFromGitHub("test-org", "test-repo", "", server.Client(), server.URL)
		if err == nil {
			t.Fatal("expected error for repo not found")
		}
	})

	t.Run("sets auth header when token provided", func(t *testing.T) {
		var gotAuthHeader string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuthHeader = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/repos/test-org/test-repo":
				json.NewEncoder(w).Encode(GitHubRepoData{Name: "test-repo", FullName: "test-org/test-repo", DefaultBranch: "main"})
			case "/orgs/test-org":
				json.NewEncoder(w).Encode(GitHubOrgData{Login: "test-org"})
			case "/repos/test-org/test-repo/community/profile":
				json.NewEncoder(w).Encode(GitHubCommunityProfile{})
			case "/repos/test-org/test-repo/contents/":
				json.NewEncoder(w).Encode([]GitHubContentEntry{})
			case "/repos/test-org/test-repo/contents/.github":
				w.WriteHeader(http.StatusNotFound)
			case "/repos/test-org/.github/contents/":
				w.WriteHeader(http.StatusNotFound)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		_, err := fetchFromGitHub("test-org", "test-repo", "my-token", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromGitHub() error = %v", err)
		}
		if gotAuthHeader != "token my-token" {
			t.Errorf("Authorization header = %q, want %q", gotAuthHeader, "token my-token")
		}
	})
}

func TestFetchFromLandscape(t *testing.T) {
	t.Run("finds project by name", func(t *testing.T) {
		landscapeYAML := `landscape:
  - category:
    name: Orchestration & Management
    subcategories:
      - subcategory:
        name: Scheduling & Orchestration
        items:
          - item:
            name: Kubernetes
            description: Production-Grade Container Orchestration
            homepage_url: https://kubernetes.io
            repo_url: https://github.com/kubernetes/kubernetes
            logo: kubernetes.svg
            twitter: https://twitter.com/kuabornetesio
            project: graduated
`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(landscapeYAML))
		}))
		defer server.Close()

		result, err := fetchFromLandscape("Kubernetes", &http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("fetchFromLandscape() error = %v", err)
		}
		if result == nil {
			t.Fatal("fetchFromLandscape() returned nil")
		}
		if result.Name != "Kubernetes" {
			t.Errorf("Name = %q, want %q", result.Name, "Kubernetes")
		}
		if result.Description != "Production-Grade Container Orchestration" {
			t.Errorf("Description = %q, want %q", result.Description, "Production-Grade Container Orchestration")
		}
		if result.HomepageURL != "https://kubernetes.io" {
			t.Errorf("HomepageURL = %q, want %q", result.HomepageURL, "https://kubernetes.io")
		}
		if result.RepoURL != "https://github.com/kubernetes/kubernetes" {
			t.Errorf("RepoURL = %q, want %q", result.RepoURL, "https://github.com/kubernetes/kubernetes")
		}
		if result.Twitter != "https://twitter.com/kuabornetesio" {
			t.Errorf("Twitter = %q, want %q", result.Twitter, "https://twitter.com/kuabornetesio")
		}
		if result.Maturity != "graduated" {
			t.Errorf("Maturity = %q, want %q", result.Maturity, "graduated")
		}
		if result.Category != "Orchestration & Management" {
			t.Errorf("Category = %q, want %q", result.Category, "Orchestration & Management")
		}
		if result.Subcategory != "Scheduling & Orchestration" {
			t.Errorf("Subcategory = %q, want %q", result.Subcategory, "Scheduling & Orchestration")
		}
	})

	t.Run("case insensitive fuzzy match", func(t *testing.T) {
		landscapeYAML := `landscape:
  - category:
    name: Provisioning
    subcategories:
      - subcategory:
        name: Container Registry
        items:
          - item:
            name: Harbor
            homepage_url: https://goharbor.io
            project: graduated
          - item:
            name: Dragonfly
            homepage_url: https://d7y.io
            project: graduated
`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(landscapeYAML))
		}))
		defer server.Close()

		result, err := fetchFromLandscape("harbor", &http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("fetchFromLandscape() error = %v", err)
		}
		if result == nil {
			t.Fatal("expected match for 'harbor'")
		}
		if result.Name != "Harbor" {
			t.Errorf("Name = %q, want %q", result.Name, "Harbor")
		}
	})

	t.Run("no match returns nil", func(t *testing.T) {
		landscapeYAML := `landscape:
  - category:
    name: Provisioning
    subcategories:
      - subcategory:
        name: Container Registry
        items:
          - item:
            name: Harbor
            homepage_url: https://goharbor.io
`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(landscapeYAML))
		}))
		defer server.Close()

		result, err := fetchFromLandscape("nonexistent-project-xyz", &http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("fetchFromLandscape() error = %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})

	t.Run("server error returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := fetchFromLandscape("test", &http.Client{}, server.URL)
		if err == nil {
			t.Fatal("expected error for server error")
		}
	})

	t.Run("only matches CNCF projects with project field", func(t *testing.T) {
		landscapeYAML := `landscape:
  - category:
    name: Provisioning
    subcategories:
      - subcategory:
        name: Automation & Configuration
        items:
          - item:
            name: Terraform
            homepage_url: https://www.terraform.io/
            repo_url: https://github.com/hashicorp/terraform
          - item:
            name: OpenTofu
            description: Open source Terraform fork
            homepage_url: https://opentofu.org/
            repo_url: https://github.com/opentofu/opentofu
            project: sandbox
`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(landscapeYAML))
		}))
		defer server.Close()

		// Terraform has no "project" field, so it's not a CNCF project
		result, err := fetchFromLandscape("Terraform", &http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("fetchFromLandscape() error = %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-CNCF project Terraform, got %+v", result)
		}

		// OpenTofu has project: sandbox, so it IS a CNCF project
		result, err = fetchFromLandscape("OpenTofu", &http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("fetchFromLandscape() error = %v", err)
		}
		if result == nil {
			t.Fatal("expected match for CNCF project OpenTofu")
		}
		if result.Maturity != "sandbox" {
			t.Errorf("Maturity = %q, want %q", result.Maturity, "sandbox")
		}
	})

	t.Run("builds logo URL from SVG filename", func(t *testing.T) {
		landscapeYAML := `landscape:
  - category:
    name: Observability
    subcategories:
      - subcategory:
        name: Monitoring
        items:
          - item:
            name: Prometheus
            homepage_url: https://prometheus.io
            logo: prometheus.svg
            project: graduated
`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(landscapeYAML))
		}))
		defer server.Close()

		result, err := fetchFromLandscape("Prometheus", &http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("fetchFromLandscape() error = %v", err)
		}
		if result == nil {
			t.Fatal("expected match")
		}
		expected := "https://landscape.cncf.io/logos/prometheus.svg"
		if result.LogoURL != expected {
			t.Errorf("LogoURL = %q, want %q", result.LogoURL, expected)
		}
	})
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		candidates []string
		wantBest   string
		wantFound  bool
	}{
		{
			name:       "exact match",
			query:      "Kubernetes",
			candidates: []string{"Kubernetes", "Prometheus", "Envoy"},
			wantBest:   "Kubernetes",
			wantFound:  true,
		},
		{
			name:       "case insensitive",
			query:      "kubernetes",
			candidates: []string{"Kubernetes", "Prometheus"},
			wantBest:   "Kubernetes",
			wantFound:  true,
		},
		{
			name:       "partial match prefers better overlap",
			query:      "argo cd",
			candidates: []string{"Argo", "Argo CD", "Argo Workflows"},
			wantBest:   "Argo CD",
			wantFound:  true,
		},
		{
			name:       "no match",
			query:      "nonexistent",
			candidates: []string{"Kubernetes", "Prometheus"},
			wantBest:   "",
			wantFound:  false,
		},
		{
			name:       "empty candidates",
			query:      "test",
			candidates: []string{},
			wantBest:   "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			best, score := fuzzyMatch(tt.query, tt.candidates)
			found := score > 0
			if found != tt.wantFound {
				t.Errorf("fuzzyMatch() found = %v, want %v (score=%f)", found, tt.wantFound, score)
			}
			if tt.wantFound && best != tt.wantBest {
				t.Errorf("fuzzyMatch() best = %q, want %q", best, tt.wantBest)
			}
		})
	}
}

func TestMergeBootstrapData(t *testing.T) {
	t.Run("merges all sources with priority", func(t *testing.T) {
		landscape := &LandscapeData{
			Name:        "Test Project",
			Description: "From landscape",
			HomepageURL: "https://test.io",
			RepoURL:     "https://github.com/test-org/test-project",
			Maturity:    "incubating",
			Category:    "Observability",
			Subcategory: "Monitoring",
			Twitter:     "https://twitter.com/testproject",
		}

		clomonitor := &CLOMonitorProject{
			DisplayName: "Test Project",
			Description: "From CLOMonitor",
			HomeURL:     "https://test-clo.io",
			Maturity:    "incubating",
			Score: &CLOMonitorScore{
				Global: 85.0,
			},
		}

		github := &GitHubData{
			Repo: &GitHubRepoData{
				FullName:    "test-org/test-project",
				Description: "From GitHub",
				HTMLURL:     "https://github.com/test-org/test-project",
				Homepage:    "https://test-gh.io",
			},
			Org: &GitHubOrgData{
				Login:       "test-org",
				TwitterUser: "testorg",
			},
			Maintainers: []string{"alice", "bob"},
			Reviewers:   []string{"carol"},
			Community: &GitHubCommunityProfile{
				HealthPercentage: 90,
			},
		}

		result := mergeBootstrapData("test-project", landscape, clomonitor, github)

		if result.Name != "Test Project" {
			t.Errorf("Name = %q, want %q", result.Name, "Test Project")
		}
		if result.Slug != "test-project" {
			t.Errorf("Slug = %q, want %q", result.Slug, "test-project")
		}
		// Landscape description takes priority
		if result.Description != "From landscape" {
			t.Errorf("Description = %q, want %q", result.Description, "From landscape")
		}
		// Landscape website takes priority
		if result.Website != "https://test.io" {
			t.Errorf("Website = %q, want %q", result.Website, "https://test.io")
		}
		if result.MaturityPhase != "incubating" {
			t.Errorf("MaturityPhase = %q, want %q", result.MaturityPhase, "incubating")
		}
		if result.LandscapeCategory != "Observability" {
			t.Errorf("LandscapeCategory = %q, want %q", result.LandscapeCategory, "Observability")
		}
		if len(result.Maintainers) != 2 {
			t.Errorf("Maintainers count = %d, want 2", len(result.Maintainers))
		}
		if result.CLOMonitorScore == nil || result.CLOMonitorScore.Global != 85.0 {
			t.Errorf("CLOMonitorScore.Global = %v, want 85.0", result.CLOMonitorScore)
		}
	})

	t.Run("handles nil sources gracefully", func(t *testing.T) {
		result := mergeBootstrapData("test-project", nil, nil, nil)

		if result.Slug != "test-project" {
			t.Errorf("Slug = %q, want %q", result.Slug, "test-project")
		}
		if len(result.TODOs) == 0 {
			t.Error("expected TODOs for missing data")
		}
	})

	t.Run("falls back to GitHub when landscape missing", func(t *testing.T) {
		github := &GitHubData{
			Repo: &GitHubRepoData{
				FullName:    "test-org/test-project",
				Description: "From GitHub",
				HTMLURL:     "https://github.com/test-org/test-project",
				Homepage:    "https://test.io",
			},
			Org: &GitHubOrgData{
				Login: "test-org",
			},
		}

		result := mergeBootstrapData("test-project", nil, nil, github)

		if result.Description != "From GitHub" {
			t.Errorf("Description = %q, want %q (fallback to GitHub)", result.Description, "From GitHub")
		}
		if result.Website != "https://test.io" {
			t.Errorf("Website = %q, want %q (fallback to GitHub)", result.Website, "https://test.io")
		}
	})
}
