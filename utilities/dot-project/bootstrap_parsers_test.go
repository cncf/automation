package projects

import (
	"testing"
)

func TestParseCodeowners(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "basic handles",
			content: `* @alice @bob
/docs/ @carol`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "deduplicates handles",
			content: `* @alice @bob
/docs/ @alice @carol`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "skips team references",
			content: `* @org/team-name @alice
/docs/ @org/reviewers @bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "skips comment lines",
			content: `# This is a comment
# @notahandle
* @alice`,
			expected: []string{"alice"},
		},
		{
			name: "skips empty lines",
			content: `

* @alice

/docs/ @bob
`,
			expected: []string{"alice", "bob"},
		},
		{
			name:     "handles with leading @ stripped",
			content:  `* @alice @bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:     "email addresses are skipped",
			content:  `* @alice someone@example.com @bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "mixed team and user refs",
			content: `* @kubernetes/sig-node-reviewers @dims @thockin
/staging/ @kubernetes/sig-api-machinery-leads @liggitt`,
			expected: []string{"dims", "liggitt", "thockin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCodeowners(tt.content)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseCodeowners() returned %d handles, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
			}
			for i, handle := range got {
				if handle != tt.expected[i] {
					t.Errorf("parseCodeowners()[%d] = %q, want %q", i, handle, tt.expected[i])
				}
			}
		})
	}
}

func TestParseOwnersFile(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		expectedApprovers []string
		expectedReviewers []string
	}{
		{
			name: "basic YAML owners file",
			content: `approvers:
  - alice
  - bob
reviewers:
  - carol
  - dave`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: []string{"carol", "dave"},
		},
		{
			name: "approvers only",
			content: `approvers:
  - alice
  - bob`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: nil,
		},
		{
			name: "reviewers only",
			content: `reviewers:
  - carol`,
			expectedApprovers: nil,
			expectedReviewers: []string{"carol"},
		},
		{
			name:              "empty content",
			content:           "",
			expectedApprovers: nil,
			expectedReviewers: nil,
		},
		{
			name: "handles with @ prefix stripped",
			content: `approvers:
  - "@alice"
  - bob
reviewers:
  - "@carol"`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: []string{"carol"},
		},
		{
			name: "extra whitespace in handles",
			content: `approvers:
  -  alice 
  - bob`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: nil,
		},
		{
			name: "ignores unknown keys",
			content: `approvers:
  - alice
reviewers:
  - bob
emeritus_approvers:
  - carol
labels:
  - sig/node`,
			expectedApprovers: []string{"alice"},
			expectedReviewers: []string{"bob"},
		},
		{
			name: "handles comment lines in YAML",
			content: `# Top-level owners
approvers:
  - alice
  # inactive: bob
  - carol
reviewers:
  - dave`,
			expectedApprovers: []string{"alice", "carol"},
			expectedReviewers: []string{"dave"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approvers, reviewers := parseOwnersFile(tt.content)
			if len(approvers) != len(tt.expectedApprovers) {
				t.Fatalf("approvers: got %d, want %d\ngot:  %v\nwant: %v",
					len(approvers), len(tt.expectedApprovers), approvers, tt.expectedApprovers)
			}
			for i, h := range approvers {
				if h != tt.expectedApprovers[i] {
					t.Errorf("approvers[%d] = %q, want %q", i, h, tt.expectedApprovers[i])
				}
			}
			if len(reviewers) != len(tt.expectedReviewers) {
				t.Fatalf("reviewers: got %d, want %d\ngot:  %v\nwant: %v",
					len(reviewers), len(tt.expectedReviewers), reviewers, tt.expectedReviewers)
			}
			for i, h := range reviewers {
				if h != tt.expectedReviewers[i] {
					t.Errorf("reviewers[%d] = %q, want %q", i, h, tt.expectedReviewers[i])
				}
			}
		})
	}
}

func TestParseMaintainersFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "plain @handle extraction",
			content: `# Maintainers

@alice - Alice Smith
@bob - Bob Jones
@carol - Carol White`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "markdown table format",
			content: `# Maintainers

| Name | GitHub Handle | Affiliation |
|------|--------------|-------------|
| Alice Smith | @alice | ACME Corp |
| Bob Jones | @bob | BigCo |
| Carol White | @carol | StartupInc |`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "plain text name (handle) format",
			content: `Alice Smith (@alice)
Bob Jones (@bob)
Carol White (@carol)`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "github URLs as handles",
			content: `# Maintainers
- Alice Smith (https://github.com/alice)
- Bob Jones (https://github.com/bob)`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "mixed formats",
			content: `# Project Maintainers

@alice - Lead
Bob Jones (@bob) - Core
| Carol White | @carol | SIG Lead |
- https://github.com/dave`,
			expected: []string{"alice", "bob", "carol", "dave"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name: "deduplicates handles",
			content: `@alice - Lead
Alice Smith (@alice) - Also lead`,
			expected: []string{"alice"},
		},
		{
			name: "skips team references",
			content: `@alice
@org/some-team
@bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "handles without @ on lines with GitHub profile URLs",
			content: `- alice https://github.com/alice
- bob https://github.com/bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "markdown list with handles",
			content: `## Maintainers
- @alice
- @bob
- @carol`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "email addresses are not treated as handles",
			content: `@alice - alice@example.com
@bob - bob@example.com`,
			expected: []string{"alice", "bob"},
		},
		{
			name:     "handles with trailing punctuation",
			content:  `@alice, @bob, and @carol are maintainers.`,
			expected: []string{"alice", "bob", "carol"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMaintainersFile(tt.content)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseMaintainersFile() returned %d handles, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
			}
			for i, handle := range got {
				if handle != tt.expected[i] {
					t.Errorf("parseMaintainersFile()[%d] = %q, want %q", i, handle, tt.expected[i])
				}
			}
		})
	}
}

func TestParsePackageManagerURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantRegistry string
		wantID       string
	}{
		{"docker hub", "https://hub.docker.com/r/envoyproxy/envoy", "docker", "envoyproxy/envoy"},
		{"docker hub trailing slash", "https://hub.docker.com/r/envoyproxy/envoy/", "docker", "envoyproxy/envoy"},
		{"pypi", "https://pypi.org/project/kubernetes/", "pypi", "kubernetes"},
		{"npm", "https://www.npmjs.com/package/@grpc/grpc-js", "npm", "@grpc/grpc-js"},
		{"crates.io", "https://crates.io/crates/tokio", "cargo", "tokio"},
		{"rubygems", "https://rubygems.org/gems/grpc", "rubygems", "grpc"},
		{"go pkg.go.dev", "https://pkg.go.dev/github.com/open-telemetry/opentelemetry-go", "go", "github.com/open-telemetry/opentelemetry-go"},
		{"artifacthub helm", "https://artifacthub.io/packages/helm/prometheus-community/kube-prometheus-stack", "helm", "prometheus-community/kube-prometheus-stack"},
		{"ghcr.io", "https://ghcr.io/envoyproxy/envoy", "docker", "envoyproxy/envoy"},
		{"quay.io", "https://quay.io/repository/prometheus/prometheus", "docker", "prometheus/prometheus"},
		{"empty", "", "", ""},
		{"unknown host", "https://example.com/foo/bar", "", ""},
		{"no path", "https://hub.docker.com", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, id := parsePackageManagerURL(tt.url)
			if reg != tt.wantRegistry {
				t.Errorf("registry = %q, want %q", reg, tt.wantRegistry)
			}
			if id != tt.wantID {
				t.Errorf("identifier = %q, want %q", id, tt.wantID)
			}
		})
	}
}

func TestDetectPackageManagerFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"package.json", "npm"},
		{"Package.json", "npm"},
		{"go.mod", "go"},
		{"Cargo.toml", "cargo"},
		{"pyproject.toml", "pypi"},
		{"setup.py", "pypi"},
		{"Chart.yaml", "helm"},
		{"pom.xml", "maven"},
		{"build.gradle", "gradle"},
		{"README.md", ""},
		{"Dockerfile", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectPackageManagerFromFilename(tt.filename)
			if got != tt.want {
				t.Errorf("detectPackageManagerFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestParseGoMod(t *testing.T) {
	content := `module github.com/open-telemetry/opentelemetry-go

go 1.21

require (
	go.opentelemetry.io/otel v1.24.0
)`
	got := parseGoMod(content)
	want := "github.com/open-telemetry/opentelemetry-go"
	if got != want {
		t.Errorf("parseGoMod() = %q, want %q", got, want)
	}

	if parseGoMod("") != "" {
		t.Error("parseGoMod empty should return empty")
	}
}

func TestParsePackageJSON(t *testing.T) {
	content := `{
  "name": "@grpc/grpc-js",
  "version": "1.10.0",
  "description": "gRPC for Node.js"
}`
	got := parsePackageJSON(content)
	want := "@grpc/grpc-js"
	if got != want {
		t.Errorf("parsePackageJSON() = %q, want %q", got, want)
	}
}

func TestParseCargoToml(t *testing.T) {
	content := `[package]
name = "tokio"
version = "1.36.0"
edition = "2021"

[dependencies]
bytes = "1"
`
	got := parseCargoToml(content)
	want := "tokio"
	if got != want {
		t.Errorf("parseCargoToml() = %q, want %q", got, want)
	}
}

func TestParsePyprojectToml(t *testing.T) {
	t.Run("standard project section", func(t *testing.T) {
		content := `[project]
name = "kubernetes"
version = "29.0.0"
`
		got := parsePyprojectToml(content)
		if got != "kubernetes" {
			t.Errorf("got %q, want %q", got, "kubernetes")
		}
	})

	t.Run("poetry section", func(t *testing.T) {
		content := `[tool.poetry]
name = "my-project"
version = "0.1.0"
`
		got := parsePyprojectToml(content)
		if got != "my-project" {
			t.Errorf("got %q, want %q", got, "my-project")
		}
	})
}

func TestParseHelmChart(t *testing.T) {
	content := `apiVersion: v2
name: kube-prometheus-stack
description: A Helm chart
version: 56.0.0
`
	got := parseHelmChart(content)
	if got != "kube-prometheus-stack" {
		t.Errorf("got %q, want %q", got, "kube-prometheus-stack")
	}
}

func TestParseManifestForIdentifier(t *testing.T) {
	if parseManifestForIdentifier("maven", "<project/>") != "" {
		t.Error("maven should return empty (not supported)")
	}
	if parseManifestForIdentifier("go", "module example.com/foo\n") != "example.com/foo" {
		t.Error("go module should be parsed")
	}
}
