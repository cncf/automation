package projects

import (
	"fmt"
	"strings"
	"time"
)

// StalenessResult contains the staleness check result for a project
type StalenessResult struct {
	ProjectSlug          string    `json:"project_slug"`
	LastMaintainerUpdate time.Time `json:"last_maintainer_update"`
	DaysSinceUpdate      int       `json:"days_since_update"`
	IsStale              bool      `json:"is_stale"`
	ProjectLead          string    `json:"project_lead,omitempty"`
	SlackChannel         string    `json:"slack_channel,omitempty"`
	Message              string    `json:"message"`
}

// CheckStaleness checks if a project's maintainer data is stale
// A project is considered stale if its maintainers haven't been updated in the given threshold
func CheckStaleness(project Project, lastUpdate time.Time, thresholdDays int) StalenessResult {
	daysSince := int(time.Since(lastUpdate).Hours() / 24)
	isStale := daysSince > thresholdDays

	result := StalenessResult{
		ProjectSlug:          project.Slug,
		LastMaintainerUpdate: lastUpdate,
		DaysSinceUpdate:      daysSince,
		IsStale:              isStale,
		ProjectLead:          project.ProjectLead,
		SlackChannel:         project.CNCFSlackChannel,
	}

	if isStale {
		result.Message = fmt.Sprintf("Project %s maintainer list has not been updated in %d days (threshold: %d days)",
			project.Slug, daysSince, thresholdDays)
	} else {
		result.Message = fmt.Sprintf("Project %s maintainer list is up to date (last updated %d days ago)",
			project.Slug, daysSince)
	}

	return result
}

// FormatStalenessResults formats staleness results as human-readable text
func FormatStalenessResults(results []StalenessResult) string {
	var b strings.Builder
	b.WriteString("Maintainer Staleness Report\n")
	b.WriteString("==========================\n\n")

	staleCount := 0
	for _, r := range results {
		if r.IsStale {
			staleCount++
			b.WriteString(fmt.Sprintf("STALE: %s\n", r.ProjectSlug))
			b.WriteString(fmt.Sprintf("  Last updated: %s (%d days ago)\n", r.LastMaintainerUpdate.Format("2006-01-02"), r.DaysSinceUpdate))
			if r.ProjectLead != "" {
				b.WriteString(fmt.Sprintf("  Contact: @%s\n", r.ProjectLead))
			}
			if r.SlackChannel != "" {
				b.WriteString(fmt.Sprintf("  Slack: %s\n", r.SlackChannel))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(fmt.Sprintf("Summary: %d projects checked, %d stale\n", len(results), staleCount))
	return b.String()
}
