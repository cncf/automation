package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type projectResult struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	Subcategory  string `json:"subcategory"`
	Maturity     string `json:"maturity,omitempty"`
	AcceptedAt   string `json:"accepted_at,omitempty"`
	IncubatingAt string `json:"incubating_at,omitempty"`
	GraduatedAt  string `json:"graduated_at,omitempty"`
	JoinedAt     string `json:"joined_at,omitempty"`
}

type orgSummary struct {
	Name            string   `json:"name,omitempty"`
	Description     string   `json:"description,omitempty"`
	City            string   `json:"city,omitempty"`
	Country         string   `json:"country,omitempty"`
	Region          string   `json:"region,omitempty"`
	CompanyType     string   `json:"company_type,omitempty"`
	NumEmployeesMin int      `json:"num_employees_min,omitempty"`
	NumEmployeesMax int      `json:"num_employees_max,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	TotalFunding    *int64   `json:"total_funding_usd,omitempty"`
	StockExchange   string   `json:"stock_exchange,omitempty"`
	Ticker          string   `json:"ticker,omitempty"`
}

type auditResult struct {
	Date   string `json:"date,omitempty"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Vendor string `json:"vendor"`
}

type projectDetailResult struct {
	Name                  string            `json:"name"`
	Category              string            `json:"category"`
	Subcategory           string            `json:"subcategory"`
	Maturity              string            `json:"maturity,omitempty"`
	Description           string            `json:"description,omitempty"`
	AcceptedAt            string            `json:"accepted_at,omitempty"`
	IncubatingAt          string            `json:"incubating_at,omitempty"`
	GraduatedAt           string            `json:"graduated_at,omitempty"`
	ArchivedAt            string            `json:"archived_at,omitempty"`
	JoinedAt              string            `json:"joined_at,omitempty"`
	HomepageURL           string            `json:"homepage_url,omitempty"`
	OSS                   bool              `json:"oss"`
	Summary               *ItemSummary      `json:"summary,omitempty"`
	Repositories          []Repository      `json:"repositories,omitempty"`
	Audits                []auditResult     `json:"audits,omitempty"`
	LatestAnnualReviewAt  string            `json:"latest_annual_review_at,omitempty"`
	LatestAnnualReviewURL string            `json:"latest_annual_review_url,omitempty"`
	AdditionalCategories  []ItemCategory    `json:"additional_categories,omitempty"`
	Links                 map[string]string `json:"links,omitempty"`
	Organization          *orgSummary       `json:"organization,omitempty"`
}

type memberResult struct {
	Name              string `json:"name"`
	Category          string `json:"category"`
	Subcategory       string `json:"subcategory"`
	MemberSubcategory string `json:"member_subcategory,omitempty"`
	JoinedAt          string `json:"joined_at,omitempty"`
}

