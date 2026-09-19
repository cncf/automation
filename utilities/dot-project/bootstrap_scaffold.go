package projects

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// projectYAMLTemplate is the template for generating project.yaml.
// It produces valid YAML with TODO comments for fields that need manual input.
const projectYAMLTemplate = `# .project metadata for {{ .Name }}
# Documentation: https://github.com/cncf/automation/tree/main/utilities/dot-project
{{ range .TODOs }}
# TODO: {{ . }}{{ end }}

schema_version: "1.0.0"
slug: "{{ .Slug }}"
name: "{{ .Name }}"
description: "{{ .Description }}"
type: "project"
{{ if .ProjectLead }}project_lead: "{{ .ProjectLead }}"{{ if isAutoDetected .Sources "project_lead" }} # TODO: AUTO-DETECTED — please verify{{ end }}{{ else }}# TODO: Set project lead GitHub handle
# project_lead: "github-handle"{{ end }}
{{ if .SlackChannels }}slack_channels:{{ if isAutoDetected .Sources "slack_channels" }} # TODO: AUTO-DETECTED — please verify the channel(s) below{{ end }}{{ range .SlackChannels }}
  - name: "{{ .Name }}"{{ if .Link }}
    link: "{{ .Link }}"{{ end }}{{ if .Workspace }}
    workspace: "{{ .Workspace }}"{{ end }}{{ if .Primary }}
    primary: true{{ end }}{{ end }}{{ else }}# TODO: Set CNCF Slack channel(s)
# slack_channels:
#   - name: "#{{ .Slug }}"
#     primary: true{{ end }}

maturity_log:
  - phase: "{{ or .MaturityPhase "sandbox" }}"
    date: "{{ formatTime .AcceptedDate }}"
    {{ if .TOCIssueURL }}issue: "{{ .TOCIssueURL }}"{{ if isAutoDetected .Sources "toc_issue_url" }} # TODO: AUTO-DETECTED — please verify{{ end }}{{ else }}issue: "https://github.com/cncf/toc/issues/XXX" # TODO: Set TOC issue URL{{ end }}

repositories:{{ if .Repositories }}{{ if isAutoDetected .Sources "primary_repo" }} # TODO: AUTO-DETECTED primary — please verify{{ end }}{{ range .Repositories }}
  - url: "{{ . }}"{{ if isPrimaryRepo $.PrimaryRepo . }}
    primary: true
    # tags: [core, sig-apps] # Optional{{ end }}{{ end }}{{ else }}
  # TODO: Add repository URLs
  - url: "https://github.com/{{ .GitHubOrg }}/{{ or .GitHubRepo .Slug }}"
    primary: true
    # tags: [core, sig-apps] # Optional{{ end }}
{{ if .Website }}
website: "{{ .Website }}"{{ else }}
# TODO: Add project website
# website: "https://{{ .Slug }}.io"{{ end }}

artwork: "{{ if .Artwork }}{{ .Artwork }}{{ else }}{{ artworkURL .Slug }}{{ end }}"
{{ if .HasAdopters }}
adopters:
  path: "{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) .DefaultBranch "ADOPTERS.md" }}"{{ else }}
# TODO: Add ADOPTERS.md if your project tracks adopters
# adopters:
#   path: "{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) .DefaultBranch "ADOPTERS.md" }}"{{ end }}

{{ if .PackageManagers }}
package_managers:{{ if isAutoDetected .Sources "package_managers" }} # AUTO-DETECTED — please verify{{ end }}{{ range $registry, $id := .PackageManagers }}
  {{ $registry }}: "{{ $id }}"{{ end }}{{ else }}
# TODO: Add package manager identifiers if your project is distributed via registries
# package_managers:
#   docker: "{{ .GitHubOrg }}/{{ or .GitHubRepo .Slug }}"{{ end }}
{{ if .Social }}
social:{{ range $platform, $url := .Social }}
  {{ $platform }}: "{{ $url }}"{{ end }}{{ end }}

security:
  policy:
    path: "{{ if .SecurityPolicyURL }}{{ .SecurityPolicyURL }}{{ else }}{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) .DefaultBranch "SECURITY.md" }}{{ end }}"{{ if .SecurityContactURL }}
  contact:
    advisory_url: "{{ .SecurityContactURL }}"{{ else }}
  contact:
    advisory_url: "{{ githubAdvisoryURL .GitHubOrg (or .GitHubRepo .Slug) }}"{{ end }}

governance:
  contributing:
    path: "{{ if .ContributingURL }}{{ .ContributingURL }}{{ else }}{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) .DefaultBranch "CONTRIBUTING.md" }}{{ end }}"
  code_of_conduct:
    path: "{{ if .CodeOfConductURL }}{{ .CodeOfConductURL }}{{ else }}https://github.com/cncf/foundation/blob/main/code-of-conduct.md{{ end }}"

legal:
  license:
    path: "{{ if .LicenseURL }}{{ .LicenseURL }}{{ else }}{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) .DefaultBranch "LICENSE" }}{{ end }}"
  identity_type:
{{ if isAutoDetected .Sources "identity_type" }}    has_dco: {{ .HasDCO }} # AUTO-DETECTED — please verify
    has_cla: {{ .HasCLA }} # AUTO-DETECTED — please verify{{ else }}    has_dco: true
    has_cla: false{{ end }}
    dco_url:
      path: "https://developercertificate.org/"
{{ if .HasReadme }}
documentation:
  readme:
    path: "{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) .DefaultBranch "README.md" }}"{{ end }}
{{ if and .LandscapeCategory .LandscapeSubcategory }}
landscape:
  category: "{{ .LandscapeCategory }}"
  subcategory: "{{ .LandscapeSubcategory }}"{{ end }}
{{ if .CLOMonitorScore }}
# CLOMonitor Score: {{ printf "%.0f" .CLOMonitorScore.Global }}/100
# Documentation: {{ printf "%.0f" .CLOMonitorScore.Documentation }} | License: {{ printf "%.0f" .CLOMonitorScore.License }} | Best Practices: {{ printf "%.0f" .CLOMonitorScore.BestPractices }} | Security: {{ printf "%.0f" .CLOMonitorScore.Security }}{{ end }}
`

