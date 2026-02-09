package projects

import (
	"testing"
	"time"
)

func TestProjectToLandscapeEntry(t *testing.T) {
	project := validBaseProject()
	project.Website = "https://test-project.io"
	project.Artwork = "https://github.com/cncf/artwork/tree/master/projects/test-project"
	project.Social = map[string]string{
		"twitter": "https://twitter.com/testproject",
		"slack":   "https://testproject.slack.com",
	}

	entry := ProjectToLandscapeEntry(project)

	if entry.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %q", entry.Name)
	}
	if entry.HomepageURL != "https://test-project.io" {
		t.Errorf("expected homepage_url 'https://test-project.io', got %q", entry.HomepageURL)
	}
	if entry.RepoURL != "https://github.com/test/repo" {
		t.Errorf("expected repo_url 'https://github.com/test/repo', got %q", entry.RepoURL)
	}
	if entry.Twitter != "https://twitter.com/testproject" {
		t.Errorf("expected twitter URL, got %q", entry.Twitter)
	}
	if entry.Project != "sandbox" {
		t.Errorf("expected project 'sandbox', got %q", entry.Project)
	}
	if entry.Extra["slug"] != "test-project" {
		t.Errorf("expected slug 'test-project', got %v", entry.Extra["slug"])
	}
}

func TestCompareLandscapeEntries(t *testing.T) {
	current := LandscapeEntry{
		Name:    "Old Name",
		Project: "sandbox",
	}
	desired := LandscapeEntry{
		Name:    "New Name",
		Project: "incubating",
		Extra:   map[string]interface{}{"slug": "test"},
	}

	diff := CompareLandscapeEntries(current, desired)
	if !diff.HasChanges {
		t.Error("expected changes")
	}
	if len(diff.Changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(diff.Changes))
	}

	// Test no changes
	same := LandscapeEntry{Name: "Same", Project: "sandbox"}
	diff2 := CompareLandscapeEntries(same, same)
	if diff2.HasChanges {
		t.Error("expected no changes")
	}
}

func TestProjectMaturityExtraction(t *testing.T) {
	project := validBaseProject()
	project.MaturityLog = []MaturityEntry{
		{Phase: "sandbox", Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/1"},
		{Phase: "incubating", Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/2"},
		{Phase: "graduated", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Issue: "https://github.com/cncf/toc/issues/3"},
	}

	entry := ProjectToLandscapeEntry(project)
	if entry.Project != "graduated" {
		t.Errorf("expected 'graduated' (last phase), got %q", entry.Project)
	}
}
