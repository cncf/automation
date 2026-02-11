package projects

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// helpers local to this test file
// ---------------------------------------------------------------------------

// newTestValidator builds a ProjectValidator backed by a temp cache dir.
func newTestValidator(t *testing.T) *ProjectValidator {
	t.Helper()
	cacheDir := filepath.Join(t.TempDir(), "cache")
	return NewValidator(cacheDir)
}

// writeFile is a short helper that writes content to path, failing the test on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile(%s): %v", path, err)
	}
}

// validProjectYAML returns a YAML string for a minimal valid project.
func validProjectYAML() string {
	return `schema_version: "1.0.0"
slug: test-project
name: Test Project
description: A valid test project
maturity_log:
  - phase: sandbox
    date: 2024-01-15
    issue: https://github.com/cncf/toc/issues/123
repositories:
  - https://github.com/test/repo
`
}

// ---------------------------------------------------------------------------
// FormatResults
// ---------------------------------------------------------------------------

func TestFormatResults(t *testing.T) {
	pv := newTestValidator(t)

	results := []ValidationResult{
		{
			URL:         "file:///tmp/proj.yaml",
			ProjectName: "proj",
			Valid:       true,
			Changed:     false,
			CurrentHash: "abc123",
			LastChecked: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			URL:         "file:///tmp/bad.yaml",
			ProjectName: "bad",
			Valid:       false,
			Errors:      []string{"name is required"},
			Changed:     true,
			CurrentHash: "def456",
			LastChecked: time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	t.Run("json format", func(t *testing.T) {
		out, err := pv.FormatResults(results, "json")
		if err != nil {
			t.Fatalf("FormatResults json: %v", err)
		}
		// Must be valid JSON
		var parsed []ValidationResult
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}
		if len(parsed) != 2 {
			t.Fatalf("expected 2 results, got %d", len(parsed))
		}
		if parsed[0].ProjectName != "proj" {
			t.Errorf("expected project name 'proj', got %q", parsed[0].ProjectName)
		}
	})

	t.Run("yaml format", func(t *testing.T) {
		out, err := pv.FormatResults(results, "yaml")
		if err != nil {
			t.Fatalf("FormatResults yaml: %v", err)
		}
		// Must be valid YAML
		var parsed []ValidationResult
		if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("output is not valid YAML: %v", err)
		}
		if len(parsed) != 2 {
			t.Fatalf("expected 2 results, got %d", len(parsed))
		}
	})

	t.Run("text format", func(t *testing.T) {
		out, err := pv.FormatResults(results, "text")
		if err != nil {
			t.Fatalf("FormatResults text: %v", err)
		}
		if !strings.Contains(out, "Project Validation Report") {
			t.Error("text output should contain report header")
		}
		if !strings.Contains(out, "INVALID") {
			t.Error("text output should flag invalid project")
		}
	})

	t.Run("default format falls back to text", func(t *testing.T) {
		out, err := pv.FormatResults(results, "unknown-format")
		if err != nil {
			t.Fatalf("FormatResults default: %v", err)
		}
		if !strings.Contains(out, "Project Validation Report") {
			t.Error("default format should produce text report")
		}
	})

	t.Run("empty results", func(t *testing.T) {
		out, err := pv.FormatResults(nil, "json")
		if err != nil {
			t.Fatalf("FormatResults empty json: %v", err)
		}
		if out != "null" {
			t.Errorf("expected 'null' for nil slice JSON, got %q", out)
		}
	})
}

// ---------------------------------------------------------------------------
// GenerateDiff
// ---------------------------------------------------------------------------

