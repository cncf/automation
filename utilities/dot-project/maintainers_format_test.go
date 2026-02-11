package projects

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// FormatMaintainersResults
// ---------------------------------------------------------------------------

func TestFormatMaintainersResults(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	pv := NewValidator(cacheDir)

	results := []MaintainerValidationResult{
		{
			ProjectID:             "proj-a",
			Org:                   "orgA",
			Valid:                 true,
			VerificationAttempted: true,
			VerificationPassed:    true,
			VerifiedHandles:       []string{"alice", "bob"},
		},
		{
			ProjectID:             "proj-b",
			Org:                   "orgB",
			Valid:                 false,
			Errors:                []string{"project_id is required"},
			VerificationAttempted: false,
		},
	}

	t.Run("json format", func(t *testing.T) {
		out, err := pv.FormatMaintainersResults(results, "json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Must be valid JSON and round-trip to the same structure
		var decoded []MaintainerValidationResult
		if err := json.Unmarshal([]byte(out), &decoded); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}
		if len(decoded) != 2 {
			t.Fatalf("expected 2 results, got %d", len(decoded))
		}
		if decoded[0].ProjectID != "proj-a" {
			t.Errorf("expected proj-a, got %s", decoded[0].ProjectID)
		}
		if decoded[1].Valid {
			t.Errorf("expected proj-b to be invalid")
		}
	})

	t.Run("yaml format", func(t *testing.T) {
		out, err := pv.FormatMaintainersResults(results, "yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var decoded []MaintainerValidationResult
		if err := yaml.Unmarshal([]byte(out), &decoded); err != nil {
			t.Fatalf("output is not valid YAML: %v", err)
		}
		if len(decoded) != 2 {
			t.Fatalf("expected 2 results, got %d", len(decoded))
		}
		if decoded[0].ProjectID != "proj-a" {
			t.Errorf("expected proj-a, got %s", decoded[0].ProjectID)
		}
	})

	t.Run("text format", func(t *testing.T) {
		out, err := pv.FormatMaintainersResults(results, "text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, "Maintainers Validation Report") {
			t.Errorf("text output missing report header")
		}
		if !strings.Contains(out, "INVALID: proj-b") {
			t.Errorf("text output missing INVALID entry for proj-b")
		}
		if !strings.Contains(out, "Summary:") {
			t.Errorf("text output missing Summary line")
		}
	})

	t.Run("default format falls back to text", func(t *testing.T) {
		out, err := pv.FormatMaintainersResults(results, "unknown-format")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, "Maintainers Validation Report") {
			t.Errorf("default format should fall back to text")
		}
	})
}

// ---------------------------------------------------------------------------
// formatMaintainersText
// ---------------------------------------------------------------------------

func TestFormatMaintainersText(t *testing.T) {
	t.Run("mix of valid and invalid", func(t *testing.T) {
		results := []MaintainerValidationResult{
			{ProjectID: "good-project", Valid: true},
			{
				ProjectID: "bad-project",
				Valid:     false,
				Errors:    []string{"project_id is required", "team 'project-maintainers' is required"},
			},
			{ProjectID: "another-good", Valid: true},
		}

		out := formatMaintainersText(results)
		if !strings.Contains(out, "Maintainers Validation Report") {
			t.Error("missing report header")
		}
		if !strings.Contains(out, "INVALID: bad-project") {
			t.Error("missing INVALID marker for bad-project")
		}
		if strings.Contains(out, "INVALID: good-project") {
			t.Error("valid project should not appear as INVALID")
		}
		// Each error should appear as a bullet point
		if !strings.Contains(out, "  - project_id is required") {
			t.Error("missing error bullet: project_id is required")
		}
		if !strings.Contains(out, "  - team 'project-maintainers' is required") {
			t.Error("missing error bullet: team 'project-maintainers' is required")
		}
		// Summary line
		if !strings.Contains(out, "Summary: 3 maintainer entries validated, 1 with issues") {
			t.Errorf("unexpected summary line in output:\n%s", out)
		}
	})

	t.Run("all valid", func(t *testing.T) {
		results := []MaintainerValidationResult{
			{ProjectID: "alpha", Valid: true},
			{ProjectID: "beta", Valid: true},
		}
		out := formatMaintainersText(results)
		if strings.Contains(out, "INVALID") {
			t.Error("should not contain INVALID when all results are valid")
		}
		if !strings.Contains(out, "Summary: 2 maintainer entries validated, 0 with issues") {
			t.Errorf("unexpected summary line in output:\n%s", out)
		}
	})

	t.Run("empty results", func(t *testing.T) {
		out := formatMaintainersText(nil)
		if !strings.Contains(out, "Summary: 0 maintainer entries validated, 0 with issues") {
			t.Errorf("unexpected summary for empty results:\n%s", out)
		}
	})
}

