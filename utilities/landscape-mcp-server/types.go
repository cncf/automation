package main

import "time"

// Dataset represents the in-memory data used by the MCP server.
type Dataset struct {
	Items          []LandscapeItem
	CrunchbaseOrgs map[string]CrunchbaseOrganization
}

// LandscapeItem captures a single entry in the landscape.
type LandscapeItem struct {
	// Identity
	Name        string
	ID          string
	Description string
	HomepageURL string
	LogoURL     string

	// Taxonomy
	Category             string
	Subcategory          string
	MemberSubcategory    string
	Maturity             string
	AdditionalCategories []ItemCategory

	// Dates
	JoinedAt     *time.Time
	AcceptedAt   *time.Time
	GraduatedAt  *time.Time
	IncubatingAt *time.Time
	ArchivedAt   *time.Time

	// Project metadata
	OSS                   bool
	Specification         bool
	EndUser               bool
	ParentProject         string
	LatestAnnualReviewAt  *time.Time
	LatestAnnualReviewURL string
	Summary               *ItemSummary
	Audits                []Audit

	// Links
	CrunchbaseURL        string
	DevStatsURL          string
	ArtworkURL           string
	BlogURL              string
	TwitterURL           string
	SlackURL             string
	DiscordURL           string
	YouTubeURL           string
	LinkedInURL          string
	StackOverflowURL     string
	ChatChannel          string
	MailingListURL       string
	DocumentationURL     string
	GithubDiscussionsURL string

	// Repositories
	Repositories []Repository
}

// ItemCategory represents an additional category assignment.
type ItemCategory struct {
	Category    string
	Subcategory string
}

// ItemSummary holds a project's summary information.
type ItemSummary struct {
	UseCase string
}

// Audit represents a security or other audit of a project.
type Audit struct {
	Date   *time.Time
	Type   string
	URL    string
	Vendor string
}

// Repository represents a source code repository.
type Repository struct {
	URL     string
	Primary bool
}

// CrunchbaseOrganization describes Crunchbase data for an organization.
type CrunchbaseOrganization struct {
	Name            string
	Description     string
	HomepageURL     string
	City            string
	Country         string
	Region          string
	CompanyType     string // "for_profit", "non_profit"
	NumEmployeesMin int
	NumEmployeesMax int
	Categories      []string
	TotalFunding    *int64
	FundingRounds   []FundingRound
	Acquisitions    []Acquisition
	LinkedInURL     string
	TwitterURL      string
	StockExchange   string
	Ticker          string
}

// Acquisition represents a company acquisition.
type Acquisition struct {
	AcquireeName string
	AnnouncedOn  *time.Time
}

// FundingRound captures a single funding event.
type FundingRound struct {
	Amount      *int64
	AnnouncedOn *time.Time
	Kind        string
}
