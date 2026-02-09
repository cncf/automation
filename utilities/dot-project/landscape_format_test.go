package projects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// FormatLandscapeDiff
// ---------------------------------------------------------------------------

func TestFormatLandscapeDiff_NoChanges(t *testing.T) {
	diff := LandscapeDiff{
		ProjectSlug: "my-project",
		HasChanges:  false,
	}

	got := FormatLandscapeDiff(diff)
	if !strings.Contains(got, "up to date") {
		t.Errorf("expected 'up to date' message, got %q", got)
	}
	if !strings.Contains(got, "my-project") {
		t.Errorf("expected project slug in output, got %q", got)
	}
}

func TestFormatLandscapeDiff_MultipleChanges(t *testing.T) {
	diff := LandscapeDiff{
		ProjectSlug: "cool-proj",
		HasChanges:  true,
		Changes: []LandscapeChange{
			{Field: "name", OldValue: "Old", NewValue: "New"},
			{Field: "homepage_url", OldValue: "https://old.io", NewValue: "https://new.io"},
			{Field: "logo", OldValue: "old-logo.svg", NewValue: "new-logo.svg"},
		},
	}

	got := FormatLandscapeDiff(diff)

	if !strings.Contains(got, "cool-proj") {
		t.Errorf("expected slug in header, got %q", got)
	}
	// Each change should appear with field name, old value, and new value.
	for _, c := range diff.Changes {
		if !strings.Contains(got, c.Field) {
			t.Errorf("expected field %q in output", c.Field)
		}
		if !strings.Contains(got, c.OldValue) {
			t.Errorf("expected old value %q in output", c.OldValue)
		}
		if !strings.Contains(got, c.NewValue) {
			t.Errorf("expected new value %q in output", c.NewValue)
		}
	}
	// The header should say "changes needed"
	if !strings.Contains(got, "changes needed") {
		t.Errorf("expected 'changes needed' header, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// LoadProjectFromFile
// ---------------------------------------------------------------------------

func TestLoadProjectFromFile_ValidYAML(t *testing.T) {
	content := `schema_version: "1.0.0"
slug: "test-proj"
name: "Test Proj"
description: "A test project"
repositories:
  - "https://github.com/test/repo"
maturity_log:
  - phase: "sandbox"
    date: 2024-01-15T00:00:00Z
    issue: "https://github.com/cncf/toc/issues/100"
website: "https://test-proj.io"
artwork: "https://artwork.example.com/logo.svg"
`
	tmp := filepath.Join(t.TempDir(), "project.yaml")
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	proj, err := LoadProjectFromFile(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.Name != "Test Proj" {
		t.Errorf("expected name 'Test Proj', got %q", proj.Name)
	}
	if proj.Slug != "test-proj" {
		t.Errorf("expected slug 'test-proj', got %q", proj.Slug)
	}
	if proj.Website != "https://test-proj.io" {
		t.Errorf("expected website, got %q", proj.Website)
	}
	if len(proj.Repositories) != 1 || proj.Repositories[0] != "https://github.com/test/repo" {
		t.Errorf("unexpected repositories: %v", proj.Repositories)
	}
	if len(proj.MaturityLog) != 1 || proj.MaturityLog[0].Phase != "sandbox" {
		t.Errorf("unexpected maturity log: %v", proj.MaturityLog)
	}
}

func TestLoadProjectFromFile_InvalidYAML(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(tmp, []byte("{{invalid yaml: [unbalanced"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadProjectFromFile(tmp)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse project YAML") {
		t.Errorf("expected parse error message, got %q", err.Error())
	}
}

func TestLoadProjectFromFile_MissingFile(t *testing.T) {
	_, err := LoadProjectFromFile("/nonexistent/path/project.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read project file") {
		t.Errorf("expected read error message, got %q", err.Error())
	}
}

func TestLoadProjectFromFile_UnknownFields(t *testing.T) {
	content := `schema_version: "1.0.0"
slug: "test-proj"
name: "Test Proj"
description: "desc"
repositories: []
maturity_log: []
totally_bogus_field: "should fail"
`
	tmp := filepath.Join(t.TempDir(), "unknown.yaml")
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadProjectFromFile(tmp)
	if err == nil {
		t.Fatal("expected error for unknown fields (KnownFields=true), got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse project YAML") {
		t.Errorf("expected parse error message, got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// CompareLandscapeEntries – extended branch coverage
// ---------------------------------------------------------------------------

func TestCompareLandscapeEntriesExtended_LogoDifference(t *testing.T) {
	current := LandscapeEntry{
		Name: "Proj",
		Logo: "old-logo.svg",
	}
	desired := LandscapeEntry{
		Name:  "Proj",
		Logo:  "new-logo.svg",
		Extra: map[string]interface{}{"slug": "proj"},
	}

	diff := CompareLandscapeEntries(current, desired)
	if !diff.HasChanges {
		t.Fatal("expected changes for logo diff")
	}
	if len(diff.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(diff.Changes))
	}
	if diff.Changes[0].Field != "logo" {
		t.Errorf("expected field 'logo', got %q", diff.Changes[0].Field)
	}
	if diff.Changes[0].OldValue != "old-logo.svg" {
		t.Errorf("expected old value 'old-logo.svg', got %q", diff.Changes[0].OldValue)
	}
	if diff.Changes[0].NewValue != "new-logo.svg" {
		t.Errorf("expected new value 'new-logo.svg', got %q", diff.Changes[0].NewValue)
	}
}

func TestCompareLandscapeEntriesExtended_ProjectDifference(t *testing.T) {
	current := LandscapeEntry{
		Name:    "Proj",
		Project: "sandbox",
	}
	desired := LandscapeEntry{
		Name:    "Proj",
		Project: "graduated",
		Extra:   map[string]interface{}{"slug": "proj"},
	}

	diff := CompareLandscapeEntries(current, desired)
	if !diff.HasChanges {
		t.Fatal("expected changes for project (maturity) diff")
	}
	found := false
	for _, c := range diff.Changes {
		if c.Field == "project" {
			found = true
			if c.OldValue != "sandbox" || c.NewValue != "graduated" {
				t.Errorf("unexpected values: old=%q new=%q", c.OldValue, c.NewValue)
			}
		}
	}
	if !found {
		t.Error("expected a 'project' change in diff")
	}
}

func TestCompareLandscapeEntriesExtended_IdenticalEntries(t *testing.T) {
	entry := LandscapeEntry{
		Name:        "Same Project",
		Description: "desc",
		HomepageURL: "https://same.io",
		RepoURL:     "https://github.com/same/repo",
		Logo:        "logo.svg",
		Project:     "incubating",
		Extra:       map[string]interface{}{"slug": "same"},
	}

	diff := CompareLandscapeEntries(entry, entry)
	if diff.HasChanges {
		t.Errorf("expected no changes for identical entries, got %d changes", len(diff.Changes))
	}
	if len(diff.Changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(diff.Changes))
	}
}

func TestCompareLandscapeEntriesExtended_AllFieldsDifferent(t *testing.T) {
	current := LandscapeEntry{
		Name:        "Old Name",
		Description: "Old desc",
		HomepageURL: "https://old.io",
		RepoURL:     "https://github.com/old/repo",
		Logo:        "old-logo.svg",
		Project:     "sandbox",
	}
	desired := LandscapeEntry{
		Name:        "New Name",
		Description: "New desc",
		HomepageURL: "https://new.io",
		RepoURL:     "https://github.com/new/repo",
		Logo:        "new-logo.svg",
		Project:     "graduated",
		Extra:       map[string]interface{}{"slug": "test"},
	}

	diff := CompareLandscapeEntries(current, desired)
	if !diff.HasChanges {
		t.Fatal("expected changes")
	}
	if len(diff.Changes) != 6 {
		t.Errorf("expected 6 changes (all fields), got %d", len(diff.Changes))
	}

	expectedFields := map[string]bool{
		"name": false, "description": false, "homepage_url": false,
		"repo_url": false, "logo": false, "project": false,
	}
	for _, c := range diff.Changes {
		expectedFields[c.Field] = true
	}
	for field, seen := range expectedFields {
		if !seen {
			t.Errorf("expected change for field %q was missing", field)
		}
	}
}
