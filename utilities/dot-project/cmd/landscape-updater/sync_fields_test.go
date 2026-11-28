package main

import (
	"strings"
	"testing"
	"time"

	"projects"
)

const syncFieldsLandscape = `landscape:
  - category:
    name: Runtime
    subcategories:
      - subcategory:
        name: Storage
        items:
          - item:
            name: Demo
            homepage_url: https://demo.io
            repo_url: https://github.com/demo/old
            logo: demo.svg
            extra:
              slack_url: https://demo.slack.com
              audits:
                - date: '2020-01-02'
                  type: security
                  url: https://example.com/audit-1
                  vendor: First Vendor
          - item:
            name: Other
            homepage_url: https://other.io
            repo_url: https://github.com/other/other
            logo: other.svg`

func TestSyncRepoURLRequiresExplicitPrimary(t *testing.T) {
	root, lines := setupTest(syncFieldsLandscape)
	project := &projects.Project{
		Name: "Demo",
		Repositories: []projects.RepositoryEntry{
			{URL: "https://github.com/demo/new"},
			{URL: "https://github.com/demo/old"},
		},
	}
	if _, status := updateLandscape(root, project, lines); status != statusNoChanges {
		t.Fatalf("unordered repositories must not re-point repo_url, got status %v", status)
	}

	project.Repositories[0].Primary = true
	newLines, status := updateLandscape(root, project, lines)
	if status != statusUpdated {
		t.Fatalf("expected statusUpdated, got %v", status)
	}
	out := strings.Join(newLines, "\n")
	if !strings.Contains(out, "            repo_url: https://github.com/demo/new\n") {
		t.Errorf("repo_url not re-pointed at the primary repository:\n%s", out)
	}
	if !strings.Contains(out, "repo_url: https://github.com/other/other") {
		t.Errorf("unrelated item was modified:\n%s", out)
	}
}

func TestSyncExtraFieldsAreDeterministic(t *testing.T) {
	project := &projects.Project{
		Name:         "Demo",
		Artwork:      "https://github.com/cncf/artwork/tree/main/projects/demo",
		Repositories: []projects.RepositoryEntry{{URL: "https://github.com/demo/old"}},
		Social: map[string]string{
			"slack":              "https://demo.slack.com",
			"stackoverflow":      "https://stackoverflow.com/questions/tagged/demo",
			"bluesky":            "https://bsky.app/profile/demo.io",
			"discord":            "https://discord.gg/demo",
			"github_discussions": "https://github.com/demo/old/discussions",
		},
	}

	want := []string{
		"              slack_url: https://demo.slack.com",
		"              audits:",
		"                - date: '2020-01-02'",
		"                  type: security",
		"                  url: https://example.com/audit-1",
		"                  vendor: First Vendor",
		"              artwork_url: https://github.com/cncf/artwork/tree/main/projects/demo",
		"              stack_overflow_url: https://stackoverflow.com/questions/tagged/demo",
		"              bluesky_url: https://bsky.app/profile/demo.io",
		"              discord_url: https://discord.gg/demo",
		"              github_discussions_url: https://github.com/demo/old/discussions",
		"          - item:",
	}

	// Map iteration order is random, so repeat to catch any order leak.
	for run := 0; run < 20; run++ {
		root, lines := setupTest(syncFieldsLandscape)
		newLines, status := updateLandscape(root, project, lines)
		if status != statusUpdated {
			t.Fatalf("expected statusUpdated, got %v", status)
		}
		at := findLine(newLines, "slack_url:")
		got := newLines[at : at+len(want)]
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("run %d: unexpected extra block:\n%s", run, strings.Join(got, "\n"))
		}
	}
}

