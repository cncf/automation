package projects

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Governance maintainer suggestions
//
// Maintainers are sourced mainly from the foundation CSV (see
// bootstrap_maintainers_csv.go). Separately, we still scan an org's governance
// files (CODEOWNERS / OWNERS / MAINTAINERS) and display the handles found there
// as *suggestions* in the GitHub onboarding Issue
// ---------------------------------------------------------------------------

// Governance roles, derived from the file a handle was discovered in.
const (
	roleMaintainer = "maintainer" // MAINTAINERS / MAINTAINERS.md
	roleApprover   = "approver"   // OWNERS: approvers
	roleReviewer   = "reviewer"   // OWNERS: reviewers
	roleCodeowner  = "code owner" // CODEOWNERS
)

// rolePriority controls the display order of roles within a row.
var rolePriority = map[string]int{
	roleMaintainer: 0,
	roleApprover:   1,
	roleReviewer:   2,
	roleCodeowner:  3,
}

// MaintainerSuggestion is a handle discovered in an org's governance files,
// annotated with the role(s) it was found under and the source location(s).
type MaintainerSuggestion struct {
	Handle  string   // GitHub handle, without @
	Roles   []string // e.g. ["maintainer", "code owner"]
	Sources []string // e.g. ["cilium/cilium:MAINTAINERS", "cilium/.github:CODEOWNERS"]
}

// suggestionCollector accumulates governance handles with their roles and
// sources, deduplicating both. Handles are matched case-insensitively but
// stored in their first-seen casing.
type suggestionCollector struct {
	byKey map[string]*MaintainerSuggestion // key = lowercase handle
	roles map[string]map[string]bool       // key -> set of roles
	srcs  map[string]map[string]bool       // key -> set of sources
}

func newSuggestionCollector() *suggestionCollector {
	return &suggestionCollector{
		byKey: make(map[string]*MaintainerSuggestion),
		roles: make(map[string]map[string]bool),
		srcs:  make(map[string]map[string]bool),
	}
}

// add records a handle discovered under role at source. Blank handles are
// ignored; the @ prefix is stripped.
func (c *suggestionCollector) add(handle, role, source string) {
	handle = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(handle), "@"))
	if handle == "" {
		return
	}
	key := strings.ToLower(handle)

	if _, ok := c.byKey[key]; !ok {
		c.byKey[key] = &MaintainerSuggestion{Handle: handle}
		c.roles[key] = make(map[string]bool)
		c.srcs[key] = make(map[string]bool)
	}
	if role != "" {
		c.roles[key][role] = true
	}
	if source != "" {
		c.srcs[key][source] = true
	}
}

// suggestions returns the collected handles excluding any whose lowercase form
// is present in exclude (e.g. the CSV roster), sorted by handle. Roles are
// ordered by rolePriority; sources are sorted alphabetically.
func (c *suggestionCollector) suggestions(exclude map[string]bool) []MaintainerSuggestion {
	var out []MaintainerSuggestion
	for key, s := range c.byKey {
		if exclude[key] {
			continue
		}
		roles := setKeys(c.roles[key])
		sort.Slice(roles, func(i, j int) bool {
			pi, oki := rolePriority[roles[i]]
			pj, okj := rolePriority[roles[j]]
			if oki && okj && pi != pj {
				return pi < pj
			}
			return roles[i] < roles[j]
		})
		sources := setKeys(c.srcs[key])
		sort.Strings(sources)

		out = append(out, MaintainerSuggestion{Handle: s.Handle, Roles: roles, Sources: sources})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Handle) < strings.ToLower(out[j].Handle)
	})
	return out
}

func setKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// renderSuggestionsTable renders suggestions as a GitHub-flavored markdown
// table. Returns "" when there are no suggestions.
func renderSuggestionsTable(suggestions []MaintainerSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("| Handle | Role(s) | Found in |\n")
	b.WriteString("|--------|---------|----------|\n")
	for _, s := range suggestions {
		roles := strings.Join(s.Roles, ", ")
		sources := strings.Join(s.Sources, ", ")
		fmt.Fprintf(&b, "| @%s | %s | %s |\n", s.Handle, roles, sources)
	}
	return b.String()
}

