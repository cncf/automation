package projects

import (
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

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
			// Strip @
			handle := strings.TrimPrefix(token, "@")
			if handle == "" {
				continue
			}
			// Skip team references (contain a slash: org/team-name)
			if strings.Contains(handle, "/") {
				continue
			}
			// Skip email-like tokens (already filtered by @ prefix check, but be safe)
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

// parseOwnersFile parses a Kubernetes-style OWNERS file (YAML with approvers/reviewers lists).
// Returns deduplicated lists of approvers and reviewers (without @ prefix, trimmed).
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