// maintainersYAMLTemplate is the template for generating maintainers.yaml.
const maintainersYAMLTemplate = `# Maintainer roster for {{ .Name }}
# Documentation: https://github.com/cncf/automation/tree/main/utilities/dot-project
{{ if not .Maintainers }}
# TODO: Add maintainer GitHub handles{{ end }}

#  Maintainers: Please connect your GitHub handle to your LFID on openprofile.dev to enable automatic access to CNCF resources. The system will use your primary email address for setup.
maintainers:
  - project_id: "{{ .Slug }}"
    {{ if .GitHubOrg }}org: "{{ .GitHubOrg }}"{{ else }}# TODO: Set GitHub organization
    # org: "my-org"{{ end }}
    teams:
      - name: "{{ .Slug }}-maintainers"
        members:{{ if .Maintainers }}{{ range .Maintainers }}
          - {{ . }}{{ end }}{{ else }}
          # TODO: Add maintainer handles
          - github-handle{{ end }}
      # Unmanaged teams: teams with "managed: false" are tracked in this
      # file for documentation but are excluded from CNCF resource
      # provisioning (mailing lists, service desk, Copilot seats, etc).
      #
      # Active contributors who do not need full CNCF resource access:
      # - name: "reviewers"
      #   managed: false
      #   members:
      #     - reviewer-handle
      #
      # Former maintainers (remove this section if not applicable):
      # - name: "emeritus"
      #   managed: false
      #   members: []
`

