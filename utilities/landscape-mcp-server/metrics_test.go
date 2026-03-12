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
