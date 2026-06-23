package projects

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Parsers
// ---------------------------------------------------------------------------

func TestParseCodeowners(t *testing.T) {
	got := parseCodeowners("* @alice @bob\n/docs/ @org/team-name @carol\n# @notahandle\nfoo someone@example.com\n")
	want := []string{"alice", "bob", "carol"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("parseCodeowners() = %v, want %v", got, want)
	}
	if parseCodeowners("") != nil {
		t.Error("expected nil for empty content")
	}
}

func TestParseOwnersFile(t *testing.T) {
	approvers, reviewers := parseOwnersFile("approvers:\n  - alice\n  - bob\nreviewers:\n  - carol\n")
	if strings.Join(approvers, ",") != "alice,bob" {
		t.Errorf("approvers = %v, want [alice bob]", approvers)
	}
	if strings.Join(reviewers, ",") != "carol" {
		t.Errorf("reviewers = %v, want [carol]", reviewers)
	}

	a, r := parseOwnersFile("not: valid: yaml: [")
	if a != nil || r != nil {
		t.Errorf("expected nil,nil for invalid YAML, got %v,%v", a, r)
	}
}

func TestParseMaintainersFile(t *testing.T) {
	content := "# Maintainers\n| Name | Handle |\n|------|--------|\n| Alice | @alice |\n" +
		"Bob (https://github.com/bob)\nContact alice@example.com\n@org/team-ref\n"
	got := parseMaintainersFile(content)
	// alice (table) + bob (url); email and team ref skipped.
	if !contains(got, "alice") || !contains(got, "bob") {
		t.Errorf("parseMaintainersFile() = %v, want alice and bob", got)
	}
	if contains(got, "team-ref") {
		t.Errorf("team ref leaked into handles: %v", got)
	}
}

// ---------------------------------------------------------------------------
// suggestionCollector
// ---------------------------------------------------------------------------

func TestSuggestionCollector_MergeRolesAndSources(t *testing.T) {
	c := newSuggestionCollector()
	c.add("Alice", roleMaintainer, "org/repo:MAINTAINERS")
	c.add("alice", roleCodeowner, "org/.github:CODEOWNERS") // same person, different casing
	c.add("@bob", roleReviewer, "org/repo2:OWNERS")

	got := c.suggestions(nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 suggestions, got %d: %+v", len(got), got)
	}

	// Sorted by handle: alice, bob
	alice := got[0]
	if strings.ToLower(alice.Handle) != "alice" {
		t.Fatalf("expected alice first, got %q", alice.Handle)
	}
	// Roles deduped + ordered by priority: maintainer before code owner
	if strings.Join(alice.Roles, ",") != "maintainer,code owner" {
		t.Errorf("alice roles = %v, want [maintainer, code owner]", alice.Roles)
	}
	if len(alice.Sources) != 2 {
		t.Errorf("alice sources = %v, want 2", alice.Sources)
	}
}

func TestSuggestionCollector_ExcludesCSVRoster(t *testing.T) {
	c := newSuggestionCollector()
	c.add("alice", roleMaintainer, "org/repo:MAINTAINERS")
	c.add("bob", roleReviewer, "org/repo:OWNERS")

	// alice is already in the CSV roster -> excluded from suggestions
	exclude := map[string]bool{"alice": true}
	got := c.suggestions(exclude)
	if len(got) != 1 || got[0].Handle != "bob" {
		t.Errorf("expected only bob suggested, got %+v", got)
	}
}

func TestSuggestionCollector_IgnoresBlankHandles(t *testing.T) {
	c := newSuggestionCollector()
	c.add("  ", roleMaintainer, "x")
	c.add("@", roleMaintainer, "x")
	if got := c.suggestions(nil); len(got) != 0 {
		t.Errorf("expected no suggestions, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// renderSuggestionsTable
// ---------------------------------------------------------------------------

func TestRenderSuggestionsTable(t *testing.T) {
	if renderSuggestionsTable(nil) != "" {
		t.Error("expected empty string for no suggestions")
	}

	table := renderSuggestionsTable([]MaintainerSuggestion{
		{Handle: "alice", Roles: []string{"maintainer", "code owner"}, Sources: []string{"org/repo:MAINTAINERS"}},
	})
	for _, want := range []string{"| Handle | Role(s) | Found in |", "| @alice |", "maintainer, code owner", "[org/repo](https://github.com/org/repo) — MAINTAINERS"} {
		if !strings.Contains(table, want) {
			t.Errorf("table missing %q\n%s", want, table)
		}
	}
}

func TestFormatSources(t *testing.T) {
	if got := formatSources(nil); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}

	got := formatSources([]string{"org/repo-a:MAINTAINERS.md", "org/repo-b:CODEOWNERS"})
	want := "[org/repo-a](https://github.com/org/repo-a) — MAINTAINERS.md" +
		"<br>" +
		"[org/repo-b](https://github.com/org/repo-b) — CODEOWNERS"
	if got != want {
		t.Errorf("formatSources =\n%q\nwant\n%q", got, want)
	}

	// Malformed entries (no colon) are emitted verbatim.
	if got := formatSources([]string{"just-a-string"}); got != "just-a-string" {
		t.Errorf("expected verbatim passthrough, got %q", got)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