// readmeTemplate generates the README.md for the .project directory.
const readmeTemplate = `# {{ if .Projects }}{{ .GitHubOrg }}{{ else }}{{ .Name }}{{ end }} ` + "`.project`" + `

` + "`.project`" + ` (dot-project) is a CNCF initiative to centralize and automate metadata management for all CNCF projects.
{{ if .Projects }}This repository holds the canonical metadata for the CNCF projects maintained in the ` + "`{{ .GitHubOrg }}`" + ` organization, and is maintained by the CNCF automation tooling.{{ else }}This repository holds the canonical metadata for [{{ .Name }}]({{ or .Website (printf "https://github.com/%s/%s" .GitHubOrg (or .GitHubRepo .Slug)) }}) and is maintained by the CNCF automation tooling.{{ end }}

## What's in this repo

| File | Purpose |
|------|---------|
{{ if .Projects }}| ` + "`org.yaml`" + ` | Index of the CNCF projects maintained in this organization |
| ` + "`<project>/project.yaml`" + ` | Canonical metadata for one project (name, maturity, repositories, governance links, …) |
| ` + "`<project>/maintainers.yaml`" + ` | Maintainer and reviewer roster for one project |
{{ else }}| ` + "`project.yaml`" + ` | Canonical project metadata (name, maturity, repositories, governance links, …) |
| ` + "`maintainers.yaml`" + ` | Maintainer and reviewer roster used for drift detection and mailing-list sync |
{{ end }}| ` + "`CODEOWNERS`" + ` | Ensures PRs to this repo require maintainer approval |
| ` + "`.github/workflows/validate.yaml`" + ` | CI — validates ` + "`project.yaml`" + ` and ` + "`maintainers.yaml`" + ` on every PR |
| ` + "`.github/workflows/update-landscape.yml`" + ` | Automatically proposes landscape updates when ` + "`project.yaml`" + ` changes |
{{ if .Projects }}
## Projects in this repository

This GitHub organization maintains more than one CNCF project. Each is a
separate project from CNCF's point of view — its own maturity, its own
landscape entry — so each owns a directory with its own metadata, and
` + "`org.yaml`" + ` indexes them:

| Project | Directory |
|---------|-----------|
{{ range .Projects }}| {{ .Name }} | ` + "`{{ .Slug }}/`" + ` |
{{ end }}
Add or remove a project by editing ` + "`org.yaml`" + ` and its directory together;
validation fails if the two disagree. See the
[repository layouts reference](https://github.com/cncf/automation/tree/main/utilities/dot-project#repository-layouts).
{{ end }}
## Keeping metadata up to date

Open a pull request against this repository to update any metadata field.
The validate workflow will check schema correctness and block merge if validation fails.

> **Note:** This repository was bootstrapped automatically from public sources (CNCF landscape, CLOMonitor, GitHub governance files).
> Some fields are best-effort guesses marked with ` + "`# TODO: AUTO-DETECTED — please verify`" + ` in the YAML files and should be confirmed by the project maintainers.

## Resources

- [` + "`.project`" + ` documentation](https://github.com/cncf/automation/tree/main/utilities/dot-project)
- [Schema reference](https://github.com/cncf/automation/blob/main/utilities/dot-project/SCHEMA.md)
- [CNCF Automation repository](https://github.com/cncf/automation)
`

// securityMDTemplate generates the SECURITY.md for the .project directory.
const securityMDTemplate = `# Security Policy

## Reporting Security Issues

The {{ .Name }} maintainers take security seriously. We appreciate your efforts to responsibly disclose your findings.

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them through our [private vulnerability reporting]({{ githubAdvisoryURL .GitHubOrg (or .GitHubRepo .Slug) }}) form.

For more details, see the [{{ .Name }} security policy]({{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) .DefaultBranch "SECURITY.md" }}).
`

// codeownersTemplate generates the CODEOWNERS file.
const codeownersTemplate = `# CODEOWNERS for .project metadata repository
# Changes to project metadata require maintainer review.
{{ if .Maintainers }}* {{ range .Maintainers }}@{{ . }} {{ end }}{{ else }}# TODO: Add CODEOWNERS
# * @maintainer-handle{{ end }}
`

// gitignoreContent is the static .gitignore content.
const gitignoreContent = `.cache/
.DS_Store
Thumbs.db
.idea/
.vscode/
*~
*.swp
`

// validateWorkflowContent is the SHA-pinned validate.yaml workflow.
const validateWorkflowContent = `name: Validate Project Metadata

on:
  pull_request:
    paths:
      - 'org.yaml'
      - 'project.yaml'
      - 'maintainers.yaml'
      # Multi-project repositories keep each project in its own directory,
      # exactly one level deep.
      - '*/project.yaml'
      - '*/maintainers.yaml'
  push:
    branches: [main]
    paths:
      - 'org.yaml'
      - 'project.yaml'
      - 'maintainers.yaml'
      - '*/project.yaml'
      - '*/maintainers.yaml'
  workflow_dispatch:

jobs:
  validate-project:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0

      - uses: cncf/automation/.github/actions/validate-project@main
        # No project_file input: the action discovers the repository layout,
        # so this step is identical for single- and multi-project repositories.

  validate-maintainers:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0

      - uses: cncf/automation/.github/actions/validate-maintainers@main
        with:
          # No maintainers_file input: the action discovers every maintainers
          # file in the repository.
          # Disabled until the LFX LLT issue is resolved. Validation is done manually for now.
          verify_maintainers: 'false'
        env:
          LFX_AUTH_TOKEN: ${{ secrets.LFX_AUTH_TOKEN }}
`

