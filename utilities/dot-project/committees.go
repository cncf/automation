package projects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CommitteesConfig is the root of a committees.yaml file: the roster of an
// organization's governance bodies (committees, boards, advisory groups,
// staff teams). It is deliberately organization-agnostic so the same schema
// serves the CNCF, Kubernetes, or any other GitHub organization.
type CommitteesConfig struct {
	SchemaVersion string      `json:"schema_version" yaml:"schema_version"`
	Org           string      `json:"org,omitempty" yaml:"org,omitempty"`
	Committees    []Committee `json:"committees" yaml:"committees"`
}

// Committee is one governance body, such as a technical oversight committee,
// a technical advisory group, a board, or a staff team.
type Committee struct {
	ID          string          `json:"id" yaml:"id"`
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string          `json:"type,omitempty" yaml:"type,omitempty"`
	URL         string          `json:"url,omitempty" yaml:"url,omitempty"`
	Teams       []CommitteeTeam `json:"teams" yaml:"teams"`
}

// CommitteeTeam is a role within a committee (members, chairs, alternates,
// tech-leads, liaisons, emeritus, ...).
type CommitteeTeam struct {
	Name        string            `json:"name" yaml:"name"`
	GitHubTeams []string          `json:"github_teams,omitempty" yaml:"github_teams,omitempty"`
	Emeritus    bool              `json:"emeritus,omitempty" yaml:"emeritus,omitempty"`
	Members     []CommitteeMember `json:"members" yaml:"members"`
}

// CommitteeMember is one seat holder. Role is an optional
// position within the team (chair, vice-chair, ...). Liaison lists the ids of
// other committees in the file that the member is liaison to. End is
// optional: nil means the term has no fixed end date (staff, board seats,
// open terms).
type CommitteeMember struct {
	GitHub    string   `json:"github" yaml:"github"`
	Role      string   `json:"role,omitempty" yaml:"role,omitempty"`
	Liaison   []string `json:"liaison,omitempty" yaml:"liaison,omitempty"`
	ElectedBy string   `json:"elected_by" yaml:"elected_by"`
	Start     string   `json:"start" yaml:"start"`
	End       *string  `json:"end,omitempty" yaml:"end,omitempty"`

	// notMapping and problems carry decode-time findings to the validator,
	// which reports them with the member's path. malformed names fields whose
	// value was already rejected, so they are not reported again as missing.
	notMapping bool
	problems   []string
	malformed  map[string]bool
}

// liaisonField is the only member field that holds a list rather than a
// single value; it is decoded by setLiaison instead of setter.
const liaisonField = "liaison"

// setter returns the function that stores a decoded single value in the
// named field, or nil when the field is not one of them. Together with
// liaisonField, this switch is the list of member fields accepted in
// committees.yaml.
func (m *CommitteeMember) setter(field string) func(string) {
	switch field {
	case "github":
		return func(v string) { m.GitHub = v }
	case "role":
		return func(v string) { m.Role = v }
	case "elected_by":
		return func(v string) { m.ElectedBy = v }
	case "start":
		return func(v string) { m.Start = v }
	case "end":
		return func(v string) {
			if v != "" {
				m.End = &v
			}
		}
	}
	return nil
}

func (m *CommitteeMember) problemf(format string, args ...any) {
	m.problems = append(m.problems, fmt.Sprintf(format, args...))
}

// setLiaison decodes the liaison list. A null value is treated as omitted,
// like end; anything other than a non-empty list of single values is
// reported.
func (m *CommitteeMember) setLiaison(v *yaml.Node) {
	switch {
	case v.ShortTag() == "!!null":
		return
	case v.Kind != yaml.SequenceNode:
		m.problemf("line %d: field %q must be a list of committee ids", v.Line, liaisonField)
		return
	case len(v.Content) == 0:
		m.problemf("line %d: field %q cannot be an empty list; omit it instead", v.Line, liaisonField)
		return
	}
	for _, item := range v.Content {
		if item.Kind != yaml.ScalarNode || item.ShortTag() == "!!null" {
			m.problemf("line %d: %s entries must be committee ids", item.Line, liaisonField)
			continue
		}
		m.Liaison = append(m.Liaison, item.Value)
	}
}

