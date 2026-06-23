package projects

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverGovernanceSuggestions(t *testing.T) {
	var base string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		// Primary repo root: a CODEOWNERS file
		case "/repos/test-org/test-repo/contents/":
			json.NewEncoder(w).Encode([]GitHubContentEntry{
				{Name: "CODEOWNERS", Type: "file", DownloadURL: base + "/raw/test-repo/CODEOWNERS"},
			})
		case "/raw/test-repo/CODEOWNERS":
			fmt.Fprint(w, "* @alice @bob\n")

		// No .github dir on the primary repo, no org .github repo
		case "/repos/test-org/test-repo/contents/.github",
			"/repos/test-org/.github/contents/":
			w.WriteHeader(http.StatusNotFound)

		// Org repo listing: one own repo + one fork (the fork must be skipped)
		case "/orgs/test-org/repos":
			json.NewEncoder(w).Encode([]map[string]any{
				{"name": "other-repo", "fork": false, "size": 100},
				{"name": "forked-repo", "fork": true, "size": 50},
			})

		// other-repo root: a MAINTAINERS file + an OWNERS with an alias + a README
		case "/repos/test-org/other-repo/contents/":
			json.NewEncoder(w).Encode([]GitHubContentEntry{
				{Name: "MAINTAINERS", Type: "file", DownloadURL: base + "/raw/other-repo/MAINTAINERS"},
				{Name: "OWNERS", Type: "file", DownloadURL: base + "/raw/other-repo/OWNERS"},
				{Name: "README.md", Type: "file", DownloadURL: base + "/raw/other-repo/README"},
			})
		case "/raw/other-repo/MAINTAINERS":
			// alice again (different role+source) + carol
			fmt.Fprint(w, "@alice\n@carol\n")
		case "/raw/other-repo/OWNERS":
			// a real reviewer (dave) + an alias group that must be filtered out
			fmt.Fprint(w, "reviewers:\n  - dave\n  - other-repo-reviewers\n")
		case "/raw/other-repo/README":
			// README referencing a CNCF Slack channel
			fmt.Fprint(w, "## Community\nJoin us on Slack at https://cloud-native.slack.com/messages/test-channel\n")

		// The fork's contents must never be requested; fail loudly if it is.
		case "/repos/test-org/forked-repo/contents/":
			t.Errorf("fork repo should not be scanned")
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	base = server.URL

	// bob is already in the CSV roster, so should be excluded.
	csv := map[string]bool{"bob": true}
	got, slack := DiscoverGovernanceSuggestions("test-org", "test-repo", "", server.Client(), server.URL, csv)

	// Expect alice (code owner + maintainer), carol (maintainer), dave (reviewer);
	// bob excluded (CSV), other-repo-reviewers excluded (alias), fork skipped.
	if len(got) != 3 {
		t.Fatalf("expected 3 suggestions, got %d: %+v", len(got), got)
	}

	// The README in other-repo references a CNCF Slack channel.
	if len(slack) != 1 || slack[0] != "#test-channel" {
		t.Errorf("slack channels = %v, want [#test-channel]", slack)
	}

	byHandle := map[string]MaintainerSuggestion{}
	for _, s := range got {
		byHandle[s.Handle] = s
	}

	alice, ok := byHandle["alice"]
	if !ok {
		t.Fatalf("expected alice in suggestions: %+v", got)
	}
	if strings.Join(alice.Roles, ",") != "maintainer,code owner" {
		t.Errorf("alice roles = %v, want [maintainer, code owner]", alice.Roles)
	}
	if len(alice.Sources) != 2 {
		t.Errorf("alice sources = %v, want 2 (CODEOWNERS + MAINTAINERS)", alice.Sources)
	}

	if _, ok := byHandle["carol"]; !ok {
		t.Errorf("expected carol in suggestions: %+v", got)
	}
	if _, ok := byHandle["dave"]; !ok {
		t.Errorf("expected dave (reviewer) in suggestions: %+v", got)
	}
	if _, ok := byHandle["bob"]; ok {
		t.Errorf("bob should be excluded (already in CSV): %+v", got)
	}
	if _, ok := byHandle["other-repo-reviewers"]; ok {
		t.Errorf("OWNERS alias should be filtered out: %+v", got)
	}
}

func TestWriteSuggestionsFile(t *testing.T) {
	dir := t.TempDir()

	// No suggestions -> no file, empty path
	path, err := WriteSuggestionsFile(dir, nil)
	if err != nil {
		t.Fatalf("WriteSuggestionsFile(nil) error = %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path for no suggestions, got %q", path)
	}

	suggestions := []MaintainerSuggestion{
		{Handle: "alice", Roles: []string{"maintainer"}, Sources: []string{"org/repo:MAINTAINERS"}},
	}
	path, err = WriteSuggestionsFile(dir, suggestions)
	if err != nil {
		t.Fatalf("WriteSuggestionsFile error = %v", err)
	}
	want := filepath.Join(dir, ".cache", "maintainer-suggestions.md")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading suggestions file: %v", err)
	}
	if !strings.Contains(string(data), "Suggested maintainers") || !strings.Contains(string(data), "@alice") {
		t.Errorf("file content missing expected section/handle:\n%s", data)
	}
}
