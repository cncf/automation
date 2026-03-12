package main

import (
	"context"
	"testing"
	"time"
)

// loadTestDataset loads the test fixture from testdata/sample.json.
func loadTestDataset(t *testing.T) *Dataset {
	t.Helper()
	ds, err := loadDataset(context.Background(), dataSource{
		filePath: "testdata/sample.json",
	})
	if err != nil {
		t.Fatalf("failed to load test dataset: %v", err)
	}
	return ds
}

// timePtr parses a "2006-01-02" date string and returns a *time.Time, or panics.
func timePtr(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic("timePtr: " + err.Error())
	}
	return &t
}

// int64Ptr returns a pointer to the given int64 value.
func int64Ptr(v int64) *int64 {
	return &v
}

// makeItem creates a LandscapeItem with the given name and maturity, applying optional modifiers.
func makeItem(name, maturity string, opts ...func(*LandscapeItem)) LandscapeItem {
	item := LandscapeItem{
		Name:     name,
		Maturity: maturity,
	}
	for _, opt := range opts {
		opt(&item)
	}
	return item
}

func TestLoadTestDataset(t *testing.T) {
	ds := loadTestDataset(t)
	if len(ds.Items) != 6 {
		t.Errorf("expected 6 items, got %d", len(ds.Items))
	}
	if len(ds.CrunchbaseOrgs) != 2 {
		t.Errorf("expected 2 crunchbase orgs, got %d", len(ds.CrunchbaseOrgs))
	}
}
