package projects

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchFromCLOMonitor(t *testing.T) {
	// helper checks that the request carries the expected search query params
	checkSearchParams := func(t *testing.T, r *http.Request, wantText string) {
		t.Helper()
		if got := r.URL.Query().Get("text"); got != wantText {
			t.Errorf("query param text = %q, want %q", got, wantText)
		}
		if got := r.URL.Query().Get("foundation"); got != "cncf" {
			t.Errorf("query param foundation = %q, want %q", got, "cncf")
		}
	}

	t.Run("finds project by display name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/projects/search" {
				checkSearchParams(t, r, "Kubernetes")
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
			checkSearchParams(t, r, "argo cd")
			// Server returns candidates matching the query (as the real API would)
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
			checkSearchParams(t, r, "nonexistent")
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

	t.Run("sends foundation=cncf query param", func(t *testing.T) {
		// Server-side filtering is now delegated to the API via foundation=cncf.
		// Verify the param is sent and that the client picks the best fuzzy match
		// from whatever the server returns.
		var gotFoundation string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotFoundation = r.URL.Query().Get("foundation")
			// Simulate server returning only CNCF results (as the real API does)
			results := []CLOMonitorProject{
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
		if gotFoundation != "cncf" {
			t.Errorf("foundation query param = %q, want %q", gotFoundation, "cncf")
		}
		if result == nil {
			t.Fatal("expected a match")
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

	t.Run("handles numeric accepted_at and updated_at from API", func(t *testing.T) {
		// The real CLOMonitor API returns accepted_at and updated_at as Unix
		// timestamps (integers), not strings. This test sends raw JSON matching
		// the real API format to verify unmarshalling works.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{
				"name": "aeraki-mesh",
				"display_name": "Aeraki Mesh",
				"description": "Manage any layer-7 protocol in Istio service mesh",
				"home_url": "https://aeraki.net",
				"maturity": "sandbox",
				"foundation": "cncf",
				"accepted_at": 1592438400,
				"updated_at": 1773817304,
				"category": "Orchestration & Management",
				"subcategory": "Service Mesh"
			}]`))
		}))
		defer server.Close()

		result, err := fetchFromCLOMonitor("Aeraki Mesh", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromCLOMonitor() error = %v", err)
		}
		if result == nil {
			t.Fatal("fetchFromCLOMonitor() returned nil")
		}
		if result.AcceptedAt != 1592438400 {
			t.Errorf("AcceptedAt = %d, want 1592438400", result.AcceptedAt)
		}
		if result.UpdatedAt != 1773817304 {
			t.Errorf("UpdatedAt = %d, want 1773817304", result.UpdatedAt)
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

func TestCommunityProfileHTMLURLs(t *testing.T) {
	t.Run("captures html_url from community profile files", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/repos/test-org/test-repo":
				json.NewEncoder(w).Encode(GitHubRepoData{Name: "test-repo", FullName: "test-org/test-repo", DefaultBranch: "main"})
			case "/orgs/test-org":
				json.NewEncoder(w).Encode(GitHubOrgData{Login: "test-org"})
			case "/repos/test-org/test-repo/community/profile":
				w.Write([]byte(`{
					"health_percentage": 90,
					"files": {
						"code_of_conduct": {"key": "contributor_covenant", "name": "Contributor Covenant", "url": "https://api.github.com/codes_of_conduct/contributor_covenant", "html_url": "https://github.com/test-org/.github/blob/main/CODE_OF_CONDUCT.md"},
						"code_of_conduct_file": {"url": "https://api.github.com/repos/test-org/.github/contents/CODE_OF_CONDUCT.md", "html_url": "https://github.com/test-org/.github/blob/main/CODE_OF_CONDUCT.md"},
						"contributing": {"url": "https://api.github.com/repos/test-org/test-repo/contents/CONTRIBUTING.md", "html_url": "https://github.com/test-org/test-repo/blob/main/CONTRIBUTING.md"},
						"license": {"key": "apache-2.0", "name": "Apache License 2.0", "spdx_id": "Apache-2.0", "url": "https://api.github.com/licenses/apache-2.0", "html_url": "https://github.com/test-org/test-repo/blob/main/LICENSE"}
					}
				}`))
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

		result, err := fetchFromGitHub("test-org", "test-repo", "", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("fetchFromGitHub() error = %v", err)
		}
		if result.ContributingURL != "https://github.com/test-org/test-repo/blob/main/CONTRIBUTING.md" {
			t.Errorf("ContributingURL = %q, want CONTRIBUTING.md html_url", result.ContributingURL)
		}
		if result.CodeOfConductURL != "https://github.com/test-org/.github/blob/main/CODE_OF_CONDUCT.md" {
			t.Errorf("CodeOfConductURL = %q, want CODE_OF_CONDUCT.md html_url", result.CodeOfConductURL)
		}
		if result.LicenseURL != "https://github.com/test-org/test-repo/blob/main/LICENSE" {
			t.Errorf("LicenseURL = %q, want LICENSE html_url", result.LicenseURL)
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

func TestFetchFromLandscape_ExtraFields(t *testing.T) {
	t.Run("extracts slack_url and accepted date from extra", func(t *testing.T) {
		landscapeYAML := `landscape:
  - category:
    name: Orchestration & Management
    subcategories:
      - subcategory:
        name: Service Mesh
        items:
          - item:
            name: Aeraki Mesh
            homepage_url: https://www.aeraki.net/
            repo_url: https://github.com/aeraki-mesh/aeraki
            project: sandbox
            extra:
              slack_url: "https://cloud-native.slack.com/messages/aeraki-mesh"
              chat_channel: "#aeraki-mesh"
              accepted: "2022-06-17"
              annual_review_url: "https://github.com/cncf/toc/pull/1143"
              package_manager_url: "https://hub.docker.com/r/aeraki/aeraki"
`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(landscapeYAML))
		}))
		defer server.Close()

		result, err := fetchFromLandscape("Aeraki Mesh", &http.Client{}, server.URL)
		if err != nil {
			t.Fatalf("fetchFromLandscape() error = %v", err)
		}
		if result == nil {
			t.Fatal("expected match")
		}
		if result.SlackURL != "https://cloud-native.slack.com/messages/aeraki-mesh" {
			t.Errorf("SlackURL = %q, want slack URL", result.SlackURL)
		}
		if result.ChatChannel != "#aeraki-mesh" {
			t.Errorf("ChatChannel = %q, want #aeraki-mesh", result.ChatChannel)
		}
		if result.AcceptedDate != "2022-06-17" {
			t.Errorf("AcceptedDate = %q, want 2022-06-17", result.AcceptedDate)
		}
		if result.AnnualReviewURL != "https://github.com/cncf/toc/pull/1143" {
			t.Errorf("AnnualReviewURL = %q, want review URL", result.AnnualReviewURL)
		}
		if result.PackageManagerURL != "https://hub.docker.com/r/aeraki/aeraki" {
			t.Errorf("PackageManagerURL = %q, want docker URL", result.PackageManagerURL)
		}
	})
}

func TestDetectDCOCLA(t *testing.T) {
	t.Run("detects DCO from signed-off-by in commits", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "/commits") {
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"commit": map[string]interface{}{"message": "feat: add feature\n\nSigned-off-by: Alice <alice@example.com>"}},
					{"commit": map[string]interface{}{"message": "fix: bug fix\n\nSigned-off-by: Bob <bob@example.com>"}},
					{"commit": map[string]interface{}{"message": "chore: update deps"}},
				})
				return
			}
			if strings.Contains(r.URL.Path, "/contents/.github") {
				json.NewEncoder(w).Encode([]GitHubContentEntry{})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		hasDCO, hasCLA, err := detectDCOCLA("test-org", "test-repo", "", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("detectDCOCLA() error = %v", err)
		}
		if !hasDCO {
			t.Error("expected hasDCO = true")
		}
		if hasCLA {
			t.Error("expected hasCLA = false")
		}
	})

	t.Run("detects CLA from config file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "/commits") {
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"commit": map[string]interface{}{"message": "feat: something"}},
				})
				return
			}
			if strings.Contains(r.URL.Path, "/contents/.github") {
				json.NewEncoder(w).Encode([]GitHubContentEntry{
					{Name: "cla.yml", Type: "file"},
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		hasDCO, hasCLA, err := detectDCOCLA("test-org", "test-repo", "", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("detectDCOCLA() error = %v", err)
		}
		if hasDCO {
			t.Error("expected hasDCO = false")
		}
		if !hasCLA {
			t.Error("expected hasCLA = true")
		}
	})

	t.Run("handles API errors gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		hasDCO, hasCLA, err := detectDCOCLA("test-org", "test-repo", "", server.Client(), server.URL)
		// Should not error — just return defaults
		if err != nil {
			t.Fatalf("detectDCOCLA() should not error, got %v", err)
		}
		if hasDCO || hasCLA {
			t.Error("expected both false on API failure")
		}
	})
}

func TestSearchTOCIssues(t *testing.T) {
	t.Run("finds onboarding issue in cncf/sandbox", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "/search/issues") {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"total_count": 1,
					"items": []map[string]interface{}{
						{
							"html_url": "https://github.com/cncf/sandbox/issues/174",
							"title":    "[PROJECT ONBOARDING] Aeraki Mesh",
							"number":   174,
						},
					},
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		url, err := SearchTOCIssues("Aeraki Mesh", "aeraki-mesh", "", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("SearchTOCIssues() error = %v", err)
		}
		if url != "https://github.com/cncf/sandbox/issues/174" {
			t.Errorf("url = %q, want sandbox issue URL", url)
		}
	})

	t.Run("returns empty when no results", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 0,
				"items":       []map[string]interface{}{},
			})
		}))
		defer server.Close()

		url, err := SearchTOCIssues("Nonexistent", "nonexistent", "", server.Client(), server.URL)
		if err != nil {
			t.Fatalf("SearchTOCIssues() error = %v", err)
		}
		if url != "" {
			t.Errorf("expected empty URL, got %q", url)
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

func TestMergeBootstrapData_DCOCLA(t *testing.T) {
	t.Run("propagates DCO/CLA from GitHub data", func(t *testing.T) {
		github := &GitHubData{
			Repo:   &GitHubRepoData{Name: "test", FullName: "test-org/test"},
			Org:    &GitHubOrgData{Login: "test-org"},
			HasDCO: true,
			HasCLA: false,
		}

		result := mergeBootstrapData("test", nil, nil, github)

		if !result.HasDCO {
			t.Error("HasDCO should be true")
		}
		if result.HasCLA {
			t.Error("HasCLA should be false")
		}
		if result.Sources["identity_type"] != "github" {
			t.Errorf("Sources[identity_type] = %q, want github", result.Sources["identity_type"])
		}
	})

	t.Run("omits identity_type TODO when DCO detected", func(t *testing.T) {
		github := &GitHubData{
			Repo:   &GitHubRepoData{Name: "test", FullName: "test-org/test"},
			Org:    &GitHubOrgData{Login: "test-org"},
			HasDCO: true,
		}

		result := mergeBootstrapData("test", nil, nil, github)

		for _, todo := range result.TODOs {
			if strings.Contains(todo, "identity_type") {
				t.Errorf("should not have identity_type TODO when DCO detected, got: %q", todo)
			}
		}
	})

	t.Run("keeps identity_type TODO when neither detected", func(t *testing.T) {
		github := &GitHubData{
			Repo: &GitHubRepoData{Name: "test", FullName: "test-org/test"},
			Org:  &GitHubOrgData{Login: "test-org"},
		}

		result := mergeBootstrapData("test", nil, nil, github)

		found := false
		for _, todo := range result.TODOs {
			if strings.Contains(todo, "identity_type") {
				found = true
			}
		}
		if !found {
			t.Error("should have identity_type TODO when neither DCO nor CLA detected")
		}
	})
}

func TestMergeBootstrapData_ExtraFields(t *testing.T) {
	t.Run("sets slack channel from landscape extra", func(t *testing.T) {
		landscape := &LandscapeData{
			Name:        "Test Project",
			Maturity:    "sandbox",
			SlackURL:    "https://cloud-native.slack.com/messages/test-project",
			ChatChannel: "#test-project",
		}

		result := mergeBootstrapData("test-project", landscape, nil, nil)

		if result.CNCFSlackChannel != "#test-project" {
			t.Errorf("CNCFSlackChannel = %q, want #test-project", result.CNCFSlackChannel)
		}
	})

	t.Run("derives slack channel from slack_url when chat_channel missing", func(t *testing.T) {
		landscape := &LandscapeData{
			Name:     "Test Project",
			Maturity: "sandbox",
			SlackURL: "https://cloud-native.slack.com/messages/test-project",
		}

		result := mergeBootstrapData("test-project", landscape, nil, nil)

		if result.CNCFSlackChannel != "#test-project" {
			t.Errorf("CNCFSlackChannel = %q, want #test-project", result.CNCFSlackChannel)
		}
	})

	t.Run("sets accepted date from landscape extra", func(t *testing.T) {
		landscape := &LandscapeData{
			Name:         "Test Project",
			Maturity:     "sandbox",
			AcceptedDate: "2022-06-17",
		}

		result := mergeBootstrapData("test-project", landscape, nil, nil)

		if result.AcceptedDate.IsZero() {
			t.Error("AcceptedDate should be set")
		}
		if result.AcceptedDate.Format("2006-01-02") != "2022-06-17" {
			t.Errorf("AcceptedDate = %v, want 2022-06-17", result.AcceptedDate)
		}
	})

	t.Run("sets TOC issue URL from landscape annual_review_url", func(t *testing.T) {
		landscape := &LandscapeData{
			Name:            "Test Project",
			Maturity:        "sandbox",
			AnnualReviewURL: "https://github.com/cncf/toc/pull/1143",
		}

		result := mergeBootstrapData("test-project", landscape, nil, nil)

		if result.TOCIssueURL != "https://github.com/cncf/toc/pull/1143" {
			t.Errorf("TOCIssueURL = %q, want toc pull URL", result.TOCIssueURL)
		}
	})
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

	t.Run("propagates discovered file URLs from GitHub", func(t *testing.T) {
		github := &GitHubData{
			Repo: &GitHubRepoData{
				Name:     "test-repo",
				FullName: "test-org/test-repo",
				HTMLURL:  "https://github.com/test-org/test-repo",
			},
			Org: &GitHubOrgData{
				Login: "test-org",
			},
			SecurityPolicyURL: "https://github.com/test-org/.github/blob/main/SECURITY.md",
			ContributingURL:   "https://github.com/test-org/test-repo/blob/main/CONTRIBUTING.md",
			CodeOfConductURL:  "https://github.com/test-org/.github/blob/main/CODE_OF_CONDUCT.md",
			LicenseURL:        "https://github.com/test-org/test-repo/blob/main/LICENSE",
		}

		result := mergeBootstrapData("test-project", nil, nil, github)

		if result.SecurityPolicyURL != "https://github.com/test-org/.github/blob/main/SECURITY.md" {
			t.Errorf("SecurityPolicyURL = %q, want org .github URL", result.SecurityPolicyURL)
		}
		if !result.HasSecurityPolicy {
			t.Error("HasSecurityPolicy should be true when SecurityPolicyURL is set")
		}
		if result.ContributingURL != "https://github.com/test-org/test-repo/blob/main/CONTRIBUTING.md" {
			t.Errorf("ContributingURL = %q, want repo CONTRIBUTING.md URL", result.ContributingURL)
		}
		if result.CodeOfConductURL != "https://github.com/test-org/.github/blob/main/CODE_OF_CONDUCT.md" {
			t.Errorf("CodeOfConductURL = %q, want org CODE_OF_CONDUCT.md URL", result.CodeOfConductURL)
		}
		if result.LicenseURL != "https://github.com/test-org/test-repo/blob/main/LICENSE" {
			t.Errorf("LicenseURL = %q, want repo LICENSE URL", result.LicenseURL)
		}
		if result.Sources["security_policy"] != "github" {
			t.Errorf("Sources[security_policy] = %q, want github", result.Sources["security_policy"])
		}
	})
}

func TestDiscoverSecurityPolicy(t *testing.T) {
	t.Run("discovers SECURITY.md in repo root", func(t *testing.T) {
		var serverURL string
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
					{Name: "SECURITY.md", Path: "SECURITY.md", Type: "file",
						DownloadURL: serverURL + "/raw/SECURITY.md",
						HTMLURL:     "https://github.com/test-org/test-repo/blob/main/SECURITY.md"},
				})
			case "/raw/SECURITY.md":
				w.Write([]byte("# Security Policy\nReport vulnerabilities..."))
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
		if result.SecurityPolicyURL != "https://github.com/test-org/test-repo/blob/main/SECURITY.md" {
			t.Errorf("SecurityPolicyURL = %q, want repo root SECURITY.md URL", result.SecurityPolicyURL)
		}
	})

	t.Run("discovers SECURITY.md in org .github repo", func(t *testing.T) {
		var serverURL string
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
				json.NewEncoder(w).Encode([]GitHubContentEntry{})
			case "/repos/test-org/test-repo/contents/.github":
				json.NewEncoder(w).Encode([]GitHubContentEntry{})
			case "/repos/test-org/.github/contents/":
				json.NewEncoder(w).Encode([]GitHubContentEntry{
					{Name: "SECURITY.md", Path: "SECURITY.md", Type: "file",
						DownloadURL: serverURL + "/raw/org-SECURITY.md",
						HTMLURL:     "https://github.com/test-org/.github/blob/main/SECURITY.md"},
				})
			case "/raw/org-SECURITY.md":
				w.Write([]byte("# Org Security Policy"))
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
		if result.SecurityPolicyURL != "https://github.com/test-org/.github/blob/main/SECURITY.md" {
			t.Errorf("SecurityPolicyURL = %q, want org .github SECURITY.md URL", result.SecurityPolicyURL)
		}
	})
}
