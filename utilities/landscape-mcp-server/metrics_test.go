package main

import (
	"encoding/json"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// 3a: isMemberTier tests
// ---------------------------------------------------------------------------

func TestIsMemberTier(t *testing.T) {
	tests := []struct {
		name string
		item LandscapeItem
		tier string
		want bool
	}{
		{
			name: "Gold member via MemberSubcategory",
			item: LandscapeItem{
				Category:          "CNCF Members",
				MemberSubcategory: "Gold",
			},
			tier: "Gold",
			want: true,
		},
		{
			name: "Silver member via Subcategory fallback",
			item: LandscapeItem{
				Category:    "CNCF Members",
				Subcategory: "Silver",
			},
			tier: "Silver",
			want: true,
		},
		{
			name: "non-member category returns false",
			item: LandscapeItem{
				Category:    "Provisioning",
				Subcategory: "Gold",
			},
			tier: "Gold",
			want: false,
		},
		{
			name: "case-insensitive MemberSubcategory",
			item: LandscapeItem{
				Category:          "CNCF Members",
				MemberSubcategory: "gold",
			},
			tier: "Gold",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMemberTier(tt.item, tt.tier)
			if got != tt.want {
				t.Errorf("isMemberTier() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3b: queryProjects tests
// ---------------------------------------------------------------------------

func TestQueryProjects(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("no filters returns all projects", func(t *testing.T) {
		result, err := queryProjects(ds, "", "", "", "", "", "", "", "", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("filter by maturity graduated", func(t *testing.T) {
		result, err := queryProjects(ds, "graduated", "", "", "", "", "", "", "", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("filter by name case-insensitive substring", func(t *testing.T) {
		result, err := queryProjects(ds, "", "kube", "", "", "", "", "", "", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
		names := jsonProjectNames(t, result)
		if len(names) == 0 || names[0] != "Kubernetes" {
			t.Errorf("expected Kubernetes, got %v", names)
		}
	})

	t.Run("filter by graduated date range", func(t *testing.T) {
		result, err := queryProjects(ds, "", "", "2018-01-01", "2019-01-01", "", "", "", "", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("limit constrains results", func(t *testing.T) {
		result, err := queryProjects(ds, "", "", "", "", "", "", "", "", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("invalid date returns error", func(t *testing.T) {
		_, err := queryProjects(ds, "", "", "bad-date", "", "", "", "", "", 100)
		if err == nil {
			t.Fatal("expected error for invalid date, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// 3c: queryMembers tests
// ---------------------------------------------------------------------------

func TestQueryMembers(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("no filters returns all members", func(t *testing.T) {
		result, err := queryMembers(ds, "", "", "", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("filter by tier Gold", func(t *testing.T) {
		result, err := queryMembers(ds, "Gold", "", "", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("filter by joined_from", func(t *testing.T) {
		result, err := queryMembers(ds, "", "2026-01-01", "", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("limit constrains results", func(t *testing.T) {
		result, err := queryMembers(ds, "", "", "", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})
}

// ---------------------------------------------------------------------------
// 3d: getProjectDetails tests
// ---------------------------------------------------------------------------

func TestGetProjectDetails(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("exact name match", func(t *testing.T) {
		result, err := getProjectDetails(ds, "Kubernetes")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("nonexistent returns error", func(t *testing.T) {
		_, err := getProjectDetails(ds, "nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent project, got nil")
		}
	})

	t.Run("case-insensitive lookup", func(t *testing.T) {
		result, err := getProjectDetails(ds, "kubernetes")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})
}

// ---------------------------------------------------------------------------
// 3e: executeMetric tests
// ---------------------------------------------------------------------------

func TestExecuteMetric(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("incubating_project_count", func(t *testing.T) {
		result, err := executeMetric("incubating_project_count", ds, time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		value := jsonInt(t, result, "value")
		// Fixture has 1 incubating project: OpenTelemetry
		if value != 1 {
			t.Errorf("value = %d, want 1", value)
		}
	})

	t.Run("sandbox_projects_joined_this_year", func(t *testing.T) {
		// Akri accepted_at = 2021-09-14
		now := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
		result, err := executeMetric("sandbox_projects_joined_this_year", ds, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		value := jsonInt(t, result, "value")
		if value != 1 {
			t.Errorf("value = %d, want 1", value)
		}
	})

	t.Run("projects_graduated_last_year", func(t *testing.T) {
		// Kubernetes graduated_at = 2018-03-06, so now must be in 2019
		now := time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)
		result, err := executeMetric("projects_graduated_last_year", ds, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		value := jsonInt(t, result, "value")
		if value != 1 {
			t.Errorf("value = %d, want 1", value)
		}
	})

	t.Run("gold_members_joined_this_year", func(t *testing.T) {
		// TestGoldCorp joined_at = 2026-01-15
		now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		result, err := executeMetric("gold_members_joined_this_year", ds, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		value := jsonInt(t, result, "value")
		if value != 1 {
			t.Errorf("value = %d, want 1", value)
		}
	})

	t.Run("silver_members_joined_this_year", func(t *testing.T) {
		// TestSilverInc joined_at = 2025-06-01
		now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		result, err := executeMetric("silver_members_joined_this_year", ds, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		value := jsonInt(t, result, "value")
		if value != 1 {
			t.Errorf("value = %d, want 1", value)
		}
	})

	t.Run("silver_members_raised_last_month", func(t *testing.T) {
		// TestSilverInc funding: seed 2026-02-01
		// Set now to March 2026 so last month = Feb 2026
		now := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		result, err := executeMetric("silver_members_raised_last_month", ds, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		value := jsonInt(t, result, "value")
		if value != 1 {
			t.Errorf("value = %d, want 1", value)
		}
	})

	t.Run("gold_members_raised_this_year", func(t *testing.T) {
		// TestGoldCorp funding: series_b 2026-01-10
		now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		result, err := executeMetric("gold_members_raised_this_year", ds, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		value := jsonInt(t, result, "value")
		if value != 1 {
			t.Errorf("value = %d, want 1", value)
		}
	})
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

// jsonInt parses a JSON string and returns the integer value at the given key.
func jsonInt(t *testing.T, jsonStr, key string) int {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	val, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in JSON", key)
	}
	// json.Unmarshal decodes numbers as float64
	num, ok := val.(float64)
	if !ok {
		t.Fatalf("key %q is not a number: %T", key, val)
	}
	return int(num)
}

// jsonProjectNames extracts project names from queryProjects JSON output.
func jsonProjectNames(t *testing.T, jsonStr string) []string {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	projects, ok := m["projects"].([]interface{})
	if !ok {
		t.Fatalf("projects is not an array")
	}
	names := make([]string, 0, len(projects))
	for _, p := range projects {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := pm["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}
