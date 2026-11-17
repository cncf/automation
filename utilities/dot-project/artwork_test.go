package projects

import "testing"

func TestCanonicalArtworkURL(t *testing.T) {
	const want = "https://github.com/cncf/artwork/tree/main/projects/demo"
	tests := map[string]string{
		"https://github.com/cncf/artwork/tree/main/projects/demo":                                                want,
		"https://github.com/cncf/artwork/tree/master/projects/demo/":                                             want,
		"https://github.com/cncf/artwork/blob/main/projects/demo/icon/color/demo-icon-color.svg":                 want,
		"https://github.com/cncf/artwork/raw/refs/heads/main/projects/demo/stacked/color/demo-stacked-color.svg": want,
		"https://raw.githubusercontent.com/cncf/artwork/main/projects/demo/icon/color/demo-icon-color.svg":       want,
		"https://raw.githubusercontent.com/cncf/artwork/refs/heads/main/projects/demo/icon/color/x.svg":          want,
		"https://github.com/cncf/artwork/blob/main/examples/graduated.md#demo-logos":                             "",
		"https://github.com/cncf/artwork/tree/main/projects/../other":                                            "",
		"https://landscape.cncf.io/logos/demo.svg":                                                               "",
		"https://github.com/someone/artwork/tree/main/projects/demo":                                             "",
		"https://demo.io/logo.svg": "",
		"":                         "",
	}
	for in, want := range tests {
		if got := CanonicalArtworkURL(in); got != want {
			t.Errorf("CanonicalArtworkURL(%q) = %q, want %q", in, got, want)
		}
	}
}