// ---------------------------------------------------------------------------
// ExtractHandles
// ---------------------------------------------------------------------------

func TestExtractHandles(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	pv := NewValidator(cacheDir)

	t.Run("extracts and normalizes handles", func(t *testing.T) {
		yamlContent := `maintainers:
- project_id: "proj1"
  teams:
    - name: "project-maintainers"
      members:
        - Alice
        - "@Bob"
        - "  carol  "
    - name: "reviewers"
      members:
        - dave
        - "@Eve"
- project_id: "proj2"
  teams:
    - name: "project-maintainers"
      members:
        - "@FRANK"
        - alice
`
		path := filepath.Join(tempDir, "maintainers.yaml")
		if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		handles, err := pv.ExtractHandles(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// All handles should be lowercased with @ stripped
		expected := []string{"alice", "bob", "carol", "dave", "eve", "frank"}
		for _, h := range expected {
			if !handles[h] {
				t.Errorf("expected handle %q to be present", h)
			}
		}

		// "alice" appears twice but the map should just have one entry
		if len(handles) != len(expected) {
			t.Errorf("expected %d unique handles, got %d", len(expected), len(handles))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := pv.ExtractHandles(filepath.Join(tempDir, "nonexistent.yaml"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		badPath := filepath.Join(tempDir, "bad.yaml")
		if err := os.WriteFile(badPath, []byte("{{{{not yaml"), 0644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		_, err := pv.ExtractHandles(badPath)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})

	t.Run("empty members are skipped", func(t *testing.T) {
		yamlContent := `maintainers:
- project_id: "proj1"
  teams:
    - name: "project-maintainers"
      members:
        - ""
        - "  "
        - "@"
        - valid
`
		path := filepath.Join(tempDir, "empty-members.yaml")
		if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		handles, err := pv.ExtractHandles(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Only "valid" should survive; empty, whitespace-only, and bare "@" should be skipped
		if !handles["valid"] {
			t.Error("expected 'valid' handle")
		}
		if len(handles) != 1 {
			t.Errorf("expected 1 handle, got %d: %v", len(handles), handles)
		}
	})
}

// ---------------------------------------------------------------------------
// checkMaintainerInLFX – hardcoded URL limits direct testing, so we test
// the env-var-not-set path.
// ---------------------------------------------------------------------------

func TestCheckMaintainerInLFX(t *testing.T) {
	t.Run("returns false when LFX_AUTH_TOKEN is empty", func(t *testing.T) {
		t.Setenv("LFX_AUTH_TOKEN", "")
		result := checkMaintainerInLFX("somehandle")
		if result {
			t.Error("expected false when LFX_AUTH_TOKEN is not set")
		}
	})
}

// ---------------------------------------------------------------------------
// verifyHandleWithExternalService
// ---------------------------------------------------------------------------

func TestVerifyHandle(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	pv := NewValidator(cacheDir)

	t.Run("skip when no env vars set", func(t *testing.T) {
		t.Setenv("LFX_AUTH_TOKEN", "")
		t.Setenv("MAINTAINER_API_ENDPOINT", "")
		t.Setenv("MAINTAINER_API_STUB", "")

		err := pv.verifyHandleWithExternalService("proj", "alice")
		if err != nil {
			t.Errorf("expected nil error when no env vars set, got: %v", err)
		}
	})

	t.Run("stub success when endpoint set and stub not fail", func(t *testing.T) {
		t.Setenv("LFX_AUTH_TOKEN", "")
		t.Setenv("MAINTAINER_API_ENDPOINT", "https://example.com/api")
		t.Setenv("MAINTAINER_API_STUB", "")

		err := pv.verifyHandleWithExternalService("proj", "alice")
		if err != nil {
			t.Errorf("expected nil error for stub success, got: %v", err)
		}
	})

	t.Run("stub failure when MAINTAINER_API_STUB=fail", func(t *testing.T) {
		t.Setenv("LFX_AUTH_TOKEN", "")
		t.Setenv("MAINTAINER_API_ENDPOINT", "https://example.com/api")
		t.Setenv("MAINTAINER_API_STUB", "fail")

		err := pv.verifyHandleWithExternalService("proj", "alice")
		if err == nil {
			t.Fatal("expected error for stubbed failure")
		}
		if !strings.Contains(err.Error(), "stubbed failure") {
			t.Errorf("error should mention stubbed failure, got: %v", err)
		}
	})

	t.Run("LFX_AUTH_TOKEN set but handle not in LFX returns error", func(t *testing.T) {
		// When LFX_AUTH_TOKEN is set, it tries to call the real LFX API.
		// Since the token is invalid, the API will return non-200, so
		// checkMaintainerInLFX returns false, and we get an error.
		t.Setenv("LFX_AUTH_TOKEN", "invalid-token-for-test")
		t.Setenv("MAINTAINER_API_ENDPOINT", "")
		t.Setenv("MAINTAINER_API_STUB", "")

		err := pv.verifyHandleWithExternalService("proj", "nonexistent-handle")
		if err == nil {
			t.Fatal("expected error when LFX returns false")
		}
		if !strings.Contains(err.Error(), "not found in LFX") {
			t.Errorf("expected 'not found in LFX' error, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ValidateMaintainersFileWithExclusion – test exclusion map behaviour
// ---------------------------------------------------------------------------

func TestValidateMaintainersWithExclusion(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	pv := NewValidator(cacheDir)

	yamlContent := `maintainers:
- project_id: "test-proj"
  teams:
    - name: "project-maintainers"
      members:
        - alice
        - bob
        - carol
`
	path := filepath.Join(tempDir, "maintainers.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Use stub endpoint so verification runs but does not call real API
	t.Setenv("LFX_AUTH_TOKEN", "")
	t.Setenv("MAINTAINER_API_ENDPOINT", "https://example.com/stub")
	t.Setenv("MAINTAINER_API_STUB", "")

	t.Run("no exclusions: all handles verified", func(t *testing.T) {
		results, err := pv.ValidateMaintainersFileWithExclusion(path, true, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		res := results[0]
		if !res.VerificationAttempted {
			t.Fatal("expected verification to be attempted")
		}
		if len(res.VerifiedHandles) != 3 {
			t.Errorf("expected 3 verified handles, got %d: %v", len(res.VerifiedHandles), res.VerifiedHandles)
		}
	})

	t.Run("exclude some handles", func(t *testing.T) {
		excluded := map[string]bool{
			"alice": true,
			"carol": true,
		}
		results, err := pv.ValidateMaintainersFileWithExclusion(path, true, excluded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		res := results[0]
		if !res.VerificationAttempted {
			t.Fatal("expected verification to be attempted")
		}
		// Only "bob" should be verified; alice and carol are excluded
		if len(res.VerifiedHandles) != 1 {
			t.Errorf("expected 1 verified handle, got %d: %v", len(res.VerifiedHandles), res.VerifiedHandles)
		}
		if len(res.VerifiedHandles) > 0 && res.VerifiedHandles[0] != "bob" {
			t.Errorf("expected 'bob' as the only verified handle, got %v", res.VerifiedHandles)
		}
	})

	t.Run("exclude all handles: still verification attempted", func(t *testing.T) {
		excluded := map[string]bool{
			"alice": true,
			"bob":   true,
			"carol": true,
		}
		results, err := pv.ValidateMaintainersFileWithExclusion(path, true, excluded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		res := results[0]
		if !res.VerificationAttempted {
			t.Fatal("expected verification to be attempted")
		}
		if len(res.VerifiedHandles) != 0 {
			t.Errorf("expected 0 verified handles, got %d", len(res.VerifiedHandles))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := pv.ValidateMaintainersFileWithExclusion(filepath.Join(tempDir, "nope.yaml"), false, nil)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		badPath := filepath.Join(tempDir, "bad-maintainers.yaml")
		if err := os.WriteFile(badPath, []byte("{{bad yaml"), 0644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		_, err := pv.ValidateMaintainersFileWithExclusion(badPath, false, nil)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})

	t.Run("empty maintainers list", func(t *testing.T) {
		emptyPath := filepath.Join(tempDir, "empty-maintainers.yaml")
		if err := os.WriteFile(emptyPath, []byte("maintainers: []\n"), 0644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		_, err := pv.ValidateMaintainersFileWithExclusion(emptyPath, false, nil)
		if err == nil {
			t.Fatal("expected error for empty maintainers list")
		}
		if !strings.Contains(err.Error(), "does not contain any entries") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// validateMaintainerEntry – cover missing project_id, empty teams, no
// project-maintainers team, and exclusion within the entry.
// ---------------------------------------------------------------------------

func TestValidateMaintainerEntry(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	pv := NewValidator(cacheDir)

	// Ensure stub verification so we control outcomes
	t.Setenv("LFX_AUTH_TOKEN", "")
	t.Setenv("MAINTAINER_API_ENDPOINT", "")
	t.Setenv("MAINTAINER_API_STUB", "")

	t.Run("empty project_id", func(t *testing.T) {
		entry := MaintainerEntry{
			ProjectID: "",
			Teams: []Team{
				{Name: "project-maintainers", Members: []string{"alice"}},
			},
		}
		result := pv.validateMaintainerEntry(entry, false, nil)
		if result.Valid {
			t.Fatal("expected invalid result for empty project_id")
		}
		found := false
		for _, e := range result.Errors {
			if strings.Contains(e, "project_id is required") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'project_id is required' error, got: %v", result.Errors)
		}
	})

	t.Run("empty teams list", func(t *testing.T) {
		entry := MaintainerEntry{
			ProjectID: "proj",
			Teams:     []Team{},
		}
		result := pv.validateMaintainerEntry(entry, false, nil)
		if result.Valid {
			t.Fatal("expected invalid result for empty teams")
		}
		foundEmpty := false
		foundMissing := false
		for _, e := range result.Errors {
			if strings.Contains(e, "teams list cannot be empty") {
				foundEmpty = true
			}
			if strings.Contains(e, "team 'project-maintainers' is required") {
				foundMissing = true
			}
		}
		if !foundEmpty {
			t.Errorf("expected 'teams list cannot be empty' error, got: %v", result.Errors)
		}
		if !foundMissing {
			t.Errorf("expected 'project-maintainers is required' error, got: %v", result.Errors)
		}
	})

	t.Run("no project-maintainers team", func(t *testing.T) {
		entry := MaintainerEntry{
			ProjectID: "proj",
			Teams: []Team{
				{Name: "reviewers", Members: []string{"alice"}},
			},
		}
		result := pv.validateMaintainerEntry(entry, false, nil)
		if result.Valid {
			t.Fatal("expected invalid when project-maintainers team is missing")
		}
		found := false
		for _, e := range result.Errors {
			if strings.Contains(e, "team 'project-maintainers' is required") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'project-maintainers is required' error, got: %v", result.Errors)
		}
	})

	t.Run("empty project-maintainers team", func(t *testing.T) {
		entry := MaintainerEntry{
			ProjectID: "proj",
			Teams: []Team{
				{Name: "project-maintainers", Members: []string{}},
			},
		}
		result := pv.validateMaintainerEntry(entry, false, nil)
		if result.Valid {
			t.Fatal("expected invalid when project-maintainers has no members")
		}
		found := false
		for _, e := range result.Errors {
			if strings.Contains(e, "team 'project-maintainers' cannot be empty") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'project-maintainers cannot be empty' error, got: %v", result.Errors)
		}
	})

	t.Run("valid entry without verification", func(t *testing.T) {
		entry := MaintainerEntry{
			ProjectID: "proj",
			Teams: []Team{
				{Name: "project-maintainers", Members: []string{"alice", "bob"}},
			},
		}
		result := pv.validateMaintainerEntry(entry, false, nil)
		if !result.Valid {
			t.Errorf("expected valid result, got errors: %v", result.Errors)
		}
		if result.VerificationAttempted {
			t.Error("verification should not be attempted when verify=false")
		}
	})

	t.Run("verification with exclusion skips excluded handles", func(t *testing.T) {
		t.Setenv("LFX_AUTH_TOKEN", "")
		t.Setenv("MAINTAINER_API_ENDPOINT", "https://example.com/api")
		t.Setenv("MAINTAINER_API_STUB", "")

		entry := MaintainerEntry{
			ProjectID: "proj",
			Teams: []Team{
				{Name: "project-maintainers", Members: []string{"alice", "bob", "carol"}},
			},
		}
		excluded := map[string]bool{"alice": true, "carol": true}
		result := pv.validateMaintainerEntry(entry, true, excluded)
		if !result.VerificationAttempted {
			t.Fatal("expected verification to be attempted")
		}
		// Only "bob" should be in verified handles
		if len(result.VerifiedHandles) != 1 {
			t.Errorf("expected 1 verified handle, got %d: %v", len(result.VerifiedHandles), result.VerifiedHandles)
		}
	})

	t.Run("verification failure marks result as failed", func(t *testing.T) {
		t.Setenv("LFX_AUTH_TOKEN", "")
		t.Setenv("MAINTAINER_API_ENDPOINT", "https://example.com/api")
		t.Setenv("MAINTAINER_API_STUB", "fail")

		entry := MaintainerEntry{
			ProjectID: "proj",
			Teams: []Team{
				{Name: "project-maintainers", Members: []string{"alice"}},
			},
		}
		result := pv.validateMaintainerEntry(entry, true, nil)
		if result.VerificationPassed {
			t.Error("expected verification to fail with stubbed failure")
		}
		if result.Valid {
			t.Error("expected invalid result when verification fails")
		}
	})
}
