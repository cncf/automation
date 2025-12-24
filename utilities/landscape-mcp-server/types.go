package main

import "time"

// Dataset represents the in-memory data used by the MCP server.
type Dataset struct {
	Items          []LandscapeItem
	CrunchbaseOrgs map[string]CrunchbaseOrganization
}

// LandscapeItem captures a single entry in the landscape with the
// fields required for the MCP queries we support.
type LandscapeItem struct {
	Name              string
	Category          string
	Subcategory       string
	MemberSubcategory string
	Maturity          string
	JoinedAt          *time.Time
	AcceptedAt        *time.Time
	GraduatedAt       *time.Time
	IncubatingAt      *time.Time
	CrunchbaseURL     string
}

// CrunchbaseOrganization describes the subset of Crunchbase data the
// server needs to compute funding-based queries.
type CrunchbaseOrganization struct {
	FundingRounds []FundingRound
}

// FundingRound captures a single funding event.
type FundingRound struct {
	Amount      *int64
	AnnouncedOn *time.Time
	Kind        string
}
