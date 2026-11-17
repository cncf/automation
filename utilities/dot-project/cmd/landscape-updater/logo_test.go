package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"projects"
)

// fakeArtwork serves files as both the raw file host and the tree listing API
// of cncf/artwork. Keys are repository paths ("projects/demo/...").
func fakeArtwork(t *testing.T, files map[string]string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is already unescaped: /api/git/trees/<ref>:<dir>
		if rest, ok := strings.CutPrefix(r.URL.Path, "/api/git/trees/"); ok {
			_, dir, _ := strings.Cut(rest, ":")
			type entry struct {
				Path string `json:"path"`
				Type string `json:"type"`
			}
			var tree []entry
			for file := range files {
				if rel, ok := strings.CutPrefix(file, dir+"/"); ok {
					tree = append(tree, entry{rel, "blob"})
				}
			}
			if tree == nil {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"tree": tree})
			return
		}
		// /raw/<ref>/<path>
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/raw/"), "/", 2)
		if len(parts) == 2 {
			if body, ok := files[parts[1]]; ok {
				_, _ = w.Write([]byte(body))
				return
			}
		}
		http.NotFound(w, r)
	}))
	previousRaw, previousAPI := artworkRawBase, artworkAPIBase
	artworkRawBase, artworkAPIBase = server.URL+"/raw", server.URL+"/api"
	t.Cleanup(func() {
		artworkRawBase, artworkAPIBase = previousRaw, previousAPI
		server.Close()
	})
}

func TestRankLogos(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{
			"conventional layout",
			[]string{"icon/color/demo-icon-color.svg", "horizontal/color/demo-horizontal-color.svg", "stacked/color/demo-stacked-color.svg", "stacked/black/demo-stacked-black.svg", "stacked/color/demo-stacked-color.png"},
			[]string{"stacked/color/demo-stacked-color.svg", "horizontal/color/demo-horizontal-color.svg", "icon/color/demo-icon-color.svg"},
		},
		{
			"capitalised directories and a -wtext variant (Copa)",
			[]string{"Stacked/Color/copa-stacked-color-wtext.svg", "Stacked/Color/copa-stacked-color.svg", "Icon/Color/copa-icon-color.svg"},
			[]string{"Stacked/Color/copa-stacked-color.svg", "Icon/Color/copa-icon-color.svg"},
		},
		{
			"underscores and reordered names (kpt, OSCAL-COMPASS)",
			[]string{"stacked/color/kpt_stacked_color.svg", "icon/color/oscal-compass-color.svg"},
			[]string{"stacked/color/kpt_stacked_color.svg", "icon/color/oscal-compass-color.svg"},
		},
		{"no colour logos", []string{"stacked/black/demo.svg", "README.md", "sub-projects/x/stacked/color/x.svg"}, nil},
	}
	for _, tt := range tests {
		if got := rankLogos(tt.files); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: rankLogos = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestLogoCandidates(t *testing.T) {
	fakeArtwork(t, map[string]string{
		"projects/demo/stacked/color/demo_stacked.svg": "x",
		"projects/demo/icon/color/demo-icon-color.svg": "x",
		"projects/demo/icon/color/demo-icon-color.png": "x",
	})
	stacked, icon := "main/projects/demo/stacked/color/demo_stacked.svg", "main/projects/demo/icon/color/demo-icon-color.svg"

	tests := []struct {
		url  string
		want []string
	}{
		{"https://github.com/cncf/artwork/tree/main/projects/demo/", []string{stacked, icon}},
		{"https://github.com/cncf/artwork/tree/master/projects/demo", []string{stacked, icon}},
		// An explicit SVG is tried first, then the directory's own logos.
		{"https://github.com/cncf/artwork/blob/main/projects/demo/icon/color/demo-icon-color.svg", []string{icon, stacked}},
		{"https://raw.githubusercontent.com/cncf/artwork/refs/heads/main/projects/demo/icon/color/gone.svg", []string{"main/projects/demo/icon/color/gone.svg", stacked, icon}},
		// A link to a PNG resolves through the project directory.
		{"https://github.com/cncf/artwork/blob/main/projects/demo/icon/color/demo-icon-color.png", []string{stacked, icon}},
		{"https://github.com/cncf/artwork/tree/main/projects/unknown", nil},
		{"https://github.com/cncf/artwork/blob/main/examples/graduated.md#demo-logos", nil},
		{"https://github.com/someone/else/tree/main/projects/demo", nil},
		{"https://github.com/cncf/artwork/tree/main/projects/../../../other/repo", nil},
		{"https://demo.io/logo.svg", nil},
		{"", nil},
	}
	for _, tt := range tests {
		got, err := logoCandidates(tt.url)
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("logoCandidates(%q) = %v, %v; want %v", tt.url, got, err, tt.want)
		}
	}
}

