package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func featureDataset() *Dataset {
	return &Dataset{
		Items: []LandscapeItem{
			makeItem("charlie", "graduated", func(i *LandscapeItem) {
				i.AcceptedAt = timePtr("2020-01-01")
				i.OSS = true
				i.ID = "charlie-id"
				i.LogoURL = "https://logo.example/charlie.svg"
				i.HomepageURL = "https://charlie.example.com/"
				i.Repositories = []Repository{{URL: "https://github.com/org/Charlie.git", Primary: true}}
			}),
			makeItem("Alpha", "sandbox", func(i *LandscapeItem) {
				i.AcceptedAt = timePtr("2022-05-05")
				i.EndUser = true
				i.ParentProject = "Charlie"
				i.Repositories = []Repository{{URL: "https://github.com/org/alpha/"}}
			}),
			makeItem("bravo", "incubating", func(i *LandscapeItem) {
				i.Specification = true
				i.OSS = true
			}),
			makeItem("Delta", "sandbox", func(i *LandscapeItem) {
				i.AcceptedAt = timePtr("2021-03-03")
				i.ParentProject = "other"
			}),
			{Name: "zeta member", Category: "CNCF Members", Subcategory: "Gold", JoinedAt: timePtr("2024-01-01")},
			{Name: "Eta member", Category: "CNCF Members", Subcategory: "Silver"},
			{Name: "alpha member", Category: "CNCF Members", Subcategory: "Gold", JoinedAt: timePtr("2023-01-01")},
		},
		CrunchbaseOrgs: map[string]CrunchbaseOrganization{},
	}
}

