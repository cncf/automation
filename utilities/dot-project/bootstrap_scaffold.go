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
{{ if .CNCFSlackChannel }}cncf_slack_channel: "{{ .CNCFSlackChannel }}"{{ if isAutoDetected .Sources "cncf_slack_channel" }} # TODO: AUTO-DETECTED — please verify{{ end }}{{ if .CNCFSlackCandidates }}
# Also detected: {{ joinChannels .CNCFSlackCandidates }} — please verify the correct channel{{ end }}{{ else }}# TODO: Set CNCF Slack channel
# cncf_slack_channel: "#{{ .Slug }}"{{ end }}

maturity_log:
  - phase: "{{ or .MaturityPhase "sandbox" }}"
    date: "{{ formatTime .AcceptedDate }}"
    {{ if .TOCIssueURL }}issue: "{{ .TOCIssueURL }}"{{ if isAutoDetected .Sources "toc_issue_url" }} # TODO: AUTO-DETECTED — please verify{{ end }}{{ else }}issue: "https://github.com/cncf/toc/issues/XXX" # TODO: Set TOC issue URL{{ end }}

repositories:{{ if .Repositories }}{{ range .Repositories }}
  - "{{ . }}"{{ end }}{{ else }}
  # TODO: Add repository URLs
  - "https://github.com/{{ .GitHubOrg }}/{{ or .GitHubRepo .Slug }}"{{ end }}
{{ if .Website }}
website: "{{ .Website }}"{{ else }}
# TODO: Add project website
# website: "https://{{ .Slug }}.io"{{ end }}

artwork: "{{ if .Artwork }}{{ .Artwork }}{{ else }}{{ artworkURL .Slug }}{{ end }}"
{{ if .HasAdopters }}
adopters:
  path: "{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) "" "ADOPTERS.md" }}"{{ else }}
# TODO: Add ADOPTERS.md if your project tracks adopters
# adopters:
#   path: "{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) "" "ADOPTERS.md" }}"{{ end }}

# TODO: Add package manager identifiers if your project is distributed via registries
# package_managers:
#   docker: "{{ .GitHubOrg }}/{{ or .GitHubRepo .Slug }}"
{{ if .Social }}
social:{{ range $platform, $url := .Social }}
  {{ $platform }}: "{{ $url }}"{{ end }}{{ end }}

security:
  policy:
    path: "{{ if .SecurityPolicyURL }}{{ .SecurityPolicyURL }}{{ else }}{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) "" "SECURITY.md" }}{{ end }}"{{ if .SecurityContactURL }}
  contact:
    advisory_url: "{{ .SecurityContactURL }}"{{ else }}
  contact:
    advisory_url: "{{ githubAdvisoryURL .GitHubOrg (or .GitHubRepo .Slug) }}"{{ end }}

governance:
  contributing:
    path: "{{ if .ContributingURL }}{{ .ContributingURL }}{{ else }}{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) "" "CONTRIBUTING.md" }}{{ end }}"
  code_of_conduct:
    path: "{{ if .CodeOfConductURL }}{{ .CodeOfConductURL }}{{ else }}https://github.com/cncf/foundation/blob/main/code-of-conduct.md{{ end }}"

legal:
  license:
    path: "{{ if .LicenseURL }}{{ .LicenseURL }}{{ else }}{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) "" "LICENSE" }}{{ end }}"
  identity_type:
{{ if isAutoDetected .Sources "identity_type" }}    has_dco: {{ .HasDCO }} # AUTO-DETECTED — please verify
    has_cla: {{ .HasCLA }} # AUTO-DETECTED — please verify{{ else }}    has_dco: true
    has_cla: false{{ end }}
    dco_url:
      path: "https://developercertificate.org/"
{{ if .HasReadme }}
documentation:
  readme:
    path: "{{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) "" "README.md" }}"{{ end }}
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

maintainers:
  - project_id: "{{ .Slug }}"
    {{ if .GitHubOrg }}org: "{{ .GitHubOrg }}"{{ else }}# TODO: Set GitHub organization
    # org: "my-org"{{ end }}
    teams:
      - name: "project-maintainers"
        members:{{ if .Maintainers }}{{ range .Maintainers }}
          - {{ . }}{{ end }}{{ else }}
          # TODO: Add maintainer handles
          - github-handle{{ end }}{{ if .Reviewers }}
      - name: "reviewers"
        members:{{ range .Reviewers }}
          - {{ . }}{{ end }}{{ end }}
`

// readmeTemplate generates the README.md for the .project directory.
const readmeTemplate = `# {{ .Name }} ` + "`.project`" + `

` + "`.project`" + ` (dot-project) is a CNCF initiative to centralize and automate metadata management for all CNCF projects.
This repository holds the canonical metadata for [{{ .Name }}]({{ or .Website (printf "https://github.com/%s/%s" .GitHubOrg (or .GitHubRepo .Slug)) }}) and is maintained by the CNCF automation tooling.

## What's in this repo

| File | Purpose |
|------|---------|
| ` + "`project.yaml`" + ` | Canonical project metadata (name, maturity, repositories, governance links, …) |
| ` + "`maintainers.yaml`" + ` | Maintainer and reviewer roster used for drift detection and mailing-list sync |
| ` + "`CODEOWNERS`" + ` | Ensures PRs to this repo require maintainer approval |
| ` + "`.github/workflows/validate.yaml`" + ` | CI — validates ` + "`project.yaml`" + ` and ` + "`maintainers.yaml`" + ` on every PR |
| ` + "`.github/workflows/update-landscape.yml`" + ` | Automatically proposes landscape updates when ` + "`project.yaml`" + ` changes |

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