// updateLandscapeWorkflowContent is the SHA-pinned update-landscape.yml workflow.
const updateLandscapeWorkflowContent = `name: Update Landscape
on:
  push:
    branches: [main]
    paths:
      - 'org.yaml'
      - 'project.yaml'
      # Multi-project repositories keep each project in its own directory,
      # exactly one level deep.
      - '*/project.yaml'
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - name: Checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0

      - name: Update Landscape
        uses: cncf/automation/.github/actions/landscape-update@main
        with:
          # No project_file input: the action discovers every project and
          # opens one landscape pull request per project.
          token: ${{ secrets.LANDSCAPE_REPO_TOKEN }}
`

// templateFuncs provides helper functions for templates.
var templateFuncs = template.FuncMap{
	"formatTime": func(t time.Time) string {
		if t.IsZero() {
			return time.Now().Format("2006-01-02T15:04:05Z")
		}
		return t.Format("2006-01-02T15:04:05Z")
	},
	"or": func(a, b string) string {
		if a != "" {
			return a
		}
		return b
	},
	"githubFileURL": func(org, repo, branch, path string) string {
		if org == "" || repo == "" {
			return path // fallback to relative if no org/repo
		}
		if branch == "" {
			branch = "main"
		}
		return fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", org, repo, branch, path)
	},
	"githubAdvisoryURL": func(org, repo string) string {
		if org == "" || repo == "" {
			return ""
		}
		return fmt.Sprintf("https://github.com/%s/%s/security/advisories/new", org, repo)
	},
	"artworkURL": func(slug string) string {
		return fmt.Sprintf("https://github.com/cncf/artwork/tree/master/projects/%s", slug)
	},
	"isAutoDetected": func(sources map[string]string, key string) bool {
		if sources == nil {
			return false
		}
		_, ok := sources[key]
		return ok
	},
	"isPrimaryRepo": func(primaryURL, repoURL string) bool {
		return primaryURL != "" && strings.EqualFold(primaryURL, repoURL)
	},
}

// writeScaffoldConfig holds options for WriteScaffold.
type writeScaffoldConfig struct {
	force bool
}

// WriteScaffoldOption configures WriteScaffold behaviour.
type WriteScaffoldOption func(*writeScaffoldConfig)

// WithForce allows WriteScaffold to overwrite auxiliary files (README.md,
// .gitignore, workflows, SECURITY.md, CODEOWNERS) but never the core
// metadata files (project.yaml, maintainers.yaml).
func WithForce() WriteScaffoldOption {
	return func(c *writeScaffoldConfig) { c.force = true }
}

// GenerateProjectYAML produces the project.yaml content from a BootstrapResult.
func GenerateProjectYAML(result *BootstrapResult) ([]byte, error) {
	tmpl, err := template.New("project").Funcs(templateFuncs).Parse(projectYAMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing project template: %w", err)
	}

	// Build a view with safe defaults
	view := struct {
		*BootstrapResult
		Name string // override to ensure non-empty
	}{
		BootstrapResult: result,
		Name:            result.Name,
	}
	if view.Name == "" {
		view.Name = result.Slug
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("executing project template: %w", err)
	}

	// Clean up excessive blank lines (more than 2 consecutive)
	output := cleanBlankLines(buf.String())

	return []byte(output), nil
}

// GenerateMaintainersYAML produces the maintainers.yaml content from a BootstrapResult.
func GenerateMaintainersYAML(result *BootstrapResult) ([]byte, error) {
	tmpl, err := template.New("maintainers").Parse(maintainersYAMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing maintainers template: %w", err)
	}

	// Build view
	view := struct {
		*BootstrapResult
		Name string
	}{
		BootstrapResult: result,
		Name:            result.Name,
	}
	if view.Name == "" {
		view.Name = result.Slug
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("executing maintainers template: %w", err)
	}

	output := cleanBlankLines(buf.String())

	return []byte(output), nil
}


