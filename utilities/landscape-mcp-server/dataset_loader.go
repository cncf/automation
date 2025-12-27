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

		item := LandscapeItem{
			Name:              it.Name,
			Category:          it.Category,
			Subcategory:       it.Subcategory,
			MemberSubcategory: strings.TrimSpace(it.MemberSubcategory),
			Maturity:          strings.TrimSpace(it.Maturity),
			JoinedAt:          joined,
			AcceptedAt:        accepted,
			GraduatedAt:       graduated,
			IncubatingAt:      incubating,
			CrunchbaseURL:     strings.TrimSpace(it.CrunchbaseURL),
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
		orgs[url] = CrunchbaseOrganization{FundingRounds: rounds}
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
		resp, err := http.DefaultClient.Do(req)
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
	Name              string `json:"name"`
	Category          string `json:"category"`
	Subcategory       string `json:"subcategory"`
	MemberSubcategory string `json:"member_subcategory"`
	Maturity          string `json:"maturity"`
	JoinedAt          string `json:"joined_at"`
	AcceptedAt        string `json:"accepted_at"`
	GraduatedAt       string `json:"graduated_at"`
	IncubatingAt      string `json:"incubating_at"`
	CrunchbaseURL     string `json:"crunchbase_url"`
}

type fullOrganization struct {
	FundingRounds []fullFundingRound `json:"funding_rounds"`
}

type fullFundingRound struct {
	Amount      *float64 `json:"amount"`
	AnnouncedOn string   `json:"announced_on"`
	Kind        string   `json:"kind"`
}
