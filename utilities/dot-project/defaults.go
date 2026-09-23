package projects

import "time"

// Centralized runtime config defaults for the dot-project tooling.

const (
	// DefaultHTTPTimeout is the timeout applied to all outbound HTTP
	// clients (validator, bootstrap sources, etc.).
	DefaultHTTPTimeout = 30 * time.Second

	// DefaultDCOCommitSampleSize is how many recent commits we fetch when
	// detecting whether a repo uses DCO (Signed-off-by).
	DefaultDCOCommitSampleSize = 20

	// DefaultDCOSignedRatio is the minimum ratio of Signed-off-by commits
	// to total sampled commits before we consider DCO "enabled".
	DefaultDCOSignedRatio = 0.5

	// DefaultFuzzyMatchWeight is the weight applied to partial word-match
	// scores when fuzzy-matching project names against landscape entries.
	DefaultFuzzyMatchWeight = 0.5

	// DefaultFoundationMaintainersCSVURL is the canonical source of truth for
	// CNCF project maintainers, published by the foundation. It is fetched
	// fresh on each run unless the caller overrides it with a local file path.
	DefaultFoundationMaintainersCSVURL = "https://raw.githubusercontent.com/cncf/foundation/main/project-maintainers.csv"

	// DefaultGitHubAPIURL is the base URL for the GitHub REST API.
	DefaultGitHubAPIURL = "https://api.github.com"

	// DefaultGitHubGraphQLURL is the GitHub GraphQL API endpoint.
	DefaultGitHubGraphQLURL = "https://api.github.com/graphql"

	// OrgFileName is the org.yaml index found at the root of a .project
	// repository whose GitHub organization hosts multiple CNCF projects.
	OrgFileName = "org.yaml"

	// ProjectFileName is the per-project metadata file. It lives at the
	// repository root in single-project repos, and inside each project
	// directory in multi-project repos.
	ProjectFileName = "project.yaml"

	// MaintainersFileName is the per-project maintainer roster, colocated
	// with its ProjectFileName.
	MaintainersFileName = "maintainers.yaml"
)