// scaffoldFile is one generated file and the rule for overwriting it.
type scaffoldFile struct {
	path     string
	generate func() ([]byte, error)
	// protected marks the core metadata files. They are never overwritten,
	// not even with --force: they hold hand-maintained content that no
	// regeneration can reproduce.
	protected bool
}

// tmplGen builds a generator that renders tmplContent against result.
//
// projects is supplied only for repository-level templates that must describe
// every project in a multi-project repository; it is empty otherwise, which is
// what those templates branch on.
func tmplGen(tmplName, tmplContent string, result *BootstrapResult, projects ...OrgProject) func() ([]byte, error) {
	return func() ([]byte, error) {
		tmpl, err := template.New(tmplName).Funcs(templateFuncs).Parse(tmplContent)
		if err != nil {
			return nil, fmt.Errorf("parsing %s template: %w", tmplName, err)
		}
		view := struct {
			*BootstrapResult
			Name     string
			Projects []OrgProject
		}{BootstrapResult: result, Name: result.Name, Projects: projects}
		if view.Name == "" {
			view.Name = result.Slug
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, view); err != nil {
			return nil, fmt.Errorf("executing %s template: %w", tmplName, err)
		}
		return []byte(cleanBlankLines(buf.String())), nil
	}
}

func staticGen(content string) func() ([]byte, error) {
	return func() ([]byte, error) { return []byte(content), nil }
}

// projectScaffoldFiles returns the files that describe a single project.
//
// prefix is empty in a single-project repository and the project's directory
// in a multi-project one, which is the only structural difference between the
// two layouts.
func projectScaffoldFiles(prefix string, result *BootstrapResult) []scaffoldFile {
	return []scaffoldFile{
		{
			path:      filepath.Join(prefix, ProjectFileName),
			generate:  func() ([]byte, error) { return GenerateProjectYAML(result) },
			protected: true,
		},
		{
			path:      filepath.Join(prefix, MaintainersFileName),
			generate:  func() ([]byte, error) { return GenerateMaintainersYAML(result) },
			protected: true,
		},
	}
}

// readmeData is the template input for the generated README. It carries the
// repository's project list alongside one project's metadata, because the
// README describes the repository rather than any single project. Projects is
// empty in a single-project repository, which is what the template branches on.
type readmeData struct {
	*BootstrapResult
	Projects []OrgProject
}

// repoScaffoldFiles returns the files that describe the repository as a whole.
// They exist once, at the root, in both layouts — including the workflows,
// which discover the layout at run time rather than being generated per
// project.
//
// entries is nil for a single-project repository.
func repoScaffoldFiles(dir string, result *BootstrapResult, entries []OrgProject) []scaffoldFile {
	files := []scaffoldFile{
		{path: "README.md", generate: tmplGen("readme", readmeTemplate, result, entries...)},
		{path: ".gitignore", generate: staticGen(gitignoreContent)},
		{path: ".github/workflows/validate.yaml", generate: staticGen(validateWorkflowContent)},
		{path: ".github/workflows/update-landscape.yml", generate: staticGen(updateLandscapeWorkflowContent)},
	}

	// Skip SECURITY.md if an existing security policy was discovered.
	if result.SecurityPolicyURL == "" {
		files = append(files, scaffoldFile{path: "SECURITY.md", generate: tmplGen("security", securityMDTemplate, result)})
	}

	// Skip CODEOWNERS if it already exists on disk.
	if _, err := os.Stat(filepath.Join(dir, "CODEOWNERS")); os.IsNotExist(err) {
		files = append(files, scaffoldFile{path: "CODEOWNERS", generate: tmplGen("codeowners", codeownersTemplate, result)})
	}

	return files
}

// WriteScaffold writes the complete .project scaffold for a single-project
// repository. It will not overwrite existing project.yaml or maintainers.yaml
// files. Other files are skipped if they exist unless force is true.
func WriteScaffold(dir string, result *BootstrapResult, opts ...WriteScaffoldOption) error {
	cfg := writeScaffoldConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	files := append(projectScaffoldFiles("", result), repoScaffoldFiles(dir, result, nil)...)
	return writeScaffoldFiles(dir, files, cfg)
}

