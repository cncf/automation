package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantErr bool
		wantY   int
		wantM   time.Month
		wantD   int
	}{
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
		{
			name:    "whitespace-only returns nil",
			input:   "  ",
			wantNil: true,
		},
		{
			name:  "valid date parses correctly",
			input: "2024-06-15",
			wantY: 2024,
			wantM: time.June,
			wantD: 15,
		},
		{
			name:    "invalid string returns error",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "wrong format returns error",
			input:   "06/15/2024",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil time, got nil")
			}
			if got.Year() != tt.wantY {
				t.Errorf("year: got %d, want %d", got.Year(), tt.wantY)
			}
			if got.Month() != tt.wantM {
				t.Errorf("month: got %v, want %v", got.Month(), tt.wantM)
			}
			if got.Day() != tt.wantD {
				t.Errorf("day: got %d, want %d", got.Day(), tt.wantD)
			}
		})
	}
}

func TestLoadExpandedFields(t *testing.T) {
	ds := loadTestDataset(t)

	// Find and verify Kubernetes item
	var k8s *LandscapeItem
	for i := range ds.Items {
		if ds.Items[i].Name == "Kubernetes" {
			k8s = &ds.Items[i]
			break
		}
	}
	if k8s == nil {
		t.Fatal("Kubernetes item not found")
	}
	if k8s.Description == "" {
		t.Error("Kubernetes: expected non-empty Description")
	}
	if k8s.HomepageURL == "" {
		t.Error("Kubernetes: expected non-empty HomepageURL")
	}
	if len(k8s.Repositories) == 0 {
		t.Error("Kubernetes: expected at least one repository")
	} else {
		foundPrimary := false
		for _, r := range k8s.Repositories {
			if r.Primary {
				foundPrimary = true
				if !strings.Contains(r.URL, "kubernetes/kubernetes") {
					t.Errorf("Kubernetes: expected primary repo URL to contain 'kubernetes/kubernetes', got %s", r.URL)
				}
			}
		}
		if !foundPrimary {
			t.Error("Kubernetes: expected a primary repository")
		}
	}
	if !k8s.OSS {
		t.Error("Kubernetes: expected OSS to be true")
	}
	if k8s.DevStatsURL == "" {
		t.Error("Kubernetes: expected non-empty DevStatsURL")
	}
	if k8s.BlogURL == "" {
		t.Error("Kubernetes: expected non-empty BlogURL")
	}
	if k8s.TwitterURL == "" {
		t.Error("Kubernetes: expected non-empty TwitterURL")
	}
	if k8s.SlackURL == "" {
		t.Error("Kubernetes: expected non-empty SlackURL")
	}
	if k8s.ChatChannel == "" {
		t.Error("Kubernetes: expected non-empty ChatChannel")
	}

	// Find and verify Akri item
	var akri *LandscapeItem
	for i := range ds.Items {
		if ds.Items[i].Name == "Akri" {
			akri = &ds.Items[i]
			break
		}
	}
	if akri == nil {
		t.Fatal("Akri item not found")
	}
	if len(akri.Audits) == 0 {
		t.Fatal("Akri: expected at least one audit")
	}
	if akri.Audits[0].Type != "security" {
		t.Errorf("Akri: expected first audit type 'security', got %q", akri.Audits[0].Type)
	}
	if akri.Audits[0].Date == nil {
		t.Error("Akri: expected first audit to have a date")
	}
	if akri.Audits[0].Vendor == "" {
		t.Error("Akri: expected first audit to have a vendor")
	}
	if akri.Summary == nil {
		t.Fatal("Akri: expected non-nil Summary")
	}
	if akri.Summary.UseCase == "" {
		t.Error("Akri: expected non-empty Summary.UseCase")
	}

	// Find and verify TestGoldCorp Crunchbase org
	org, ok := ds.CrunchbaseOrgs["https://www.crunchbase.com/organization/testgoldcorp"]
	if !ok {
		t.Fatal("TestGoldCorp crunchbase org not found")
	}
	if org.Name == "" {
		t.Error("TestGoldCorp org: expected non-empty Name")
	}
	if org.City == "" {
		t.Error("TestGoldCorp org: expected non-empty City")
	}
	if org.Country == "" {
		t.Error("TestGoldCorp org: expected non-empty Country")
	}
	if org.NumEmployeesMin <= 0 {
		t.Errorf("TestGoldCorp org: expected NumEmployeesMin > 0, got %d", org.NumEmployeesMin)
	}
	if len(org.Acquisitions) == 0 {
		t.Fatal("TestGoldCorp org: expected at least one acquisition")
	}
	if org.Acquisitions[0].AcquireeName == "" {
		t.Error("TestGoldCorp org: expected non-empty acquiree name")
	}
	if org.Acquisitions[0].AnnouncedOn == nil {
		t.Error("TestGoldCorp org: expected acquisition to have an announced date")
	}
	if org.CompanyType != "for_profit" {
		t.Errorf("TestGoldCorp org: expected CompanyType 'for_profit', got %q", org.CompanyType)
	}
	if len(org.Categories) == 0 {
		t.Error("TestGoldCorp org: expected at least one category")
	}
	if org.LinkedInURL == "" {
		t.Error("TestGoldCorp org: expected non-empty LinkedInURL")
	}
	if org.TwitterURL == "" {
		t.Error("TestGoldCorp org: expected non-empty TwitterURL")
	}
}