func executeMetric(metric string, ds *Dataset, now time.Time) (string, error) {
	switch metric {
	case "incubating_project_count":
		count := 0
		for _, item := range ds.Items {
			if strings.EqualFold(item.Maturity, "incubating") {
				count++
			}
		}
		return encodeResult(metric, map[string]interface{}{
			"value":       count,
			"description": "Total CNCF projects currently in incubating maturity",
		})

	case "sandbox_projects_joined_this_year":
		year := now.Year()
		count := 0
		for _, item := range ds.Items {
			if !strings.EqualFold(item.Maturity, "sandbox") {
				continue
			}
			if item.AcceptedAt != nil && item.AcceptedAt.Year() == year {
				count++
			}
		}
		return encodeResult(metric, map[string]interface{}{
			"value":       count,
			"year":        year,
			"description": "Sandbox projects accepted into CNCF this year",
		})

	case "projects_graduated_last_year":
		targetYear := now.AddDate(-1, 0, 0).Year()
		count := 0
		for _, item := range ds.Items {
			if item.GraduatedAt != nil && item.GraduatedAt.Year() == targetYear {
				count++
			}
		}
		return encodeResult(metric, map[string]interface{}{
			"value":       count,
			"year":        targetYear,
			"description": "Projects that achieved graduated status last year",
		})

	case "gold_members_joined_this_year":
		year := now.Year()
		members := membersJoinedByTier(ds, "Gold", year)
		return encodeResult(metric, map[string]interface{}{
			"value":       len(members),
			"year":        year,
			"description": "Gold members that joined CNCF this year",
			"members":     members,
		})

	case "silver_members_joined_this_year":
		year := now.Year()
		members := membersJoinedByTier(ds, "Silver", year)
		return encodeResult(metric, map[string]interface{}{
			"value":       len(members),
			"year":        year,
			"description": "Silver members that joined CNCF this year",
			"members":     members,
		})

	case "silver_members_raised_last_month":
		prev := now.AddDate(0, -1, 0)
		events := fundingEventsForPeriod(ds, "Silver", prev.Year(), prev.Month())
		members := uniqueMembers(events)
		return encodeResult(metric, map[string]interface{}{
			"value":       len(members),
			"year":        prev.Year(),
			"month":       int(prev.Month()),
			"description": "Silver members with funding rounds last month",
			"members":     members,
			"events":      events,
		})

	case "gold_members_raised_this_year":
		year := now.Year()
		events := fundingEventsForPeriod(ds, "Gold", year, 0)
		members := uniqueMembers(events)
		return encodeResult(metric, map[string]interface{}{
			"value":       len(members),
			"year":        year,
			"description": "Gold members with funding rounds announced this year",
			"members":     members,
			"events":      events,
		})

	default:
		return "", fmt.Errorf("unknown metric: %s", metric)
	}
}