func TestSyncAuditsAppendOnly(t *testing.T) {
	project := &projects.Project{
		Name:         "Demo",
		Repositories: []projects.RepositoryEntry{{URL: "https://github.com/demo/old"}},
		Audits: []projects.Audit{
			// Already listed: kept as curated, even though the vendor differs.
			{Date: time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC), Type: "security", URL: "https://example.com/audit-1", Vendor: "Renamed"},
			// No vendor: the landscape schema requires one, so it is skipped.
			{Date: time.Date(2021, 3, 4, 0, 0, 0, 0, time.UTC), Type: "security", URL: "https://example.com/audit-2"},
			{Date: time.Date(2022, 5, 6, 0, 0, 0, 0, time.UTC), Type: "performance", URL: "https://example.com/audit-3", Vendor: "Vendor: Three"},
		},
	}

	root, lines := setupTest(syncFieldsLandscape)
	newLines, status := updateLandscape(root, project, lines)
	if status != statusUpdated {
		t.Fatalf("expected statusUpdated, got %v", status)
	}

	want := []string{
		"              audits:",
		"                - date: '2020-01-02'",
		"                  type: security",
		"                  url: https://example.com/audit-1",
		"                  vendor: First Vendor",
		"                - date: '2022-05-06'",
		"                  type: performance",
		"                  url: https://example.com/audit-3",
		`                  vendor: "Vendor: Three"`,
		"          - item:",
	}
	at := findLine(newLines, "audits:")
	if got := strings.Join(newLines[at:at+len(want)], "\n"); got != strings.Join(want, "\n") {
		t.Fatalf("unexpected audits block:\n%s", got)
	}
	if len(newLines) != len(lines)+4 {
		t.Errorf("expected exactly 4 added lines, got %d", len(newLines)-len(lines))
	}

	// Applying the result again is a no-op.
	root2, lines2 := setupTest(strings.Join(newLines, "\n"))
	if _, status := updateLandscape(root2, project, lines2); status != statusNoChanges {
		t.Errorf("second run should be a no-op, got status %v", status)
	}
}

func TestSyncAuditsCreatesExtraBlock(t *testing.T) {
	project := &projects.Project{
		Name:         "Other",
		Repositories: []projects.RepositoryEntry{{URL: "https://github.com/other/other"}},
		Audits: []projects.Audit{
			{Date: time.Date(2023, 7, 8, 0, 0, 0, 0, time.UTC), Type: "security", URL: "https://example.com/other", Vendor: "Vendor"},
		},
	}

	root, lines := setupTest(syncFieldsLandscape)
	newLines, status := updateLandscape(root, project, lines)
	if status != statusUpdated {
		t.Fatalf("expected statusUpdated, got %v", status)
	}
	want := []string{
		"            logo: other.svg",
		"            extra:",
		"              audits:",
		"                - date: '2023-07-08'",
		"                  type: security",
		"                  url: https://example.com/other",
		"                  vendor: Vendor",
	}
	at := findLine(newLines, "logo: other.svg")
	if got := strings.Join(newLines[at:], "\n"); got != strings.Join(want, "\n") {
		t.Fatalf("unexpected item tail:\n%s", got)
	}

	root2, lines2 := setupTest(strings.Join(newLines, "\n"))
	if _, status := updateLandscape(root2, project, lines2); status != statusNoChanges {
		t.Errorf("second run should be a no-op, got status %v", status)
	}
}

func TestFindLogoName(t *testing.T) {
	root, _ := setupTest(syncFieldsLandscape)
	project := &projects.Project{
		Name:         "Other",
		Repositories: []projects.RepositoryEntry{{URL: "https://github.com/other/other"}},
	}
	if got := findLogoName(root, project); got != "other.svg" {
		t.Errorf("findLogoName = %q, want other.svg", got)
	}
	project.Name = "Missing"
	if got := findLogoName(root, project); got != "" {
		t.Errorf("findLogoName for an unmatched project = %q, want empty", got)
	}
}

func TestSyncArtworkOnlyFromCNCFArtwork(t *testing.T) {
	for _, artwork := range []string{
		"https://landscape.cncf.io/logos/demo.svg",
		"https://demo.io/brand/logo.png",
		"https://raw.githubusercontent.com/demo/old/main/logo.svg",
	} {
		root, lines := setupTest(syncFieldsLandscape)
		project := &projects.Project{
			Name:         "Demo",
			Artwork:      artwork,
			Repositories: []projects.RepositoryEntry{{URL: "https://github.com/demo/old"}},
		}
		if _, status := updateLandscape(root, project, lines); status != statusNoChanges {
			t.Errorf("artwork %q is outside cncf/artwork and must not be synced, got status %v", artwork, status)
		}
	}
}