func TestGenerateDiff(t *testing.T) {
	pv := newTestValidator(t)

	t.Run("changed project", func(t *testing.T) {
		results := []ValidationResult{
			{
				URL:          "file:///x.yaml",
				ProjectName:  "proj-a",
				Valid:        true,
				Changed:      true,
				PreviousHash: "aaa",
				CurrentHash:  "bbb",
			},
		}
		diff := pv.GenerateDiff(results)
		if !strings.Contains(diff, "CHANGED: proj-a") {
			t.Error("diff should contain CHANGED line")
		}
		if !strings.Contains(diff, "Previous Hash: aaa") {
			t.Error("diff should show previous hash")
		}
		if !strings.Contains(diff, "1 changed") {
			t.Error("summary should report 1 changed")
		}
	})

	t.Run("invalid project", func(t *testing.T) {
		results := []ValidationResult{
			{
				URL:         "file:///bad.yaml",
				ProjectName: "bad-proj",
				Valid:       false,
				Errors:      []string{"name is required", "slug is required"},
			},
		}
		diff := pv.GenerateDiff(results)
		if !strings.Contains(diff, "INVALID: bad-proj") {
			t.Error("diff should contain INVALID line")
		}
		if !strings.Contains(diff, "- name is required") {
			t.Error("diff should list error details")
		}
		if !strings.Contains(diff, "1 with errors") {
			t.Error("summary should report 1 with errors")
		}
	})

	t.Run("all valid, no changes", func(t *testing.T) {
		results := []ValidationResult{
			{URL: "file:///ok.yaml", ProjectName: "ok", Valid: true, Changed: false},
		}
		diff := pv.GenerateDiff(results)
		if strings.Contains(diff, "CHANGED") || strings.Contains(diff, "INVALID") {
			t.Error("diff should not contain CHANGED or INVALID for all-valid unchanged results")
		}
		if !strings.Contains(diff, "0 changed") {
			t.Error("summary should report 0 changed")
		}
		if !strings.Contains(diff, "0 with errors") {
			t.Error("summary should report 0 with errors")
		}
	})

	t.Run("changed and invalid", func(t *testing.T) {
		results := []ValidationResult{
			{
				URL:          "file:///x.yaml",
				ProjectName:  "proj-x",
				Valid:        false,
				Changed:      true,
				PreviousHash: "111",
				CurrentHash:  "222",
				Errors:       []string{"description is required"},
			},
		}
		diff := pv.GenerateDiff(results)
		if !strings.Contains(diff, "CHANGED: proj-x") {
			t.Error("diff should contain CHANGED")
		}
		if !strings.Contains(diff, "INVALID: proj-x") {
			t.Error("diff should contain INVALID")
		}
	})
}

// ---------------------------------------------------------------------------
// loadConfig
// ---------------------------------------------------------------------------

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		writeFile(t, cfgPath, `
project_list_url: "testdata/projectlist.yaml"
cache_dir: "/tmp/my-cache"
output_format: "yaml"
`)
		cfg, err := loadConfig(cfgPath)
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		if cfg.ProjectListURL != "testdata/projectlist.yaml" {
			t.Errorf("unexpected ProjectListURL: %q", cfg.ProjectListURL)
		}
		if cfg.CacheDir != "/tmp/my-cache" {
			t.Errorf("expected explicit cache_dir, got %q", cfg.CacheDir)
		}
		if cfg.OutputFormat != "yaml" {
			t.Errorf("expected explicit output_format, got %q", cfg.OutputFormat)
		}
	})

	t.Run("defaults applied", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		writeFile(t, cfgPath, `project_list_url: "list.yaml"`)
		cfg, err := loadConfig(cfgPath)
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		if cfg.CacheDir != ".cache" {
			t.Errorf("expected default CacheDir '.cache', got %q", cfg.CacheDir)
		}
		if cfg.OutputFormat != "json" {
			t.Errorf("expected default OutputFormat 'json', got %q", cfg.OutputFormat)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loadConfig("/nonexistent/path/config.yaml")
		if err == nil {
			t.Fatal("expected error for missing config file")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "bad.yaml")
		writeFile(t, cfgPath, `[invalid`)
		_, err := loadConfig(cfgPath)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})
}

// ---------------------------------------------------------------------------
// Cache round-trip (loadCache + save)
// ---------------------------------------------------------------------------

