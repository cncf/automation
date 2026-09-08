package main

import (
	"errors"
	"testing"
)

func TestCanonicalProjectName(t *testing.T) {
	tests := map[string]string{
		"Kube Bind":   "kubebind",
		"kube-bind":   "kubebind",
		" kube_bind ": "kubebind",
	}

	for input, want := range tests {
		if got := canonicalProjectName(input); got != want {
			t.Errorf("canonicalProjectName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
		ok   bool
	}{
		{name: "repository", url: "https://github.com/kbind-dev/kbind", want: "kbind-dev/kbind", ok: true},
		{name: "git suffix", url: "https://github.com/KBind-Dev/KBind.git", want: "kbind-dev/kbind", ok: true},
		{name: "organization only", url: "https://github.com/kbind-dev", ok: false},
		{name: "other host", url: "https://gitlab.com/kbind-dev/kbind", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseGitHubRepo(tt.url)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseGitHubRepo(%q) = (%q, %t), want (%q, %t)", tt.url, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestFindMissingProjects(t *testing.T) {
	landscape := &Landscape{Landscape: []LandscapeCategory{{
		Subcategories: []LandscapeSubcategory{{Items: []LandscapeProject{
			{Name: "kbind", RepoURL: "https://github.com/kbind-dev/kbind"},
			{Name: "Open Choreo", RepoURL: "https://github.com/openchoreo/openchoreo"},
		}}},
	}}}
	index := extractLandscapeProjects(landscape)

	tests := []struct {
		name        string
		issue       SandboxIssue
		resolver    repoResolver
		wantMissing bool
	}{
		{
			name:  "normalized name match",
			issue: SandboxIssue{ProjectName: "open-choreo"},
		},
		{
			name:  "direct repository match",
			issue: SandboxIssue{ProjectName: "different name", RepoURL: "https://github.com/openchoreo/openchoreo"},
		},
		{
			name:  "renamed repository match",
			issue: SandboxIssue{ProjectName: "kube-bind", RepoURL: "https://github.com/kube-bind/kube-bind"},
			resolver: func(string) (string, error) {
				return "kbind-dev/kbind", nil
			},
		},
		{
			name:  "resolver failure remains missing",
			issue: SandboxIssue{ProjectName: "missing", RepoURL: "https://github.com/example/missing"},
			resolver: func(string) (string, error) {
				return "", errors.New("not found")
			},
			wantMissing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := findMissingProjects([]SandboxIssue{tt.issue}, index, tt.resolver, false)
			if got := len(missing) == 1; got != tt.wantMissing {
				t.Fatalf("missing = %t, want %t", got, tt.wantMissing)
			}
		})
	}
}
