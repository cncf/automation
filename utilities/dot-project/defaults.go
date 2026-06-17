package projects

import "time"

// Centralized runtime config defaults for the dot-project tooling.

const (
	// DefaultHTTPTimeout is the timeout applied to all outbound HTTP
	// clients (validator, bootstrap sources, etc.).
	DefaultHTTPTimeout = 30 * time.Second

	// DefaultStalenessThresholdDays is the number of days after which a
	// project's maintainer data is considered stale.
	DefaultStalenessThresholdDays = 180

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
)