func TestCache(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "cache")

		cache, err := loadCache(dir)
		if err != nil {
			t.Fatalf("loadCache (new): %v", err)
		}
		if len(cache.Entries) != 0 {
			t.Fatalf("expected empty cache, got %d entries", len(cache.Entries))
		}

		// Insert an entry and save
		cache.Entries["file:///a.yaml"] = CacheEntry{
			URL:         "file:///a.yaml",
			Hash:        "abc",
			LastChecked: time.Now(),
			Content:     "hello",
		}
		if err := cache.save(); err != nil {
			t.Fatalf("cache.save: %v", err)
		}

		// Reload and verify
		cache2, err := loadCache(dir)
		if err != nil {
			t.Fatalf("loadCache (reload): %v", err)
		}
		entry, ok := cache2.Entries["file:///a.yaml"]
		if !ok {
			t.Fatal("expected entry not found after reload")
		}
		if entry.Hash != "abc" {
			t.Errorf("expected hash 'abc', got %q", entry.Hash)
		}
		if entry.Content != "hello" {
			t.Errorf("expected content 'hello', got %q", entry.Content)
		}
	})

	t.Run("missing cache file returns empty cache", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "fresh-cache")
		cache, err := loadCache(dir)
		if err != nil {
			t.Fatalf("loadCache (no file): %v", err)
		}
		if len(cache.Entries) != 0 {
			t.Fatalf("expected empty cache for missing file, got %d entries", len(cache.Entries))
		}
	})

	t.Run("corrupted cache file", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "bad-cache")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "cache.json"), `{not valid json}`)
		_, err := loadCache(dir)
		if err == nil {
			t.Fatal("expected error for corrupted cache file")
		}
	})
}

// ---------------------------------------------------------------------------
// fetchContent
// ---------------------------------------------------------------------------

