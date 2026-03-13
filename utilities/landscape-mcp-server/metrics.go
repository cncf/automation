package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	defaultTopFundedLimit = 5
	maxCompareProjects    = 10
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

func isMember(item LandscapeItem) bool {
	return strings.Contains(strings.ToLower(item.Category), "member")
}

func isMemberTier(item LandscapeItem, tier string) bool {
	if !isMember(item) {
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
	Offset         int
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
	skipped := 0

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

		// Apply offset — skip matching items until we've skipped enough
		if q.Offset > 0 && skipped < q.Offset {
			skipped++
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
	Offset     int
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
	skipped := 0

	for _, item := range ds.Items {
		// Only include members
		if !isMember(item) {
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

		// Apply offset — skip matching items until we've skipped enough
		if q.Offset > 0 && skipped < q.Offset {
			skipped++
			continue
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

// ---------------------------------------------------------------------------
// Discovery tools
// ---------------------------------------------------------------------------

type categoryResult struct {
	Name          string              `json:"name"`
	ItemCount     int                 `json:"item_count"`
	Subcategories []subcategoryResult `json:"subcategories"`
}

type subcategoryResult struct {
	Name      string `json:"name"`
	ItemCount int    `json:"item_count"`
}

// listCategories returns all categories and subcategories with item counts.
func listCategories(ds *Dataset) (string, error) {
	catMap := make(map[string]map[string]int) // category -> subcategory -> count

	for _, item := range ds.Items {
		if _, ok := catMap[item.Category]; !ok {
			catMap[item.Category] = make(map[string]int)
		}
		catMap[item.Category][item.Subcategory]++
	}

	// Build sorted result
	catNames := make([]string, 0, len(catMap))
	for name := range catMap {
		catNames = append(catNames, name)
	}
	sort.Strings(catNames)

	categories := make([]categoryResult, 0, len(catNames))
	for _, catName := range catNames {
		subs := catMap[catName]
		subNames := make([]string, 0, len(subs))
		for subName := range subs {
			subNames = append(subNames, subName)
		}
		sort.Strings(subNames)

		totalCount := 0
		subcategories := make([]subcategoryResult, 0, len(subNames))
		for _, subName := range subNames {
			count := subs[subName]
			totalCount += count
			subcategories = append(subcategories, subcategoryResult{
				Name:      subName,
				ItemCount: count,
			})
		}

		categories = append(categories, categoryResult{
			Name:          catName,
			ItemCount:     totalCount,
			Subcategories: subcategories,
		})
	}

	response := map[string]interface{}{
		"categories": categories,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// landscapeSummary provides a high-level statistical overview of the landscape.
func landscapeSummary(ds *Dataset) (string, error) {
	totalItems := len(ds.Items)
	totalProjects := 0
	totalMembers := 0
	projectsByMaturity := make(map[string]int)
	membersByTier := make(map[string]int)
	categoriesSet := make(map[string]struct{})
	itemsWithCB := 0

	for _, item := range ds.Items {
		categoriesSet[item.Category] = struct{}{}

		// Count projects (items with non-empty maturity)
		if item.Maturity != "" {
			totalProjects++
			projectsByMaturity[item.Maturity]++
		}

		// Count members (items in a category containing "member")
		if isMember(item) {
			totalMembers++
			tier := item.MemberSubcategory
			if tier == "" {
				tier = item.Subcategory
			}
			if tier != "" {
				membersByTier[tier]++
			}
		}

		// Count items with crunchbase data
		if item.CrunchbaseURL != "" {
			if _, ok := ds.CrunchbaseOrgs[item.CrunchbaseURL]; ok {
				itemsWithCB++
			}
		}
	}

	response := map[string]interface{}{
		"total_items":                totalItems,
		"total_projects":             totalProjects,
		"total_members":              totalMembers,
		"projects_by_maturity":       projectsByMaturity,
		"members_by_tier":            membersByTier,
		"categories_count":           len(categoriesSet),
		"items_with_crunchbase_data": itemsWithCB,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type searchResultItem struct {
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Subcategory       string   `json:"subcategory"`
	Maturity          string   `json:"maturity,omitempty"`
	MemberSubcategory string   `json:"member_subcategory,omitempty"`
	HomepageURL       string   `json:"homepage_url,omitempty"`
	MatchFields       []string `json:"match_fields"`
}

// searchLandscape performs a case-insensitive substring search across items.
func searchLandscape(ds *Dataset, query string, limit int) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query parameter is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	queryLower := strings.ToLower(query)
	results := make([]searchResultItem, 0)

	for _, item := range ds.Items {
		var matchFields []string

		if strings.Contains(strings.ToLower(item.Name), queryLower) {
			matchFields = append(matchFields, "name")
		}
		if strings.Contains(strings.ToLower(item.Description), queryLower) {
			matchFields = append(matchFields, "description")
		}
		if strings.Contains(strings.ToLower(item.Category), queryLower) {
			matchFields = append(matchFields, "category")
		}
		if strings.Contains(strings.ToLower(item.Subcategory), queryLower) {
			matchFields = append(matchFields, "subcategory")
		}
		if strings.Contains(strings.ToLower(item.HomepageURL), queryLower) {
			matchFields = append(matchFields, "homepage_url")
		}

		if len(matchFields) == 0 {
			continue
		}

		result := searchResultItem{
			Name:        item.Name,
			Category:    item.Category,
			Subcategory: item.Subcategory,
			HomepageURL: item.HomepageURL,
			MatchFields: matchFields,
		}
		if item.Maturity != "" {
			result.Maturity = item.Maturity
		}
		if isMember(item) {
			tier := item.MemberSubcategory
			if tier == "" {
				tier = item.Subcategory
			}
			result.MemberSubcategory = tier
		}

		results = append(results, result)
		if len(results) >= limit {
			break
		}
	}

	response := map[string]interface{}{
		"count": len(results),
		"items": results,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type memberDetailResult struct {
	Name              string            `json:"name"`
	Category          string            `json:"category"`
	Subcategory       string            `json:"subcategory"`
	MemberSubcategory string            `json:"member_subcategory,omitempty"`
	JoinedAt          string            `json:"joined_at,omitempty"`
	HomepageURL       string            `json:"homepage_url,omitempty"`
	Links             map[string]string `json:"links,omitempty"`
	Organization      *orgSummary       `json:"organization,omitempty"`
}

// ---------------------------------------------------------------------------
// Analytical tools
// ---------------------------------------------------------------------------

type comparisonProject struct {
	Name         string            `json:"name"`
	Category     string            `json:"category"`
	Subcategory  string            `json:"subcategory"`
	Maturity     string            `json:"maturity,omitempty"`
	AcceptedAt   string            `json:"accepted_at,omitempty"`
	IncubatingAt string            `json:"incubating_at,omitempty"`
	GraduatedAt  string            `json:"graduated_at,omitempty"`
	HomepageURL  string            `json:"homepage_url,omitempty"`
	OSS          bool              `json:"oss"`
	Description  string            `json:"description,omitempty"`
	Repositories []Repository      `json:"repositories,omitempty"`
	Links        map[string]string `json:"links,omitempty"`
	Organization *orgSummary       `json:"organization,omitempty"`
}

// compareProjects returns a side-by-side comparison of named projects.
func compareProjects(ds *Dataset, names []string) (string, error) {
	if len(names) < 2 {
		return "", fmt.Errorf("at least 2 project names are required for comparison")
	}
	if len(names) > maxCompareProjects {
		return "", fmt.Errorf("maximum %d projects can be compared at once", maxCompareProjects)
	}
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return "", fmt.Errorf("project name cannot be empty")
		}
	}

	results := make([]comparisonProject, 0, len(names))
	for _, name := range names {
		nameLower := strings.ToLower(name)
		var found *LandscapeItem
		for i := range ds.Items {
			item := &ds.Items[i]
			if item.Maturity == "" {
				continue
			}
			if strings.Contains(strings.ToLower(item.Name), nameLower) {
				found = item
				break
			}
		}
		if found == nil {
			return "", fmt.Errorf("project not found: %s", name)
		}

		cp := comparisonProject{
			Name:        found.Name,
			Category:    found.Category,
			Subcategory: found.Subcategory,
			Maturity:    found.Maturity,
			HomepageURL: found.HomepageURL,
			OSS:         found.OSS,
			Description: found.Description,
		}
		if found.AcceptedAt != nil {
			cp.AcceptedAt = found.AcceptedAt.Format("2006-01-02")
		}
		if found.IncubatingAt != nil {
			cp.IncubatingAt = found.IncubatingAt.Format("2006-01-02")
		}
		if found.GraduatedAt != nil {
			cp.GraduatedAt = found.GraduatedAt.Format("2006-01-02")
		}
		if len(found.Repositories) > 0 {
			cp.Repositories = found.Repositories
		}
		cp.Links = buildLinks(*found)
		if found.CrunchbaseURL != "" {
			if org, ok := ds.CrunchbaseOrgs[found.CrunchbaseURL]; ok {
				cp.Organization = buildOrgSummary(org)
			}
		}
		results = append(results, cp)
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

type fundingKindSummary struct {
	Count       int   `json:"count"`
	TotalAmount int64 `json:"total_amount"`
}

type topFundedMember struct {
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	TotalAmount int64  `json:"total_amount"`
	RoundCount  int    `json:"round_count"`
}

// fundingAnalysis analyzes funding data from Crunchbase for landscape members.
func fundingAnalysis(ds *Dataset, tier string, year int) (string, error) {
	totalMembersAnalyzed := 0
	totalFundingRounds := 0
	var totalFundingAmount int64
	roundsByKind := make(map[string]*fundingKindSummary)
	var memberFunding []topFundedMember

	for _, item := range ds.Items {
		if !isMember(item) {
			continue
		}
		if tier != "" && !isMemberTier(item, tier) {
			continue
		}
		if item.CrunchbaseURL == "" {
			continue
		}
		org, ok := ds.CrunchbaseOrgs[item.CrunchbaseURL]
		if !ok {
			continue
		}

		totalMembersAnalyzed++
		memberRoundCount := 0
		var memberTotal int64

		for _, round := range org.FundingRounds {
			if year > 0 && round.AnnouncedOn != nil && round.AnnouncedOn.Year() != year {
				continue
			}
			if year > 0 && round.AnnouncedOn == nil {
				continue
			}

			totalFundingRounds++
			memberRoundCount++

			amt := int64(0)
			if round.Amount != nil {
				amt = *round.Amount
			}
			totalFundingAmount += amt
			memberTotal += amt

			kind := round.Kind
			if kind == "" {
				kind = "unknown"
			}
			if _, ok := roundsByKind[kind]; !ok {
				roundsByKind[kind] = &fundingKindSummary{}
			}
			roundsByKind[kind].Count++
			roundsByKind[kind].TotalAmount += amt
		}

		if memberRoundCount > 0 {
			memberTier := item.MemberSubcategory
			if memberTier == "" {
				memberTier = item.Subcategory
			}
			memberFunding = append(memberFunding, topFundedMember{
				Name:        item.Name,
				Tier:        memberTier,
				TotalAmount: memberTotal,
				RoundCount:  memberRoundCount,
			})
		}
	}

	// Sort top_funded descending by total_amount
	sort.Slice(memberFunding, func(i, j int) bool {
		return memberFunding[i].TotalAmount > memberFunding[j].TotalAmount
	})

	// Limit to top funded members
	topFunded := memberFunding
	if len(topFunded) > defaultTopFundedLimit {
		topFunded = topFunded[:defaultTopFundedLimit]
	}

	response := map[string]interface{}{
		"total_members_analyzed": totalMembersAnalyzed,
		"total_funding_rounds":   totalFundingRounds,
		"total_funding_amount":   totalFundingAmount,
		"rounds_by_kind":         roundsByKind,
		"top_funded":             topFunded,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type geoDistributionEntry struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// geographicDistribution returns a geographic breakdown of landscape items by Crunchbase location.
func geographicDistribution(ds *Dataset, groupBy string) (string, error) {
	if groupBy == "" {
		groupBy = "country"
	}
	switch groupBy {
	case "country", "region", "city":
	default:
		return "", fmt.Errorf("invalid group_by value: %q (must be country, region, or city)", groupBy)
	}

	totalAnalyzed := 0
	counts := make(map[string]int)

	for _, item := range ds.Items {
		if item.CrunchbaseURL == "" {
			continue
		}
		org, ok := ds.CrunchbaseOrgs[item.CrunchbaseURL]
		if !ok {
			continue
		}
		totalAnalyzed++

		var field string
		switch groupBy {
		case "country":
			field = org.Country
		case "region":
			field = org.Region
		case "city":
			field = org.City
		}
		if field != "" {
			counts[field]++
		}
	}

	totalWithLocation := 0
	for _, c := range counts {
		totalWithLocation += c
	}
	totalWithoutLocation := totalAnalyzed - totalWithLocation

	// Build sorted distribution
	distribution := make([]geoDistributionEntry, 0, len(counts))
	for name, count := range counts {
		pct := 0.0
		if totalWithLocation > 0 {
			pct = float64(count) / float64(totalWithLocation) * 100.0
			// Round to 1 decimal place
			pct = math.Round(pct*10) / 10
		}
		distribution = append(distribution, geoDistributionEntry{
			Name:       name,
			Count:      count,
			Percentage: pct,
		})
	}
	sort.Slice(distribution, func(i, j int) bool {
		return distribution[i].Count > distribution[j].Count
	})

	response := map[string]interface{}{
		"total_with_location":    totalWithLocation,
		"total_without_location": totalWithoutLocation,
		"distribution":           distribution,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getMemberDetails returns detailed information about a specific member
func getMemberDetails(ds *Dataset, name string) (string, error) {
	nameLower := strings.ToLower(name)
	var matches []memberDetailResult

	for _, item := range ds.Items {
		if !isMember(item) {
			continue
		}
		if !strings.Contains(strings.ToLower(item.Name), nameLower) {
			continue
		}

		detail := memberDetailResult{
			Name:              item.Name,
			Category:          item.Category,
			Subcategory:       item.Subcategory,
			MemberSubcategory: item.MemberSubcategory,
			HomepageURL:       item.HomepageURL,
		}
		if item.JoinedAt != nil {
			detail.JoinedAt = item.JoinedAt.Format("2006-01-02")
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

	if len(matches) == 0 {
		return "", fmt.Errorf("no member found matching name: %s", name)
	}

	response := map[string]interface{}{
		"count":   len(matches),
		"members": matches,
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
