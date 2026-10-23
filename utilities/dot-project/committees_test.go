package projects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

var committeesTestNow = time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)

func writeCommitteesFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), CommitteesFileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write committees file: %v", err)
	}
	return path
}

func validateCommitteesYAML(t *testing.T, content string) CommitteesValidationResult {
	t.Helper()
	res, err := ValidateCommitteesFile(writeCommitteesFile(t, content), committeesTestNow)
	if err != nil {
		t.Fatalf("ValidateCommitteesFile: %v", err)
	}
	return res
}

func anyContains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

const validCommitteesHeader = `schema_version: "1.0.0"
org: cncf
committees:
`

func TestExampleCommitteesFileIsValid(t *testing.T) {
	res, err := ValidateCommitteesFile(filepath.Join("example", CommitteesFileName), committeesTestNow)
	if err != nil {
		t.Fatalf("ValidateCommitteesFile: %v", err)
	}
	if !res.Valid {
		t.Fatalf("example committees.yaml should be valid, got errors: %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("example committees.yaml should have no warnings, got: %v", res.Warnings)
	}
	if res.Committees != 2 {
		t.Errorf("expected 2 committees, got %d", res.Committees)
	}
}

func TestCommitteesValidation(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantErrors []string
		wantWarns  []string
		wantValid  bool
	}{
		{
			name: "minimal valid",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: Governing Board
            start: 2025-03-04
            end: 2027-03-04
`,
			wantValid: true,
		},
		{
			name: "quoted dates are accepted",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: Governing Board
            start: "2025-03-04"
            end: "2027-03-04"
`,
			wantValid: true,
		},
		{
			name: "bare handle is rejected",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - alice
`,
			wantErrors: []string{`member "alice" must be a mapping`},
		},
		{
			name: "required member fields",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - role: chair
`,
			wantErrors: []string{"github is required", "elected_by is required", "start is required"},
		},
		{
			name: "name and company are rejected; they come from LFID",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            name: Alice Example
            company: Example Corp
            elected_by: TOC
            start: 2025-03-04
`,
			wantErrors: []string{`unknown field "name"`, `unknown field "company"`},
		},
		{
			name: "role is optional and must be a slug",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            role: chair
            elected_by: TOC
            start: 2025-03-04
          - github: bob
            role: Vice Chair
            elected_by: TOC
            start: 2025-03-04
          - github: carol
            elected_by: TOC
            start: 2025-03-04
`,
			wantErrors: []string{`members[bob]: role "Vice Chair" must be lowercase`},
		},
		{
			name: "liaison lists other committees, even ones defined later",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            role: chair
            liaison: [tag-a, tag-b]
            elected_by: TOC
            start: 2025-03-04
          - github: bob
            liaison: ~
            elected_by: TOC
            start: 2025-03-04
  - id: tag-a
    name: TAG A
    teams:
      - name: chairs
        members:
          - github: carol
            elected_by: TOC
            start: 2025-03-04
  - id: tag-b
    name: TAG B
    teams:
      - name: chairs
        members:
          - github: dave
            elected_by: TOC
            start: 2025-03-04
`,
			wantValid: true,
		},
		{
			name: "liaison entries must be distinct ids of other committees",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            liaison: [tag-a, tag-a, tag-missing, toc]
            elected_by: TOC
            start: 2025-03-04
  - id: tag-a
    name: TAG A
    teams:
      - name: chairs
        members:
          - github: carol
            elected_by: TOC
            start: 2025-03-04
`,
			wantErrors: []string{
				`members[alice]: liaison "tag-a" is listed more than once`,
				`members[alice]: liaison "tag-missing" is not a committee id in this file`,
				`members[alice]: liaison "toc" is the member's own committee`,
			},
		},
		{
			name: "liaison must be a non-empty list of single values",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            liaison: tag-a
            elected_by: TOC
            start: 2025-03-04
          - github: bob
            liaison: []
            elected_by: TOC
            start: 2025-03-04
          - github: carol
            liaison: [[tag-a]]
            elected_by: TOC
            start: 2025-03-04
          - github: dave
            liaison: [tag-a]
            liaison: [tag-a]
            elected_by: TOC
            start: 2025-03-04
`,
			wantErrors: []string{
				`members[alice]: line 11: field "liaison" must be a list of committee ids`,
				`members[bob]: line 15: field "liaison" cannot be an empty list`,
				`members[carol]: line 19: liaison entries must be committee ids`,
				`members[dave]: line 24: field "liaison" is set more than once`,
			},
		},
		{
			name: "unknown member field such as email is rejected",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            email: alice@example.com
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
`,
			wantErrors: []string{`unknown field "email"`},
		},
		{
			name: "end is optional",
			yaml: validCommitteesHeader + `
  - id: staff
    name: Staff
    teams:
      - name: members
        members:
          - github: alice
            elected_by: Hired
            start: 2025-03-04
          - github: bob
            elected_by: Hired
            start: 2025-03-04
            end: 2099-01-01
`,
			wantValid: true,
		},
		{
			name: "emeritus members need an end",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: bob
            elected_by: TOC
            start: 2025-03-04
      - name: emeritus
        emeritus: true
        members:
          - github: alice
            elected_by: TOC
            start: 2020-03-04
`,
			wantErrors: []string{"members[alice]: end is required for emeritus members"},
		},
		{
			name: "bad dates and ordering",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 3/4/2025
            end: 2027-03-04
          - github: bob
            elected_by: TOC
            start: 2027-03-04
            end: 2025-03-04
`,
			wantErrors: []string{`start "3/4/2025" must be a YYYY-MM-DD date`, "end 2025-03-04 must be after start 2027-03-04"},
		},
		{
			name: "handle format and duplicates",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: "@alice"
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
          - github: Bob
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
          - github: bob
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
          - github: not/valid
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
`,
			wantErrors: []string{"must not start with @", "members[bob]: duplicate handle in team with overlapping terms", `"not/valid" is not a valid GitHub handle`},
		},
		{
			name: "repeat handle with disjoint terms records re-election",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: carol
            elected_by: TOC
            start: 2027-03-04
            end: 2029-03-04
      - name: emeritus
        emeritus: true
        members:
          - github: carol
            elected_by: TOC
            start: 2020-02-04
            end: 2020-11-02
          - github: Carol
            elected_by: GB
            start: 2022-03-18
            end: 2026-02-04
`,
			wantValid: true,
		},
		{
			name: "repeat handle with no-end overlap is rejected",
			yaml: validCommitteesHeader + `
  - id: staff
    name: Staff
    teams:
      - name: staff
        members:
          - github: dana
            elected_by: CNCF
            start: 2020-01-01
          - github: dana
            elected_by: CNCF
            start: 2024-01-01
            end: 2025-01-01
`,
			wantErrors: []string{"members[dana]: duplicate handle in team with overlapping terms"},
		},
		{
			name: "same person in two teams is fine",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: GB
            start: 2025-03-04
            end: 2027-03-04
      - name: liaisons
        members:
          - github: alice
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
`,
			wantValid: true,
		},
		{
			name: "committee level checks",
			yaml: validCommitteesHeader + `
  - id: TOC
    name: ""
    url: not-a-url
    teams: []
  - id: TOC
    name: dup
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
`,
			wantErrors: []string{
				"id must be lowercase",
				"name is required",
				`url "not-a-url" is not a valid`,
				"teams list cannot be empty",
				"duplicate id",
			},
		},
		{
			name: "team level checks",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: Members
        github_teams: [Bad_Team, ok-team, ok-team]
        members:
          - github: alice
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
      - name: Members
        members: []
`,
			wantErrors: []string{
				"name must be lowercase",
				`github_teams entry "Bad_Team"`,
				`github_teams entry "ok-team" is listed more than once`,
				"duplicate team name",
			},
		},
		{
			name: "emeritus constraints",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: emeritus
        emeritus: true
        github_teams: [cncf-toc]
        members:
          - github: alice
            elected_by: TOC
            start: 2020-03-04
            end: 2022-03-04
`,
			wantErrors: []string{
				"emeritus teams cannot grant github_teams",
				"at least one team must be non-emeritus",
			},
		},
		{
			name: "empty or missing members list is rejected",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2025-03-04
      - name: alternates
        members: []
      - name: liaisons
`,
			wantErrors: []string{
				"teams[alternates]: members list cannot be empty",
				"teams[liaisons]: members list cannot be empty",
			},
		},
		{
			name: "expired term and future emeritus end are warnings only",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2023-03-04
            end: 2025-03-04
      - name: emeritus
        emeritus: true
        members:
          - github: bob
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
`,
			wantValid: true,
			wantWarns: []string{
				"term ended on 2025-03-04",
				"emeritus member's term ends in the future",
			},
		},
		{
			name: "term ending today is not expired",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2024-10-21
            end: 2026-10-21
`,
			wantValid: true,
		},
		{
			name: "impossible calendar dates and timestamps are rejected",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2025-02-30
          - github: bob
            elected_by: TOC
            start: 2025-03-04T10:00:00Z
            end: 2027-13-01
`,
			wantErrors: []string{
				`start "2025-02-30" must be a YYYY-MM-DD date`,
				`start "2025-03-04T10:00:00Z" must be a YYYY-MM-DD date`,
				`end "2027-13-01" must be a YYYY-MM-DD date`,
			},
		},
		{
			name: "back-to-back terms do not overlap",
			yaml: validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2027-03-04
          - github: alice
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
      - name: emeritus
        emeritus: true
        members:
          - github: bob
            elected_by: TOC
            start: 2020-03-04
            end: 2022-03-04
`,
			wantValid: true,
		},
		{
			name: "schema version",
			yaml: `schema_version: "9.9.9"
committees:
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2025-03-04
            end: 2027-03-04
`,
			wantErrors: []string{`schema_version "9.9.9" is not supported`},
		},
		{
			name:       "empty committees",
			yaml:       `schema_version: "1.0.0"` + "\ncommittees: []\n",
			wantErrors: []string{"committees list cannot be empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := validateCommitteesYAML(t, tt.yaml)
			if tt.wantValid != res.Valid {
				t.Fatalf("Valid = %v, want %v; errors: %v", res.Valid, tt.wantValid, res.Errors)
			}
			for _, want := range tt.wantErrors {
				if !anyContains(res.Errors, want) {
					t.Errorf("missing error containing %q; got %v", want, res.Errors)
				}
			}
			for _, want := range tt.wantWarns {
				if !anyContains(res.Warnings, want) {
					t.Errorf("missing warning containing %q; got %v", want, res.Warnings)
				}
			}
			if tt.wantValid && len(tt.wantWarns) == 0 && len(res.Warnings) > 0 {
				t.Errorf("unexpected warnings: %v", res.Warnings)
			}
		})
	}
}