func TestFetchContent(t *testing.T) {
	pv := newTestValidator(t)

	t.Run("file:// prefix", func(t *testing.T) {
		dir := t.TempDir()
		fpath := filepath.Join(dir, "project.yaml")
		writeFile(t, fpath, "hello: world")

		content, err := pv.fetchContent("file://" + fpath)
		if err != nil {
			t.Fatalf("fetchContent file://: %v", err)
		}
		if content != "hello: world" {
			t.Errorf("unexpected content: %q", content)
		}
	})

	t.Run("plain local path", func(t *testing.T) {
		dir := t.TempDir()
		fpath := filepath.Join(dir, "project.yaml")
		writeFile(t, fpath, "key: value")

		content, err := pv.fetchContent(fpath)
		if err != nil {
			t.Fatalf("fetchContent local path: %v", err)
		}
		if content != "key: value" {
			t.Errorf("unexpected content: %q", content)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := pv.fetchContent("file:///nonexistent/file.yaml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("missing file plain path", func(t *testing.T) {
		_, err := pv.fetchContent("/nonexistent/file.yaml")
		if err == nil {
			t.Fatal("expected error for missing plain path")
		}
	})
}

// ---------------------------------------------------------------------------
// loadProjectList
// ---------------------------------------------------------------------------

func TestLoadProjectList(t *testing.T) {
	t.Run("basic list", func(t *testing.T) {
		dir := t.TempDir()
		listPath := filepath.Join(dir, "projectlist.yaml")
		writeFile(t, listPath, `projects:
  - url: "file:///path/a.yaml"
  - url: "file:///path/b.yaml"
`)
		pv := newTestValidator(t)
		pv.config.ProjectListURL = listPath

		urls, err := pv.loadProjectList()
		if err != nil {
			t.Fatalf("loadProjectList: %v", err)
		}
		if len(urls) != 2 {
			t.Fatalf("expected 2 urls, got %d", len(urls))
		}
		if urls[0] != "file:///path/a.yaml" {
			t.Errorf("unexpected first url: %q", urls[0])
		}
	})

	t.Run("env var expansion", func(t *testing.T) {
		dir := t.TempDir()
		listPath := filepath.Join(dir, "projectlist.yaml")
		writeFile(t, listPath, `projects:
  - url: "${REPO_ROOT}/project.yaml"
`)
		t.Setenv("REPO_ROOT", "/my/repo")

		pv := newTestValidator(t)
		pv.config.ProjectListURL = listPath

		urls, err := pv.loadProjectList()
		if err != nil {
			t.Fatalf("loadProjectList: %v", err)
		}
		if len(urls) != 1 {
			t.Fatalf("expected 1 url, got %d", len(urls))
		}
		if urls[0] != "/my/repo/project.yaml" {
			t.Errorf("env var not expanded, got %q", urls[0])
		}
	})

	t.Run("unset env var expands to empty", func(t *testing.T) {
		dir := t.TempDir()
		listPath := filepath.Join(dir, "projectlist.yaml")
		writeFile(t, listPath, `projects:
  - url: "${UNSET_VAR_FOR_TEST}/project.yaml"
`)
		// Make sure the env var is not set
		t.Setenv("UNSET_VAR_FOR_TEST", "")

		pv := newTestValidator(t)
		pv.config.ProjectListURL = listPath

		urls, err := pv.loadProjectList()
		if err != nil {
			t.Fatalf("loadProjectList: %v", err)
		}
		if len(urls) != 1 {
			t.Fatalf("expected 1 url, got %d", len(urls))
		}
		if urls[0] != "/project.yaml" {
			t.Errorf("expected empty expansion, got %q", urls[0])
		}
	})

	t.Run("missing project list file", func(t *testing.T) {
		pv := newTestValidator(t)
		pv.config.ProjectListURL = "/nonexistent/projectlist.yaml"

		_, err := pv.loadProjectList()
		if err == nil {
			t.Fatal("expected error for missing project list file")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		listPath := filepath.Join(dir, "bad.yaml")
		writeFile(t, listPath, `:::invalid yaml`)

		pv := newTestValidator(t)
		pv.config.ProjectListURL = listPath

		_, err := pv.loadProjectList()
		if err == nil {
			t.Fatal("expected error for invalid project list YAML")
		}
	})
}

// ---------------------------------------------------------------------------
// validateProject
// ---------------------------------------------------------------------------

func TestValidateProject(t *testing.T) {
	t.Run("valid YAML project", func(t *testing.T) {
		dir := t.TempDir()
		projPath := filepath.Join(dir, "project.yaml")
		writeFile(t, projPath, validProjectYAML())

		pv := newTestValidator(t)
		result, err := pv.validateProject("file://" + projPath)
		if err != nil {
			t.Fatalf("validateProject: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected valid, got errors: %v", result.Errors)
		}
		if result.ProjectName != "Test Project" {
			t.Errorf("expected project name 'Test Project', got %q", result.ProjectName)
		}
		if result.CurrentHash == "" {
			t.Error("expected non-empty hash")
		}
		if !result.Changed {
			t.Error("expected Changed=true for new project (not in cache)")
		}
	})

	t.Run("second validation detects unchanged", func(t *testing.T) {
		dir := t.TempDir()
		projPath := filepath.Join(dir, "project.yaml")
		writeFile(t, projPath, validProjectYAML())

		pv := newTestValidator(t)
		// First validation: should be Changed=true
		_, err := pv.validateProject("file://" + projPath)
		if err != nil {
			t.Fatalf("first validateProject: %v", err)
		}
		// Second validation: same content, should be Changed=false
		result2, err := pv.validateProject("file://" + projPath)
		if err != nil {
			t.Fatalf("second validateProject: %v", err)
		}
		if result2.Changed {
			t.Error("expected Changed=false for unchanged project")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		projPath := filepath.Join(dir, "bad.yaml")
		writeFile(t, projPath, `:::not valid yaml`)

		pv := newTestValidator(t)
		result, err := pv.validateProject("file://" + projPath)
		if err != nil {
			t.Fatalf("validateProject: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid result for bad YAML")
		}
		if len(result.Errors) == 0 {
			t.Error("expected errors for bad YAML")
		}
		found := false
		for _, e := range result.Errors {
			if strings.Contains(e, "YAML parsing error") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected YAML parsing error, got: %v", result.Errors)
		}
	})

	t.Run("YAML with validation errors", func(t *testing.T) {
		dir := t.TempDir()
		projPath := filepath.Join(dir, "incomplete.yaml")
		// Valid YAML but missing required fields
		writeFile(t, projPath, `schema_version: "1.0.0"
slug: test-proj
name: ""
description: ""
maturity_log: []
repositories: []
`)
		pv := newTestValidator(t)
		result, err := pv.validateProject("file://" + projPath)
		if err != nil {
			t.Fatalf("validateProject: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid result for incomplete project")
		}
		// Should have multiple validation errors
		if len(result.Errors) < 2 {
			t.Errorf("expected multiple validation errors, got %d: %v", len(result.Errors), result.Errors)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		pv := newTestValidator(t)
		result, err := pv.validateProject("file:///nonexistent/project.yaml")
		if err != nil {
			t.Fatalf("validateProject should not return error, got: %v", err)
		}
		// The error should be captured in result.Errors, not as a returned error
		if len(result.Errors) == 0 {
			t.Error("expected errors for missing file")
		}
		found := false
		for _, e := range result.Errors {
			if strings.Contains(e, "Failed to fetch content") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'Failed to fetch content' error, got: %v", result.Errors)
		}
	})
}

// ---------------------------------------------------------------------------
// ValidateProjects / ValidateAll (end-to-end)
// ---------------------------------------------------------------------------

func TestValidateProjects(t *testing.T) {
	t.Run("end to end success", func(t *testing.T) {
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, "cache")

		// Write a valid project file
		projPath := filepath.Join(dir, "project.yaml")
		writeFile(t, projPath, validProjectYAML())

		// Write project list
		listPath := filepath.Join(dir, "projectlist.yaml")
		writeFile(t, listPath, "projects:\n  - url: \"file://"+projPath+"\"\n")

		// Write config
		cfgPath := filepath.Join(dir, "config.yaml")
		writeFile(t, cfgPath, "project_list_url: \""+listPath+"\"\ncache_dir: \""+cacheDir+"\"\noutput_format: \"json\"\n")

		pv, err := NewProjectValidator(cfgPath)
		if err != nil {
			t.Fatalf("NewProjectValidator: %v", err)
		}

		results, err := pv.ValidateProjects()
		if err != nil {
			t.Fatalf("ValidateProjects: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Valid {
			t.Errorf("expected valid result, got errors: %v", results[0].Errors)
		}

		// Cache file should now exist
		cachePath := filepath.Join(cacheDir, "cache.json")
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Error("expected cache file to be created after validation")
		}
	})

	t.Run("missing project list", func(t *testing.T) {
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, "cache")

		cfgPath := filepath.Join(dir, "config.yaml")
		writeFile(t, cfgPath, "project_list_url: \"/nonexistent/list.yaml\"\ncache_dir: \""+cacheDir+"\"\n")

		pv, err := NewProjectValidator(cfgPath)
		if err != nil {
			t.Fatalf("NewProjectValidator: %v", err)
		}

		_, err = pv.ValidateProjects()
		if err == nil {
			t.Fatal("expected error for missing project list")
		}
	})

	t.Run("ValidateAll with explicit path", func(t *testing.T) {
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, "cache")

		projPath := filepath.Join(dir, "project.yaml")
		writeFile(t, projPath, validProjectYAML())

		listPath := filepath.Join(dir, "projectlist.yaml")
		writeFile(t, listPath, "projects:\n  - url: \"file://"+projPath+"\"\n")

		pv := NewValidator(cacheDir)
		results, err := pv.ValidateAll(listPath)
		if err != nil {
			t.Fatalf("ValidateAll: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if !results[0].Valid {
			t.Errorf("expected valid, got errors: %v", results[0].Errors)
		}
	})

	t.Run("ValidateAll with empty path uses config default", func(t *testing.T) {
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, "cache")

		projPath := filepath.Join(dir, "project.yaml")
		writeFile(t, projPath, validProjectYAML())

		listPath := filepath.Join(dir, "projectlist.yaml")
		writeFile(t, listPath, "projects:\n  - url: \"file://"+projPath+"\"\n")

		pv := NewValidator(cacheDir)
		pv.config.ProjectListURL = listPath

		results, err := pv.ValidateAll("")
		if err != nil {
			t.Fatalf("ValidateAll empty path: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})

	t.Run("project with fetch error is captured in results", func(t *testing.T) {
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, "cache")

		listPath := filepath.Join(dir, "projectlist.yaml")
		writeFile(t, listPath, "projects:\n  - url: \"file:///does/not/exist.yaml\"\n")

		pv := NewValidator(cacheDir)
		results, err := pv.ValidateAll(listPath)
		if err != nil {
			t.Fatalf("ValidateAll: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Valid {
			t.Error("expected invalid result for missing file")
		}
	})
}

// ---------------------------------------------------------------------------
// NewProjectValidator
// ---------------------------------------------------------------------------

func TestNewProjectValidator(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, "cache")
		cfgPath := filepath.Join(dir, "config.yaml")
		writeFile(t, cfgPath, "project_list_url: \"list.yaml\"\ncache_dir: \""+cacheDir+"\"\n")

		pv, err := NewProjectValidator(cfgPath)
		if err != nil {
			t.Fatalf("NewProjectValidator: %v", err)
		}
		if pv == nil {
			t.Fatal("expected non-nil validator")
		}
		if pv.config == nil {
			t.Fatal("expected non-nil config")
		}
		if pv.cache == nil {
			t.Fatal("expected non-nil cache")
		}
		if pv.client == nil {
			t.Fatal("expected non-nil http client")
		}
	})

	t.Run("missing config file", func(t *testing.T) {
		_, err := NewProjectValidator("/nonexistent/config.yaml")
		if err == nil {
			t.Fatal("expected error for missing config")
		}
		if !strings.Contains(err.Error(), "failed to load config") {
			t.Errorf("expected 'failed to load config' in error, got: %v", err)
		}
	})

	t.Run("invalid config YAML", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "bad.yaml")
		writeFile(t, cfgPath, `:::bad yaml`)
		_, err := NewProjectValidator(cfgPath)
		if err == nil {
			t.Fatal("expected error for invalid config YAML")
		}
	})
}

// ---------------------------------------------------------------------------
// fetchContent – HTTP paths (success, non-200, unreachable)
// ---------------------------------------------------------------------------

func TestFetchContentHTTP(t *testing.T) {
	pv := newTestValidator(t)

	t.Run("HTTP success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("hello from server"))
		}))
		defer srv.Close()

		content, err := pv.fetchContent(srv.URL + "/project.yaml")
		if err != nil {
			t.Fatalf("fetchContent HTTP: %v", err)
		}
		if content != "hello from server" {
			t.Errorf("unexpected content: %q", content)
		}
	})

	t.Run("HTTP non-200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		_, err := pv.fetchContent(srv.URL + "/missing.yaml")
		if err == nil {
			t.Fatal("expected error for HTTP 404")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("expected 404 in error, got: %v", err)
		}
	})

	t.Run("HTTP unreachable", func(t *testing.T) {
		_, err := pv.fetchContent("http://127.0.0.1:1/unreachable")
		if err == nil {
			t.Fatal("expected error for unreachable server")
		}
	})
}

