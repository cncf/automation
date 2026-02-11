package projects

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LandscapeEntry represents a project entry in the CNCF landscape
type LandscapeEntry struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description,omitempty"`
	HomepageURL string                 `yaml:"homepage_url,omitempty"`
	RepoURL     string                 `yaml:"repo_url,omitempty"`
	Logo        string                 `yaml:"logo,omitempty"`
	Twitter     string                 `yaml:"twitter,omitempty"`
	Project     string                 `yaml:"project,omitempty"`
	Extra       map[string]interface{} `yaml:"extra,omitempty"`
}

// LandscapeDiff represents changes needed to sync landscape with project.yaml
type LandscapeDiff struct {
	ProjectSlug string            `json:"project_slug"`
	Changes     []LandscapeChange `json:"changes"`
	HasChanges  bool              `json:"has_changes"`
}

// LandscapeChange represents a single field change
type LandscapeChange struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// ProjectToLandscapeEntry converts a Project to a LandscapeEntry
func ProjectToLandscapeEntry(project Project) LandscapeEntry {
	entry := LandscapeEntry{
		Name:        project.Name,
		Description: project.Description,
		HomepageURL: project.Website,
		Logo:        project.Artwork,
		Extra:       make(map[string]interface{}),
	}

	// Use first repository as repo_url
	if len(project.Repositories) > 0 {
		entry.RepoURL = project.Repositories[0]
	}

	// Get twitter URL from social
	if twitter, ok := project.Social["twitter"]; ok {
		entry.Twitter = twitter
	}

	// Get current maturity (last entry in log)
	if len(project.MaturityLog) > 0 {
		entry.Project = project.MaturityLog[len(project.MaturityLog)-1].Phase
	}

	// Set slug in extra
	if project.Slug != "" {
		entry.Extra["slug"] = project.Slug
	}

	return entry
}

// CompareLandscapeEntries compares current landscape entry with project-derived entry
func CompareLandscapeEntries(current, desired LandscapeEntry) LandscapeDiff {
	diff := LandscapeDiff{
		ProjectSlug: fmt.Sprintf("%v", desired.Extra["slug"]),
	}

	if current.Name != desired.Name {
		diff.Changes = append(diff.Changes, LandscapeChange{"name", current.Name, desired.Name})
	}
	if current.Description != desired.Description {
		diff.Changes = append(diff.Changes, LandscapeChange{"description", current.Description, desired.Description})
	}
	if current.HomepageURL != desired.HomepageURL {
		diff.Changes = append(diff.Changes, LandscapeChange{"homepage_url", current.HomepageURL, desired.HomepageURL})
	}
	if current.RepoURL != desired.RepoURL {
		diff.Changes = append(diff.Changes, LandscapeChange{"repo_url", current.RepoURL, desired.RepoURL})
	}
	if current.Logo != desired.Logo {
		diff.Changes = append(diff.Changes, LandscapeChange{"logo", current.Logo, desired.Logo})
	}
	if current.Project != desired.Project {
		diff.Changes = append(diff.Changes, LandscapeChange{"project", current.Project, desired.Project})
	}

	diff.HasChanges = len(diff.Changes) > 0
	return diff
}

// FormatLandscapeDiff formats a diff as human-readable text
func FormatLandscapeDiff(diff LandscapeDiff) string {
	if !diff.HasChanges {
		return fmt.Sprintf("Landscape entry for %s is up to date.\n", diff.ProjectSlug)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Landscape changes needed for %s:\n", diff.ProjectSlug))
	for _, change := range diff.Changes {
		b.WriteString(fmt.Sprintf("  %s: %q -> %q\n", change.Field, change.OldValue, change.NewValue))
	}
	return b.String()
}

// LoadProjectFromFile reads and parses a project.yaml file
func LoadProjectFromFile(path string) (Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Project{}, fmt.Errorf("failed to read project file: %w", err)
	}

	var project Project
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&project); err != nil {
		return Project{}, fmt.Errorf("failed to parse project YAML: %w", err)
	}

	return project, nil
}
