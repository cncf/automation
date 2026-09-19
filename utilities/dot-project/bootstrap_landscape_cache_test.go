package projects

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// The landscape is several megabytes and every project bootstrapped from it
// reads the same copy, so it must be downloaded once per URL per process.
// Before caching, an organization with two projects fetched it three times:
// once to find the projects and once per project to fill in the metadata.
func TestFetchLandscapeRootCaches(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Orchestration & Management
    subcategories:
      - subcategory:
        name: Scheduling & Orchestration
        items:
          - item:
            name: Kubernetes
            repo_url: https://github.com/kubernetes/kubernetes
            project: graduated
`
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(landscapeYAML))
	}))
	defer server.Close()

	ResetLandscapeCache()
	defer ResetLandscapeCache()

	for i := 0; i < 3; i++ {
		if _, err := fetchLandscapeRoot(&http.Client{}, server.URL); err != nil {
			t.Fatalf("fetchLandscapeRoot() error = %v", err)
		}
	}

	// FindOrgProjects and fetchFromLandscape must share the same cached copy.
	found, err := FindOrgProjects("kubernetes", &http.Client{}, server.URL)
	if err != nil {
		t.Fatalf("FindOrgProjects() error = %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("FindOrgProjects() returned %d projects, want 1", len(found))
	}
	if _, err := fetchFromLandscape("Kubernetes", &http.Client{}, server.URL); err != nil {
		t.Fatalf("fetchFromLandscape() error = %v", err)
	}

	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("landscape downloaded %d times, want 1", got)
	}
}

// A failed fetch must not be cached, otherwise a transient network error would
// poison every later lookup in the same process.
func TestFetchLandscapeRootDoesNotCacheFailures(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("landscape: []\n"))
	}))
	defer server.Close()

	ResetLandscapeCache()
	defer ResetLandscapeCache()

	if _, err := fetchLandscapeRoot(&http.Client{}, server.URL); err == nil {
		t.Fatal("expected an error for HTTP 500")
	}
	if _, err := fetchLandscapeRoot(&http.Client{}, server.URL); err != nil {
		t.Fatalf("retry after a failure should succeed, got %v", err)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("landscape requested %d times, want 2", got)
	}
}
