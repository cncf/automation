package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LayoutMode describes how a .project repository is laid out.
type LayoutMode string

const (
	// LayoutSingle is the original layout: project.yaml and maintainers.yaml
	// at the repository root, representing exactly one CNCF project.
	LayoutSingle LayoutMode = "single"

	// LayoutMulti is the layout for a GitHub organization that hosts more
	// than one CNCF project: an org.yaml index at the root plus one
	// directory per project.
	LayoutMulti LayoutMode = "multi"
)

// DiscoveredProject is one project found in a .project repository, with paths
// resolved against the repository root.
type DiscoveredProject struct {
	// ID is the declared project id from org.yaml. It is empty in
	// single-project repos, where the identity comes from the project.yaml
	// slug instead.
	ID string

	// Dir is the project directory relative to the repository root. It is
	// empty for the root project in a single-project repo.
	Dir string

	// ProjectPath is the path to project.yaml. It is always set, even when
	// the file is missing, so callers can report precisely what is absent.
	ProjectPath string

	// MaintainersPath is the path to maintainers.yaml. In a single-project
	// repo it is empty when the file does not exist, because the roster is
	// optional there. In a multi-project repo it is always set.
	MaintainersPath string
}

// Discovery is the resolved layout of a .project repository.
type Discovery struct {
	RepoRoot string
	Mode     LayoutMode

	// Org is the parsed org.yaml, or nil in single-project repos.
	Org *OrgConfig

	Projects []DiscoveredProject
}

// Discover resolves the layout of the .project repository rooted at repoRoot.
//
// Precedence:
//
//  1. org.yaml exists       -> multi-project; org.yaml is authoritative
//  2. root project.yaml     -> single project (the original layout)
//  3. neither               -> error
//
// Discover reports layout, not correctness: a repository whose org.yaml
// declares a directory that does not exist still discovers successfully, so
// that ValidateRepo can report exactly which files are missing. Discover only
// fails when the layout itself cannot be determined or org.yaml cannot be read.
func Discover(repoRoot string) (*Discovery, error) {
	if repoRoot == "" {
		repoRoot = "."
	}

	orgPath := filepath.Join(repoRoot, OrgFileName)
	if fileExists(orgPath) {
		org, err := LoadOrgFromFile(orgPath)
		if err != nil {
			return nil, err
		}

		d := &Discovery{RepoRoot: repoRoot, Mode: LayoutMulti, Org: &org}
		for _, entry := range org.Projects {
			dir := entry.ResolvedPath()
			d.Projects = append(d.Projects, DiscoveredProject{
				ID:              entry.ID,
				Dir:             dir,
				ProjectPath:     filepath.Join(repoRoot, dir, ProjectFileName),
				MaintainersPath: filepath.Join(repoRoot, dir, MaintainersFileName),
			})
		}
		return d, nil
	}

	projectPath := filepath.Join(repoRoot, ProjectFileName)
	if fileExists(projectPath) {
		p := DiscoveredProject{ProjectPath: projectPath}
		maintainersPath := filepath.Join(repoRoot, MaintainersFileName)
		if fileExists(maintainersPath) {
			p.MaintainersPath = maintainersPath
		}
		return &Discovery{
			RepoRoot: repoRoot,
			Mode:     LayoutSingle,
			Projects: []DiscoveredProject{p},
		}, nil
	}

	return nil, fmt.Errorf("no %s or %s found in %s: not a .project repository", OrgFileName, ProjectFileName, repoRoot)
}

// ProjectPaths returns the project.yaml path of every discovered project.
func (d *Discovery) ProjectPaths() []string {
	paths := make([]string, 0, len(d.Projects))
	for _, p := range d.Projects {
		paths = append(paths, p.ProjectPath)
	}
	return paths
}

// MaintainersPaths returns the maintainers.yaml path of every discovered
// project, skipping projects that have none.
func (d *Discovery) MaintainersPaths() []string {
	paths := make([]string, 0, len(d.Projects))
	for _, p := range d.Projects {
		if p.MaintainersPath != "" {
			paths = append(paths, p.MaintainersPath)
		}
	}
	return paths
}

// undeclaredProjectDirs returns directories directly under the repository root
// that contain a project.yaml but are not declared in org.yaml. Such a
// directory is almost always a project that was added on disk without being
// registered, so its files would be silently skipped by every tool.
func (d *Discovery) undeclaredProjectDirs() ([]string, error) {
	declared := make(map[string]bool, len(d.Projects))
	for _, p := range d.Projects {
		declared[p.Dir] = true
	}

	entries, err := os.ReadDir(d.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository root: %w", err)
	}

	var undeclared []string
	for _, entry := range entries {
		if !entry.IsDir() || declared[entry.Name()] {
			continue
		}
		if validateProjectDirName(entry.Name()) != "" {
			continue // skip dot-directories such as .github and .git
		}
		if fileExists(filepath.Join(d.RepoRoot, entry.Name(), ProjectFileName)) {
			undeclared = append(undeclared, entry.Name())
		}
	}
	return undeclared, nil
}

// FindMaintainersFiles returns every maintainers.yaml at the root of root or
// one level below it, sorted by path.
//
// Unlike Discover, this does not require a valid layout. It exists for the
// base side of a diff, where the checkout may predate a layout change (for
// example a roster that has since moved from the root into a project
// directory) and therefore cannot be discovered through org.yaml.
func FindMaintainersFiles(root string) ([]string, error) {
	if root == "" {
		root = "."
	}

	var found []string
	if fileExists(filepath.Join(root, MaintainersFileName)) {
		found = append(found, filepath.Join(root, MaintainersFileName))
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", root, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || validateProjectDirName(entry.Name()) != "" {
			continue
		}
		candidate := filepath.Join(root, entry.Name(), MaintainersFileName)
		if fileExists(candidate) {
			found = append(found, candidate)
		}
	}

	sort.Strings(found)
	return found, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
