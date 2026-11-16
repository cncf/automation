package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

type dataSource struct {
	filePath string
	url      string
}

// loadDataset reads and parses the data source. prev carries the validators
// from the last successful fetch so remote sources can answer 304; the
// returned validators belong to the freshly fetched body. When the remote
// reports the dataset unchanged the error wraps errNotModified.
func loadDataset(ctx context.Context, src dataSource, prev fetchValidators) (*Dataset, fetchValidators, error) {
	raw, validators, err := readSource(ctx, src, prev)
	if err != nil {
		return nil, fetchValidators{}, fmt.Errorf("load data source: %w", err)
	}

	var full fullDataset
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fetchValidators{}, fmt.Errorf("parse full dataset: %w", err)
	}

	var warns dateWarnings
	defer warns.log()

	items := make([]LandscapeItem, 0, len(full.Items))
	for _, it := range full.Items {
		accepted, _ := warns.parse(it.Name, "accepted_at", it.AcceptedAt)
		graduated, _ := warns.parse(it.Name, "graduated_at", it.GraduatedAt)
		incubating, _ := warns.parse(it.Name, "incubating_at", it.IncubatingAt)
		joined, _ := warns.parse(it.Name, "joined_at", it.JoinedAt)
		archived, _ := warns.parse(it.Name, "archived_at", it.ArchivedAt)
		latestAnnualReview, _ := warns.parse(it.Name, "latest_annual_review_at", it.LatestAnnualReviewAt)

		// Map audits
		audits := make([]Audit, 0, len(it.Audits))
		for _, a := range it.Audits {
			auditDate, ok := warns.parse(it.Name, "audit date", a.Date)
			if !ok {
				continue
			}
			audits = append(audits, Audit{
				Date:   auditDate,
				Type:   a.Type,
				URL:    a.URL,
				Vendor: a.Vendor,
			})
		}

		// Map repositories
		repos := make([]Repository, 0, len(it.Repositories))
		for _, r := range it.Repositories {
			repos = append(repos, Repository(r))
		}

		// Map additional categories
		addlCategories := make([]ItemCategory, 0, len(it.AdditionalCategories))
		for _, ac := range it.AdditionalCategories {
			addlCategories = append(addlCategories, ItemCategory(ac))
		}

		// Map summary
		var summary *ItemSummary
		if it.Summary != nil {
			summary = &ItemSummary{
				UseCase: it.Summary.UseCase,
			}
		}

		item := LandscapeItem{
			// Identity
			Name:        it.Name,
			ID:          it.ID,
			Description: it.Description,
			HomepageURL: it.HomepageURL,
			LogoURL:     it.Logo,

			// Taxonomy
			Category:             it.Category,
			Subcategory:          it.Subcategory,
			MemberSubcategory:    strings.TrimSpace(it.MemberSubcategory),
			Maturity:             strings.TrimSpace(it.Maturity),
			AdditionalCategories: addlCategories,

			// Dates
			JoinedAt:     joined,
			AcceptedAt:   accepted,
			GraduatedAt:  graduated,
			IncubatingAt: incubating,
			ArchivedAt:   archived,

			// Project metadata
			OSS:                   it.OSS,
			Specification:         it.Specification,
			EndUser:               it.EndUser,
			ParentProject:         it.ParentProject,
			LatestAnnualReviewAt:  latestAnnualReview,
			LatestAnnualReviewURL: it.LatestAnnualReviewURL,
			Summary:               summary,
			Audits:                audits,

			// Links
			CrunchbaseURL:        strings.TrimSpace(it.CrunchbaseURL),
			DevStatsURL:          it.DevStatsURL,
			ArtworkURL:           it.ArtworkURL,
			BlogURL:              it.BlogURL,
			TwitterURL:           it.TwitterURL,
			SlackURL:             it.SlackURL,
			DiscordURL:           it.DiscordURL,
			YouTubeURL:           it.YouTubeURL,
			LinkedInURL:          it.LinkedInURL,
			StackOverflowURL:     it.StackOverflowURL,
			ChatChannel:          it.ChatChannel,
			MailingListURL:       it.MailingListURL,
			DocumentationURL:     it.DocumentationURL,
			GithubDiscussionsURL: it.GithubDiscussionsURL,

			// Repositories
			Repositories: repos,
		}
		items = append(items, item)
	}

	orgs := make(map[string]CrunchbaseOrganization, len(full.CrunchbaseData))
	for url, org := range full.CrunchbaseData {
		rounds := make([]FundingRound, 0, len(org.FundingRounds))
		for _, round := range org.FundingRounds {
			date, ok := warns.parse(url, "announced_on", round.AnnouncedOn)
			if !ok {
				continue
			}

			var amountPtr *int64
			if round.Amount != nil {
				amt := int64(math.Round(*round.Amount))
				amountPtr = &amt
			}

			rounds = append(rounds, FundingRound{
				Amount:      amountPtr,
				AnnouncedOn: date,
				Kind:        strings.TrimSpace(round.Kind),
			})
		}

		// Map acquisitions
		acquisitions := make([]Acquisition, 0, len(org.Acquisitions))
		for _, acq := range org.Acquisitions {
			acqDate, ok := warns.parse(url, "acquisition announced_on", acq.AnnouncedOn)
			if !ok {
				continue
			}
			acquisitions = append(acquisitions, Acquisition{
				AcquireeName: acq.AcquireeName,
				AnnouncedOn:  acqDate,
			})
		}

		// Map total funding
		var totalFunding *int64
		if org.Funding != nil {
			f := int64(math.Round(*org.Funding))
			totalFunding = &f
		}

		orgs[url] = CrunchbaseOrganization{
			Name:            org.Name,
			Description:     org.Description,
			HomepageURL:     org.HomepageURL,
			City:            org.City,
			Country:         org.Country,
			Region:          org.Region,
			CompanyType:     org.CompanyType,
			NumEmployeesMin: org.NumEmployeesMin,
			NumEmployeesMax: org.NumEmployeesMax,
			Categories:      org.Categories,
			TotalFunding:    totalFunding,
			FundingRounds:   rounds,
			Acquisitions:    acquisitions,
			LinkedInURL:     org.LinkedInURL,
			TwitterURL:      org.TwitterURL,
			StockExchange:   org.StockExchange,
			Ticker:          org.Ticker,
		}
	}

	return &Dataset{
		Items:          items,
		CrunchbaseOrgs: orgs,
	}, validators, nil
}

