package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"projects"
)

const (
	userAgent = "cncf-dot-project-onboarding-report"
)

// GraphQL types

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type orgNode struct {
	Login string `json:"login"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphQLResponse struct {
	Data struct {
		Enterprise struct {
			Organizations struct {
				Nodes    []orgNode `json:"nodes"`
				PageInfo pageInfo  `json:"pageInfo"`
			} `json:"organizations"`
		} `json:"enterprise"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func main() {
	var (
		enterprise = flag.String("enterprise", projects.DefaultEnterprise, "GitHub Enterprise slug")
		outputFile = flag.String("output", "ONBOARDED.md", "Output markdown file path")
		token      = flag.String("token", "", "GitHub token (or set GITHUB_TOKEN env)")
	)
	flag.Parse()

	ghToken := *token
	if ghToken == "" {
		ghToken = os.Getenv("GITHUB_TOKEN")
	}
	if ghToken == "" {
		log.Fatal("GitHub token is required: set GITHUB_TOKEN or use -token flag")
	}

	client := &http.Client{Timeout: projects.DefaultHTTPTimeout}

	log.Printf("Fetching organizations from enterprise: %s", *enterprise)
	orgs, err := listEnterpriseOrgs(client, ghToken, *enterprise)
	if err != nil {
		log.Fatalf("Failed to list enterprise orgs: %v", err)
	}
	log.Printf("Found %d organizations", len(orgs))

	var onboarded []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrency (avoid hitting rate limits)
	concurrency := 20
	sem := make(chan struct{}, concurrency)

	for _, o := range orgs {
		wg.Add(1)
		go func(org string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			has, err := hasDotProject(client, ghToken, org)
			if err != nil {
				log.Printf("Warning: error checking %s/.project: %v", org, err)
				return
			}
			if has {
				mu.Lock()
				onboarded = append(onboarded, org)
				mu.Unlock()
			}
		}(o)
	}
	wg.Wait()

	sort.Strings(onboarded)
	log.Printf("Found %d onboarded organizations (with .project repo)", len(onboarded))

	report := generateMarkdown(onboarded, len(orgs))

	if err := os.WriteFile(*outputFile, []byte(report), 0644); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}
	log.Printf("Report written to %s", *outputFile)
}

// listEnterpriseOrgs uses the GitHub GraphQL API to paginate through all
// organizations in the enterprise.
func listEnterpriseOrgs(client *http.Client, token, enterprise string) ([]string, error) {
	var allOrgs []string
	var cursor *string

	query := `
query($enterprise: String!, $first: Int!, $after: String) {
  enterprise(slug: $enterprise) {
    organizations(first: $first, after: $after) {
      nodes {
        login
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}`

	for {
		variables := map[string]any{
			"enterprise": enterprise,
			"first":      100,
		}
		if cursor != nil {
			variables["after"] = *cursor
		}

		reqBody := graphQLRequest{
			Query:     query,
			Variables: variables,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}

		req, err := http.NewRequest("POST", projects.DefaultGitHubGraphQLURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GraphQL API returned %d: %s", resp.StatusCode, string(respBody))
		}

		var gqlResp graphQLResponse
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		if len(gqlResp.Errors) > 0 {
			msgs := make([]string, len(gqlResp.Errors))
			for i, e := range gqlResp.Errors {
				msgs[i] = e.Message
			}
			return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(msgs, "; "))
		}

		// If the enterprise field is null/empty, the token likely lacks read:enterprise scope.
		if gqlResp.Data.Enterprise.Organizations.Nodes == nil && cursor == nil {
			return nil, fmt.Errorf(
				"enterprise '%s' returned no organizations — your token likely lacks the 'read:enterprise' scope.\n"+
					"  Run: gh auth refresh -s read:enterprise",
				enterprise,
			)
		}

		nodes := gqlResp.Data.Enterprise.Organizations.Nodes
		for _, n := range nodes {
			allOrgs = append(allOrgs, n.Login)
		}

		pi := gqlResp.Data.Enterprise.Organizations.PageInfo
		if !pi.HasNextPage {
			break
		}
		cursor = &pi.EndCursor
	}

	return allOrgs, nil
}

// hasDotProject checks if an org has a .project repository via REST HEAD request.
func hasDotProject(client *http.Client, token, org string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/.project", projects.DefaultGitHubAPIURL, org)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		log.Printf("Warning: failed to close response body: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound, http.StatusMovedPermanently:
		return false, nil
	case http.StatusForbidden, http.StatusTooManyRequests:
		return false, fmt.Errorf("rate limited or forbidden (status %d)", resp.StatusCode)
	default:
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
}

// generateMarkdown produces the onboarding report.
func generateMarkdown(onboarded []string, totalOrgs int) string {
	var sb strings.Builder

	sb.WriteString("# .project Onboarding Report\n\n")
	sb.WriteString(fmt.Sprintf("_Last updated: %s UTC_\n\n", time.Now().UTC().Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("> Auto-generated by the [onboarding-report](./cmd/onboarding-report) tool.\n\n"))
	sb.WriteString(fmt.Sprintf("**Total organizations checked:** %d  \n", totalOrgs))
	sb.WriteString(fmt.Sprintf("**Onboarded (have `.project` repo):** %d  \n\n", len(onboarded)))

	sb.WriteString("## Onboarded Organizations\n\n")
	sb.WriteString("| # | Organization | Repository |\n")
	sb.WriteString("|---|---|---|\n")

	for i, org := range onboarded {
		sb.WriteString(fmt.Sprintf("| %d | [%s](https://github.com/%s) | [%s/.project](https://github.com/%s/.project) |\n",
			i+1, org, org, org, org))
	}

	sb.WriteString("\n---\n")

	return sb.String()
}