func TestSyncLogo(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg>`
	fakeArtwork(t, map[string]string{
		// No stacked variant: the horizontal one is the first that exists.
		"projects/demo/horizontal/color/demo-horizontal-color.svg": svg,
		"projects/demo/icon/color/demo-icon-color.svg":             `<svg xmlns="http://www.w3.org/2000/svg"><circle/></svg>`,
		"projects/evil/stacked/color/evil-stacked-color.svg":       `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`,
		"projects/html/stacked/color/html-stacked-color.svg":       `<html><body>not a logo</body></html>`,
	})

	dir := t.TempDir()
	target := filepath.Join(dir, "demo.svg")
	if err := os.WriteFile(target, []byte("<svg>old</svg>"), 0644); err != nil {
		t.Fatal(err)
	}
	artwork := func(name string) *projects.Project {
		return &projects.Project{Name: name, Artwork: "https://github.com/cncf/artwork/tree/main/projects/" + name}
	}

	changed, err := syncLogo(artwork("demo"), "demo.svg", dir)
	if err != nil || !changed {
		t.Fatalf("first sync: changed=%v err=%v, want changed with no error", changed, err)
	}
	if got, _ := os.ReadFile(target); string(got) != svg {
		t.Errorf("logo content = %q, want the horizontal artwork SVG", got)
	}

	changed, err = syncLogo(artwork("demo"), "demo.svg", dir)
	if err != nil || changed {
		t.Errorf("second sync: changed=%v err=%v, want an unchanged no-op", changed, err)
	}

	for _, name := range []string{"evil", "html", "missing"} {
		if changed, err := syncLogo(artwork(name), "demo.svg", dir); err == nil || changed {
			t.Errorf("syncLogo(%s): changed=%v err=%v, want an error and no change", name, changed, err)
		}
	}
	if changed, err := syncLogo(artwork("demo"), "../demo.svg", dir); err == nil || changed {
		t.Errorf("a logo name with a path must be rejected, got changed=%v err=%v", changed, err)
	}
	if got, _ := os.ReadFile(target); string(got) != svg {
		t.Errorf("rejected logos must not overwrite the file, got %q", got)
	}

	// Without a cncf/artwork link there is nothing to do, and no request.
	other := &projects.Project{Name: "demo", Artwork: "https://demo.io/brand"}
	if changed, err := syncLogo(other, "demo.svg", dir); err != nil || changed {
		t.Errorf("non-artwork link: changed=%v err=%v, want a silent no-op", changed, err)
	}
}

func TestValidateSVG(t *testing.T) {
	if err := validateSVG([]byte(`<?xml version="1.0"?><!-- c --><svg xmlns="http://www.w3.org/2000/svg"/>`)); err != nil {
		t.Errorf("valid SVG rejected: %v", err)
	}
	if err := validateSVG([]byte("plain text")); err == nil || !strings.Contains(err.Error(), "no <svg>") {
		t.Errorf("plain text accepted or wrong error: %v", err)
	}
}