For more details, see the [{{ .Name }} security policy]({{ githubFileURL .GitHubOrg (or .GitHubRepo .Slug) "" "SECURITY.md" }}).
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
      - 'project.yaml'
      - 'maintainers.yaml'
  push:
    branches: [main]
    paths:
      - 'project.yaml'
      - 'maintainers.yaml'
  workflow_dispatch:

jobs:
  validate-project:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v4
        with:
          fetch-depth: 0

      - uses: cncf/automation/.github/actions/validate-project@95d25b12337a14e4a74f690c856f6903584e839e
        with:
          project_file: 'project.yaml'

  validate-maintainers:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v4
        with:
          fetch-depth: 0

      - uses: cncf/automation/.github/actions/validate-maintainers@95d25b12337a14e4a74f690c856f6903584e839e
        with:
          maintainers_file: 'maintainers.yaml'
          verify_maintainers: 'true'
        env:
          LFX_AUTH_TOKEN: ${{ secrets.LFX_AUTH_TOKEN }}
`

// updateLandscapeWorkflowContent is the SHA-pinned update-landscape.yml workflow.
const updateLandscapeWorkflowContent = `name: Update Landscape
on:
  push:
    branches: [main]
    paths:
      - 'project.yaml'
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - name: Checkout
        uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v4
        with:
          fetch-depth: 0

      - name: Update Landscape
        uses: cncf/automation/.github/actions/landscape-update@95d25b12337a14e4a74f690c856f6903584e839e
        with:
          project_file: 'project.yaml'
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
	"joinChannels": func(channels []string) string {
		quoted := make([]string, len(channels))
		for i, ch := range channels {
			quoted[i] = `"` + ch + `"`
		}
		return strings.Join(quoted, ", ")
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

// protectedFiles are never overwritten, even with --force.
var protectedFiles = map[string]bool{
	"project.yaml":     true,
	"maintainers.yaml": true,
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

// WriteScaffold writes the complete .project scaffold (8 files) to the
// specified directory. It will not overwrite existing project.yaml or
// maintainers.yaml files. Other files are skipped if they exist unless
// force is true.
func WriteScaffold(dir string, result *BootstrapResult, opts ...WriteScaffoldOption) error {
	cfg := writeScaffoldConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	// Helper: generate content from a Go template string
	tmplGen := func(tmplName, tmplContent string) func() ([]byte, error) {
		return func() ([]byte, error) {
			tmpl, err := template.New(tmplName).Funcs(templateFuncs).Parse(tmplContent)
			if err != nil {
				return nil, fmt.Errorf("parsing %s template: %w", tmplName, err)
			}
			view := struct {
				*BootstrapResult
				Name string
			}{BootstrapResult: result, Name: result.Name}
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

	// Helper: return static content
	staticGen := func(content string) func() ([]byte, error) {
		return func() ([]byte, error) { return []byte(content), nil }
	}

	type scaffoldFile struct {
		path     string
		generate func() ([]byte, error)
	}

	// Core files: always generated
	files := []scaffoldFile{
		{"project.yaml", func() ([]byte, error) { return GenerateProjectYAML(result) }},
		{"maintainers.yaml", func() ([]byte, error) { return GenerateMaintainersYAML(result) }},
		{"README.md", tmplGen("readme", readmeTemplate)},
		{".gitignore", staticGen(gitignoreContent)},
		{".github/workflows/validate.yaml", staticGen(validateWorkflowContent)},
		{".github/workflows/update-landscape.yml", staticGen(updateLandscapeWorkflowContent)},
	}

	// Conditional: SECURITY.md — skip if an existing security policy was discovered
	if result.SecurityPolicyURL == "" {
		files = append(files, scaffoldFile{"SECURITY.md", tmplGen("security", securityMDTemplate)})
	}

	// Conditional: CODEOWNERS — skip if it already exists on disk
	if _, err := os.Stat(filepath.Join(dir, "CODEOWNERS")); os.IsNotExist(err) {
		files = append(files, scaffoldFile{"CODEOWNERS", tmplGen("codeowners", codeownersTemplate)})
	}

	// Check if any protected files exist
	protectedExist := false
	for f := range protectedFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			protectedExist = true
			break
		}
	}

	// If protected files exist and force is NOT set, error out
	if protectedExist && !cfg.force {
		for f := range protectedFiles {
			if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
				return fmt.Errorf("%s already exists in %s; refusing to overwrite (use --force to regenerate auxiliary files)", f, dir)
			}
		}
	}

	for _, f := range files {
		fullPath := filepath.Join(dir, f.path)

		// Check if file already exists on disk
		existingData, existsErr := os.ReadFile(fullPath)
		fileExists := existsErr == nil

		if fileExists {
			// Generate the content so we can compare
			newContent, err := f.generate()
			if err != nil {
				return fmt.Errorf("generating %s: %w", f.path, err)
			}

			identical := bytes.Equal(existingData, newContent)

			// Never overwrite protected files
			if protectedFiles[f.path] {
				if !identical {
					log.Printf("Skipping %s: protected file differs from generated version", f.path)
					logDiffSummary(f.path, existingData, newContent)
				}
				continue
			}

			// Skip existing auxiliary files unless force is set
			if !cfg.force {
				if !identical {
					log.Printf("Skipping %s: file differs from generated version (use --force to overwrite)", f.path)
					logDiffSummary(f.path, existingData, newContent)
				}
				continue
			}

			// force is set — overwrite auxiliary, but skip if identical
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

		// File does not exist — generate and write
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
