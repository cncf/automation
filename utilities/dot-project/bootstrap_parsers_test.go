package projects

import (
	"testing"
)

func TestExtractSlackChannel(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "messages URL",
			content:  "Join us at https://cloud-native.slack.com/messages/my-project",
			expected: "#my-project",
		},
		{
			name:     "archives URL",
			content:  "Join us at https://cloud-native.slack.com/archives/my-project",
			expected: "#my-project",
		},
		{
			name:     "channels URL",
			content:  "Join us at https://cloud-native.slack.com/channels/my-project",
			expected: "#my-project",
		},
		{
			name:     "channels URL in markdown link",
			content:  "[#my-project](https://cloud-native.slack.com/channels/my-project)",
			expected: "#my-project",
		},
		{
			name:     "channels URL with trailing slash",
			content:  "https://cloud-native.slack.com/channels/my-project/",
			expected: "#my-project",
		},
		{
			name:     "hash channel on same line as Slack keyword",
			content:  "Slack: #my-project",
			expected: "#my-project",
		},
		{
			name:     "hash channel on same line as CNCF keyword",
			content:  "CNCF Slack channel: #my-project",
			expected: "#my-project",
		},
		{
			name: "channel on separate line near Slack mention",
			content: `## Community
Join our Slack workspace
Channel: #my-project`,
			expected: "#my-project",
		},
		{
			name: "channel on line 3 lines after Slack mention",
			content: `Join us on Slack:
Some info
More info
Channel: #my-project`,
			expected: "#my-project",
		},
		{
			name:     "no slack reference at all",
			content:  "This project is great!",
			expected: "",
		},
		{
			name: "channel too far from Slack mention",
			content: `Join us on Slack
line1
line2
line3
line4
Channel: #my-project`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSlackChannel(tt.content)
			if got != tt.expected {
				t.Errorf("extractSlackChannel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractSlackChannels(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:     "single channel from URL",
			content:  "Join us at https://cloud-native.slack.com/messages/my-project",
			expected: []string{"#my-project"},
		},
		{
			name: "multiple channels from URLs",
			content: `Join [#envoy](https://cloud-native.slack.com/channels/envoy) or
[#envoy-dev](https://cloud-native.slack.com/channels/envoy-dev) for development discussions.`,
			expected: []string{"#envoy", "#envoy-dev"},
		},
		{
			name: "deduplicates same channel from URL and context",
			content: `Slack: #my-project
More info at https://cloud-native.slack.com/messages/my-project`,
			expected: []string{"#my-project"},
		},
		{
			name: "mixed URL and keyword sources",
			content: `Join [#envoy](https://cloud-native.slack.com/channels/envoy)
For mobile: Slack channel #envoy-mobile`,
			expected: []string{"#envoy", "#envoy-mobile"},
		},
		{
			name: "multiple keyword-based channels",
			content: `Slack: #project-general
CNCF Slack: #project-dev`,
			expected: []string{"#project-general", "#project-dev"},
		},
		{
			name: "filters out Slack channel IDs from archives URLs",
			content: `Join us on [Slack](https://cloud-native.slack.com/archives/C01N7PP1THC)
or [#opentelemetry](https://cloud-native.slack.com/messages/opentelemetry)
See also https://cloud-native.slack.com/archives/D03FAB6GN0K`,
			expected: []string{"#opentelemetry"},
		},
		{
			name:     "filters out generic #slack keyword",
			content:  "Join our Slack community at #slack for discussions",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSlackChannels(tt.content)
			if len(got) != len(tt.expected) {
				t.Fatalf("extractSlackChannels() returned %d channels, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
			}
			for i, ch := range got {
				if ch != tt.expected[i] {
					t.Errorf("extractSlackChannels()[%d] = %q, want %q", i, ch, tt.expected[i])
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