// UnmarshalYAML decodes a member by hand so that the validator can report
// every malformed member instead of stopping at the first.
func (m *CommitteeMember) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
	case yaml.ScalarNode:
		m.notMapping = true
		m.GitHub = node.Value
		m.problemf("line %d: member %q must be a mapping with github, elected_by and start; "+
			"bare handles are not allowed", node.Line, node.Value)
		return nil
	default:
		m.notMapping = true
		m.problemf("line %d: member must be a mapping with github, elected_by and start", node.Line)
		return nil
	}

	seen := make(map[string]bool)
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		set := m.setter(k.Value)
		switch {
		case set == nil && k.Value != liaisonField:
			m.problemf("line %d: unknown field %q", k.Line, k.Value)
		case seen[k.Value]:
			m.problemf("line %d: field %q is set more than once", k.Line, k.Value)
		case k.Value == liaisonField:
			seen[k.Value] = true
			m.setLiaison(v)
		case v.Kind != yaml.ScalarNode:
			seen[k.Value] = true
			if m.malformed == nil {
				m.malformed = make(map[string]bool)
			}
			m.malformed[k.Value] = true
			m.problemf("line %d: field %q must be a single value", v.Line, k.Value)
		case v.ShortTag() == "!!null":
			seen[k.Value] = true
			set("")
		default:
			seen[k.Value] = true
			set(v.Value)
		}
	}
	return nil
}

