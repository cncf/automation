package projects

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := `# a comment line
export GITHUB_TOKEN=ghp_fromfile
GH_TOKEN="quoted_value"
SINGLE='single_quoted'
EMPTY=
SPACED  =  spaced_value
NO_EQUALS_LINE
ALREADY_SET=should_not_override
`
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	// Pre-set one var to prove file does not override real env.
	t.Setenv("ALREADY_SET", "real_value")

	applied, err := LoadDotEnv(envPath)
	if err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	checks := map[string]string{
		"GITHUB_TOKEN": "ghp_fromfile",
		"GH_TOKEN":     "quoted_value",
		"SINGLE":       "single_quoted",
		"EMPTY":        "",
		"SPACED":       "spaced_value",
	}
	for k, want := range checks {
		if got := os.Getenv(k); got != want {
			t.Errorf("env %s = %q, want %q", k, got, want)
		}
		// Clean up process env we set via the file.
		os.Unsetenv(k)
	}

	// Real env must win.
	if got := os.Getenv("ALREADY_SET"); got != "real_value" {
		t.Errorf("ALREADY_SET = %q, want real_value (file must not override)", got)
	}
	for _, k := range applied {
		if k == "ALREADY_SET" {
			t.Errorf("ALREADY_SET should not be reported as applied")
		}
	}
}

func TestLoadDotEnvMissingFileIsNoError(t *testing.T) {
	applied, err := LoadDotEnv(filepath.Join(t.TempDir(), "does-not-exist.env"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if applied != nil {
		t.Errorf("expected nil applied for missing file, got %v", applied)
	}
}
