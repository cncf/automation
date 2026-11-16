package projects

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SupportedOrgSchemaVersions lists the org.yaml schema versions this tooling
// supports. org.yaml is a new file with its own schema line, independent of
// the project.yaml schema version.
var SupportedOrgSchemaVersions = []string{"1.0.0"}

// OrgConfig is the root of org.yaml. Its presence in a .project repository
// declares that the GitHub organization hosts more than one CNCF project, and
// it is the authoritative index of those projects.
//
// org.yaml is a pure index: it carries no project metadata and no inherited
// defaults. Every project.yaml remains complete and standalone so that any
// consumer can read a single file to get a project's full metadata.
type OrgConfig struct {
	SchemaVersion string            `json:"schema_version" yaml:"schema_version"`
	Org           string            `json:"org" yaml:"org"`
	Projects      []OrgProjectEntry `json:"projects" yaml:"projects"`
}

// OrgProjectEntry declares one CNCF project hosted in the organization.
type OrgProjectEntry struct {
	// ID is the project identifier. It must match the slug in the project's
	// project.yaml and the project_id in its maintainers.yaml.
	ID string `json:"id" yaml:"id"`

	// Path is the directory holding the project's files, relative to the
	// repository root. It is a single path segment and defaults to ID.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// ResolvedPath returns the directory for this project, defaulting to the ID
// when path is omitted.
func (e OrgProjectEntry) ResolvedPath() string {
	if e.Path != "" {
		return e.Path
	}
	return e.ID
}

// LoadOrgFromFile reads and parses an org.yaml file. Unknown fields are
// rejected, matching the behaviour of LoadProjectFromFile.
func LoadOrgFromFile(path string) (OrgConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return OrgConfig{}, fmt.Errorf("failed to read org file: %w", err)
	}

	var org OrgConfig
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&org); err != nil {
		return OrgConfig{}, fmt.Errorf("failed to parse org YAML: %w", err)
	}

	return org, nil
}

// ValidateOrgStruct validates an org.yaml document in isolation. Cross-file
// rules (directories exist, slugs match, teams agree) live in ValidateRepo.
func ValidateOrgStruct(org OrgConfig) []string {
	var errors []string

	if org.SchemaVersion == "" {
		errors = append(errors, "schema_version is required")
	} else {
		supported := false
		for _, v := range SupportedOrgSchemaVersions {
			if org.SchemaVersion == v {
				supported = true
				break
			}
		}
		if !supported {
			errors = append(errors, fmt.Sprintf("unsupported org schema_version %q (supported: %s)",
				org.SchemaVersion, strings.Join(SupportedOrgSchemaVersions, ", ")))
		}
	}

	if strings.TrimSpace(org.Org) == "" {
		errors = append(errors, "org is required")
	} else if strings.ContainsAny(org.Org, "/\\ ") {
		errors = append(errors, fmt.Sprintf("org must be a GitHub organization name without slashes or spaces, got: %s", org.Org))
	}

	if len(org.Projects) == 0 {
		errors = append(errors, "projects must list at least one project")
	}

	seenIDs := make(map[string]bool)
	seenPaths := make(map[string]bool)
	for i, entry := range org.Projects {
		if entry.ID == "" {
			errors = append(errors, fmt.Sprintf("projects[%d].id is required", i))
			continue
		}
		if !isValidSlug(entry.ID) {
			errors = append(errors, fmt.Sprintf("projects[%d].id must be lowercase alphanumeric with hyphens, got: %s", i, entry.ID))
		}
		if seenIDs[entry.ID] {
			errors = append(errors, fmt.Sprintf("duplicate project id %q in org.yaml", entry.ID))
		}
		seenIDs[entry.ID] = true

		path := entry.ResolvedPath()
		if msg := validateProjectDirName(path); msg != "" {
			errors = append(errors, fmt.Sprintf("projects[%d].path %s", i, msg))
		}
		if seenPaths[path] {
			errors = append(errors, fmt.Sprintf("duplicate project path %q in org.yaml", path))
		}
		seenPaths[path] = true
	}

	return errors
}

// validateProjectDirName enforces that a project directory is a single path
// segment directly under the repository root. Keeping project directories one
// level deep is what allows the shipped workflows to use explicit path filters
// (for example '*/project.yaml') instead of ambiguous '**' globs.
func validateProjectDirName(path string) string {
	switch {
	case path == "":
		return "cannot be empty"
	case strings.ContainsAny(path, "/\\"):
		return fmt.Sprintf("must be a single directory name directly under the repository root, got: %s", path)
	case path == "." || path == "..":
		return fmt.Sprintf("must be a directory name, got: %s", path)
	case strings.HasPrefix(path, "."):
		return fmt.Sprintf("must not start with a dot, got: %s", path)
	}
	return ""
}