func encodeResult(metric string, payload map[string]interface{}) (string, error) {
	payload["metric"] = metric
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func isMemberTier(item LandscapeItem, tier string) bool {
	if !strings.Contains(strings.ToLower(item.Category), "member") {
		return false
	}
	if strings.EqualFold(item.MemberSubcategory, tier) {
		return true
	}
	return strings.EqualFold(item.Subcategory, tier)
}

type fundingEvent struct {
	Member      string `json:"member"`
	AnnouncedOn string `json:"announced_on"`
	AmountUSD   *int64 `json:"amount_usd,omitempty"`
	FundingType string `json:"funding_type,omitempty"`
}

func fundingEventsForPeriod(ds *Dataset, tier string, year int, month time.Month) []fundingEvent {
	var events []fundingEvent
	for _, item := range ds.Items {
		if !isMemberTier(item, tier) {
			continue
		}
		org, ok := ds.CrunchbaseOrgs[item.CrunchbaseURL]
		if !ok {
			continue
		}
		for _, round := range org.FundingRounds {
			if round.AnnouncedOn == nil {
				continue
			}
			if year > 0 && round.AnnouncedOn.Year() != year {
				continue
			}
			if month != 0 && round.AnnouncedOn.Month() != month {
				continue
			}
			// Capture only one event per round, but multiple per org if needed
			ev := fundingEvent{
				Member:      item.Name,
				AnnouncedOn: round.AnnouncedOn.Format("2006-01-02"),
			}
			if round.Amount != nil {
				ev.AmountUSD = round.Amount
			}
			if round.Kind != "" {
				ev.FundingType = round.Kind
			}
			events = append(events, ev)
		}
	}
	return events
}

func uniqueMembers(events []fundingEvent) []string {
	seen := make(map[string]struct{})
	members := make([]string, 0)
	for _, ev := range events {
		name := strings.TrimSpace(ev.Member)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		members = append(members, name)
	}
	sort.Strings(members)
	return members
}

func membersJoinedByTier(ds *Dataset, tier string, year int) []string {
	members := make([]string, 0)
	for _, item := range ds.Items {
		if !isMemberTier(item, tier) {
			continue
		}
		if item.JoinedAt == nil || item.JoinedAt.Year() != year {
			continue
		}
		members = append(members, item.Name)
	}
	sort.Strings(members)
	return members
}

// ProjectQuery holds the parameters for querying projects.
type ProjectQuery struct {
	Maturity       string
	Name           string
	Category       string
	Subcategory    string
	GraduatedFrom  string
	GraduatedTo    string
	IncubatingFrom string
	IncubatingTo   string
	AcceptedFrom   string
	AcceptedTo     string
	Limit          int
}

// queryProjects filters and returns projects based on various criteria
func queryProjects(ds *Dataset, q ProjectQuery) (string, error) {
	// Parse date filters
	var gradFromDate, gradToDate, incFromDate, incToDate, accFromDate, accToDate *time.Time

	if q.GraduatedFrom != "" {
		d, err := time.Parse("2006-01-02", q.GraduatedFrom)
		if err != nil {
			return "", fmt.Errorf("invalid graduated_from date: %w", err)
		}
		gradFromDate = &d
	}
	if q.GraduatedTo != "" {
		d, err := time.Parse("2006-01-02", q.GraduatedTo)
		if err != nil {
			return "", fmt.Errorf("invalid graduated_to date: %w", err)
		}
		gradToDate = &d
	}
	if q.IncubatingFrom != "" {
		d, err := time.Parse("2006-01-02", q.IncubatingFrom)
		if err != nil {
			return "", fmt.Errorf("invalid incubating_from date: %w", err)
		}
		incFromDate = &d
	}
	if q.IncubatingTo != "" {
		d, err := time.Parse("2006-01-02", q.IncubatingTo)
		if err != nil {
			return "", fmt.Errorf("invalid incubating_to date: %w", err)
		}
		incToDate = &d
	}
	if q.AcceptedFrom != "" {
		d, err := time.Parse("2006-01-02", q.AcceptedFrom)
		if err != nil {
			return "", fmt.Errorf("invalid accepted_from date: %w", err)
		}
		accFromDate = &d
	}
	if q.AcceptedTo != "" {
		d, err := time.Parse("2006-01-02", q.AcceptedTo)
		if err != nil {
			return "", fmt.Errorf("invalid accepted_to date: %w", err)
		}
		accToDate = &d
	}

	results := make([]projectResult, 0)
	nameLower := strings.ToLower(q.Name)

	for _, item := range ds.Items {
		// Filter by maturity
		if q.Maturity != "" && !strings.EqualFold(item.Maturity, q.Maturity) {
			continue
		}

		// Filter by category
		if q.Category != "" && !strings.EqualFold(item.Category, q.Category) {
			continue
		}
		// Filter by subcategory
		if q.Subcategory != "" && !strings.EqualFold(item.Subcategory, q.Subcategory) {
			continue
		}

		// Filter by name
		if q.Name != "" && !strings.Contains(strings.ToLower(item.Name), nameLower) {
			continue
		}

		// Filter by graduated dates
		if gradFromDate != nil {
			if item.GraduatedAt == nil || item.GraduatedAt.Before(*gradFromDate) {
				continue
			}
		}
		if gradToDate != nil {
			if item.GraduatedAt == nil || item.GraduatedAt.After(*gradToDate) {
				continue
			}
		}

		// Filter by incubating dates
		if incFromDate != nil {
			if item.IncubatingAt == nil || item.IncubatingAt.Before(*incFromDate) {
				continue
			}
		}
		if incToDate != nil {
			if item.IncubatingAt == nil || item.IncubatingAt.After(*incToDate) {
				continue
			}
		}

		// Filter by accepted dates
		if accFromDate != nil {
			if item.AcceptedAt == nil || item.AcceptedAt.Before(*accFromDate) {
				continue
			}
		}
		if accToDate != nil {
			if item.AcceptedAt == nil || item.AcceptedAt.After(*accToDate) {
				continue
			}
		}

		// Only include projects (items with maturity)
		if item.Maturity == "" {
			continue
		}

		result := projectResult{
			Name:        item.Name,
			Category:    item.Category,
			Subcategory: item.Subcategory,
			Maturity:    item.Maturity,
		}
		if item.AcceptedAt != nil {
			result.AcceptedAt = item.AcceptedAt.Format("2006-01-02")
		}
		if item.IncubatingAt != nil {
			result.IncubatingAt = item.IncubatingAt.Format("2006-01-02")
		}
		if item.GraduatedAt != nil {
			result.GraduatedAt = item.GraduatedAt.Format("2006-01-02")
		}
		if item.JoinedAt != nil {
			result.JoinedAt = item.JoinedAt.Format("2006-01-02")
		}

		results = append(results, result)
		if len(results) >= q.Limit {
			break
		}
	}

	response := map[string]interface{}{
		"count":    len(results),
		"projects": results,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MemberQuery holds the parameters for querying members.
type MemberQuery struct {
	Tier       string
	Category   string
	JoinedFrom string
	JoinedTo   string
	Limit      int
}

// queryMembers filters and returns members based on tier and join dates
func queryMembers(ds *Dataset, q MemberQuery) (string, error) {
	// Parse date filters
	var joinFromDate, joinToDate *time.Time

	if q.JoinedFrom != "" {
		d, err := time.Parse("2006-01-02", q.JoinedFrom)
		if err != nil {
			return "", fmt.Errorf("invalid joined_from date: %w", err)
		}
		joinFromDate = &d
	}
	if q.JoinedTo != "" {
		d, err := time.Parse("2006-01-02", q.JoinedTo)
		if err != nil {
			return "", fmt.Errorf("invalid joined_to date: %w", err)
		}
		joinToDate = &d
	}

	results := make([]memberResult, 0)

	for _, item := range ds.Items {
		// Only include members
		if !strings.Contains(strings.ToLower(item.Category), "member") {
			continue
		}

		// Filter by tier
		if q.Tier != "" && !isMemberTier(item, q.Tier) {
			continue
		}

		// Filter by category
		if q.Category != "" && !strings.EqualFold(item.Category, q.Category) {
			continue
		}

		// Filter by joined dates
		if joinFromDate != nil {
			if item.JoinedAt == nil || item.JoinedAt.Before(*joinFromDate) {
				continue
			}
		}
		if joinToDate != nil {
			if item.JoinedAt == nil || item.JoinedAt.After(*joinToDate) {
				continue
			}
		}

		result := memberResult{
			Name:              item.Name,
			Category:          item.Category,
			Subcategory:       item.Subcategory,
			MemberSubcategory: item.MemberSubcategory,
		}
		if item.JoinedAt != nil {
			result.JoinedAt = item.JoinedAt.Format("2006-01-02")
		}

		results = append(results, result)
		if len(results) >= q.Limit {
			break
		}
	}

	response := map[string]interface{}{
		"count":   len(results),
		"members": results,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// buildLinks constructs a map of non-empty link URLs from a LandscapeItem.
func buildLinks(item LandscapeItem) map[string]string {
	links := make(map[string]string)
	if item.DevStatsURL != "" {
		links["devstats"] = item.DevStatsURL
	}
	if item.BlogURL != "" {
		links["blog"] = item.BlogURL
	}
	if item.TwitterURL != "" {
		links["twitter"] = item.TwitterURL
	}
	if item.SlackURL != "" {
		links["slack"] = item.SlackURL
	}
	if item.DiscordURL != "" {
		links["discord"] = item.DiscordURL
	}
	if item.YouTubeURL != "" {
		links["youtube"] = item.YouTubeURL
	}
	if item.LinkedInURL != "" {
		links["linkedin"] = item.LinkedInURL
	}
	if item.StackOverflowURL != "" {
		links["stackoverflow"] = item.StackOverflowURL
	}
	if item.ChatChannel != "" {
		links["chat_channel"] = item.ChatChannel
	}
	if item.MailingListURL != "" {
		links["mailing_list"] = item.MailingListURL
	}
	if item.DocumentationURL != "" {
		links["documentation"] = item.DocumentationURL
	}
	if item.GithubDiscussionsURL != "" {
		links["github_discussions"] = item.GithubDiscussionsURL
	}
	if item.ArtworkURL != "" {
		links["artwork"] = item.ArtworkURL
	}
	if len(links) == 0 {
		return nil
	}
	return links
}

// buildOrgSummary creates an orgSummary from a CrunchbaseOrganization.
func buildOrgSummary(org CrunchbaseOrganization) *orgSummary {
	return &orgSummary{
		Name:            org.Name,
		Description:     org.Description,
		City:            org.City,
		Country:         org.Country,
		Region:          org.Region,
		CompanyType:     org.CompanyType,
		NumEmployeesMin: org.NumEmployeesMin,
		NumEmployeesMax: org.NumEmployeesMax,
		Categories:      org.Categories,
		TotalFunding:    org.TotalFunding,
		StockExchange:   org.StockExchange,
		Ticker:          org.Ticker,
	}
}

// getProjectDetails returns detailed information about a specific project
func getProjectDetails(ds *Dataset, name string) (string, error) {
	nameLower := strings.ToLower(name)
	var matches []projectDetailResult

	for _, item := range ds.Items {
		// Only include projects (items with maturity)
		if item.Maturity == "" {
			continue
		}

		if strings.Contains(strings.ToLower(item.Name), nameLower) {
			detail := projectDetailResult{
				Name:        item.Name,
				Category:    item.Category,
				Subcategory: item.Subcategory,
				Maturity:    item.Maturity,
				Description: item.Description,
				HomepageURL: item.HomepageURL,
				OSS:         item.OSS,
				Summary:     item.Summary,
			}
			if item.AcceptedAt != nil {
				detail.AcceptedAt = item.AcceptedAt.Format("2006-01-02")
			}
			if item.IncubatingAt != nil {
				detail.IncubatingAt = item.IncubatingAt.Format("2006-01-02")
			}
			if item.GraduatedAt != nil {
				detail.GraduatedAt = item.GraduatedAt.Format("2006-01-02")
			}
			if item.ArchivedAt != nil {
				detail.ArchivedAt = item.ArchivedAt.Format("2006-01-02")
			}
			if item.JoinedAt != nil {
				detail.JoinedAt = item.JoinedAt.Format("2006-01-02")
			}
			if item.LatestAnnualReviewAt != nil {
				detail.LatestAnnualReviewAt = item.LatestAnnualReviewAt.Format("2006-01-02")
			}
			detail.LatestAnnualReviewURL = item.LatestAnnualReviewURL

			if len(item.Repositories) > 0 {
				detail.Repositories = item.Repositories
			}

			// Map audits
			if len(item.Audits) > 0 {
				audits := make([]auditResult, 0, len(item.Audits))
				for _, a := range item.Audits {
					ar := auditResult{
						Type:   a.Type,
						URL:    a.URL,
						Vendor: a.Vendor,
					}
					if a.Date != nil {
						ar.Date = a.Date.Format("2006-01-02")
					}
					audits = append(audits, ar)
				}
				detail.Audits = audits
			}

			if len(item.AdditionalCategories) > 0 {
				detail.AdditionalCategories = item.AdditionalCategories
			}

			// Build links
			detail.Links = buildLinks(item)

			// Look up Crunchbase org
			if item.CrunchbaseURL != "" {
				if org, ok := ds.CrunchbaseOrgs[item.CrunchbaseURL]; ok {
					detail.Organization = buildOrgSummary(org)
				}
			}

			matches = append(matches, detail)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no project found matching name: %s", name)
	}

	response := map[string]interface{}{
		"count":    len(matches),
		"projects": matches,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
