package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

type dataSource struct {
	filePath string
	url      string
}

func loadDataset(ctx context.Context, src dataSource) (*Dataset, error) {
	raw, err := readSource(ctx, src.filePath, src.url)
	if err != nil {
		return nil, fmt.Errorf("load data source: %w", err)
	}

	var full fullDataset
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("parse full dataset: %w", err)
	}

	items := make([]LandscapeItem, 0, len(full.Items))
	for _, it := range full.Items {
		accepted, err := parseDate(it.AcceptedAt)
		if err != nil {
			log.Printf("warning: invalid accepted_at for %s: %v", it.Name, err)
		}
		graduated, err := parseDate(it.GraduatedAt)
		if err != nil {
			log.Printf("warning: invalid graduated_at for %s: %v", it.Name, err)
		}
		incubating, err := parseDate(it.IncubatingAt)
		if err != nil {
			log.Printf("warning: invalid incubating_at for %s: %v", it.Name, err)
		}
		joined, err := parseDate(it.JoinedAt)
		if err != nil {
			log.Printf("warning: invalid joined_at for %s: %v", it.Name, err)
		}
		archived, err := parseDate(it.ArchivedAt)
		if err != nil {
			log.Printf("warning: invalid archived_at for %s: %v", it.Name, err)
		}
		latestAnnualReview, err := parseDate(it.LatestAnnualReviewAt)
		if err != nil {
			log.Printf("warning: invalid latest_annual_review_at for %s: %v", it.Name, err)
		}

		// Map audits
		audits := make([]Audit, 0, len(it.Audits))
		for _, a := range it.Audits {
			auditDate, err := parseDate(a.Date)
			if err != nil {
				log.Printf("warning: invalid audit date for %s: %v", it.Name, err)
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
			repos = append(repos, Repository{
				URL:     r.URL,
				Primary: r.Primary,
			})
		}

		// Map additional categories
		addlCategories := make([]ItemCategory, 0, len(it.AdditionalCategories))
		for _, ac := range it.AdditionalCategories {
			addlCategories = append(addlCategories, ItemCategory{
				Category:    ac.Category,
				Subcategory: ac.Subcategory,
			})
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
			date, err := parseDate(round.AnnouncedOn)
			if err != nil {
				log.Printf("warning: invalid announced_on for %s: %v", url, err)
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
			acqDate, err := parseDate(acq.AnnouncedOn)
			if err != nil {
				log.Printf("warning: invalid acquisition announced_on for %s: %v", url, err)
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
	}, nil
}

func readSource(ctx context.Context, filePath, url string) ([]byte, error) {
	switch {
	case filePath != "":
		return os.ReadFile(filePath)
	case url != "":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	default:
		return nil, errors.New("either file or url must be provided")
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
