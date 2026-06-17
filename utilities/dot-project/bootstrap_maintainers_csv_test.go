package projects

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleCSV mirrors the real foundation CSV quirks:
//   - header where col 0 is empty
//   - sparse status/project columns (forward-fill)
//   - sub-group blocks ("Kubernetes maintainers", "Envoy: Gateway (non-voting)")
//   - a bare continuation label ("Steering Committee" after Linkerd)
//   - a quoted company field containing a comma
//   - a CamelCase project name (CloudEvents) and a parenthetical (TUF)
const sampleCSV = `,Project,Maintainer Name,Company,Github Name,OWNERS/MAINTAINERS
Graduated,Kubernetes steering,Antonio Ojea,Google,aojea,https://git.k8s.io/steering#members
,,Benjamin Elder,Google,BenTheElder,
,Kubernetes maintainers,Adolfo García Veytia,"Carabiner Systems, Inc",puerco,
,,Arnaud Meukam,Independent,ameukam,
Graduated,Envoy,Alyssa Wilk,Google,alyssawilk,
,Envoy: Gateway (non-voting),Arko Dasgupta,Tetrate,arkodg,
Graduated,Linkerd,William Morgan,Buoyant,wmorgan,
,Steering Committee,Some Person,Acme,steerperson,
Graduated,CloudEvents,Doug Davis,Microsoft,duglin,
Graduated,TUF (The Update Framework),Justin Cappos,NYU,JustinCappos,
`

func parseSample(t *testing.T) []MaintainerBlock {
	t.Helper()
	blocks, err := parseFoundationMaintainersCSV(strings.NewReader(sampleCSV))
	if err != nil {
		t.Fatalf("parseFoundationMaintainersCSV() error = %v", err)
	}
	return blocks
}

func findBlock(blocks []MaintainerBlock, project string) *MaintainerBlock {
	for i := range blocks {
		if blocks[i].Project == project {
			return &blocks[i]
		}
	}
	return nil
}

func TestParseFoundationMaintainersCSV_ForwardFill(t *testing.T) {
	blocks := parseSample(t)

	steering := findBlock(blocks, "Kubernetes steering")
	if steering == nil {
		t.Fatal("expected 'Kubernetes steering' block")
	}
	if got := strings.Join(steering.Handles, ","); got != "aojea,BenTheElder" {
		t.Errorf("steering handles = %q, want aojea,BenTheElder", got)
	}
	if steering.Status != "Graduated" {
		t.Errorf("steering status = %q, want Graduated", steering.Status)
	}

	maint := findBlock(blocks, "Kubernetes maintainers")
	if maint == nil {
		t.Fatal("expected 'Kubernetes maintainers' block")
	}
	if got := strings.Join(maint.Handles, ","); got != "puerco,ameukam" {
		t.Errorf("maintainers handles = %q, want puerco,ameukam", got)
	}
}

func TestParseFoundationMaintainersCSV_QuotedCompany(t *testing.T) {
	blocks := parseSample(t)
	// The quoted "Carabiner Systems, Inc" must not shift the Github Name column.
	maint := findBlock(blocks, "Kubernetes maintainers")
	if maint == nil || len(maint.Handles) == 0 || maint.Handles[0] != "puerco" {
		t.Fatalf("expected first maintainer handle 'puerco', got %+v", maint)
	}
}

func TestParseFoundationMaintainersCSV_ContinuationLabel(t *testing.T) {
	blocks := parseSample(t)

	// A bare "Steering Committee" must merge into Linkerd, not stand alone.
	if findBlock(blocks, "Steering Committee") != nil {
		t.Error("bare 'Steering Committee' should not be a standalone block")
	}
	linkerd := findBlock(blocks, "Linkerd")
	if linkerd == nil {
		t.Fatal("expected 'Linkerd' block")
	}
	if got := strings.Join(linkerd.Handles, ","); got != "wmorgan,steerperson" {
		t.Errorf("linkerd handles = %q, want wmorgan,steerperson", got)
	}
}