// BuildSuggestionsSection renders the full markdown "Suggested maintainers"
// section (heading + explanation + table) used in the onboarding issue and the
// bootstrap output. Returns "" when there are no suggestions.
func BuildSuggestionsSection(suggestions []MaintainerSuggestion) string {
	table := renderSuggestionsTable(suggestions)
	if table == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Suggested maintainers (from your org's governance files)\n\n")
	b.WriteString("These GitHub handles were found in your org's CODEOWNERS / OWNERS / MAINTAINERS " +
		"files but are **not yet** in the [CNCF foundation maintainers CSV]" +
		"(https://github.com/cncf/foundation/blob/main/project-maintainers.csv). " +
		"If they are current maintainers, please add them to `project-maintainers.csv` by opening a PR against `cncf/foundation` and `maintainers.yaml`. " +
		"Otherwise, you can ignore this list - they won't be added to the maintainers.yml.\n\n")
	b.WriteString(table)
	return b.String()
}

// ---------------------------------------------------------------------------
// Governance file parsers (handle extraction)
// ---------------------------------------------------------------------------

// parseCodeowners extracts individual GitHub handles from a CODEOWNERS file.
// It skips comment lines, team references (org/team), and email addresses.
// Returns a sorted, deduplicated list of handles (without @ prefix).
func parseCodeowners(content string) []string {
	if content == "" {
		return nil
	}

	seen := make(map[string]bool)
	var handles []string

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Tokens after the file pattern are the owners
		tokens := strings.Fields(line)
		for _, token := range tokens {
			if !strings.HasPrefix(token, "@") {
				continue
			}
			handle := strings.TrimPrefix(token, "@")
			if handle == "" {
				continue
			}
			// Skip team references (contain a slash: org/team-name)
			if strings.Contains(handle, "/") {
				continue
			}
			// Skip email-like tokens
			if strings.Contains(handle, ".") && strings.Contains(handle, "@") {
				continue
			}
			if !seen[handle] {
				seen[handle] = true
				handles = append(handles, handle)
			}
		}
	}

	sort.Strings(handles)
	return handles
}

// ownersFileData represents the YAML structure of a Kubernetes-style OWNERS file.
type ownersFileData struct {
	Approvers []string `yaml:"approvers"`
	Reviewers []string `yaml:"reviewers"`
}

// parseOwnersFile parses a Kubernetes-style OWNERS file (YAML with
// approvers/reviewers lists). Returns deduplicated lists of approvers and
// reviewers (without @ prefix, trimmed).
func parseOwnersFile(content string) (approvers []string, reviewers []string) {
	if content == "" {
		return nil, nil
	}

	var data ownersFileData
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return nil, nil
	}

	approvers = normalizeHandleList(data.Approvers)
	reviewers = normalizeHandleList(data.Reviewers)

	if len(approvers) == 0 {
		approvers = nil
	}
	if len(reviewers) == 0 {
		reviewers = nil
	}
	return approvers, reviewers
}

// normalizeHandleList strips @ prefixes and whitespace from a list of handles.
func normalizeHandleList(handles []string) []string {
	var result []string
	for _, h := range handles {
		h = strings.TrimSpace(h)
		h = strings.TrimPrefix(h, "@")
		h = strings.TrimSpace(h)
		if h != "" {
			result = append(result, h)
		}
	}
	return result
}

// handlePattern matches @handle tokens (GitHub usernames: alphanumeric, hyphens, no slash).
var handlePattern = regexp.MustCompile(`@([a-zA-Z0-9](?:[a-zA-Z0-9\-]*[a-zA-Z0-9])?)`)

// githubURLPattern matches https://github.com/<username> in text.
var githubURLPattern = regexp.MustCompile(`https?://github\.com/([a-zA-Z0-9](?:[a-zA-Z0-9\-]*[a-zA-Z0-9])?)(?:\s|$|\)|,)`)

// parseMaintainersFile heuristically extracts GitHub handles from a MAINTAINERS file.
// It handles multiple common formats:
//   - @handle extraction from prose
//   - Markdown table rows with @handle
//   - "Name (@handle)" patterns
//   - GitHub profile URLs (https://github.com/username)
//
// Returns a sorted, deduplicated list of handles (without @ prefix).
func parseMaintainersFile(content string) []string {
	if content == "" {
		return nil
	}

	seen := make(map[string]bool)
	var handles []string

	addHandle := func(h string) {
		h = strings.ToLower(h)
		if h == "" || seen[h] {
			return
		}
		// Skip team references
		if strings.Contains(h, "/") {
			return
		}
		seen[h] = true
		handles = append(handles, h)
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip markdown header separator lines
		if strings.Count(line, "-") > len(line)/2 && strings.Contains(line, "|") {
			continue
		}

		// Extract @handle tokens
		matches := handlePattern.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			handle := line[m[2]:m[3]]
			fullMatchEnd := m[1]
			// Skip if followed by '/' (team reference like @org/team-name)
			if fullMatchEnd < len(line) && line[fullMatchEnd] == '/' {
				continue
			}
			// Skip if this @handle is actually part of an email (preceded by word chars)
			matchStart := m[0]
			if matchStart > 0 {
				prev := line[matchStart-1]
				if prev != ' ' && prev != '(' && prev != '|' && prev != ',' && prev != '-' && prev != '\t' {
					continue
				}
			}
			addHandle(handle)
		}

		// Extract GitHub profile URLs
		urlMatches := githubURLPattern.FindAllStringSubmatch(line, -1)
		for _, m := range urlMatches {
			addHandle(m[1])
		}
	}

	sort.Strings(handles)
	if len(handles) == 0 {
		return nil
	}
	return handles
}