// dateWarnings collects invalid-date occurrences during one load so they can
// be reported as a single log line instead of one per item.
type dateWarnings struct {
	count int
	first string
}

// parse parses value as a date. ok is false when value is non-empty and
// malformed; the occurrence is recorded for the summary.
func (w *dateWarnings) parse(subject, field, value string) (t *time.Time, ok bool) {
	t, err := parseDate(value)
	if err != nil {
		if w.count == 0 {
			w.first = fmt.Sprintf("%s %s", subject, field)
		}
		w.count++
		return nil, false
	}
	return t, true
}

func (w *dateWarnings) log() {
	if w.count > 0 {
		log.Printf("warning: %d invalid dates in dataset (first: %s)", w.count, w.first)
	}
}

func parseDate(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	tt := t
	return &tt, nil
}

type fullDataset struct {
	Items          []fullItem                  `json:"items"`
	CrunchbaseData map[string]fullOrganization `json:"crunchbase_data"`
}

type fullItem struct {
	Name                  string             `json:"name"`
	ID                    string             `json:"id"`
	Description           string             `json:"description"`
	HomepageURL           string             `json:"homepage_url"`
	Logo                  string             `json:"logo"`
	Category              string             `json:"category"`
	Subcategory           string             `json:"subcategory"`
	MemberSubcategory     string             `json:"member_subcategory"`
	Maturity              string             `json:"maturity"`
	AdditionalCategories  []fullItemCategory `json:"additional_categories"`
	JoinedAt              string             `json:"joined_at"`
	AcceptedAt            string             `json:"accepted_at"`
	GraduatedAt           string             `json:"graduated_at"`
	IncubatingAt          string             `json:"incubating_at"`
	ArchivedAt            string             `json:"archived_at"`
	OSS                   bool               `json:"oss"`
	Specification         bool               `json:"specification"`
	EndUser               bool               `json:"enduser"`
	ParentProject         string             `json:"parent_project"`
	LatestAnnualReviewAt  string             `json:"latest_annual_review_at"`
	LatestAnnualReviewURL string             `json:"latest_annual_review_url"`
	Summary               *fullItemSummary   `json:"summary"`
	Audits                []fullAudit        `json:"audits"`
	CrunchbaseURL         string             `json:"crunchbase_url"`
	DevStatsURL           string             `json:"devstats_url"`
	ArtworkURL            string             `json:"artwork_url"`
	BlogURL               string             `json:"blog_url"`
	TwitterURL            string             `json:"twitter_url"`
	SlackURL              string             `json:"slack_url"`
	DiscordURL            string             `json:"discord_url"`
	YouTubeURL            string             `json:"youtube_url"`
	LinkedInURL           string             `json:"linkedin_url"`
	StackOverflowURL      string             `json:"stack_overflow_url"`
	ChatChannel           string             `json:"chat_channel"`
	MailingListURL        string             `json:"mailing_list_url"`
	DocumentationURL      string             `json:"documentation_url"`
	GithubDiscussionsURL  string             `json:"github_discussions_url"`
	Repositories          []fullRepository   `json:"repositories"`
}

type fullItemCategory struct {
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
}

type fullItemSummary struct {
	UseCase string `json:"use_case"`
}

type fullAudit struct {
	Date   string `json:"date"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Vendor string `json:"vendor"`
}

type fullRepository struct {
	URL     string `json:"url"`
	Primary bool   `json:"primary"`
}

type fullOrganization struct {
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	HomepageURL     string             `json:"homepage_url"`
	City            string             `json:"city"`
	Country         string             `json:"country"`
	Region          string             `json:"region"`
	CompanyType     string             `json:"company_type"`
	NumEmployeesMin int                `json:"num_employees_min"`
	NumEmployeesMax int                `json:"num_employees_max"`
	Categories      []string           `json:"categories"`
	Funding         *float64           `json:"funding"`
	FundingRounds   []fullFundingRound `json:"funding_rounds"`
	Acquisitions    []fullAcquisition  `json:"acquisitions"`
	LinkedInURL     string             `json:"linkedin_url"`
	TwitterURL      string             `json:"twitter_url"`
	StockExchange   string             `json:"stock_exchange"`
	Ticker          string             `json:"ticker"`
}

type fullAcquisition struct {
	AcquireeName string `json:"acquiree_name"`
	AnnouncedOn  string `json:"announced_on"`
}

type fullFundingRound struct {
	Amount      *float64 `json:"amount"`
	AnnouncedOn string   `json:"announced_on"`
	Kind        string   `json:"kind"`
}
