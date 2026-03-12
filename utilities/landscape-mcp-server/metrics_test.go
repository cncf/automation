package main

import (
	"context"
	"encoding/json"
	"strings"
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
		result, err := queryProjects(ds, ProjectQuery{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("filter by maturity graduated", func(t *testing.T) {
		result, err := queryProjects(ds, ProjectQuery{Maturity: "graduated", Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("filter by name case-insensitive substring", func(t *testing.T) {
		result, err := queryProjects(ds, ProjectQuery{Name: "kube", Limit: 100})
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
		result, err := queryProjects(ds, ProjectQuery{GraduatedFrom: "2018-01-01", GraduatedTo: "2019-01-01", Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("limit constrains results", func(t *testing.T) {
		result, err := queryProjects(ds, ProjectQuery{Limit: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("invalid date returns error", func(t *testing.T) {
		_, err := queryProjects(ds, ProjectQuery{GraduatedFrom: "bad-date", Limit: 100})
		if err == nil {
			t.Fatal("expected error for invalid date, got nil")
		}
	})

	t.Run("filter by category", func(t *testing.T) {
		result, err := queryProjects(ds, ProjectQuery{Category: "Provisioning", Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
		names := jsonProjectNames(t, result)
		if len(names) == 0 || names[0] != "Akri" {
			t.Errorf("expected Akri, got %v", names)
		}
	})

	t.Run("filter by subcategory", func(t *testing.T) {
		result, err := queryProjects(ds, ProjectQuery{Subcategory: "Monitoring", Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
		names := jsonProjectNames(t, result)
		if len(names) == 0 || names[0] != "OpenTelemetry" {
			t.Errorf("expected OpenTelemetry, got %v", names)
		}
	})

	t.Run("offset skips first results", func(t *testing.T) {
		// Get all 3 projects without offset
		all, err := queryProjects(ds, ProjectQuery{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		allNames := jsonProjectNames(t, all)
		if len(allNames) < 3 {
			t.Fatalf("expected at least 3 projects, got %d", len(allNames))
		}

		// Offset=1 should skip the first project
		result, err := queryProjects(ds, ProjectQuery{Limit: 100, Offset: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
		names := jsonProjectNames(t, result)
		// The first result after offset should be the second project from the full list
		if len(names) == 0 || names[0] != allNames[1] {
			t.Errorf("first project after offset = %v, want %q", names, allNames[1])
		}
	})

	t.Run("offset beyond results returns empty", func(t *testing.T) {
		result, err := queryProjects(ds, ProjectQuery{Limit: 100, Offset: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("offset with limit paginates correctly", func(t *testing.T) {
		// Get all projects
		all, err := queryProjects(ds, ProjectQuery{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		allNames := jsonProjectNames(t, all)

		// Page 1: offset=0, limit=1
		p1, err := queryProjects(ds, ProjectQuery{Limit: 1, Offset: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p1Names := jsonProjectNames(t, p1)

		// Page 2: offset=1, limit=1
		p2, err := queryProjects(ds, ProjectQuery{Limit: 1, Offset: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p2Names := jsonProjectNames(t, p2)

		// Page 3: offset=2, limit=1
		p3, err := queryProjects(ds, ProjectQuery{Limit: 1, Offset: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p3Names := jsonProjectNames(t, p3)

		// The three pages should cover all projects in order
		if len(p1Names) != 1 || p1Names[0] != allNames[0] {
			t.Errorf("page 1 = %v, want [%s]", p1Names, allNames[0])
		}
		if len(p2Names) != 1 || p2Names[0] != allNames[1] {
			t.Errorf("page 2 = %v, want [%s]", p2Names, allNames[1])
		}
		if len(p3Names) != 1 || p3Names[0] != allNames[2] {
			t.Errorf("page 3 = %v, want [%s]", p3Names, allNames[2])
		}
	})
}

// ---------------------------------------------------------------------------
// 3c: queryMembers tests
// ---------------------------------------------------------------------------

func TestQueryMembers(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("no filters returns all members", func(t *testing.T) {
		result, err := queryMembers(ds, MemberQuery{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("filter by tier Gold", func(t *testing.T) {
		result, err := queryMembers(ds, MemberQuery{Tier: "Gold", Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("filter by joined_from", func(t *testing.T) {
		result, err := queryMembers(ds, MemberQuery{JoinedFrom: "2026-01-01", Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("limit constrains results", func(t *testing.T) {
		result, err := queryMembers(ds, MemberQuery{Limit: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("filter by category", func(t *testing.T) {
		result, err := queryMembers(ds, MemberQuery{Category: "CNCF Members", Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("offset skips first results", func(t *testing.T) {
		all, err := queryMembers(ds, MemberQuery{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		allNames := jsonMemberNames(t, all)
		if len(allNames) < 3 {
			t.Fatalf("expected at least 3 members, got %d", len(allNames))
		}

		result, err := queryMembers(ds, MemberQuery{Limit: 100, Offset: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
		names := jsonMemberNames(t, result)
		if len(names) == 0 || names[0] != allNames[1] {
			t.Errorf("first member after offset = %v, want %q", names, allNames[1])
		}
	})

	t.Run("offset beyond results returns empty", func(t *testing.T) {
		result, err := queryMembers(ds, MemberQuery{Limit: 100, Offset: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("offset with limit paginates correctly", func(t *testing.T) {
		all, err := queryMembers(ds, MemberQuery{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		allNames := jsonMemberNames(t, all)

		p1, err := queryMembers(ds, MemberQuery{Limit: 1, Offset: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p1Names := jsonMemberNames(t, p1)

		p2, err := queryMembers(ds, MemberQuery{Limit: 1, Offset: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p2Names := jsonMemberNames(t, p2)

		p3, err := queryMembers(ds, MemberQuery{Limit: 1, Offset: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p3Names := jsonMemberNames(t, p3)

		if len(p1Names) != 1 || p1Names[0] != allNames[0] {
			t.Errorf("page 1 = %v, want [%s]", p1Names, allNames[0])
		}
		if len(p2Names) != 1 || p2Names[0] != allNames[1] {
			t.Errorf("page 2 = %v, want [%s]", p2Names, allNames[1])
		}
		if len(p3Names) != 1 || p3Names[0] != allNames[2] {
			t.Errorf("page 3 = %v, want [%s]", p3Names, allNames[2])
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

	t.Run("enriched response fields", func(t *testing.T) {
		result, err := getProjectDetails(ds, "Kubernetes")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Count    int `json:"count"`
			Projects []struct {
				Name         string            `json:"name"`
				Description  string            `json:"description"`
				HomepageURL  string            `json:"homepage_url"`
				OSS          bool              `json:"oss"`
				Repositories []Repository      `json:"repositories"`
				Links        map[string]string `json:"links"`
			} `json:"projects"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if resp.Count != 1 {
			t.Fatalf("count = %d, want 1", resp.Count)
		}
		p := resp.Projects[0]
		if p.Description == "" {
			t.Error("expected non-empty description")
		}
		if p.HomepageURL != "https://kubernetes.io/" {
			t.Errorf("homepage_url = %q, want %q", p.HomepageURL, "https://kubernetes.io/")
		}
		if !p.OSS {
			t.Error("expected oss = true")
		}
		if len(p.Repositories) != 2 {
			t.Errorf("repositories count = %d, want 2", len(p.Repositories))
		}
		if p.Links == nil {
			t.Fatal("expected non-nil links")
		}
		if _, ok := p.Links["devstats"]; !ok {
			t.Error("expected devstats link")
		}
		if _, ok := p.Links["blog"]; !ok {
			t.Error("expected blog link")
		}
		if _, ok := p.Links["slack"]; !ok {
			t.Error("expected slack link")
		}
	})
}

// ---------------------------------------------------------------------------
// 3d2: getMemberDetails tests
// ---------------------------------------------------------------------------

func TestGetMemberDetails(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("exact name match", func(t *testing.T) {
		result, err := getMemberDetails(ds, "TestGoldCorp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("case-insensitive lookup", func(t *testing.T) {
		result, err := getMemberDetails(ds, "testgoldcorp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("substring match returns multiple", func(t *testing.T) {
		// "Test" matches TestGoldCorp, TestSilverInc, TestPlatinum
		result, err := getMemberDetails(ds, "Test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("nonexistent returns error", func(t *testing.T) {
		_, err := getMemberDetails(ds, "nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent member, got nil")
		}
	})

	t.Run("enriched response fields", func(t *testing.T) {
		result, err := getMemberDetails(ds, "TestGoldCorp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Count   int `json:"count"`
			Members []struct {
				Name              string            `json:"name"`
				Category          string            `json:"category"`
				Subcategory       string            `json:"subcategory"`
				MemberSubcategory string            `json:"member_subcategory"`
				JoinedAt          string            `json:"joined_at"`
				Links             map[string]string `json:"links"`
				Organization      *orgSummary       `json:"organization"`
			} `json:"members"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if resp.Count != 1 {
			t.Fatalf("count = %d, want 1", resp.Count)
		}
		m := resp.Members[0]
		if m.Name != "TestGoldCorp" {
			t.Errorf("name = %q, want %q", m.Name, "TestGoldCorp")
		}
		if m.Category != "CNCF Members" {
			t.Errorf("category = %q, want %q", m.Category, "CNCF Members")
		}
		if m.MemberSubcategory != "Gold" {
			t.Errorf("member_subcategory = %q, want %q", m.MemberSubcategory, "Gold")
		}
		if m.JoinedAt != "2026-01-15" {
			t.Errorf("joined_at = %q, want %q", m.JoinedAt, "2026-01-15")
		}
		// Crunchbase org should be populated
		if m.Organization == nil {
			t.Fatal("expected non-nil organization")
		}
		if m.Organization.Description != "A leading cloud infrastructure company." {
			t.Errorf("org description = %q", m.Organization.Description)
		}
		if m.Organization.City != "San Francisco" {
			t.Errorf("org city = %q, want %q", m.Organization.City, "San Francisco")
		}
	})

	t.Run("member without crunchbase has nil organization", func(t *testing.T) {
		result, err := getMemberDetails(ds, "TestPlatinum")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Members []struct {
				Name         string      `json:"name"`
				Organization *orgSummary `json:"organization"`
			} `json:"members"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if len(resp.Members) != 1 {
			t.Fatalf("expected 1 member, got %d", len(resp.Members))
		}
		if resp.Members[0].Organization != nil {
			t.Error("expected nil organization for member without crunchbase_url")
		}
	})

	t.Run("does not match non-member items", func(t *testing.T) {
		// "kube" matches Kubernetes (a project), not a member
		_, err := getMemberDetails(ds, "Kubernetes")
		if err == nil {
			t.Fatal("expected error since Kubernetes is a project not a member")
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
// 5a: listCategories tests
// ---------------------------------------------------------------------------

func TestListCategories(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("returns all categories with subcategories", func(t *testing.T) {
		result, err := listCategories(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Categories []struct {
				Name          string `json:"name"`
				ItemCount     int    `json:"item_count"`
				Subcategories []struct {
					Name      string `json:"name"`
					ItemCount int    `json:"item_count"`
				} `json:"subcategories"`
			} `json:"categories"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Fixture has 4 categories: CNCF Members, Observability and Analysis, Orchestration & Management, Provisioning
		if len(resp.Categories) != 4 {
			t.Errorf("categories count = %d, want 4", len(resp.Categories))
		}

		// Find CNCF Members category
		var membersCat *struct {
			Name          string `json:"name"`
			ItemCount     int    `json:"item_count"`
			Subcategories []struct {
				Name      string `json:"name"`
				ItemCount int    `json:"item_count"`
			} `json:"subcategories"`
		}
		for i := range resp.Categories {
			if resp.Categories[i].Name == "CNCF Members" {
				membersCat = &resp.Categories[i]
				break
			}
		}
		if membersCat == nil {
			t.Fatal("CNCF Members category not found")
		}
		if membersCat.ItemCount != 3 {
			t.Errorf("CNCF Members item_count = %d, want 3", membersCat.ItemCount)
		}
		// Should have 3 subcategories: Gold, Silver, Platinum
		if len(membersCat.Subcategories) != 3 {
			t.Errorf("CNCF Members subcategories count = %d, want 3", len(membersCat.Subcategories))
		}
	})

	t.Run("categories are sorted alphabetically", func(t *testing.T) {
		result, err := listCategories(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Categories []struct {
				Name string `json:"name"`
			} `json:"categories"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		for i := 1; i < len(resp.Categories); i++ {
			if resp.Categories[i].Name < resp.Categories[i-1].Name {
				t.Errorf("categories not sorted: %q before %q", resp.Categories[i-1].Name, resp.Categories[i].Name)
			}
		}
	})

	t.Run("item counts are correct per subcategory", func(t *testing.T) {
		result, err := listCategories(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Categories []struct {
				Name          string `json:"name"`
				ItemCount     int    `json:"item_count"`
				Subcategories []struct {
					Name      string `json:"name"`
					ItemCount int    `json:"item_count"`
				} `json:"subcategories"`
			} `json:"categories"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Find Orchestration & Management → Scheduling & Orchestration → 1 (Kubernetes)
		for _, cat := range resp.Categories {
			if cat.Name == "Orchestration & Management" {
				if cat.ItemCount != 1 {
					t.Errorf("Orchestration & Management item_count = %d, want 1", cat.ItemCount)
				}
				if len(cat.Subcategories) != 1 {
					t.Errorf("Orchestration & Management subcategories = %d, want 1", len(cat.Subcategories))
				}
				if cat.Subcategories[0].Name != "Scheduling & Orchestration" {
					t.Errorf("subcategory name = %q, want %q", cat.Subcategories[0].Name, "Scheduling & Orchestration")
				}
				if cat.Subcategories[0].ItemCount != 1 {
					t.Errorf("Scheduling & Orchestration item_count = %d, want 1", cat.Subcategories[0].ItemCount)
				}
			}
		}
	})
}

// ---------------------------------------------------------------------------
// 5b: landscapeSummary tests
// ---------------------------------------------------------------------------

func TestLandscapeSummary(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("returns correct total counts", func(t *testing.T) {
		result, err := landscapeSummary(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		totalItems := jsonInt(t, result, "total_items")
		if totalItems != 6 {
			t.Errorf("total_items = %d, want 6", totalItems)
		}

		totalProjects := jsonInt(t, result, "total_projects")
		if totalProjects != 3 {
			t.Errorf("total_projects = %d, want 3", totalProjects)
		}

		totalMembers := jsonInt(t, result, "total_members")
		if totalMembers != 3 {
			t.Errorf("total_members = %d, want 3", totalMembers)
		}

		categoriesCount := jsonInt(t, result, "categories_count")
		if categoriesCount != 4 {
			t.Errorf("categories_count = %d, want 4", categoriesCount)
		}
	})

	t.Run("projects_by_maturity breakdown", func(t *testing.T) {
		result, err := landscapeSummary(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			ProjectsByMaturity map[string]int `json:"projects_by_maturity"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.ProjectsByMaturity["graduated"] != 1 {
			t.Errorf("graduated = %d, want 1", resp.ProjectsByMaturity["graduated"])
		}
		if resp.ProjectsByMaturity["incubating"] != 1 {
			t.Errorf("incubating = %d, want 1", resp.ProjectsByMaturity["incubating"])
		}
		if resp.ProjectsByMaturity["sandbox"] != 1 {
			t.Errorf("sandbox = %d, want 1", resp.ProjectsByMaturity["sandbox"])
		}
	})

	t.Run("members_by_tier breakdown", func(t *testing.T) {
		result, err := landscapeSummary(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			MembersByTier map[string]int `json:"members_by_tier"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.MembersByTier["Gold"] != 1 {
			t.Errorf("Gold = %d, want 1", resp.MembersByTier["Gold"])
		}
		if resp.MembersByTier["Silver"] != 1 {
			t.Errorf("Silver = %d, want 1", resp.MembersByTier["Silver"])
		}
		if resp.MembersByTier["Platinum"] != 1 {
			t.Errorf("Platinum = %d, want 1", resp.MembersByTier["Platinum"])
		}
	})

	t.Run("items_with_crunchbase_data count", func(t *testing.T) {
		result, err := landscapeSummary(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cbCount := jsonInt(t, result, "items_with_crunchbase_data")
		// TestGoldCorp and TestSilverInc have crunchbase data
		if cbCount != 2 {
			t.Errorf("items_with_crunchbase_data = %d, want 2", cbCount)
		}
	})
}

// ---------------------------------------------------------------------------
// 5c: searchLandscape tests
// ---------------------------------------------------------------------------

func TestSearchLandscape(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("search by name", func(t *testing.T) {
		result, err := searchLandscape(ds, "kubernetes", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Count int `json:"count"`
			Items []struct {
				Name        string   `json:"name"`
				MatchFields []string `json:"match_fields"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if resp.Count != 1 {
			t.Errorf("count = %d, want 1", resp.Count)
		}
		if resp.Items[0].Name != "Kubernetes" {
			t.Errorf("name = %q, want %q", resp.Items[0].Name, "Kubernetes")
		}
		// Should match on "name"
		found := false
		for _, f := range resp.Items[0].MatchFields {
			if f == "name" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected match_fields to contain 'name', got %v", resp.Items[0].MatchFields)
		}
	})

	t.Run("search by description", func(t *testing.T) {
		result, err := searchLandscape(ds, "containerized", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("search by category", func(t *testing.T) {
		result, err := searchLandscape(ds, "Provisioning", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Count int `json:"count"`
			Items []struct {
				Name        string   `json:"name"`
				MatchFields []string `json:"match_fields"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if resp.Count != 1 {
			t.Errorf("count = %d, want 1", resp.Count)
		}
		if resp.Items[0].Name != "Akri" {
			t.Errorf("name = %q, want %q", resp.Items[0].Name, "Akri")
		}
	})

	t.Run("search by subcategory", func(t *testing.T) {
		result, err := searchLandscape(ds, "Monitoring", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("search by homepage_url", func(t *testing.T) {
		result, err := searchLandscape(ds, "kubernetes.io", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("case-insensitive search", func(t *testing.T) {
		result, err := searchLandscape(ds, "KUBERNETES", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count := jsonInt(t, result, "count")
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("limit constrains results", func(t *testing.T) {
		// "Test" matches 3 members
		result, err := searchLandscape(ds, "Test", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count := jsonInt(t, result, "count")
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		result, err := searchLandscape(ds, "zzzznonexistent", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count := jsonInt(t, result, "count")
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})

	t.Run("empty query returns error", func(t *testing.T) {
		_, err := searchLandscape(ds, "", 20)
		if err == nil {
			t.Fatal("expected error for empty query")
		}
	})

	t.Run("result includes maturity for projects", func(t *testing.T) {
		result, err := searchLandscape(ds, "kubernetes", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Items []struct {
				Maturity string `json:"maturity"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Items))
		}
		if resp.Items[0].Maturity != "graduated" {
			t.Errorf("maturity = %q, want %q", resp.Items[0].Maturity, "graduated")
		}
	})

	t.Run("result includes member_subcategory for members", func(t *testing.T) {
		result, err := searchLandscape(ds, "TestGoldCorp", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Items []struct {
				MemberSubcategory string `json:"member_subcategory"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Items))
		}
		if resp.Items[0].MemberSubcategory != "Gold" {
			t.Errorf("member_subcategory = %q, want %q", resp.Items[0].MemberSubcategory, "Gold")
		}
	})

	t.Run("default limit is 20 and max is 100", func(t *testing.T) {
		// With limit 0, should use default 20
		result, err := searchLandscape(ds, "Test", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		// only 3 items match "Test", so all 3 returned within default limit
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("multiple match_fields", func(t *testing.T) {
		// "kubernetes.io" appears in homepage_url and name doesn't match it
		// But "kubernetes" appears in name, description etc.
		result, err := searchLandscape(ds, "kubernetes", 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Items []struct {
				Name        string   `json:"name"`
				MatchFields []string `json:"match_fields"`
			} `json:"items"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(resp.Items))
		}
		// "kubernetes" matches name, homepage_url ("kubernetes.io")
		fields := resp.Items[0].MatchFields
		if len(fields) < 2 {
			t.Errorf("expected at least 2 match_fields, got %v", fields)
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

func jsonMemberNames(t *testing.T, jsonStr string) []string {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	members, ok := m["members"].([]interface{})
	if !ok {
		t.Fatalf("members is not an array")
	}
	names := make([]string, 0, len(members))
	for _, p := range members {
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

// ---------------------------------------------------------------------------
// 6a: compareProjects tests
// ---------------------------------------------------------------------------

func TestCompareProjects(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("compare two projects", func(t *testing.T) {
		result, err := compareProjects(ds, []string{"Kubernetes", "OpenTelemetry"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}

		var resp struct {
			Count    int `json:"count"`
			Projects []struct {
				Name        string `json:"name"`
				Maturity    string `json:"maturity"`
				Category    string `json:"category"`
				Subcategory string `json:"subcategory"`
			} `json:"projects"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if resp.Projects[0].Name != "Kubernetes" {
			t.Errorf("first project = %q, want %q", resp.Projects[0].Name, "Kubernetes")
		}
		if resp.Projects[1].Name != "OpenTelemetry" {
			t.Errorf("second project = %q, want %q", resp.Projects[1].Name, "OpenTelemetry")
		}
		if resp.Projects[0].Maturity != "graduated" {
			t.Errorf("Kubernetes maturity = %q, want %q", resp.Projects[0].Maturity, "graduated")
		}
		if resp.Projects[1].Maturity != "incubating" {
			t.Errorf("OpenTelemetry maturity = %q, want %q", resp.Projects[1].Maturity, "incubating")
		}
	})

	t.Run("compare three projects", func(t *testing.T) {
		result, err := compareProjects(ds, []string{"Kubernetes", "OpenTelemetry", "Akri"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("case-insensitive lookup", func(t *testing.T) {
		result, err := compareProjects(ds, []string{"kubernetes", "opentelemetry"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := jsonInt(t, result, "count")
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("nonexistent project returns error", func(t *testing.T) {
		_, err := compareProjects(ds, []string{"Kubernetes", "NonExistent"})
		if err == nil {
			t.Fatal("expected error for nonexistent project, got nil")
		}
	})

	t.Run("fewer than 2 names returns error", func(t *testing.T) {
		_, err := compareProjects(ds, []string{"Kubernetes"})
		if err == nil {
			t.Fatal("expected error for single project, got nil")
		}
	})

	t.Run("empty name returns error", func(t *testing.T) {
		_, err := compareProjects(ds, []string{"", "Kubernetes"})
		if err == nil {
			t.Fatal("expected error for empty project name, got nil")
		}
	})

	t.Run("more than 10 names returns error", func(t *testing.T) {
		names := make([]string, 11)
		for i := range names {
			names[i] = "Kubernetes"
		}
		_, err := compareProjects(ds, names)
		if err == nil {
			t.Fatal("expected error for too many names, got nil")
		}
	})

	t.Run("includes enriched fields", func(t *testing.T) {
		result, err := compareProjects(ds, []string{"Kubernetes", "Akri"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			Projects []struct {
				Name         string            `json:"name"`
				HomepageURL  string            `json:"homepage_url"`
				OSS          bool              `json:"oss"`
				Description  string            `json:"description"`
				Repositories []Repository      `json:"repositories"`
				Links        map[string]string `json:"links"`
				Organization *orgSummary       `json:"organization"`
			} `json:"projects"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		// Kubernetes should have homepage, repos, links
		k := resp.Projects[0]
		if k.HomepageURL != "https://kubernetes.io/" {
			t.Errorf("Kubernetes homepage_url = %q", k.HomepageURL)
		}
		if !k.OSS {
			t.Error("expected Kubernetes oss = true")
		}
		if len(k.Repositories) != 2 {
			t.Errorf("Kubernetes repositories count = %d, want 2", len(k.Repositories))
		}
		if k.Links == nil {
			t.Error("expected non-nil links for Kubernetes")
		}
	})

	t.Run("does not match members", func(t *testing.T) {
		_, err := compareProjects(ds, []string{"Kubernetes", "TestGoldCorp"})
		if err == nil {
			t.Fatal("expected error since TestGoldCorp is a member, not a project")
		}
	})
}

// ---------------------------------------------------------------------------
// 6b: fundingAnalysis tests
// ---------------------------------------------------------------------------

func TestFundingAnalysis(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("all members no filters", func(t *testing.T) {
		result, err := fundingAnalysis(ds, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalMembersAnalyzed int                        `json:"total_members_analyzed"`
			TotalFundingRounds   int                        `json:"total_funding_rounds"`
			TotalFundingAmount   int64                      `json:"total_funding_amount"`
			RoundsByKind         map[string]json.RawMessage `json:"rounds_by_kind"`
			TopFunded            []struct {
				Name        string `json:"name"`
				Tier        string `json:"tier"`
				TotalAmount int64  `json:"total_amount"`
				RoundCount  int    `json:"round_count"`
			} `json:"top_funded"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// 2 members have crunchbase data (TestGoldCorp, TestSilverInc)
		if resp.TotalMembersAnalyzed != 2 {
			t.Errorf("total_members_analyzed = %d, want 2", resp.TotalMembersAnalyzed)
		}
		// 3 total rounds: series_b, series_a, seed
		if resp.TotalFundingRounds != 3 {
			t.Errorf("total_funding_rounds = %d, want 3", resp.TotalFundingRounds)
		}
		// 50M + 20M + 10M = 80M
		if resp.TotalFundingAmount != 80000000 {
			t.Errorf("total_funding_amount = %d, want 80000000", resp.TotalFundingAmount)
		}
		// 3 kinds: series_a, series_b, seed
		if len(resp.RoundsByKind) != 3 {
			t.Errorf("rounds_by_kind count = %d, want 3", len(resp.RoundsByKind))
		}
		// Top funded: TestGoldCorp (70M) first, TestSilverInc (10M) second
		if len(resp.TopFunded) != 2 {
			t.Fatalf("top_funded count = %d, want 2", len(resp.TopFunded))
		}
		if resp.TopFunded[0].Name != "TestGoldCorp" {
			t.Errorf("top_funded[0].name = %q, want %q", resp.TopFunded[0].Name, "TestGoldCorp")
		}
		if resp.TopFunded[0].TotalAmount != 70000000 {
			t.Errorf("top_funded[0].total_amount = %d, want 70000000", resp.TopFunded[0].TotalAmount)
		}
		if resp.TopFunded[0].RoundCount != 2 {
			t.Errorf("top_funded[0].round_count = %d, want 2", resp.TopFunded[0].RoundCount)
		}
		if resp.TopFunded[0].Tier != "Gold" {
			t.Errorf("top_funded[0].tier = %q, want %q", resp.TopFunded[0].Tier, "Gold")
		}
		if resp.TopFunded[1].Name != "TestSilverInc" {
			t.Errorf("top_funded[1].name = %q, want %q", resp.TopFunded[1].Name, "TestSilverInc")
		}
		if resp.TopFunded[1].TotalAmount != 10000000 {
			t.Errorf("top_funded[1].total_amount = %d, want 10000000", resp.TopFunded[1].TotalAmount)
		}
	})

	t.Run("filter by tier Gold", func(t *testing.T) {
		result, err := fundingAnalysis(ds, "Gold", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalMembersAnalyzed int   `json:"total_members_analyzed"`
			TotalFundingRounds   int   `json:"total_funding_rounds"`
			TotalFundingAmount   int64 `json:"total_funding_amount"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.TotalMembersAnalyzed != 1 {
			t.Errorf("total_members_analyzed = %d, want 1", resp.TotalMembersAnalyzed)
		}
		if resp.TotalFundingRounds != 2 {
			t.Errorf("total_funding_rounds = %d, want 2", resp.TotalFundingRounds)
		}
		// 50M + 20M = 70M
		if resp.TotalFundingAmount != 70000000 {
			t.Errorf("total_funding_amount = %d, want 70000000", resp.TotalFundingAmount)
		}
	})

	t.Run("filter by year 2026", func(t *testing.T) {
		result, err := fundingAnalysis(ds, "", 2026)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalMembersAnalyzed int   `json:"total_members_analyzed"`
			TotalFundingRounds   int   `json:"total_funding_rounds"`
			TotalFundingAmount   int64 `json:"total_funding_amount"`
			TopFunded            []struct {
				Name       string `json:"name"`
				RoundCount int    `json:"round_count"`
			} `json:"top_funded"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Both members analyzed (they have CB data), but only 2026 rounds count
		// TestGoldCorp: series_b 2026-01-10 ($50M) — matches
		// TestGoldCorp: series_a 2025-06-15 ($20M) — does NOT match
		// TestSilverInc: seed 2026-02-01 ($10M) — matches
		if resp.TotalFundingRounds != 2 {
			t.Errorf("total_funding_rounds = %d, want 2", resp.TotalFundingRounds)
		}
		// 50M + 10M = 60M
		if resp.TotalFundingAmount != 60000000 {
			t.Errorf("total_funding_amount = %d, want 60000000", resp.TotalFundingAmount)
		}
	})

	t.Run("filter by tier and year", func(t *testing.T) {
		result, err := fundingAnalysis(ds, "Gold", 2026)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalMembersAnalyzed int   `json:"total_members_analyzed"`
			TotalFundingRounds   int   `json:"total_funding_rounds"`
			TotalFundingAmount   int64 `json:"total_funding_amount"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.TotalMembersAnalyzed != 1 {
			t.Errorf("total_members_analyzed = %d, want 1", resp.TotalMembersAnalyzed)
		}
		// Only series_b from 2026 matches
		if resp.TotalFundingRounds != 1 {
			t.Errorf("total_funding_rounds = %d, want 1", resp.TotalFundingRounds)
		}
		if resp.TotalFundingAmount != 50000000 {
			t.Errorf("total_funding_amount = %d, want 50000000", resp.TotalFundingAmount)
		}
	})

	t.Run("tier with no crunchbase data", func(t *testing.T) {
		result, err := fundingAnalysis(ds, "Platinum", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalMembersAnalyzed int `json:"total_members_analyzed"`
			TotalFundingRounds   int `json:"total_funding_rounds"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.TotalMembersAnalyzed != 0 {
			t.Errorf("total_members_analyzed = %d, want 0", resp.TotalMembersAnalyzed)
		}
		if resp.TotalFundingRounds != 0 {
			t.Errorf("total_funding_rounds = %d, want 0", resp.TotalFundingRounds)
		}
	})

	t.Run("rounds_by_kind breakdown", func(t *testing.T) {
		result, err := fundingAnalysis(ds, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			RoundsByKind map[string]struct {
				Count       int   `json:"count"`
				TotalAmount int64 `json:"total_amount"`
			} `json:"rounds_by_kind"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		seriesB, ok := resp.RoundsByKind["series_b"]
		if !ok {
			t.Fatal("series_b not found in rounds_by_kind")
		}
		if seriesB.Count != 1 || seriesB.TotalAmount != 50000000 {
			t.Errorf("series_b = {count:%d, total:%d}, want {1, 50000000}", seriesB.Count, seriesB.TotalAmount)
		}

		seriesA, ok := resp.RoundsByKind["series_a"]
		if !ok {
			t.Fatal("series_a not found in rounds_by_kind")
		}
		if seriesA.Count != 1 || seriesA.TotalAmount != 20000000 {
			t.Errorf("series_a = {count:%d, total:%d}, want {1, 20000000}", seriesA.Count, seriesA.TotalAmount)
		}

		seed, ok := resp.RoundsByKind["seed"]
		if !ok {
			t.Fatal("seed not found in rounds_by_kind")
		}
		if seed.Count != 1 || seed.TotalAmount != 10000000 {
			t.Errorf("seed = {count:%d, total:%d}, want {1, 10000000}", seed.Count, seed.TotalAmount)
		}
	})
}

// ---------------------------------------------------------------------------
// 6c: geographicDistribution tests
// ---------------------------------------------------------------------------

func TestGeographicDistribution(t *testing.T) {
	ds := loadTestDataset(t)

	t.Run("default group_by country", func(t *testing.T) {
		result, err := geographicDistribution(ds, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalWithLocation    int `json:"total_with_location"`
			TotalWithoutLocation int `json:"total_without_location"`
			Distribution         []struct {
				Name       string  `json:"name"`
				Count      int     `json:"count"`
				Percentage float64 `json:"percentage"`
			} `json:"distribution"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// TestGoldCorp has country, TestSilverInc does not
		if resp.TotalWithLocation != 1 {
			t.Errorf("total_with_location = %d, want 1", resp.TotalWithLocation)
		}
		if resp.TotalWithoutLocation != 1 {
			t.Errorf("total_without_location = %d, want 1", resp.TotalWithoutLocation)
		}
		if len(resp.Distribution) != 1 {
			t.Fatalf("distribution count = %d, want 1", len(resp.Distribution))
		}
		if resp.Distribution[0].Name != "United States" {
			t.Errorf("distribution[0].name = %q, want %q", resp.Distribution[0].Name, "United States")
		}
		if resp.Distribution[0].Count != 1 {
			t.Errorf("distribution[0].count = %d, want 1", resp.Distribution[0].Count)
		}
		if resp.Distribution[0].Percentage != 100.0 {
			t.Errorf("distribution[0].percentage = %f, want 100.0", resp.Distribution[0].Percentage)
		}
	})

	t.Run("group_by region", func(t *testing.T) {
		result, err := geographicDistribution(ds, "region")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalWithLocation int `json:"total_with_location"`
			Distribution      []struct {
				Name string `json:"name"`
			} `json:"distribution"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.TotalWithLocation != 1 {
			t.Errorf("total_with_location = %d, want 1", resp.TotalWithLocation)
		}
		if len(resp.Distribution) != 1 {
			t.Fatalf("distribution count = %d, want 1", len(resp.Distribution))
		}
		if resp.Distribution[0].Name != "North America" {
			t.Errorf("distribution[0].name = %q, want %q", resp.Distribution[0].Name, "North America")
		}
	})

	t.Run("group_by city", func(t *testing.T) {
		result, err := geographicDistribution(ds, "city")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalWithLocation int `json:"total_with_location"`
			Distribution      []struct {
				Name string `json:"name"`
			} `json:"distribution"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.TotalWithLocation != 1 {
			t.Errorf("total_with_location = %d, want 1", resp.TotalWithLocation)
		}
		if len(resp.Distribution) != 1 {
			t.Fatalf("distribution count = %d, want 1", len(resp.Distribution))
		}
		if resp.Distribution[0].Name != "San Francisco" {
			t.Errorf("distribution[0].name = %q, want %q", resp.Distribution[0].Name, "San Francisco")
		}
	})

	t.Run("invalid group_by returns error", func(t *testing.T) {
		_, err := geographicDistribution(ds, "invalid")
		if err == nil {
			t.Fatal("expected error for invalid group_by, got nil")
		}
	})

	t.Run("explicit country group_by", func(t *testing.T) {
		result, err := geographicDistribution(ds, "country")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resp struct {
			TotalWithLocation int `json:"total_with_location"`
		}
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if resp.TotalWithLocation != 1 {
			t.Errorf("total_with_location = %d, want 1", resp.TotalWithLocation)
		}
	})
}

// ---------------------------------------------------------------------------
// MCP Resources handler tests
// ---------------------------------------------------------------------------

func TestHandleResourcesList(t *testing.T) {
	cfg := LandscapeConfig{Name: "TestLandscape", Description: "Test"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "resources/list",
		ID:      json.RawMessage(`1`),
	}

	resp := handleResourcesList(req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Resources []struct {
			URI         string `json:"uri"`
			Name        string `json:"name"`
			Description string `json:"description"`
			MimeType    string `json:"mimeType"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(result.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(result.Resources))
	}

	// Verify categories resource
	cat := result.Resources[0]
	if cat.URI != "landscape://categories" {
		t.Errorf("resource[0].uri = %q, want %q", cat.URI, "landscape://categories")
	}
	if cat.Name != "TestLandscape Categories" {
		t.Errorf("resource[0].name = %q, want %q", cat.Name, "TestLandscape Categories")
	}
	if cat.MimeType != "application/json" {
		t.Errorf("resource[0].mimeType = %q, want %q", cat.MimeType, "application/json")
	}

	// Verify summary resource
	sum := result.Resources[1]
	if sum.URI != "landscape://summary" {
		t.Errorf("resource[1].uri = %q, want %q", sum.URI, "landscape://summary")
	}
	if sum.Name != "TestLandscape Summary" {
		t.Errorf("resource[1].name = %q, want %q", sum.Name, "TestLandscape Summary")
	}
	if sum.MimeType != "application/json" {
		t.Errorf("resource[1].mimeType = %q, want %q", sum.MimeType, "application/json")
	}
}

func TestHandleResourcesReadCategories(t *testing.T) {
	ds := loadTestDataset(t)
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)
	state.setDataset(ds, nil)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "resources/read",
		Params:  json.RawMessage(`{"uri":"landscape://categories"}`),
		ID:      json.RawMessage(`2`),
	}

	resp := handleResourcesRead(context.Background(), req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Contents []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Contents))
	}

	content := result.Contents[0]
	if content.URI != "landscape://categories" {
		t.Errorf("content.uri = %q, want %q", content.URI, "landscape://categories")
	}
	if content.MimeType != "application/json" {
		t.Errorf("content.mimeType = %q, want %q", content.MimeType, "application/json")
	}

	// Verify the text is valid JSON containing categories
	var catData struct {
		Categories []struct {
			Name string `json:"name"`
		} `json:"categories"`
	}
	if err := json.Unmarshal([]byte(content.Text), &catData); err != nil {
		t.Fatalf("content.text is not valid JSON: %v", err)
	}
	if len(catData.Categories) == 0 {
		t.Error("expected at least one category")
	}
}

func TestHandleResourcesReadSummary(t *testing.T) {
	ds := loadTestDataset(t)
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)
	state.setDataset(ds, nil)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "resources/read",
		Params:  json.RawMessage(`{"uri":"landscape://summary"}`),
		ID:      json.RawMessage(`3`),
	}

	resp := handleResourcesRead(context.Background(), req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Contents []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Contents))
	}

	content := result.Contents[0]
	if content.URI != "landscape://summary" {
		t.Errorf("content.uri = %q, want %q", content.URI, "landscape://summary")
	}
	if content.MimeType != "application/json" {
		t.Errorf("content.mimeType = %q, want %q", content.MimeType, "application/json")
	}

	// Verify the text is valid JSON containing summary data
	var summaryData struct {
		TotalItems    int `json:"total_items"`
		TotalProjects int `json:"total_projects"`
	}
	if err := json.Unmarshal([]byte(content.Text), &summaryData); err != nil {
		t.Fatalf("content.text is not valid JSON: %v", err)
	}
	if summaryData.TotalItems == 0 {
		t.Error("expected total_items > 0")
	}
}

func TestHandleResourcesReadUnknownURI(t *testing.T) {
	ds := loadTestDataset(t)
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)
	state.setDataset(ds, nil)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "resources/read",
		Params:  json.RawMessage(`{"uri":"landscape://unknown"}`),
		ID:      json.RawMessage(`4`),
	}

	resp := handleResourcesRead(context.Background(), req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown URI, got success")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want %d", resp.Error.Code, -32602)
	}
}

func TestHandleResourcesReadInvalidParams(t *testing.T) {
	ds := loadTestDataset(t)
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)
	state.setDataset(ds, nil)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "resources/read",
		Params:  json.RawMessage(`{invalid`),
		ID:      json.RawMessage(`5`),
	}

	resp := handleResourcesRead(context.Background(), req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error for invalid params, got success")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want %d", resp.Error.Code, -32602)
	}
}

func TestInitializeIncludesResourcesCapability(t *testing.T) {
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      json.RawMessage(`1`),
	}

	resp := handleRequest(context.Background(), req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Capabilities struct {
			Tools     map[string]interface{} `json:"tools"`
			Resources map[string]interface{} `json:"resources"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability to be present")
	}
	if result.Capabilities.Resources == nil {
		t.Error("expected resources capability to be present")
	}
}

func TestResourcesDispatchViaHandleRequest(t *testing.T) {
	ds := loadTestDataset(t)
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)
	state.setDataset(ds, nil)

	// Test resources/list dispatch
	listReq := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "resources/list",
		ID:      json.RawMessage(`10`),
	}
	listResp := handleRequest(context.Background(), listReq, state)
	if listResp == nil {
		t.Fatal("resources/list: expected response, got nil")
	}
	if listResp.Error != nil {
		t.Fatalf("resources/list: unexpected error: %s", listResp.Error.Message)
	}

	// Test resources/read dispatch
	readReq := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "resources/read",
		Params:  json.RawMessage(`{"uri":"landscape://summary"}`),
		ID:      json.RawMessage(`11`),
	}
	readResp := handleRequest(context.Background(), readReq, state)
	if readResp == nil {
		t.Fatal("resources/read: expected response, got nil")
	}
	if readResp.Error != nil {
		t.Fatalf("resources/read: unexpected error: %s", readResp.Error.Message)
	}
}

// ---------------------------------------------------------------------------
// Prompts tests
// ---------------------------------------------------------------------------

func TestHandlePromptsList(t *testing.T) {
	cfg := LandscapeConfig{Name: "TestLandscape", Description: "Test"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "prompts/list",
		ID:      json.RawMessage(`1`),
	}

	resp := handlePromptsList(req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Prompts []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Arguments   []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Required    bool   `json:"required"`
			} `json:"arguments"`
		} `json:"prompts"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(result.Prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(result.Prompts))
	}

	// Verify analyze_landscape prompt
	analyze := result.Prompts[0]
	if analyze.Name != "analyze_landscape" {
		t.Errorf("prompts[0].name = %q, want %q", analyze.Name, "analyze_landscape")
	}
	if analyze.Description != "Analyze the current state of the TestLandscape landscape" {
		t.Errorf("prompts[0].description = %q, want %q", analyze.Description, "Analyze the current state of the TestLandscape landscape")
	}
	if len(analyze.Arguments) != 0 {
		t.Errorf("prompts[0].arguments length = %d, want 0", len(analyze.Arguments))
	}

	// Verify compare_projects prompt
	compare := result.Prompts[1]
	if compare.Name != "compare_projects" {
		t.Errorf("prompts[1].name = %q, want %q", compare.Name, "compare_projects")
	}
	if compare.Description != "Compare specific projects in the TestLandscape landscape" {
		t.Errorf("prompts[1].description = %q, want %q", compare.Description, "Compare specific projects in the TestLandscape landscape")
	}
	if len(compare.Arguments) != 1 {
		t.Fatalf("prompts[1].arguments length = %d, want 1", len(compare.Arguments))
	}
	if compare.Arguments[0].Name != "project_names" {
		t.Errorf("prompts[1].arguments[0].name = %q, want %q", compare.Arguments[0].Name, "project_names")
	}
	if !compare.Arguments[0].Required {
		t.Error("prompts[1].arguments[0].required should be true")
	}
}

func TestHandlePromptsGetAnalyzeLandscape(t *testing.T) {
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "prompts/get",
		Params:  json.RawMessage(`{"name":"analyze_landscape"}`),
		ID:      json.RawMessage(`1`),
	}

	resp := handlePromptsGet(req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Description string `json:"description"`
		Messages    []struct {
			Role    string `json:"role"`
			Content struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result.Description != "Analyze the current state of the CNCF landscape" {
		t.Errorf("description = %q, want %q", result.Description, "Analyze the current state of the CNCF landscape")
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	msg := result.Messages[0]
	if msg.Role != "user" {
		t.Errorf("message.role = %q, want %q", msg.Role, "user")
	}
	if msg.Content.Type != "text" {
		t.Errorf("message.content.type = %q, want %q", msg.Content.Type, "text")
	}
	if !strings.Contains(msg.Content.Text, "CNCF landscape") {
		t.Error("message text should contain 'CNCF landscape'")
	}
	if !strings.Contains(msg.Content.Text, "landscape_summary") {
		t.Error("message text should reference landscape_summary tool")
	}
}

func TestHandlePromptsGetCompareProjects(t *testing.T) {
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "prompts/get",
		Params:  json.RawMessage(`{"name":"compare_projects","arguments":{"project_names":"Kubernetes,Envoy"}}`),
		ID:      json.RawMessage(`2`),
	}

	resp := handlePromptsGet(req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Description string `json:"description"`
		Messages    []struct {
			Role    string `json:"role"`
			Content struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result.Description != "Compare specific projects in the CNCF landscape" {
		t.Errorf("description = %q, want %q", result.Description, "Compare specific projects in the CNCF landscape")
	}
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}
	msg := result.Messages[0]
	if msg.Role != "user" {
		t.Errorf("message.role = %q, want %q", msg.Role, "user")
	}
	if msg.Content.Type != "text" {
		t.Errorf("message.content.type = %q, want %q", msg.Content.Type, "text")
	}
	if !strings.Contains(msg.Content.Text, "Kubernetes,Envoy") {
		t.Error("message text should contain project names 'Kubernetes,Envoy'")
	}
	if !strings.Contains(msg.Content.Text, "compare_projects") {
		t.Error("message text should reference compare_projects tool")
	}
}

func TestHandlePromptsGetUnknown(t *testing.T) {
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "prompts/get",
		Params:  json.RawMessage(`{"name":"nonexistent_prompt"}`),
		ID:      json.RawMessage(`3`),
	}

	resp := handlePromptsGet(req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown prompt, got success")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want %d", resp.Error.Code, -32602)
	}
	if !strings.Contains(resp.Error.Message, "Unknown prompt") {
		t.Errorf("error message = %q, want it to contain 'Unknown prompt'", resp.Error.Message)
	}
}

func TestHandlePromptsGetMissingArgument(t *testing.T) {
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "prompts/get",
		Params:  json.RawMessage(`{"name":"compare_projects"}`),
		ID:      json.RawMessage(`4`),
	}

	resp := handlePromptsGet(req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error for missing project_names argument, got success")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want %d", resp.Error.Code, -32602)
	}
	if !strings.Contains(resp.Error.Message, "project_names") {
		t.Errorf("error message = %q, want it to contain 'project_names'", resp.Error.Message)
	}
}

func TestInitializeIncludesPromptsCapability(t *testing.T) {
	cfg := LandscapeConfig{Name: "CNCF", Description: "Cloud Native Computing Foundation"}
	state := newServerState(cfg)

	req := &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      json.RawMessage(`1`),
	}

	resp := handleRequest(context.Background(), req, state)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	var result struct {
		Capabilities struct {
			Tools     map[string]interface{} `json:"tools"`
			Resources map[string]interface{} `json:"resources"`
			Prompts   map[string]interface{} `json:"prompts"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability to be present")
	}
	if result.Capabilities.Resources == nil {
		t.Error("expected resources capability to be present")
	}
	if result.Capabilities.Prompts == nil {
		t.Error("expected prompts capability to be present")
	}
}
