package projects

import (
	"regexp"
	"strings"
)

// cncfArtworkPattern matches the forms a link into github.com/cncf/artwork
// takes: the tree/blob/raw pages on github.com and the raw file host, with the
// ref given either bare ("main") or as "refs/heads/main". It captures the ref
// and the path inside the repository.
var cncfArtworkPattern = regexp.MustCompile(`^https://(?:github\.com/cncf/artwork/(?:tree|blob|raw)|raw\.githubusercontent\.com/cncf/artwork)/(?:refs/heads/)?([^/?#]+)/([^?#]+?)/?$`)

// ParseCNCFArtworkURL splits a link into the cncf/artwork repository into its
// ref and repository path. ok is false for any other URL.
func ParseCNCFArtworkURL(rawURL string) (ref, path string, ok bool) {
	m := cncfArtworkPattern.FindStringSubmatch(strings.TrimSpace(rawURL))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// CanonicalArtworkURL returns the link to a project's directory in
// cncf/artwork ("https://github.com/cncf/artwork/tree/main/projects/<dir>")
// for any link that points at or below that directory, and "" otherwise.
//
// project.yaml's artwork field should carry this form: cncf/artwork is the
// reviewed source of CNCF project logos, and the directory holds every variant
// rather than one arbitrarily chosen file.
func CanonicalArtworkURL(rawURL string) string {
	_, path, ok := ParseCNCFArtworkURL(rawURL)
	if !ok {
		return ""
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "projects" || parts[1] == "" || parts[1] == ".." {
		return ""
	}
	return "https://github.com/cncf/artwork/tree/main/projects/" + parts[1]
}
