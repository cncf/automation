package projects

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// OrgProject is one CNCF project found in a GitHub organization.
//
// A GitHub organization usually maps one-to-one onto a CNCF project, but a
// handful host several — SPIFFE and SPIRE are separately graduated projects
// that both live under github.com/spiffe. The landscape is the authority on
// which projects CNCF recognizes, so it is what decides whether a repository
// needs the multi-project layout.
type OrgProject struct {
	// Name is the landscape item name. The landscape updater matches on it,
	// so it must be carried through to project.yaml unchanged.
	Name     string
	Slug     string
	Maturity string
	RepoURL  string
	Repo     string
}

// FindOrgProjects returns every CNCF project in the landscape whose primary
// repository belongs to org, sorted by name.
//
// Archived projects are excluded: they no longer receive metadata updates, so
// counting them would push a repository into the multi-project layout for a
// project nobody maintains.
func FindOrgProjects(org string, client *http.Client, baseURL string) ([]OrgProject, error) {
	org = strings.TrimSpace(strings.ToLower(org))
	if org == "" {
		return nil, nil
	}

	root, err := fetchLandscapeRoot(client, baseURL)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var found []OrgProject

	for _, cat := range root.Landscape {
		for _, subcat := range cat.Subcategories {
			for _, item := range subcat.Items {
				if item.Project == "" || strings.EqualFold(item.Project, "archived") {
					continue
				}
				itemOrg, repo := splitGitHubRepoURL(item.RepoURL)
				if itemOrg == "" || !strings.EqualFold(itemOrg, org) {
					continue
				}
				if seen[item.Name] {
					continue
				}
				seen[item.Name] = true

				found = append(found, OrgProject{
					Name:     item.Name,
					Slug:     Slugify(item.Name),
					Maturity: item.Project,
					RepoURL:  item.RepoURL,
					Repo:     repo,
				})
			}
		}
	}

	sort.Slice(found, func(i, j int) bool { return found[i].Name < found[j].Name })
	return found, nil
}

// splitGitHubRepoURL extracts the organization and repository from a GitHub
// repository URL. It returns empty strings for URLs hosted elsewhere.
func splitGitHubRepoURL(repoURL string) (string, string) {
	u := strings.TrimSpace(repoURL)
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "git@github.com:", "github.com/"} {
		if strings.HasPrefix(strings.ToLower(u), strings.ToLower(prefix)) {
			u = u[len(prefix):]
			parts := strings.Split(strings.Trim(u, "/"), "/")
			if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
				return "", ""
			}
			return parts[0], strings.TrimSuffix(parts[1], ".git")
		}
	}
	return "", ""
}

// Slugify converts a project display name into the lowercase, hyphenated form
// used for the slug, the org.yaml project id, and the project directory name.
func Slugify(name string) string {
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}

// GenerateOrgYAML renders the org.yaml index for a multi-project repository.
//
// org.yaml is deliberately only an index: it names the projects and where they
// live, and nothing else. Shared metadata is not hoisted into it, because a
// value that looks shared today (a security contact, an adopters list) is
// routinely project-specific, and hoisting it would mean no single file fully
// describes a project.
func GenerateOrgYAML(org string, entries []OrgProject) ([]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("cannot generate %s with no projects", OrgFileName)
	}

	var b strings.Builder
	b.WriteString("# Index of the CNCF projects maintained in this GitHub organization.\n")
	b.WriteString("# Documentation: https://github.com/cncf/automation/tree/main/utilities/dot-project\n")
	b.WriteString("#\n")
	b.WriteString("# Each project owns a directory containing its own project.yaml and\n")
	b.WriteString("# maintainers.yaml. Remove an entry here only when CNCF archives the project.\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("schema_version: %q\n", SupportedOrgSchemaVersions[0]))
	b.WriteString(fmt.Sprintf("org: %q\n", org))
	b.WriteString("\n")
	b.WriteString("projects:\n")
	for _, e := range entries {
		if e.Name != "" {
			label := e.Name
			if e.Maturity != "" {
				label = fmt.Sprintf("%s (%s)", e.Name, e.Maturity)
			}
			b.WriteString(fmt.Sprintf("  # %s\n", label))
		}
		b.WriteString(fmt.Sprintf("  - id: %q\n", e.Slug))
	}

	return []byte(b.String()), nil
}