func TestMatchProjectMaintainers_SubGroups(t *testing.T) {
	blocks := parseSample(t)

	// "Kubernetes" should gather both steering and maintainers sub-groups.
	got := MatchProjectMaintainers(blocks, "Kubernetes")
	want := map[string]bool{"aojea": true, "BenTheElder": true, "puerco": true, "ameukam": true}
	if len(got) != len(want) {
		t.Fatalf("Kubernetes handles = %v, want %d entries", got, len(want))
	}
	for _, h := range got {
		if !want[h] {
			t.Errorf("unexpected handle %q in Kubernetes match", h)
		}
	}
}

func TestMatchProjectMaintainers_Variants(t *testing.T) {
	blocks := parseSample(t)

	cases := []struct {
		name      string
		variants  []string
		wantFirst string
	}{
		{"camelcase", []string{"cloudevents"}, "duglin"},      // CloudEvents
		{"parenthetical", []string{"TUF"}, "JustinCappos"},    // TUF (The Update Framework)
		{"sub-group prefix", []string{"Envoy"}, "alyssawilk"}, // Envoy + Envoy: Gateway
		{"org-style hyphen", []string{"kubernetes"}, "aojea"}, // forward-filled blocks
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MatchProjectMaintainers(blocks, tc.variants...)
			if len(got) == 0 {
				t.Fatalf("no handles matched for %v", tc.variants)
			}
			if got[0] != tc.wantFirst {
				t.Errorf("first handle = %q, want %q (all: %v)", got[0], tc.wantFirst, got)
			}
		})
	}
}

func TestMatchProjectMaintainers_EnvoyGathersBoth(t *testing.T) {
	blocks := parseSample(t)
	got := MatchProjectMaintainers(blocks, "Envoy")
	want := map[string]bool{"alyssawilk": true, "arkodg": true}
	if len(got) != len(want) {
		t.Fatalf("Envoy handles = %v, want %d", got, len(want))
	}
	for _, h := range got {
		if !want[h] {
			t.Errorf("unexpected handle %q", h)
		}
	}
}

func TestMatchProjectMaintainers_NoFalsePositivePrefix(t *testing.T) {
	blocks := []MaintainerBlock{
		{Project: "ArgoCD", Handles: []string{"argoperson"}},
	}
	// "argo" must NOT match "ArgoCD" via prefix (no word boundary) or compact.
	if got := MatchProjectMaintainers(blocks, "argo"); len(got) != 0 {
		t.Errorf("expected no match for argo->ArgoCD, got %v", got)
	}
}

func TestMatchProjectMaintainers_NoMatch(t *testing.T) {
	blocks := parseSample(t)
	if got := MatchProjectMaintainers(blocks, "nonexistent-project-xyz"); got != nil {
		t.Errorf("expected nil for unknown project, got %v", got)
	}
}

func TestFetchFoundationMaintainers_LocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-maintainers.csv")
	if err := os.WriteFile(path, []byte(sampleCSV), 0o644); err != nil {
		t.Fatalf("writing temp CSV: %v", err)
	}

	blocks, err := FetchFoundationMaintainers(path, nil)
	if err != nil {
		t.Fatalf("FetchFoundationMaintainers(local) error = %v", err)
	}
	if findBlock(blocks, "Linkerd") == nil {
		t.Error("expected Linkerd block from local file")
	}
}

func TestFetchFoundationMaintainers_URL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(sampleCSV))
	}))
	defer server.Close()

	blocks, err := fetchMaintainersCSVFromURL(server.URL, server.Client())
	if err != nil {
		t.Fatalf("fetchMaintainersCSVFromURL() error = %v", err)
	}
	got := MatchProjectMaintainers(blocks, "CloudEvents")
	if len(got) != 1 || got[0] != "duglin" {
		t.Errorf("CloudEvents handles = %v, want [duglin]", got)
	}
}

func TestFetchFoundationMaintainers_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := fetchMaintainersCSVFromURL(server.URL, server.Client()); err == nil {
		t.Error("expected error on HTTP 404, got nil")
	}
}