// CommitteesValidationResult is the outcome of validating one committees file.
type CommitteesValidationResult struct {
	File       string   `json:"file" yaml:"file"`
	Valid      bool     `json:"valid" yaml:"valid"`
	Committees int      `json:"committees" yaml:"committees"`
	Members    int      `json:"members" yaml:"members"`
	Errors     []string `json:"errors,omitempty" yaml:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// LoadCommitteesFromFile reads and strictly parses a committees.yaml file.
func LoadCommitteesFromFile(path string) (CommitteesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CommitteesConfig{}, fmt.Errorf("failed to read committees file: %w", err)
	}

	var cfg CommitteesConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		if err == io.EOF {
			return CommitteesConfig{}, fmt.Errorf("committees file %s is empty", path)
		}
		return CommitteesConfig{}, fmt.Errorf("failed to parse committees YAML: %w", err)
	}

	// Anything after a second "---" would otherwise be dropped silently. A
	// trailing separator with nothing after it is harmless and allowed.
	for {
		var extra yaml.Node
		err := decoder.Decode(&extra)
		if err == io.EOF {
			break
		}
		if err != nil {
			return CommitteesConfig{}, fmt.Errorf("failed to parse committees YAML: %w", err)
		}
		if len(extra.Content) > 0 && extra.Content[0].ShortTag() != "!!null" {
			return CommitteesConfig{}, fmt.Errorf("committees file %s must contain a single YAML document "+
				"(found another at line %d)", path, extra.Line)
		}
	}
	return cfg, nil
}

// ValidateCommitteesFile loads and validates a committees.yaml file. now
// decides which terms have expired; pass time.Now() outside of tests.
func ValidateCommitteesFile(path string, now time.Time) (CommitteesValidationResult, error) {
	cfg, err := LoadCommitteesFromFile(path)
	if err != nil {
		return CommitteesValidationResult{}, err
	}
	result := ValidateCommittees(cfg, now)
	result.File = path
	return result, nil
}

// ValidateCommittees checks a parsed committees configuration. Errors fail
// validation; warnings flag data that is valid but likely stale, such as a
// term that has already ended, so that CI does not start failing on a date.
func ValidateCommittees(cfg CommitteesConfig, now time.Time) CommitteesValidationResult {
	u := now.UTC()
	v := &committeesValidator{
		today: time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC),
		ids:   make(map[string]bool),
	}

	switch cfg.SchemaVersion {
	case "":
		v.errorf("schema_version is required")
	case CommitteesSchemaVersion:
	default:
		v.errorf("schema_version %q is not supported (expected %q)", cfg.SchemaVersion, CommitteesSchemaVersion)
	}

	if len(cfg.Committees) == 0 {
		v.errorf("committees list cannot be empty")
	}

	// Collected up front so a liaison may name a committee defined later.
	for _, c := range cfg.Committees {
		if c.ID != "" {
			if v.ids[c.ID] {
				v.errorf("committees[%s]: duplicate id", c.ID)
			}
			v.ids[c.ID] = true
		}
	}

	for i, c := range cfg.Committees {
		v.committee(i, c)
	}

	v.res.Valid = len(v.res.Errors) == 0
	return v.res
}

// committeesValidator accumulates the findings of one ValidateCommittees run.
type committeesValidator struct {
	res   CommitteesValidationResult
	today time.Time
	ids   map[string]bool // every committee id in the file
	cid   string          // id of the committee being validated
}

func (v *committeesValidator) errorf(format string, args ...any) {
	v.res.Errors = append(v.res.Errors, fmt.Sprintf(format, args...))
}

func (v *committeesValidator) warnf(format string, args ...any) {
	v.res.Warnings = append(v.res.Warnings, fmt.Sprintf(format, args...))
}

func (v *committeesValidator) committee(i int, c Committee) {
	v.res.Committees++
	v.cid = c.ID
	cpath := fmt.Sprintf("committees[%s]", c.ID)
	if c.ID == "" {
		cpath = fmt.Sprintf("committees[%d]", i)
		v.errorf("%s: id is required", cpath)
	} else if !isValidSlug(c.ID) {
		v.errorf("%s: id must be lowercase letters, digits and hyphens", cpath)
	}
	if strings.TrimSpace(c.Name) == "" {
		v.errorf("%s: name is required", cpath)
	}
	if c.URL != "" && !isValidURL(c.URL) {
		v.errorf("%s: url %q is not a valid http(s) URL", cpath, c.URL)
	}

	if len(c.Teams) == 0 {
		v.errorf("%s: teams list cannot be empty", cpath)
		return
	}

	teamNames := make(map[string]bool)
	hasActiveTeam := false
	for j, t := range c.Teams {
		tpath := fmt.Sprintf("%s.teams[%s]", cpath, t.Name)
		if t.Name == "" {
			tpath = fmt.Sprintf("%s.teams[%d]", cpath, j)
		} else if teamNames[t.Name] {
			v.errorf("%s: duplicate team name", tpath)
		}
		teamNames[t.Name] = true
		if !t.Emeritus {
			hasActiveTeam = true
		}
		v.team(tpath, t)
	}
	if !hasActiveTeam {
		v.errorf("%s: at least one team must be non-emeritus", cpath)
	}
}

func (v *committeesValidator) team(tpath string, t CommitteeTeam) {
	if t.Name == "" {
		v.errorf("%s: name is required", tpath)
	} else if !isValidSlug(t.Name) {
		v.errorf("%s: name must be lowercase letters, digits and hyphens", tpath)
	}

	ghTeams := make(map[string]bool)
	for _, gt := range t.GitHubTeams {
		if !isValidSlug(gt) {
			v.errorf("%s: github_teams entry %q is not a valid GitHub team slug", tpath, gt)
		}
		if ghTeams[gt] {
			v.errorf("%s: github_teams entry %q is listed more than once", tpath, gt)
		}
		ghTeams[gt] = true
	}
	if t.Emeritus && len(t.GitHubTeams) > 0 {
		v.errorf("%s: emeritus teams cannot grant github_teams", tpath)
	}

	if len(t.Members) == 0 {
		v.errorf("%s: members list cannot be empty; remove the team until it has members", tpath)
	}
	for k, m := range t.Members {
		v.member(tpath, k, m, t.Emeritus)
	}
	for _, dup := range overlappingTerms(t.Members) {
		v.errorf("%s.members[%s]: duplicate handle in team with overlapping terms", tpath, dup)
	}
}

func (v *committeesValidator) member(tpath string, k int, m CommitteeMember, emeritus bool) {
	v.res.Members++
	mpath := fmt.Sprintf("%s.members[%s]", tpath, m.GitHub)
	if m.GitHub == "" {
		mpath = fmt.Sprintf("%s.members[%d]", tpath, k)
	}
	for _, p := range m.problems {
		v.errorf("%s: %s", mpath, p)
	}
	if m.notMapping {
		// The field-level errors below would only restate that the entry is
		// not a mapping.
		return
	}

	switch {
	case m.malformed["github"]:
	case m.GitHub == "":
		v.errorf("%s: github is required", mpath)
	case strings.HasPrefix(m.GitHub, "@"):
		v.errorf("%s: github handle must not start with @", mpath)
	case !githubHandleRe.MatchString(m.GitHub):
		v.errorf("%s: github %q is not a valid GitHub handle", mpath, m.GitHub)
	}

	if m.Role != "" && !isValidSlug(m.Role) {
		v.errorf("%s: role %q must be lowercase letters, digits and hyphens", mpath, m.Role)
	}

	listed := make(map[string]bool)
	for _, id := range m.Liaison {
		switch {
		case listed[id]:
			v.errorf("%s: liaison %q is listed more than once", mpath, id)
		case !v.ids[id]:
			v.errorf("%s: liaison %q is not a committee id in this file", mpath, id)
		case id == v.cid:
			v.errorf("%s: liaison %q is the member's own committee", mpath, id)
		}
		listed[id] = true
	}

	if strings.TrimSpace(m.ElectedBy) == "" && !m.malformed["elected_by"] {
		v.errorf("%s: elected_by is required", mpath)
	}

	start, startOK := parseCommitteeDate(m.Start)
	switch {
	case m.malformed["start"]:
	case m.Start == "":
		v.errorf("%s: start is required", mpath)
	case !startOK:
		v.errorf("%s: start %q must be a YYYY-MM-DD date", mpath, m.Start)
	}

	if m.End == nil {
		if emeritus && !m.malformed["end"] {
			v.errorf("%s: end is required for emeritus members", mpath)
		}
		return
	}
	end, endOK := parseCommitteeDate(*m.End)
	switch {
	case !endOK:
		v.errorf("%s: end %q must be a YYYY-MM-DD date", mpath, *m.End)
	case startOK && !end.After(start):
		v.errorf("%s: end %s must be after start %s", mpath, *m.End, m.Start)
	case emeritus && end.After(v.today):
		v.warnf("%s: emeritus member's term ends in the future (%s)", mpath, *m.End)
	case !emeritus && end.Before(v.today):
		v.warnf("%s: term ended on %s; renew it or move the member to an emeritus team", mpath, *m.End)
	}
}
func parseCommitteeDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(CommitteeDateLayout, s)
	return t, err == nil
}

// overlappingTerms returns, in first-seen order, the handles that appear more
// than once in a team with overlapping terms. Repeat entries with disjoint
// terms are allowed so re-elections and separate stints can be recorded. A
// missing or unparsable start counts as the beginning of time and a missing or
// unparsable end as open, so malformed entries still clash.
func overlappingTerms(members []CommitteeMember) []string {
	type term struct{ start, end time.Time }
	byHandle := make(map[string][]term)
	var dups []string
	reported := make(map[string]bool)
	for _, m := range members {
		if m.notMapping || m.GitHub == "" {
			continue
		}
		key := strings.ToLower(m.GitHub)
		start, _ := parseCommitteeDate(m.Start)
		end := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
		if m.End != nil {
			if e, ok := parseCommitteeDate(*m.End); ok {
				end = e
			}
		}
		cur := term{start, end}
		for _, prev := range byHandle[key] {
			if cur.start.Before(prev.end) && prev.start.Before(cur.end) && !reported[key] {
				reported[key] = true
				dups = append(dups, m.GitHub)
			}
		}
		byHandle[key] = append(byHandle[key], cur)
	}
	return dups
}

// CollectCommitteeLinks returns one link per distinct member handle, plus each
// committee URL, so -check-links catches renamed or deleted GitHub accounts.
func CollectCommitteeLinks(cfg CommitteesConfig, source string) []Link {
	var links []Link
	seen := make(map[string]bool)
	for _, c := range cfg.Committees {
		if c.URL != "" {
			links = append(links, newLink(source, fmt.Sprintf("committees[%s].url", c.ID), c.URL, c.URL))
		}
		for _, t := range c.Teams {
			for _, m := range t.Members {
				h := strings.TrimSpace(m.GitHub)
				key := strings.ToLower(h)
				if h == "" || seen[key] {
					continue
				}
				seen[key] = true
				field := fmt.Sprintf("committees[%s].teams[%s]", c.ID, t.Name)
				switch {
				case isPlaceholder(h):
					links = append(links, Link{Source: source, Field: field, URL: h, Note: "placeholder handle has not been replaced"})
				case !githubHandleRe.MatchString(h):
					links = append(links, Link{Source: source, Field: field, URL: h, Resolved: h})
				default:
					links = append(links, Link{Source: source, Field: field, URL: h, Resolved: "https://github.com/" + h})
				}
			}
		}
	}
	return links
}

// FormatCommitteesResult renders a validation result as text, json or yaml.
func FormatCommitteesResult(res CommitteesValidationResult, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	case "yaml":
		data, err := yaml.Marshal(res)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		var b strings.Builder
		b.WriteString("Committees Validation Report\n")
		b.WriteString("============================\n\n")
		fmt.Fprintf(&b, "File: %s\n", res.File)
		fmt.Fprintf(&b, "Committees: %d, member entries: %d\n\n", res.Committees, res.Members)
		if len(res.Errors) > 0 {
			b.WriteString("Errors:\n")
			for _, e := range res.Errors {
				fmt.Fprintf(&b, "  - %s\n", e)
			}
			b.WriteString("\n")
		}
		if len(res.Warnings) > 0 {
			b.WriteString("Warnings:\n")
			for _, w := range res.Warnings {
				fmt.Fprintf(&b, "  - %s\n", w)
			}
			b.WriteString("\n")
		}
		if res.Valid {
			b.WriteString("Result: VALID\n")
		} else {
			fmt.Fprintf(&b, "Result: INVALID (%d errors)\n", len(res.Errors))
		}
		return b.String(), nil
	}
}
