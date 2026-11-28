package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"projects"
)

// artworkRawBase is where files of the cncf/artwork repository are fetched
// from. It is a variable so tests can point it at a local server.
var artworkRawBase = "https://raw.githubusercontent.com/cncf/artwork"

// maxLogoBytes bounds a downloaded logo; landscape logos are small SVGs.
const maxLogoBytes = 2 << 20

// artworkAPIBase is the GitHub API root of the cncf/artwork repository, used
// to list a project's directory. It is a variable for the same reason.
var artworkAPIBase = "https://api.github.com/repos/cncf/artwork"

// logoCandidates returns the cncf/artwork files ("<ref>/<path>") that may hold
// the project's logo, most preferred first. A link to an .svg file comes
// first; it is followed by the logos found by listing the project's directory,
// which is what a directory link, a link to a non-SVG file, or a stale file
// link resolves through.
func logoCandidates(artworkURL string) ([]string, error) {
	// Only cncf/artwork is accepted: its contents are reviewed by CNCF staff,
	// whereas an arbitrary URL in project.yaml is not.
	ref, p, ok := projects.ParseCNCFArtworkURL(artworkURL)
	if !ok {
		return nil, nil
	}
	p = path.Clean(p)
	if strings.HasPrefix(p, "..") {
		return nil, nil
	}
	// cncf/artwork renamed its default branch; old links still say master,
	// which the raw file host redirects but the listing API does not.
	if ref == "master" {
		ref = "main"
	}

	var candidates []string
	if strings.HasSuffix(strings.ToLower(p), ".svg") {
		candidates = append(candidates, ref+"/"+p)
	}

	parts := strings.Split(p, "/")
	if len(parts) < 2 || parts[0] != "projects" {
		return candidates, nil
	}
	dir := "projects/" + parts[1]
	files, err := listArtworkDir(ref, dir)
	if err != nil {
		return nil, err
	}
	for _, file := range rankLogos(files) {
		candidate := ref + "/" + dir + "/" + file
		if len(candidates) == 0 || candidates[0] != candidate {
			candidates = append(candidates, candidate)
		}
	}
	return candidates, nil
}

// rankLogos picks the colour SVG of each layout from a project directory
// listing, preferring the layouts that include the project name, as the
// landscape guidelines ask. Directory and file naming in cncf/artwork is only
// loosely conventional (Stacked/Color, name_stacked_color.svg, ...), so the
// layout and colour are read from the directory names, case-insensitively,
// rather than assumed from a file name pattern. Within a directory the
// shortest name wins, which skips variants such as "-wtext".
func rankLogos(files []string) []string {
	var ranked []string
	for _, layout := range []string{"stacked", "horizontal", "icon"} {
		best := ""
		for _, file := range files {
			lower := strings.ToLower(file)
			if !strings.HasSuffix(lower, ".svg") {
				continue
			}
			dirs := strings.Split(path.Dir(lower), "/")
			if len(dirs) != 2 || dirs[0] != layout || dirs[1] != "color" {
				continue
			}
			if best == "" || len(file) < len(best) || (len(file) == len(best) && file < best) {
				best = file
			}
		}
		if best != "" {
			ranked = append(ranked, best)
		}
	}
	return ranked
}

// listArtworkDir returns the files below dir in cncf/artwork at ref, relative
// to dir. A directory that does not exist yields an empty list.
func listArtworkDir(ref, dir string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/git/trees/%s?recursive=1", artworkAPIBase, url.PathEscape(ref+":"+dir))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// Optional: anonymous API calls are limited to 60 per hour.
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := os.Getenv(name); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			break
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s in cncf/artwork: %w", dir, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("listing %s in cncf/artwork returned %s", dir, resp.Status)
	}

	var listing struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&listing); err != nil {
		return nil, fmt.Errorf("failed to decode the cncf/artwork listing of %s: %w", dir, err)
	}
	var files []string
	for _, entry := range listing.Tree {
		if entry.Type == "blob" {
			files = append(files, entry.Path)
		}
	}
	return files, nil
}

// syncLogo replaces <logosDir>/<logoName> with the project's logo from
// cncf/artwork when the two differ. It reports whether the file was rewritten.
// A project without a usable cncf/artwork link is left alone.
func syncLogo(project *projects.Project, logoName, logosDir string) (bool, error) {
	if logoName == "" {
		return false, nil
	}
	candidates, err := logoCandidates(project.Artwork)
	if err != nil {
		return false, err
	}
	if _, _, ok := projects.ParseCNCFArtworkURL(project.Artwork); !ok {
		return false, nil
	}
	if len(candidates) == 0 {
		return false, fmt.Errorf("no logo found in cncf/artwork for %s", project.Artwork)
	}
	if logoName != filepath.Base(logoName) || !strings.HasSuffix(strings.ToLower(logoName), ".svg") {
		return false, fmt.Errorf("landscape logo %q is not a plain .svg file name", logoName)
	}

	var logo []byte
	for _, candidate := range candidates {
		data, found, err := fetchLogo(artworkRawBase + "/" + candidate)
		if err != nil {
			return false, err
		}
		if found {
			logo = data
			break
		}
	}
	if logo == nil {
		return false, fmt.Errorf("no logo found in cncf/artwork for %s", project.Artwork)
	}
	if err := validateSVG(logo); err != nil {
		return false, err
	}

	target := filepath.Join(logosDir, logoName)
	current, err := os.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to read %s: %w", target, err)
	}
	if err == nil && bytes.Equal(current, logo) {
		return false, nil
	}
	if err := os.WriteFile(target, logo, 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", target, err)
	}
	return true, nil
}

// fetchLogo downloads url. A 404 is reported as not found rather than an
// error so the caller can try the next candidate.
func fetchLogo(url string) ([]byte, bool, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("fetching %s returned %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxLogoBytes+1))
	if err != nil {
		return nil, false, fmt.Errorf("failed to read %s: %w", url, err)
	}
	if len(data) > maxLogoBytes {
		return nil, false, fmt.Errorf("%s is larger than %d bytes", url, maxLogoBytes)
	}
	return data, true, nil
}

// validateSVG checks that data is a well-formed SVG document without script
// elements, since the landscape serves these files to browsers.
func validateSVG(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	sawRoot := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("logo is not well-formed XML: %w", err)
		}
		element, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(element.Name.Local)
		if !sawRoot {
			if name != "svg" {
				return fmt.Errorf("logo root element is <%s>, not <svg>", element.Name.Local)
			}
			sawRoot = true
		}
		if name == "script" {
			return fmt.Errorf("logo contains a <script> element")
		}
	}
	if !sawRoot {
		return fmt.Errorf("logo contains no <svg> element")
	}
	return nil
}
