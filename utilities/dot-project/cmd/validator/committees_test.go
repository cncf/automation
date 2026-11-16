package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"projects"
)

func writeCommittees(t *testing.T, dir string, handles ...string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("schema_version: \"1.0.0\"\ncommittees:\n  - id: toc\n    name: TOC\n    teams:\n      - name: members\n        members:\n")
	for _, h := range handles {
		b.WriteString("          - github: " + h + "\n            elected_by: TOC\n            start: 2025-03-04\n")
	}
	path := filepath.Join(dir, projects.CommitteesFileName)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// runBinary runs the validator with a dummy token so that neither the env file
// nor the gh CLI is consulted.
func runBinary(t *testing.T, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GITHUB_TOKEN=test-token", "GITHUB_ACTIONS=")
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("run validator: %v", err)
	}
	return out.String(), errOut.String(), code
}

// The CLI cases use a placeholder handle, which the link checker reports
// without any network request, to observe whether link checking ran.
func TestCommitteesCLI(t *testing.T) {
	bin := buildValidator(t)
	dir := t.TempDir()
	valid := writeCommittees(t, dir, "TODO")

	t.Run("links are checked by default", func(t *testing.T) {
		out, _, code := runBinary(t, bin, "-committees", valid)
		if code != 0 {
			t.Fatalf("exit code %d, want 0\n%s", code, out)
		}
		if !strings.Contains(out, "Result: VALID") || !strings.Contains(out, "Link check: 1 checked") {
			t.Errorf("expected validation and link check output, got:\n%s", out)
		}
	})

	t.Run("-check-links=false opts out", func(t *testing.T) {
		out, _, code := runBinary(t, bin, "-committees", valid, "-check-links=false")
		if code != 0 || strings.Contains(out, "Link check:") {
			t.Errorf("exit %d; link check should not run, got:\n%s", code, out)
		}
	})

	t.Run("json output stays parseable with links on", func(t *testing.T) {
		out, errOut, code := runBinary(t, bin, "-committees", valid, "-output", "json")
		var res projects.CommitteesValidationResult
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("stdout is not JSON: %v\n%s", err, out)
		}
		if code != 0 || !res.Valid || res.Members != 1 {
			t.Errorf("exit %d, result %+v", code, res)
		}
		if !strings.Contains(errOut, "Link check: 1 checked") {
			t.Errorf("link summary should go to stderr, got:\n%s", errOut)
		}
	})

	t.Run("schema errors exit 1", func(t *testing.T) {
		bad := writeCommittees(t, t.TempDir(), "bad_handle")
		out, _, code := runBinary(t, bin, "-committees", bad, "-check-links=false")
		if code != 1 || !strings.Contains(out, `"bad_handle" is not a valid GitHub handle`) {
			t.Errorf("exit %d, want 1, got:\n%s", code, out)
		}
	})

	t.Run("unreadable file exits 1", func(t *testing.T) {
		_, errOut, code := runBinary(t, bin, "-committees", filepath.Join(dir, "missing.yaml"))
		if code != 1 || !strings.Contains(errOut, "failed to read committees file") {
			t.Errorf("exit %d, want 1, stderr:\n%s", code, errOut)
		}
	})
}