// WriteMultiScaffold writes a repository that holds several CNCF projects:
// an org.yaml index, one directory per project containing that project's
// metadata, and a single shared set of repository-level files.
//
// results must be in the same order as entries, one per project.
func WriteMultiScaffold(dir, org string, entries []OrgProject, results []*BootstrapResult, opts ...WriteScaffoldOption) error {
	if len(entries) != len(results) {
		return fmt.Errorf("got %d projects but %d bootstrap results", len(entries), len(results))
	}
	if len(entries) == 0 {
		return fmt.Errorf("cannot write a multi-project scaffold with no projects")
	}

	cfg := writeScaffoldConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	// org.yaml is protected for the same reason as project.yaml: once a
	// repository declares its projects, regenerating the index could silently
	// drop one that was added by hand.
	files := []scaffoldFile{{
		path:      OrgFileName,
		generate:  func() ([]byte, error) { return GenerateOrgYAML(org, entries) },
		protected: true,
	}}

	for i, entry := range entries {
		if msg := validateProjectDirName(entry.Slug); msg != "" {
			return fmt.Errorf("project %q: %s", entry.Name, msg)
		}
		files = append(files, projectScaffoldFiles(entry.Slug, results[i])...)
	}

	// Repository-level files are generated from the first project only
	// because they describe the repository, not any one project. The README
	// lists every project separately.
	files = append(files, repoScaffoldFiles(dir, results[0], entries)...)

	return writeScaffoldFiles(dir, files, cfg)
}

func writeScaffoldFiles(dir string, files []scaffoldFile, cfg writeScaffoldConfig) error {
	// If any protected file already exists and force is not set, refuse the
	// whole run rather than half-writing a repository.
	if !cfg.force {
		for _, f := range files {
			if !f.protected {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, f.path)); err == nil {
				return fmt.Errorf("%s already exists in %s; refusing to overwrite (use --force to regenerate auxiliary files)", f.path, dir)
			}
		}
	}

	for _, f := range files {
		fullPath := filepath.Join(dir, f.path)

		existingData, existsErr := os.ReadFile(fullPath)
		fileExists := existsErr == nil

		if fileExists {
			newContent, err := f.generate()
			if err != nil {
				return fmt.Errorf("generating %s: %w", f.path, err)
			}

			identical := bytes.Equal(existingData, newContent)

			if f.protected {
				if !identical {
					log.Printf("Skipping %s: protected file differs from generated version", f.path)
					logDiffSummary(f.path, existingData, newContent)
				}
				continue
			}

			if !cfg.force {
				if !identical {
					log.Printf("Skipping %s: file differs from generated version (use --force to overwrite)", f.path)
					logDiffSummary(f.path, existingData, newContent)
				}
				continue
			}

			if identical {
				continue
			}
			log.Printf("Overwriting %s (--force)", f.path)

			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("creating directory for %s: %w", f.path, err)
			}
			if err := os.WriteFile(fullPath, newContent, 0644); err != nil {
				return fmt.Errorf("writing %s: %w", f.path, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.path, err)
		}
		content, err := f.generate()
		if err != nil {
			return fmt.Errorf("generating %s: %w", f.path, err)
		}
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.path, err)
		}
	}
	return nil
}

// logDiffSummary logs a concise summary of differences between existing and
// generated file content so the user can see what would change.
func logDiffSummary(path string, existing, generated []byte) {
	oldLines := strings.Split(string(existing), "\n")
	newLines := strings.Split(string(generated), "\n")

	// Count added/removed/changed lines with a simple LCS-free approach
	added, removed := 0, 0
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}
	for i := 0; i < maxLen; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		if oldLine != newLine {
			if i >= len(oldLines) {
				added++
			} else if i >= len(newLines) {
				removed++
			} else {
				added++
				removed++
			}
		}
	}

	if added == 0 && removed == 0 {
		return
	}

	log.Printf("  %s: %d line(s) differ (+%d/-%d)", path, added+removed, added, removed)

	// Show first few differing lines (max 5) as context
	shown := 0
	for i := 0; i < maxLen && shown < 5; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		if oldLine != newLine {
			if i < len(oldLines) && oldLine != "" {
				log.Printf("  - %s", truncate(oldLine, 120))
			}
			if i < len(newLines) && newLine != "" {
				log.Printf("  + %s", truncate(newLine, 120))
			}
			shown++
		}
	}
	remaining := (added + removed) - shown
	if remaining > 0 {
		log.Printf("  ... and %d more difference(s)", remaining)
	}
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// cleanBlankLines reduces runs of 3+ consecutive blank lines to at most 2.
func cleanBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	blankCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}
