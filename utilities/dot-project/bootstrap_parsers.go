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

// slackChannelURLPattern matches cloud-native.slack.com/{messages,archives,channels}/<channel> URLs.
var slackChannelURLPattern = regexp.MustCompile(`cloud-native\.slack\.com/(?:messages|archives|channels)/#?([a-zA-Z0-9][a-zA-Z0-9_-]*)`)

// slackContextPattern matches "#channel-name" near Slack-related keywords.
// Looks for lines containing "slack" (case-insensitive) with a #channel reference.
var slackContextPattern = regexp.MustCompile(`(?i)(?:slack|cncf).*?(#[a-z][a-z0-9_-]+)`)

// slackChannelLinePattern matches a line like "Channel: #channel-name" or "- #channel-name" standalone.
var slackChannelLinePattern = regexp.MustCompile(`(?i)channel\s*[:\-]\s*(#[a-z][a-z0-9_-]+)`)

// slackChannelIDPattern matches internal Slack channel IDs like "C01N7PP1THC" or "D03FAB6GN0K".
// These appear in /archives/ URLs and are not human-readable channel names.
var slackChannelIDPattern = regexp.MustCompile(`^#?[A-Z][A-Z0-9]{8,}$`)

// extractSlackChannel extracts the first CNCF Slack channel name from content.
// Convenience wrapper around extractSlackChannels.
// Returns a channel name with "#" prefix (e.g., "#tokenetes") or "" if not found.
func extractSlackChannel(content string) string {
	channels := extractSlackChannels(content)
	if len(channels) > 0 {
		return channels[0]
	}
	return ""
}

// extractSlackChannels extracts all unique CNCF Slack channel names from content.
// It looks for cloud-native.slack.com/{messages,archives,channels}/<channel> URLs,
// then #channel-name references near Slack keywords,
// then standalone "Channel: #name" patterns near Slack context.
// Returns deduplicated channel names with "#" prefix, in discovery order.
func extractSlackChannels(content string) []string {
	if content == "" {
		return nil
	}

	seen := make(map[string]bool)
	var channels []string

	addChannel := func(ch string) {
		if ch == "" || seen[ch] {
			return
		}
		// Skip internal Slack channel IDs (e.g., #C01N7PP1THC, #D03FAB6GN0K)
		if slackChannelIDPattern.MatchString(ch) {
			return
		}
		// Skip generic words that aren't real channel names
		lower := strings.ToLower(ch)
		switch lower {
		case "#slack", "#cncf", "#channel", "#channels":
			return
		}
		seen[ch] = true
		channels = append(channels, ch)
	}

	// Priority 1: cloud-native.slack.com/{messages,archives,channels}/<channel> URL
	if matches := slackChannelURLPattern.FindAllStringSubmatch(content, -1); len(matches) > 0 {
		for _, m := range matches {
			channel := strings.TrimRight(m[1], "/)")
			if channel != "" {
				addChannel("#" + channel)
			}
		}
	}

	// Priority 2: #channel-name on the same line as Slack/CNCF keywords
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if matches := slackContextPattern.FindStringSubmatch(line); len(matches) > 1 {
			addChannel(matches[1]) // already has "#" prefix
		}
	}

	// Priority 3: "Channel: #name" pattern on a line near a Slack mention (within 3 lines)
	for i, line := range lines {
		if matches := slackChannelLinePattern.FindStringSubmatch(line); len(matches) > 1 {
			start := i - 3
			if start < 0 {
				start = 0
			}
			context := strings.ToLower(strings.Join(lines[start:i+1], " "))
			if strings.Contains(context, "slack") {
				addChannel(matches[1])
			}
		}
	}

	return channels
}

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