// fakeGitHub serves /api/users/<handle> and the github.com page /web/<handle>:
// handles in missing return 404 on both, every other handle 200. It records
// the handles looked up through the API.
func fakeGitHub(t *testing.T, missing ...string) *[]string {
	t.Helper()
	var (
		mu        sync.Mutex
		requested []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle, isAPI := strings.CutPrefix(r.URL.Path, "/api/users/")
		if !isAPI {
			handle = strings.TrimPrefix(r.URL.Path, "/web/")
		} else {
			mu.Lock()
			requested = append(requested, handle)
			mu.Unlock()
		}
		for _, m := range missing {
			if handle == m {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	orig := newLinkChecker
	newLinkChecker = func(token string) *projects.LinkChecker {
		c := projects.NewLinkChecker(token)
		c.Client = srv.Client()
		c.APIBase = srv.URL + "/api"
		c.WebBase = srv.URL + "/web"
		c.RecheckDelay = 0
		return c
	}
	t.Cleanup(func() { newLinkChecker = orig })
	return &requested
}

// distinct returns the sorted unique values of list.
func distinct(list []string) []string {
	set := make(map[string]bool)
	var out []string
	for _, v := range list {
		if !set[v] {
			set[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

func TestRunCommitteesLinkCheck(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")

	t.Run("unknown handle fails and is reported", func(t *testing.T) {
		requested := fakeGitHub(t, "ghost")
		dir := t.TempDir()
		path := writeCommittees(t, dir, "alice", "ghost")
		report := filepath.Join(dir, "report.md")
		if code := runCommittees(path, "json", &linkOptions{report: report}); code != 1 {
			t.Fatalf("exit code %d, want 1", code)
		}
		body, err := os.ReadFile(report)
		if err != nil || !strings.Contains(string(body), "ghost") {
			t.Errorf("report should name ghost: %v\n%s", err, body)
		}
		// A 404 is fetched twice so a momentary outage is not reported.
		if got := strings.Join(distinct(*requested), ","); got != "alice,ghost" {
			t.Errorf("expected alice and ghost to be looked up, got %v", *requested)
		}
	})

	t.Run("handles already in the base are not checked", func(t *testing.T) {
		requested := fakeGitHub(t, "ghost")
		baseDir, headDir := t.TempDir(), t.TempDir()
		writeCommittees(t, baseDir, "ghost")
		path := writeCommittees(t, headDir, "ghost", "alice")
		report := filepath.Join(headDir, "report.md")
		// The base may be given as a directory or as the file itself.
		for _, base := range []string{baseDir, filepath.Join(baseDir, projects.CommitteesFileName)} {
			*requested = nil
			if code := runCommittees(path, "json", &linkOptions{base: base, report: report}); code != 0 {
				t.Fatalf("base %s: exit code %d, want 0", base, code)
			}
			if strings.Join(*requested, ",") != "alice" {
				t.Errorf("base %s: only alice should be checked, got %v", base, *requested)
			}
			if body, _ := os.ReadFile(report); len(body) != 0 {
				t.Errorf("base %s: report should be empty, got:\n%s", base, body)
			}
		}
	})

	t.Run("unreadable base checks every handle", func(t *testing.T) {
		requested := fakeGitHub(t)
		path := writeCommittees(t, t.TempDir(), "alice", "bob")
		if code := runCommittees(path, "json", &linkOptions{base: filepath.Join(t.TempDir(), "nope")}); code != 0 {
			t.Fatalf("exit code %d, want 0", code)
		}
		if len(*requested) != 2 {
			t.Errorf("expected every handle to be checked, got %v", *requested)
		}
	})

	t.Run("disabled link check makes no requests", func(t *testing.T) {
		requested := fakeGitHub(t, "ghost")
		path := writeCommittees(t, t.TempDir(), "ghost")
		if code := runCommittees(path, "json", nil); code != 0 {
			t.Fatalf("exit code %d, want 0", code)
		}
		if len(*requested) != 0 {
			t.Errorf("expected no requests, got %v", *requested)
		}
	})
}

func TestGitHubToken(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GITHUB_TOKEN=from-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	origGH := ghAuthToken
	t.Cleanup(func() { ghAuthToken = origGH })

	// unset clears a variable for the rest of the test; t.Setenv restores it.
	unset := func(keys ...string) {
		for _, k := range keys {
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
	}

	cases := []struct {
		name, env, ghEnv, gh, envFile, want string
	}{
		{name: "environment wins over env file", env: "from-env", envFile: envFile, gh: "from-gh", want: "from-env"},
		{name: "env file", envFile: envFile, gh: "from-gh", want: "from-file"},
		{name: "GH_TOKEN", ghEnv: "from-gh-token", gh: "from-gh", want: "from-gh-token"},
		{name: "gh auth token fallback", gh: "from-gh", want: "from-gh"},
		{name: "no token", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unset("GITHUB_TOKEN", "GH_TOKEN")
			if tc.env != "" {
				t.Setenv("GITHUB_TOKEN", tc.env)
			}
			if tc.ghEnv != "" {
				t.Setenv("GH_TOKEN", tc.ghEnv)
			}
			ghAuthToken = func() string { return tc.gh }
			if got := githubToken(tc.envFile); got != tc.want {
				t.Errorf("githubToken = %q, want %q", got, tc.want)
			}
		})
	}
}