func TestCommitteesBareHandleReportsOnlyOneError(t *testing.T) {
	res := validateCommitteesYAML(t, validCommitteesHeader+`
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - alice
          - bob
`)
	if len(res.Errors) != 2 {
		t.Fatalf("expected exactly one error per bare handle, got %d: %v", len(res.Errors), res.Errors)
	}
}

func TestCommitteesUnknownFieldsAreParseErrors(t *testing.T) {
	cases := map[string]string{
		"nam": validCommitteesHeader + "  - id: toc\n    nam: TOC\n",
		"votng": validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        votng: true
`,
		"organisation": `schema_version: "1.0.0"` + "\norganisation: cncf\n",
	}
	for field, content := range cases {
		_, err := ValidateCommitteesFile(writeCommitteesFile(t, content), committeesTestNow)
		if err == nil || !strings.Contains(err.Error(), field) {
			t.Errorf("expected unknown field error for %q, got %v", field, err)
		}
	}
}

func TestCommitteesEmptyFile(t *testing.T) {
	_, err := ValidateCommitteesFile(writeCommitteesFile(t, ""), committeesTestNow)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty file error, got %v", err)
	}
}

// A field that is already reported as malformed must not be reported again
// as missing, and an inverted term must not also be reported as expired.
func TestCommitteesReportEachProblemOnce(t *testing.T) {
	res := validateCommitteesYAML(t, validCommitteesHeader+`
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: [TOC]
            start: 2025-03-04
          - github: bob
            elected_by: TOC
            start: {date: 2025-03-04}
          - github: carol
            elected_by: TOC
            start: 2024-03-04
            end: 2024-03-04
      - name: emeritus
        emeritus: true
        members:
          - github: dave
            elected_by: TOC
            start: 2020-03-04
            end: [2022-03-04]
`)
	want := []string{
		`members[alice]: line 11: field "elected_by" must be a single value`,
		`members[bob]: line 15: field "start" must be a single value`,
		"members[carol]: end 2024-03-04 must be after start 2024-03-04",
		`members[dave]: line 26: field "end" must be a single value`,
	}
	if len(res.Errors) != len(want) {
		t.Fatalf("expected %d errors, got %d: %v", len(want), len(res.Errors), res.Errors)
	}
	for _, w := range want {
		if !anyContains(res.Errors, w) {
			t.Errorf("missing error containing %q; got %v", w, res.Errors)
		}
	}
	if len(res.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", res.Warnings)
	}
}

func TestCommitteesSecondYAMLDocumentIsRejected(t *testing.T) {
	doc := validCommitteesHeader + `
  - id: toc
    name: TOC
    teams:
      - name: members
        members:
          - github: alice
            elected_by: TOC
            start: 2025-03-04
`
	_, err := ValidateCommitteesFile(writeCommitteesFile(t, doc+"---\n"+doc), committeesTestNow)
	if err == nil || !strings.Contains(err.Error(), "single YAML document") {
		t.Errorf("expected a single-document error, got %v", err)
	}

	for _, trailer := range []string{"---\n", "---\n# end of file\n", "...\n"} {
		res, err := ValidateCommitteesFile(writeCommitteesFile(t, doc+trailer), committeesTestNow)
		if err != nil || !res.Valid {
			t.Errorf("trailer %q: expected a valid file, got err=%v errors=%v", trailer, err, res.Errors)
		}
	}
}

func TestCollectCommitteeLinksDeduplicatesHandles(t *testing.T) {
	cfg, err := LoadCommitteesFromFile(filepath.Join("example", CommitteesFileName))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	links := CollectCommitteeLinks(cfg, CommitteesFileName)
	seen := map[string]int{}
	for _, l := range links {
		seen[l.URL]++
	}
	if seen["alice"] != 1 {
		t.Errorf("alice should be checked once, got %d", seen["alice"])
	}
	if seen["https://github.com/example-org/toc"] != 1 {
		t.Errorf("committee url should be collected once, got %d", seen["https://github.com/example-org/toc"])
	}
	for _, l := range links {
		if l.URL == "alice" && l.Resolved != "https://github.com/alice" {
			t.Errorf("alice resolved to %q", l.Resolved)
		}
	}
}

func TestFormatCommitteesResult(t *testing.T) {
	res := CommitteesValidationResult{File: "committees.yaml", Committees: 1, Members: 2, Errors: []string{"boom"}, Warnings: []string{"hmm"}}
	text, err := FormatCommitteesResult(res, "text")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Errors:", "boom", "Warnings:", "hmm", "INVALID"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output missing %q:\n%s", want, text)
		}
	}
	js, err := FormatCommitteesResult(res, "json")
	if err != nil || !strings.Contains(js, `"errors"`) {
		t.Errorf("json output unexpected: %v %s", err, js)
	}
	y, err := FormatCommitteesResult(res, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	var back CommitteesValidationResult
	if err := yaml.Unmarshal([]byte(y), &back); err != nil {
		t.Fatalf("yaml output does not parse: %v\n%s", err, y)
	}
	if back.File != res.File || back.Members != 2 || len(back.Errors) != 1 || len(back.Warnings) != 1 {
		t.Errorf("yaml output round-trip mismatch: %+v", back)
	}
	valid, err := FormatCommitteesResult(CommitteesValidationResult{File: "c.yaml", Valid: true}, "text")
	if err != nil || !strings.Contains(valid, "Result: VALID") || strings.Contains(valid, "Errors:") {
		t.Errorf("valid text output unexpected: %v\n%s", err, valid)
	}
}