// ---------------------------------------------------------------------------
// ValidateProjectStruct – exported wrapper
// ---------------------------------------------------------------------------

func TestValidateProjectStructExported(t *testing.T) {
	t.Run("valid project", func(t *testing.T) {
		p := validBaseProject()
		errs := ValidateProjectStruct(p)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got: %v", errs)
		}
	})

	t.Run("invalid project", func(t *testing.T) {
		p := Project{}
		errs := ValidateProjectStruct(p)
		if len(errs) == 0 {
			t.Error("expected errors for empty project")
		}
	})
}

// ---------------------------------------------------------------------------
// loadProjectList – HTTP URL path
// ---------------------------------------------------------------------------

func TestLoadProjectListHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`projects:
  - url: "file:///tmp/a.yaml"
  - url: "file:///tmp/b.yaml"
`))
	}))
	defer srv.Close()

	pv := newTestValidator(t)
	pv.config.ProjectListURL = srv.URL + "/projectlist.yaml"

	urls, err := pv.loadProjectList()
	if err != nil {
		t.Fatalf("loadProjectList HTTP: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %d", len(urls))
	}
}

// ---------------------------------------------------------------------------
// ValidateProjects with HTTP-served project files
// ---------------------------------------------------------------------------

func TestValidateProjectsHTTP(t *testing.T) {
	// Serve a valid project YAML over HTTP
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(validProjectYAML()))
	}))
	defer srv.Close()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	listPath := filepath.Join(dir, "projectlist.yaml")
	writeFile(t, listPath, "projects:\n  - url: \""+srv.URL+"/project.yaml\"\n")

	pv := NewValidator(cacheDir)
	results, err := pv.ValidateAll(listPath)
	if err != nil {
		t.Fatalf("ValidateAll: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Valid {
		t.Errorf("expected valid, got errors: %v", results[0].Errors)
	}
}

// ---------------------------------------------------------------------------
// ValidateProjects – error path (project fetch error captured as result)
// ---------------------------------------------------------------------------

func TestValidateProjectsErrorCaptured(t *testing.T) {
	// Serve a 500 error for the project file
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")

	listPath := filepath.Join(dir, "projectlist.yaml")
	writeFile(t, listPath, "projects:\n  - url: \""+srv.URL+"/project.yaml\"\n")

	pv := NewValidator(cacheDir)
	results, err := pv.ValidateAll(listPath)
	if err != nil {
		t.Fatalf("ValidateAll: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Result should have errors captured
	if len(results[0].Errors) == 0 {
		t.Error("expected errors for HTTP 500 project fetch")
	}
}
