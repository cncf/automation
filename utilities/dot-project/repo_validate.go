package projects

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// RepoValidationResult captures the repository-wide checks that no single-file
// validator can perform: layout consistency, identifier agreement between
// project.yaml and maintainers.yaml, and team definitions shared across
// projects in the same GitHub organization.
type RepoValidationResult struct {
	RepoRoot string     `json:"repo_root" yaml:"repo_root"`
	Mode     LayoutMode `json:"mode" yaml:"mode"`
	Projects []string   `json:"projects,omitempty" yaml:"projects,omitempty"`
	Valid    bool       `json:"valid" yaml:"valid"`
	Errors   []string   `json:"errors,omitempty" yaml:"errors,omitempty"`
	Warnings []string   `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// teamDefinition records how one project defines a named team, so that
// definitions of the same team name can be compared across projects.
type teamDefinition struct {
	// name is the team name exactly as written, kept so the warning can show
	// the spellings a maintainer will recognize in their file.
	name      string
	projectID string
	members   []string
}

// teamKey normalizes a team name the way GitHub does when it derives a team's
// slug: teams are organization-scoped and matched by slug, so "Maintainers"
// and "maintainers" are one team even though the two files spell them
// differently.
func teamKey(name string) string {
	if slug := Slugify(name); slug != "" {
		return slug
	}
	return strings.ToLower(strings.TrimSpace(name))
}

// ValidateRepo runs the repository-wide checks for a .project repository.
//
// It does not re-run per-file validation: ValidateProjectStruct and
// ValidateMaintainersFile still own that. ValidateRepo owns only the rules
// that span files.
func ValidateRepo(repoRoot string) (*RepoValidationResult, error) {
	d, err := Discover(repoRoot)
	if err != nil {
		return nil, err
	}

	result := &RepoValidationResult{RepoRoot: d.RepoRoot, Mode: d.Mode}

	if d.Mode == LayoutMulti {
		validateMultiLayout(d, result)
	}

	validateProjectIdentities(d, result)

	result.Valid = len(result.Errors) == 0
	return result, nil
}

// validateMultiLayout checks the structural rules that only apply when an
// org.yaml is present.
func validateMultiLayout(d *Discovery, result *RepoValidationResult) {
	for _, err := range ValidateOrgStruct(*d.Org) {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", OrgFileName, err))
	}

	// A root project.yaml alongside org.yaml is ambiguous: tools could not
	// tell which landscape entry it owns, and designating one project as the
	// "root" project would impose a parent/child hierarchy on projects that
	// CNCF treats as peers.
	if fileExists(filepath.Join(d.RepoRoot, ProjectFileName)) {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"%s must not exist at the repository root when %s is present: each project owns its own %s inside its directory",
			ProjectFileName, OrgFileName, ProjectFileName))
	}

	if fileExists(filepath.Join(d.RepoRoot, MaintainersFileName)) {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"%s must not exist at the repository root when %s is present: each project owns its own %s inside its directory",
			MaintainersFileName, OrgFileName, MaintainersFileName))
	}

	for _, p := range d.Projects {
		if !fileExists(p.ProjectPath) {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"project %q: missing %s", p.ID, filepath.ToSlash(filepath.Join(p.Dir, ProjectFileName))))
		}
		if !fileExists(p.MaintainersPath) {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"project %q: missing %s", p.ID, filepath.ToSlash(filepath.Join(p.Dir, MaintainersFileName))))
		}
	}

	undeclared, err := d.undeclaredProjectDirs()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return
	}
	for _, dir := range undeclared {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"directory %q contains a %s but is not declared in %s: every project must be listed there or its files are ignored",
			dir, ProjectFileName, OrgFileName))
	}
}

// validateProjectIdentities loads each discovered project and checks that the
// identifiers used to route CNCF resources agree across files.
func validateProjectIdentities(d *Discovery, result *RepoValidationResult) {
	seenSlugs := make(map[string]string)
	teamsByName := make(map[string][]teamDefinition)

	for _, p := range d.Projects {
		label := p.ID
		if label == "" {
			label = filepath.ToSlash(p.ProjectPath)
		}

		if !fileExists(p.ProjectPath) {
			continue // already reported by validateMultiLayout
		}

		project, err := LoadProjectFromFile(p.ProjectPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", fileLabel(p, p.ProjectPath), err))
			continue
		}

		slug := project.Slug
		if slug != "" {
			// Report the declared id where there is one: that is the key every
			// tool routes on, so listing a mismatched slug here would imply a
			// project the repository does not actually declare.
			listed := slug
			if p.ID != "" {
				listed = p.ID
			}
			result.Projects = append(result.Projects, listed)

			if previous, ok := seenSlugs[slug]; ok {
				result.Errors = append(result.Errors, fmt.Sprintf(
					"duplicate slug %q: declared by both %q and %q", slug, previous, label))
			}
			seenSlugs[slug] = label
		}

		// In a multi-project repo the directory id is the key every tool
		// routes on, so a slug that disagrees with it would silently send a
		// project's metadata to the wrong place.
		if p.ID != "" && slug != "" && p.ID != slug {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"project %q: slug in %s is %q but %s declares id %q; they must match",
				p.ID, filepath.ToSlash(filepath.Join(p.Dir, ProjectFileName)), slug, OrgFileName, p.ID))
		}

		if p.MaintainersPath == "" || !fileExists(p.MaintainersPath) {
			continue
		}

		config, err := LoadMaintainersFromFile(p.MaintainersPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", fileLabel(p, p.MaintainersPath), err))
			continue
		}

		validateMaintainerIdentity(d, p, project, config, result, teamsByName)
	}

	reportSharedTeams(teamsByName, result)
}

// fileLabel identifies the exact file an error came from. In a multi-project
// repo the project id is the clearest handle, but it is ambiguous between the
// two files in a project directory, and in a legacy single-project repo there
// is no id at all — so always name the file, qualified by the id when there is
// one.
func fileLabel(p DiscoveredProject, path string) string {
	if p.ID == "" {
		return filepath.ToSlash(path)
	}
	return fmt.Sprintf("%s (%s)", filepath.ToSlash(filepath.Join(p.Dir, filepath.Base(path))), p.ID)
}

// validateMaintainerIdentity checks one maintainers.yaml against the
// project.yaml beside it and against the organization it belongs to.
func validateMaintainerIdentity(
	d *Discovery,
	p DiscoveredProject,
	project Project,
	config MaintainersConfig,
	result *RepoValidationResult,
	teamsByName map[string][]teamDefinition,
) {
	rel := filepath.ToSlash(filepath.Join(p.Dir, MaintainersFileName))

	for _, entry := range config.Maintainers {
		if entry.ProjectID != "" && project.Slug != "" && entry.ProjectID != project.Slug {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: project_id %q does not match the slug %q in %s",
				rel, entry.ProjectID, project.Slug,
				filepath.ToSlash(filepath.Join(p.Dir, ProjectFileName))))
		}

		if d.Org != nil && entry.Org != "" && !strings.EqualFold(entry.Org, d.Org.Org) {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s: org %q does not match the org %q declared in %s",
				rel, entry.Org, d.Org.Org, OrgFileName))
		}

		projectID := entry.ProjectID
		if projectID == "" {
			projectID = project.Slug
		}
		for _, team := range entry.Teams {
			if team.Name == "" {
				continue
			}
			members, _ := normalizeHandles(team.Members)
			lowered := make([]string, 0, len(members))
			for _, m := range members {
				lowered = append(lowered, strings.ToLower(m))
			}
			sort.Strings(lowered)
			key := teamKey(team.Name)
			teamsByName[key] = append(teamsByName[key], teamDefinition{
				name:      team.Name,
				projectID: projectID,
				members:   lowered,
			})
		}
	}
}

// reportSharedTeams warns when the same team is defined by more than one
// project with different members.
//
// GitHub teams are organization-scoped and addressed by slug, so two projects
// naming the same team describe one team — even if they spell it differently.
// That is legitimate and intentional for groups whose remit spans projects,
// and the effective membership is the union of the definitions. The warning
// exists so that a human confirms the sharing is deliberate rather than an
// accidental collision on a generic name.
func reportSharedTeams(teamsByName map[string][]teamDefinition, result *RepoValidationResult) {
	names := make([]string, 0, len(teamsByName))
	for name := range teamsByName {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		defs := teamsByName[name]
		projectIDs := make(map[string]bool)
		for _, def := range defs {
			projectIDs[def.projectID] = true
		}
		if len(projectIDs) < 2 {
			continue
		}

		identical := true
		for _, def := range defs[1:] {
			if strings.Join(def.members, ",") != strings.Join(defs[0].members, ",") {
				identical = false
				break
			}
		}
		if identical {
			continue
		}

		owners := make([]string, 0, len(projectIDs))
		for id := range projectIDs {
			owners = append(owners, id)
		}
		sort.Strings(owners)

		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"team %q is defined with different members by projects %s; GitHub teams are org-scoped, so its effective membership is the union of those definitions",
			teamLabel(name, defs), strings.Join(owners, ", ")))
	}
}

// teamLabel names a team in a warning. When the projects spell it differently
// every spelling is listed, because otherwise a maintainer would search their
// file for a name that is not in it.
func teamLabel(key string, defs []teamDefinition) string {
	seen := make(map[string]bool)
	var spellings []string
	for _, def := range defs {
		if def.name == "" || seen[def.name] {
			continue
		}
		seen[def.name] = true
		spellings = append(spellings, def.name)
	}
	sort.Strings(spellings)

	switch len(spellings) {
	case 0:
		return key
	case 1:
		return spellings[0]
	default:
		return fmt.Sprintf("%s (spelled %s)", key, strings.Join(spellings, ", "))
	}
}

// FormatRepoResult renders a repository validation result in the requested
// output format (text, json or yaml).
func FormatRepoResult(result *RepoValidationResult, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal repo result as JSON: %w", err)
		}
		return string(data) + "\n", nil
	case "yaml":
		data, err := yaml.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to marshal repo result as YAML: %w", err)
		}
		return string(data), nil
	default:
		return FormatRepoValidationResult(result), nil
	}
}

// FormatRepoValidationResult renders a repository validation result as
// human-readable text.
func FormatRepoValidationResult(result *RepoValidationResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Repository layout: %s\n", result.Mode))
	if len(result.Projects) > 0 {
		b.WriteString(fmt.Sprintf("Projects: %s\n", strings.Join(result.Projects, ", ")))
	}

	for _, w := range result.Warnings {
		b.WriteString(fmt.Sprintf("WARNING: %s\n", w))
	}

	if result.Valid {
		b.WriteString("Repository structure is valid.\n")
		return b.String()
	}

	b.WriteString("Repository structure is INVALID:\n")
	for _, e := range result.Errors {
		b.WriteString(fmt.Sprintf("  - %s\n", e))
	}
	return b.String()
}