func projectNames(t *testing.T, out string) []string {
	t.Helper()
	var resp struct {
		Projects []struct {
			Name string `json:"name"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	names := []string{}
	for _, p := range resp.Projects {
		names = append(names, p.Name)
	}
	return names
}

func memberNames(t *testing.T, out string) []string {
	t.Helper()
	var resp struct {
		Members []struct {
			Name string `json:"name"`
		} `json:"members"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	names := []string{}
	for _, m := range resp.Members {
		names = append(names, m.Name)
	}
	return names
}

func boolPtr(b bool) *bool { return &b }

func TestConfigDefaultsAndOverride(t *testing.T) {
	cfg := defaultConfig()
	if cfg.MemberCategoryMatch != "member" {
		t.Errorf("MemberCategoryMatch = %q, want member", cfg.MemberCategoryMatch)
	}
	if !reflect.DeepEqual(cfg.Tiers, []string{"Gold", "Silver"}) {
		t.Errorf("Tiers = %v, want [Gold Silver]", cfg.Tiers)
	}

	custom := LandscapeConfig{Tiers: []string{"Premier", "General"}}
	high, second := custom.metricTiers()
	if high != "Premier" || second != "General" {
		t.Errorf("metricTiers = %q,%q", high, second)
	}
	high, second = LandscapeConfig{}.metricTiers()
	if high != "Gold" || second != "Silver" {
		t.Errorf("zero-config metricTiers = %q,%q, want defaults", high, second)
	}
}

func TestIsMemberCustomMatch(t *testing.T) {
	t.Cleanup(func() { setMemberCategoryMatch("") })
	item := LandscapeItem{Category: "Supporters Circle"}
	if isMember(item) {
		t.Fatal("default match must not treat 'Supporters Circle' as member")
	}
	setMemberCategoryMatch("  Supporters ")
	if !isMember(item) {
		t.Error("custom match 'supporters' should match category")
	}
	if isMember(LandscapeItem{Category: "CNCF Members"}) {
		t.Error("custom match must replace the default")
	}
	setMemberCategoryMatch("")
	if !isMember(LandscapeItem{Category: "CNCF Members"}) {
		t.Error("empty match restores default")
	}
}

func TestMetricDescriptionsUseConfig(t *testing.T) {
	ds := featureDataset()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	cfg := LandscapeConfig{Name: "Acme", Tiers: []string{"Premier", "General"}}

	cases := map[string]string{
		"incubating_project_count":          "Total Acme projects currently in incubating maturity",
		"sandbox_projects_joined_this_year": "Sandbox projects accepted into Acme this year",
		"gold_members_joined_this_year":     "Premier members that joined Acme this year",
		"silver_members_joined_this_year":   "General members that joined Acme this year",
		"silver_members_raised_last_month":  "General members with funding rounds last month",
		"gold_members_raised_this_year":     "Premier members with funding rounds announced this year",
	}
	for metric, want := range cases {
		out, err := executeMetric(metric, ds, now, cfg)
		if err != nil {
			t.Fatalf("%s: %v", metric, err)
		}
		var got struct {
			Description string `json:"description"`
		}
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		if got.Description != want {
			t.Errorf("%s description = %q, want %q", metric, got.Description, want)
		}
	}

	out, err := executeMetric("gold_members_joined_this_year", &Dataset{Items: []LandscapeItem{
		{Name: "P", Category: "Members", Subcategory: "Premier", JoinedAt: timePtr("2026-02-02")},
	}}, now, cfg)
	if err != nil || !strings.Contains(out, `"value": 1`) {
		t.Errorf("custom tier should back gold_ metric: %v %s", err, out)
	}
}

func TestAcquisitionsInOrganization(t *testing.T) {
	ds := loadTestDataset(t)

	out, err := getMemberDetails(ds, "TestGoldCorp")
	if err != nil {
		t.Fatal(err)
	}
	var resp struct {
		Members []struct {
			Organization struct {
				Acquisitions []struct {
					Name string `json:"acquiree_name"`
					On   string `json:"announced_on"`
				} `json:"acquisitions"`
			} `json:"organization"`
		} `json:"members"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatal(err)
	}
	acqs := resp.Members[0].Organization.Acquisitions
	if len(acqs) != 1 || acqs[0].Name != "SmallStartup" || acqs[0].On != "2025-03-20" {
		t.Errorf("acquisitions = %+v", acqs)
	}

	out, err = getMemberDetails(ds, "TestSilverInc")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "acquisitions") {
		t.Errorf("acquisitions must be omitted when empty: %s", out)
	}
}

func TestProjectDetailsNewFields(t *testing.T) {
	ds := featureDataset()

	out, err := getProjectDetails(ds, "Alpha")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"end_user": true`, `"specification": false`, `"parent_project": "Charlie"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
	if strings.Contains(out, `"logo_url"`) || strings.Contains(out, `"id"`) {
		t.Errorf("empty id/logo_url must be omitted: %s", out)
	}

	out, err = getProjectDetails(ds, "charlie")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"id": "charlie-id"`, `"logo_url": "https://logo.example/charlie.svg"`, `"end_user": false`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
	if strings.Contains(out, "parent_project") {
		t.Errorf("empty parent_project must be omitted: %s", out)
	}
}

func TestQueryProjectsFlagFilters(t *testing.T) {
	ds := featureDataset()
	tests := []struct {
		name string
		q    ProjectQuery
		want []string
	}{
		{"oss true", ProjectQuery{OSS: boolPtr(true)}, []string{"bravo", "charlie"}},
		{"oss false", ProjectQuery{OSS: boolPtr(false)}, []string{"Alpha", "Delta"}},
		{"end_user true", ProjectQuery{EndUser: boolPtr(true)}, []string{"Alpha"}},
		{"specification true", ProjectQuery{Specification: boolPtr(true)}, []string{"bravo"}},
		{"parent exact case-insensitive", ProjectQuery{ParentProject: "CHARLIE"}, []string{"Alpha"}},
		{"parent is not substring", ProjectQuery{ParentProject: "char"}, []string{}},
		{"absent filters keep all", ProjectQuery{}, []string{"Alpha", "bravo", "charlie", "Delta"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.Limit = 100
			out, err := queryProjects(ds, tt.q)
			if err != nil {
				t.Fatal(err)
			}
			if got := projectNames(t, out); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("names = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryProjectsSorting(t *testing.T) {
	ds := featureDataset()
	tests := []struct {
		name string
		q    ProjectQuery
		want []string
	}{
		{"default name asc case-insensitive", ProjectQuery{}, []string{"Alpha", "bravo", "charlie", "Delta"}},
		{"name desc", ProjectQuery{SortOrder: "desc"}, []string{"Delta", "charlie", "bravo", "Alpha"}},
		{"accepted_at asc nil last", ProjectQuery{SortBy: "accepted_at"}, []string{"charlie", "Delta", "Alpha", "bravo"}},
		{"accepted_at desc nil still last", ProjectQuery{SortBy: "accepted_at", SortOrder: "desc"}, []string{"Alpha", "Delta", "charlie", "bravo"}},
		{"all-nil dates tie-break by name", ProjectQuery{SortBy: "graduated_at", SortOrder: "desc"}, []string{"Alpha", "bravo", "charlie", "Delta"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.Limit = 100
			out, err := queryProjects(ds, tt.q)
			if err != nil {
				t.Fatal(err)
			}
			if got := projectNames(t, out); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("names = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("sorting happens before pagination", func(t *testing.T) {
		out, err := queryProjects(ds, ProjectQuery{SortBy: "accepted_at", SortOrder: "desc", Limit: 2, Offset: 1})
		if err != nil {
			t.Fatal(err)
		}
		if got := projectNames(t, out); !reflect.DeepEqual(got, []string{"Delta", "charlie"}) {
			t.Errorf("page = %v", got)
		}
		if !strings.Contains(out, `"total": 4`) {
			t.Errorf("total must ignore pagination: %s", out)
		}
	})
}

func TestQueryMembersSorting(t *testing.T) {
	ds := featureDataset()
	tests := []struct {
		name string
		q    MemberQuery
		want []string
	}{
		{"default name asc", MemberQuery{}, []string{"alpha member", "Eta member", "zeta member"}},
		{"name desc", MemberQuery{SortOrder: "desc"}, []string{"zeta member", "Eta member", "alpha member"}},
		{"joined_at asc nil last", MemberQuery{SortBy: "joined_at"}, []string{"alpha member", "zeta member", "Eta member"}},
		{"joined_at desc nil last", MemberQuery{SortBy: "joined_at", SortOrder: "desc"}, []string{"zeta member", "alpha member", "Eta member"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.q.Limit = 100
			out, err := queryMembers(ds, tt.q)
			if err != nil {
				t.Fatal(err)
			}
			if got := memberNames(t, out); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("names = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidSortIsErrorResult(t *testing.T) {
	state := newTestState(t, featureDataset())
	tests := []struct {
		tool, args, want string
	}{
		{"query_projects", `{"sort_by":"bogus"}`, "name, accepted_at, incubating_at, graduated_at"},
		{"query_projects", `{"sort_order":"up"}`, "asc, desc"},
		{"query_members", `{"sort_by":"graduated_at"}`, "name, joined_at"},
		{"query_members", `{"sort_order":"sideways"}`, "asc, desc"},
	}
	for _, tt := range tests {
		t.Run(tt.tool+tt.args, func(t *testing.T) {
			res := decodeToolResult(t, callTool(t, state, tt.tool, tt.args))
			if !res.IsError || !strings.Contains(res.Content[0].Text, tt.want) {
				t.Errorf("isError=%v text=%q, want error listing %q", res.IsError, res.Content[0].Text, tt.want)
			}
		})
	}
}

func TestQueryProjectsToolFlagArguments(t *testing.T) {
	state := newTestState(t, featureDataset())
	res := decodeToolResult(t, callTool(t, state, "query_projects", `{"oss":false,"sort_by":"accepted_at","sort_order":"desc"}`))
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Content[0].Text)
	}
	if got := projectNames(t, res.Content[0].Text); !reflect.DeepEqual(got, []string{"Alpha", "Delta"}) {
		t.Errorf("names = %v", got)
	}
}

func TestGetProjectDetailsByRepoURL(t *testing.T) {
	state := newTestState(t, featureDataset())
	tests := []struct {
		name, args, wantName string
	}{
		{"exact with .git stripped", `{"repo_url":"https://github.com/org/charlie"}`, "charlie"},
		{"stored .git and case-insensitive", `{"repo_url":"HTTPS://GITHUB.COM/org/CHARLIE.git/"}`, "charlie"},
		{"stored trailing slash", `{"repo_url":"https://github.com/org/alpha"}`, "Alpha"},
		{"homepage match", `{"repo_url":"https://charlie.example.com"}`, "charlie"},
		{"no scheme", `{"repo_url":"github.com/org/charlie"}`, "charlie"},
		{"http scheme", `{"repo_url":"http://github.com/org/charlie.git"}`, "charlie"},
		{"blank name falls back to repo_url", `{"name":"  ","repo_url":"https://github.com/org/alpha"}`, "Alpha"},
		{"name wins over repo_url", `{"name":"bravo","repo_url":"https://github.com/org/alpha"}`, "bravo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := decodeToolResult(t, callTool(t, state, "get_project_details", tt.args))
			if res.IsError {
				t.Fatalf("unexpected error: %s", res.Content[0].Text)
			}
			if got := projectNames(t, res.Content[0].Text); !reflect.DeepEqual(got, []string{tt.wantName}) {
				t.Errorf("names = %v, want %s", got, tt.wantName)
			}
		})
	}

	t.Run("unknown url", func(t *testing.T) {
		res := decodeToolResult(t, callTool(t, state, "get_project_details", `{"repo_url":"https://github.com/nobody/nothing"}`))
		if !res.IsError || !strings.Contains(res.Content[0].Text, "no project found") {
			t.Errorf("want not-found error, got %+v", res)
		}
	})
	t.Run("both empty", func(t *testing.T) {
		for _, args := range []string{`{}`, `{"name":"","repo_url":"  "}`} {
			res := decodeToolResult(t, callTool(t, state, "get_project_details", args))
			if !res.IsError || !strings.Contains(res.Content[0].Text, "repo_url") {
				t.Errorf("args %s: want required error, got %+v", args, res)
			}
		}
	})
}

func TestToolSchemasForNewOptions(t *testing.T) {
	cat := buildToolCatalog(defaultConfig()).catalog

	details := cat["get_project_details"].InputSchema
	if _, has := details["required"]; has {
		t.Errorf("get_project_details must not require individual args: %v", details["required"])
	}
	props := details["properties"].(map[string]interface{})
	if _, ok := props["repo_url"]; !ok {
		t.Error("get_project_details missing repo_url")
	}

	qp := cat["query_projects"].InputSchema["properties"].(map[string]interface{})
	for _, k := range []string{"oss", "end_user", "specification"} {
		if qp[k].(map[string]interface{})["type"] != "boolean" {
			t.Errorf("%s should be boolean", k)
		}
	}
	if got := qp["sort_by"].(map[string]interface{})["enum"]; !reflect.DeepEqual(got, projectSortFields) {
		t.Errorf("query_projects sort_by enum = %v", got)
	}
	qm := cat["query_members"].InputSchema["properties"].(map[string]interface{})
	if got := qm["sort_by"].(map[string]interface{})["enum"]; !reflect.DeepEqual(got, memberSortFields) {
		t.Errorf("query_members sort_by enum = %v", got)
	}
}
