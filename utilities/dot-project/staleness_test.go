package projects

import (
	"strings"
	"testing"
	"time"
)

func TestCheckStaleness(t *testing.T) {
	project := validBaseProject()
	project.ProjectLead = "jdoe"
	project.CNCFSlackChannel = "#test-project"

	// Fresh update - not stale
	recent := time.Now().Add(-24 * time.Hour * 30) // 30 days ago
	result := CheckStaleness(project, recent, 180)
	if result.IsStale {
		t.Error("expected project updated 30 days ago to not be stale at 180-day threshold")
	}

	// Old update - stale
	old := time.Now().Add(-24 * time.Hour * 200) // 200 days ago
	result = CheckStaleness(project, old, 180)
	if !result.IsStale {
		t.Error("expected project updated 200 days ago to be stale at 180-day threshold")
	}
	if result.ProjectLead != "jdoe" {
		t.Errorf("expected project_lead 'jdoe', got %q", result.ProjectLead)
	}
	if result.SlackChannel != "#test-project" {
		t.Errorf("expected slack channel '#test-project', got %q", result.SlackChannel)
	}
}

func TestFormatStalenessResults(t *testing.T) {
	results := []StalenessResult{
		{
			ProjectSlug:          "stale-project",
			LastMaintainerUpdate: time.Now().Add(-24 * time.Hour * 200),
			DaysSinceUpdate:      200,
			IsStale:              true,
			ProjectLead:          "lead",
			SlackChannel:         "#stale",
			Message:              "stale",
		},
		{
			ProjectSlug:     "fresh-project",
			DaysSinceUpdate: 10,
			IsStale:         false,
			Message:         "fresh",
		},
	}

	output := FormatStalenessResults(results)
	if !strings.Contains(output, "STALE: stale-project") {
		t.Error("expected stale project in output")
	}
	if !strings.Contains(output, "1 stale") {
		t.Error("expected '1 stale' in summary")
	}
}
