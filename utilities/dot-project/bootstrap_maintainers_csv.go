package projects

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// CSV column indices in project-maintainers.csv.
// Header: ,Project,Maintainer Name,Company,Github Name,OWNERS/MAINTAINERS
const (
	csvColStatus     = 0 // maturity, only set on a project block's first row
	csvColProject    = 1 // project label, only set on a block's first row
	csvColGithubName = 4 // the GitHub handle (no @ prefix)
)

// MaintainerBlock is a contiguous group of maintainers listed under a single
// project label in the foundation CSV.
type MaintainerBlock struct {
	Project string
	Status  string
	Handles []string
}

// continuationLabels are generic sub-group labels that occasionally appear in
// the Project column without the owning project's name (e.g. Linkerd is
// followed by a bare "Steering Committee" block). When a new block's label
// normalizes to one of these, its handles are merged into the preceding block
// rather than starting a standalone, unmatchable block.
var continuationLabels = map[string]bool{
	"steering committee":   true,
	"governance committee": true,
	"technical committee":  true,
	"maintainers":          true,
	"committers":           true,
	"approvers":            true,
	"reviewers":            true,
}

// FetchFoundationMaintainers loads and parses the foundation maintainers CSV.
//
// csvPath is an optional local file path. When empty, the CSV is fetched fresh
// from DefaultFoundationMaintainersCSVURL.
func FetchFoundationMaintainers(csvPath string, client *http.Client) ([]MaintainerBlock, error) {
	if csvPath != "" {
		f, err := os.Open(csvPath)
		if err != nil {
			return nil, fmt.Errorf("opening maintainers CSV %q: %w", csvPath, err)
		}
		defer func() { _ = f.Close() }()
		return parseFoundationMaintainersCSV(f)
	}
	return fetchMaintainersCSVFromURL(DefaultFoundationMaintainersCSVURL, client)
}

// fetchMaintainersCSVFromURL downloads and parses the CSV from url.
func fetchMaintainersCSVFromURL(url string, client *http.Client) ([]MaintainerBlock, error) {
	if client == nil {
		client = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching maintainers CSV from %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("maintainers CSV returned HTTP %d for %s", resp.StatusCode, url)
	}
	return parseFoundationMaintainersCSV(resp.Body)
}

// parseFoundationMaintainersCSV parses the foundation CSV, forward-filling the
// sparse status/project columns and associating bare continuation sub-groups
// with their owning project block.
func parseFoundationMaintainersCSV(r io.Reader) ([]MaintainerBlock, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // rows have a variable number of columns
	reader.LazyQuotes = true

	var blocks []MaintainerBlock
	current := -1 // index into blocks of the block currently being filled

	for i := 0; ; i++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing maintainers CSV at row %d: %w", i+1, err)
		}
		if i == 0 {
			continue // header row
		}

		project := strings.TrimSpace(field(record, csvColProject))
		status := strings.TrimSpace(field(record, csvColStatus))
		handle := strings.TrimSpace(strings.TrimPrefix(field(record, csvColGithubName), "@"))

		if project != "" {
			// A bare continuation label belongs to the preceding project block.
			if current >= 0 && continuationLabels[normalizeProjectKey(project)] {
				// keep filling the current block; do not start a new one
			} else {
				blocks = append(blocks, MaintainerBlock{Project: project, Status: status})
				current = len(blocks) - 1
			}
		}

		if handle != "" && current >= 0 {
			blocks[current].Handles = append(blocks[current].Handles, handle)
		}
	}

	return blocks, nil
}

// MatchProjectMaintainers returns the merged, de-duplicated set of handles for
// every block matching any of the supplied candidate names (typically the
// project name, GitHub org, repo, CLOMonitor name, and landscape name).
func MatchProjectMaintainers(blocks []MaintainerBlock, names ...string) []string {
	candidates := normalizeCandidates(names...)
	if len(candidates) == 0 {
		return nil
	}

	var handles []string
	for _, b := range blocks {
		if blockMatches(b.Project, candidates) {
			handles = mergeStringSlices(handles, b.Handles)
		}
	}
	return handles
}

// normalizeCandidates returns the de-duplicated set of normalized candidate
// names (length >= 2).
func normalizeCandidates(names ...string) map[string]bool {
	set := make(map[string]bool)
	for _, n := range names {
		if k := normalizeProjectKey(n); len(k) >= 2 {
			set[k] = true
		}
	}
	return set
}

// blockMatches reports whether a CSV project label matches any candidate name,
// either exactly or as a boundary-prefixed sub-group.
func blockMatches(project string, candidates map[string]bool) bool {
	label := normalizeProjectKey(project)
	if label == "" {
		return false
	}
	for c := range candidates {
		if label == c || hasBoundaryPrefix(label, c) {
			return true
		}
	}
	return false
}

// hasBoundaryPrefix reports whether s starts with prefix followed by a word
// boundary (space, ':', '(', '-' or end of string). Requires the prefix to be
// at least 3 chars to avoid spurious short-token matches.
func hasBoundaryPrefix(s, prefix string) bool {
	if len(prefix) < 3 || !strings.HasPrefix(s, prefix) {
		return false
	}
	if len(s) == len(prefix) {
		return true
	}
	switch s[len(prefix)] {
	case ' ', ':', '(', '-':
		return true
	}
	return false
}

// normalizeProjectKey lowercases, trims, and collapses internal whitespace.
func normalizeProjectKey(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// field safely returns the column at idx, or "" if the row is shorter.
func field(record []string, idx int) string {
	if idx < len(record) {
		return record[idx]
	}
	return ""
}
