package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"projects"
)

// buildValidator compiles this package once per test binary so the tests can
// drive the real command line, flags and exit code.
func buildValidator(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "validator")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

// copyFixture copies the multi-project testdata repo into a scratch dir so a
// test can corrupt one project without touching the fixture.
func copyFixture(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "multiproject")
	dst := filepath.Join(t.TempDir(), "repo")
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copying fixture: %v", err)
	}
	return dst
}

func TestRepoRootReportsEmptyRosterAndValidSibling(t *testing.T) {
	bin := buildValidator(t)
	repo := copyFixture(t)

	if err := os.WriteFile(filepath.Join(repo, "alpha", "maintainers.yaml"), []byte("maintainers: []\n"), 0o644); err != nil {
		t.Fatalf("writing empty roster: %v", err)
	}

	cmd := exec.Command(bin, "-repo-root", ".", "-config=/dev/null", "-cache", t.TempDir(), "-output", "json")
	cmd.Dir = repo
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit status 1, got err=%v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if code := exitErr.ExitCode(); code != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout:\n%s", code, stdout.String())
	}

	// The command prints three JSON documents in -output json mode: the repo
	// structure result, the project results, then the maintainers results.
	dec := json.NewDecoder(strings.NewReader(stdout.String()))
	var repoResult projects.RepoValidationResult
	if err := dec.Decode(&repoResult); err != nil {
		t.Fatalf("decoding repo result: %v\nstdout:\n%s", err, stdout.String())
	}
	var projectResults json.RawMessage
	if err := dec.Decode(&projectResults); err != nil {
		t.Fatalf("decoding project results: %v", err)
	}
	var maintainerResults []projects.MaintainerValidationResult
	if err := dec.Decode(&maintainerResults); err != nil {
		t.Fatalf("decoding maintainer results: %v\nstdout:\n%s", err, stdout.String())
	}

	if repoResult.Valid {
		t.Errorf("repo result should be invalid, errors: %v", repoResult.Errors)
	}

	byID := make(map[string]projects.MaintainerValidationResult, len(maintainerResults))
	for _, r := range maintainerResults {
		byID[r.ProjectID] = r
	}

	alpha, ok := byID["alpha/maintainers.yaml"]
	if !ok {
		t.Fatalf("empty roster missing from maintainers report: %+v", maintainerResults)
	}
	if alpha.Valid || !strings.Contains(strings.Join(alpha.Errors, "\n"), "does not contain any entries") {
		t.Errorf("empty roster should be reported invalid with an entries error, got %+v", alpha)
	}

	beta, ok := byID["beta"]
	if !ok {
		t.Fatalf("valid sibling missing from maintainers report: %+v", maintainerResults)
	}
	if !beta.Valid {
		t.Errorf("valid sibling should still be reported valid, got %+v", beta)
	}
}
