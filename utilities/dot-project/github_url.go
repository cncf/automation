package projects

import (
	"fmt"
	"regexp"
	"strings"
)

var githubURLRe = regexp.MustCompile(`^https?://github\.com/([^/]+)(?:/([^/]+?))?/?$`)

// ParseGitHubURL extracts org and repo slugs from a GitHub URL or a plain slug.
//
// Accepted forms:
//
//	"kubernetes"                              → org="kubernetes", repo=""
//	"https://github.com/kubernetes"          → org="kubernetes", repo=""
//	"https://github.com/kubernetes/kubernetes" → org="kubernetes", repo="kubernetes"
//
// Returns an error if the value looks like a URL but is not a valid GitHub URL.
func ParseGitHubURL(value string) (org, repo string, err error) {
	value = strings.TrimRight(value, "/")
	if m := githubURLRe.FindStringSubmatch(value); m != nil {
		return m[1], m[2], nil
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return "", "", fmt.Errorf("not a valid GitHub URL (expected https://github.com/<org>[/<repo>]): %s", value)
	}
	return value, "", nil
}
