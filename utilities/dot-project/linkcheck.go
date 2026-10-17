package projects

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Link is one URL taken from a .project file, with enough context to tell a
// maintainer exactly which line to fix.
type Link struct {
	// Source is the file the link came from, e.g. "spire/project.yaml".
	Source string
	// Field is the YAML path of the link, e.g. "security.policy".
	Field string
	// URL is the value as written in the file.
	URL string
	// Resolved is the absolute URL that is actually checked. It differs from
	// URL when a relative path was resolved against the primary repository,
	// and is empty when there is nothing to fetch.
	Resolved string
	// Note, when set, means the link is not fetched and is reported as a
	// warning with this explanation (placeholders, unresolvable paths).
	Note string
}

// LinkStatus is the outcome of checking a single link.
type LinkStatus int

const (
	// LinkOK means the target exists.
	LinkOK LinkStatus = iota
	// LinkWarning means the target could not be confirmed either way (rate
	// limits, bot blocking, timeouts, placeholders). Warnings never fail a run.
	LinkWarning
	// LinkBroken means the target is definitely gone (404/410) or the value
	// is not a usable URL at all.
	LinkBroken
)

func (s LinkStatus) String() string {
	switch s {
	case LinkOK:
		return "ok"
	case LinkWarning:
		return "warning"
	default:
		return "broken"
	}
}

// LinkResult is a Link together with its check outcome.
type LinkResult struct {
	Link
	Status LinkStatus
	Detail string
}

var (
	placeholderRe  = regexp.MustCompile(`XXX|\bTODO\b|github-handle|example\.(com|org)`)
	githubHandleRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)
)

// isPlaceholder reports whether v is scaffold filler that the project has not
// replaced yet. Placeholders are expected right after onboarding, so they are
// surfaced as warnings rather than fetched and reported as broken.
func isPlaceholder(v string) bool {
	return placeholderRe.MatchString(v)
}

// CollectProjectLinks returns every link in a project.yaml. Relative PathRef
// values are resolved against the primary repository so they can be checked.
// source labels the file in reports.
func CollectProjectLinks(p Project, source string) []Link {
	primary := PrimaryRepositoryURL(p.Repositories)
	var links []Link

	addURL := func(field, v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		links = append(links, newLink(source, field, v, v))
	}
	addPath := func(field string, ref *PathRef) {
		if ref == nil || strings.TrimSpace(ref.Path) == "" {
			return
		}
		v := strings.TrimSpace(ref.Path)
		if hasScheme(v) || isPlaceholder(v) {
			links = append(links, newLink(source, field, v, v))
			return
		}
		resolved, ok := resolveRepoPath(primary, v)
		if !ok {
			links = append(links, Link{Source: source, Field: field, URL: v,
				Note: "relative path cannot be resolved because the primary repository is not on GitHub; use a full URL"})
			return
		}
		links = append(links, newLink(source, field, v, resolved))
	}

	addURL("website", p.Website)
	addURL("artwork", p.Artwork)
	for i, r := range p.Repositories {
		addURL(fmt.Sprintf("repositories[%d]", i), r.URL)
	}
	for i, m := range p.MaturityLog {
		addURL(fmt.Sprintf("maturity_log[%d].issue", i), m.Issue)
	}
	for _, k := range sortedKeys(p.Social) {
		addURL("social."+k, p.Social[k])
	}
	for i, m := range p.MailingLists {
		// Mailing lists are frequently written as bare addresses, which
		// have nothing to fetch.
		if hasScheme(m) || isPlaceholder(m) {
			addURL(fmt.Sprintf("mailing_lists[%d]", i), m)
		}
	}
	for i, a := range p.Audits {
		addURL(fmt.Sprintf("audits[%d].url", i), a.URL)
	}
	for i, s := range p.SlackChannels {
		addURL(fmt.Sprintf("slack_channels[%d].link", i), s.Link)
	}
	addPath("adopters", p.Adopters)

	if s := p.Security; s != nil {
		addPath("security.policy", s.Policy)
		addPath("security.threat_model", s.ThreatModel)
		if s.Contact != nil {
			addURL("security.contact.advisory_url", s.Contact.AdvisoryURL)
		}
	}
	if g := p.Governance; g != nil {
		for _, f := range []struct {
			name string
			ref  *PathRef
		}{
			{"contributing", g.Contributing},
			{"codeowners", g.Codeowners},
			{"governance_doc", g.GovernanceDoc},
			{"gitvote_config", g.GitVoteConfig},
			{"vendor_neutrality_statement", g.VendorNeutralityStatement},
			{"decision_making_process", g.DecisionMakingProcess},
			{"roles_and_teams", g.RolesAndTeams},
			{"code_of_conduct", g.CodeOfConduct},
			{"sub_project_list", g.SubProjectList},
			{"sub_project_docs", g.SubProjectDocs},
			{"contributor_ladder", g.ContributorLadder},
			{"change_process", g.ChangeProcess},
			{"comms_channels", g.CommsChannels},
			{"community_calendar", g.CommunityCalendar},
			{"contributor_guide", g.ContributorGuide},
		} {
			addPath("governance."+f.name, f.ref)
		}
		ml := g.MaintainerLifecycle
		addPath("governance.maintainer_lifecycle.onboarding_doc", ml.OnboardingDoc)
		addPath("governance.maintainer_lifecycle.progression_ladder", ml.ProgressionLadder)
		addPath("governance.maintainer_lifecycle.offboarding_policy", ml.OffboardingPolicy)
		for i, m := range ml.MentoringProgram {
			addURL(fmt.Sprintf("governance.maintainer_lifecycle.mentoring_program[%d]", i), m)
		}
	}
	if l := p.Legal; l != nil {
		addPath("legal.license", l.License)
		if it := l.IdentityType; it != nil {
			addPath("legal.identity_type.dco_url", it.DCOURL)
			addPath("legal.identity_type.cla_url", it.CLAURL)
		}
	}
	if d := p.Documentation; d != nil {
		addPath("documentation.readme", d.Readme)
		addPath("documentation.support", d.Support)
		addPath("documentation.architecture", d.Architecture)
		addPath("documentation.api", d.API)
	}

	return links
}

// CollectMaintainerLinks returns one link per maintainer handle, pointing at
// the GitHub profile, so a renamed or deleted account is caught the same way
// as a dead URL. Unmanaged teams are included: a stale handle is wrong
// regardless of whether CNCF provisions resources for it.
func CollectMaintainerLinks(cfg MaintainersConfig, source string) []Link {
	var links []Link
	for _, entry := range cfg.Maintainers {
		for _, team := range entry.Teams {
			for _, m := range team.Members {
				h := strings.TrimPrefix(strings.TrimSpace(m), "@")
				if h == "" {
					continue
				}
				field := fmt.Sprintf("maintainers[%s].teams[%s]", entry.ProjectID, team.Name)
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

// CollectLinksFromFiles parses each project and maintainers file and returns
// all their links. Paths in reports are made relative to root when possible.
//
// Files that cannot be read or parsed are skipped: schema validation already
// reports them, and one bad file should not hide broken links in the others.
func CollectLinksFromFiles(root string, projectFiles, maintainersFiles []string) []Link {
	var links []Link
	for _, path := range projectFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var p Project
		if err := yaml.Unmarshal(data, &p); err != nil {
			continue
		}
		links = append(links, CollectProjectLinks(p, relTo(root, path))...)
	}
	for _, path := range maintainersFiles {
		cfg, err := LoadMaintainersFromFile(path)
		if err != nil {
			continue
		}
		links = append(links, CollectMaintainerLinks(cfg, relTo(root, path))...)
	}
	return links
}

// CollectRepoLinks discovers the .project repository at root and returns the
// links of every project and maintainers file in it.
func CollectRepoLinks(root string) ([]Link, error) {
	d, err := Discover(root)
	if err != nil {
		return nil, err
	}
	var projectFiles []string
	for _, p := range d.ProjectPaths() {
		if fileExists(p) {
			projectFiles = append(projectFiles, p)
		}
	}
	return CollectLinksFromFiles(root, projectFiles, d.MaintainersPaths()), nil
}

// LocalProjectFiles returns the local paths of the entries in a project list,
// using the same rule as the validator: anything that is not http(s) is a
// file, with or without a file:// prefix.
func LocalProjectFiles(listPath string) ([]string, error) {
	if listPath == "" || listPath == "/dev/null" {
		return nil, nil
	}
	data, err := os.ReadFile(listPath)
	if err != nil {
		return nil, err
	}
	var list ProjectListConfig
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to parse project list: %w", err)
	}
	var files []string
	for _, e := range list.Projects {
		u := os.ExpandEnv(e.URL)
		if !isHTTPURL(u) {
			files = append(files, strings.TrimPrefix(u, "file://"))
		}
	}
	return files, nil
}

// ExcludeKnownLinks drops links whose URL already appears in base, so a pull
// request is only held responsible for the links it introduces.
func ExcludeKnownLinks(links, base []Link) []Link {
	known := make(map[string]bool, len(base))
	for _, l := range base {
		known[l.URL] = true
	}
	var out []Link
	for _, l := range links {
		if !known[l.URL] {
			out = append(out, l)
		}
	}
	return out
}

// LinkChecker verifies that links resolve. GitHub links are checked through
// the REST API, which is authenticated and far less likely to be rate limited
// or bot-blocked than scraping github.com.
type LinkChecker struct {
	Client       *http.Client
	Token        string
	APIBase      string
	WebBase      string
	Concurrency  int
	RecheckDelay time.Duration
}

// NewLinkChecker returns a checker with production defaults. token may be
// empty, at the cost of GitHub's unauthenticated rate limit.
func NewLinkChecker(token string) *LinkChecker {
	return &LinkChecker{
		Client:       &http.Client{Timeout: DefaultHTTPTimeout},
		Token:        token,
		APIBase:      DefaultGitHubAPIURL,
		WebBase:      "https://github.com",
		Concurrency:  DefaultLinkCheckConcurrency,
		RecheckDelay: DefaultLinkCheckRecheckDelay,
	}
}

// Check verifies every link and returns results in input order. Each distinct
// URL is fetched once. A link that looks broken is fetched once more after
// RecheckDelay, so a momentary outage does not open an issue or fail a PR.
func (c *LinkChecker) Check(ctx context.Context, links []Link) []LinkResult {
	type outcome struct {
		status LinkStatus
		detail string
	}

	var unique []string
	seen := map[string]bool{}
	for _, l := range links {
		if l.Note == "" && !seen[l.Resolved] {
			seen[l.Resolved] = true
			unique = append(unique, l.Resolved)
		}
	}

	conc := c.Concurrency
	if conc < 1 {
		conc = 1
	}
	outcomes := make(map[string]outcome, len(unique))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, conc)
	var suspects []string

	for _, u := range unique {
		wg.Add(1)
		sem <- struct{}{}
		go func(u string) {
			defer wg.Done()
			defer func() { <-sem }()
			s, d := c.checkOne(ctx, u)
			mu.Lock()
			outcomes[u] = outcome{s, d}
			if s == LinkBroken && isHTTPURL(u) {
				suspects = append(suspects, u)
			}
			mu.Unlock()
		}(u)
	}
	wg.Wait()

	if len(suspects) > 0 && c.RecheckDelay > 0 {
		select {
		case <-time.After(c.RecheckDelay):
		case <-ctx.Done():
		}
	}
	for _, u := range suspects {
		wg.Add(1)
		sem <- struct{}{}
		go func(u string) {
			defer wg.Done()
			defer func() { <-sem }()
			s, d := c.checkOne(ctx, u)
			if s != LinkBroken {
				mu.Lock()
				outcomes[u] = outcome{s, d}
				mu.Unlock()
			}
		}(u)
	}
	wg.Wait()

	results := make([]LinkResult, len(links))
	for i, l := range links {
		if l.Note != "" {
			results[i] = LinkResult{Link: l, Status: LinkWarning, Detail: l.Note}
			continue
		}
		o := outcomes[l.Resolved]
		results[i] = LinkResult{Link: l, Status: o.status, Detail: o.detail}
	}
	return results
}

func (c *LinkChecker) checkOne(ctx context.Context, raw string) (LinkStatus, string) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.Contains(u.Host, ".") {
		if !hasScheme(raw) && githubHandleLike(raw) {
			return LinkBroken, "not a valid GitHub handle"
		}
		return LinkBroken, "not a valid http(s) URL"
	}

	var status LinkStatus
	var detail string
	if strings.EqualFold(u.Host, "github.com") || strings.EqualFold(u.Host, "www.github.com") {
		status, detail = c.checkGitHub(ctx, u)
	} else {
		status, detail = c.checkWeb(ctx, raw)
	}
	if status == LinkOK && u.Scheme == "http" {
		return LinkWarning, "uses http://; switch to https://"
	}
	return status, detail
}

// checkGitHub maps a github.com URL onto the REST endpoint that proves the
// target exists.
func (c *LinkChecker) checkGitHub(ctx context.Context, u *url.URL) (LinkStatus, string) {
	seg := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
	var api string
	contents := false
	switch {
	case len(seg) == 0:
		return LinkOK, ""
	case len(seg) >= 2 && seg[0] == "orgs":
		api = "/orgs/" + url.PathEscape(seg[1])
	case len(seg) == 1:
		api = "/users/" + url.PathEscape(seg[0])
	default:
		owner, repo := url.PathEscape(seg[0]), url.PathEscape(strings.TrimSuffix(seg[1], ".git"))
		api = "/repos/" + owner + "/" + repo
		switch {
		case len(seg) >= 4 && (seg[2] == "blob" || seg[2] == "tree"):
			// Refs may contain slashes, but nearly every .project link
			// points at a single-segment branch or HEAD; anything the API
			// cannot resolve is confirmed against github.com below.
			api += "/contents/" + escapePath(seg[4:]) + "?ref=" + url.QueryEscape(seg[3])
			contents = true
		case len(seg) >= 4 && (seg[2] == "issues" || seg[2] == "pull"):
			// Pull requests are issues in the REST API.
			api += "/issues/" + url.PathEscape(seg[3])
		}
		// Any other repository page (security advisories, discussions,
		// releases, ...) is accepted when the repository exists.
	}

	code, err := c.get(ctx, strings.TrimRight(c.APIBase, "/")+api, true)
	if err != nil {
		return LinkWarning, "could not be checked: " + err.Error()
	}
	if (code == http.StatusNotFound) && contents {
		// The API does not follow branch renames (master -> main), but
		// github.com does, and a renamed-branch link still works for readers.
		web := strings.TrimRight(c.WebBase, "/") + u.EscapedPath()
		if wc, err := c.get(ctx, web, false); err == nil {
			code = wc
		}
	}
	return classify(code)
}

func (c *LinkChecker) checkWeb(ctx context.Context, raw string) (LinkStatus, string) {
	code, err := c.do(ctx, http.MethodHead, raw, false)
	switch {
	case err != nil,
		code == http.StatusMethodNotAllowed,
		code == http.StatusForbidden,
		code == http.StatusBadRequest,
		code == http.StatusNotImplemented,
		code == http.StatusNotFound:
		// Many servers mishandle HEAD, including some that answer 404 to
		// HEAD and 200 to GET, so GET has the final say.
		code, err = c.get(ctx, raw, false)
	}
	if err != nil {
		return LinkWarning, "could not be checked: " + err.Error()
	}
	return classify(code)
}

func (c *LinkChecker) get(ctx context.Context, target string, api bool) (int, error) {
	return c.do(ctx, http.MethodGet, target, api)
}

func (c *LinkChecker) do(ctx context.Context, method, target string, api bool) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", bootstrapUserAgent)
	if api {
		req.Header.Set("Accept", "application/vnd.github+json")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, nil
}

// classify treats only 404 and 410 as broken. Everything else that is not a
// success (bot walls, rate limits, server errors) says nothing reliable about
// whether the page exists, so it is a warning.
func classify(code int) (LinkStatus, string) {
	switch {
	case code >= 200 && code < 400:
		return LinkOK, ""
	case code == http.StatusNotFound || code == http.StatusGone:
		return LinkBroken, fmt.Sprintf("%d %s", code, http.StatusText(code))
	default:
		return LinkWarning, fmt.Sprintf("could not be confirmed (%d %s)", code, http.StatusText(code))
	}
}

// LinkSummary counts results by status.
func LinkSummary(results []LinkResult) (broken, warnings int) {
	for _, r := range results {
		switch r.Status {
		case LinkBroken:
			broken++
		case LinkWarning:
			warnings++
		}
	}
	return broken, warnings
}

// FormatLinkResultsText renders broken links and warnings for a terminal.
func FormatLinkResultsText(results []LinkResult) string {
	broken, warnings := LinkSummary(results)
	var b strings.Builder
	fmt.Fprintf(&b, "Link check: %d checked, %d broken, %d warnings\n", len(results), broken, warnings)
	for _, status := range []LinkStatus{LinkBroken, LinkWarning} {
		for _, r := range results {
			if r.Status != status {
				continue
			}
			mark := "✗"
			if status == LinkWarning {
				mark = "!"
			}
			fmt.Fprintf(&b, "  %s %s: %s: %s (%s)\n", mark, r.Source, r.Field, r.URL, r.Detail)
		}
	}
	return b.String()
}

// FormatLinkAnnotations renders GitHub Actions workflow commands so problems
// show up inline on the pull request's file view.
func FormatLinkAnnotations(results []LinkResult) string {
	var b strings.Builder
	for _, r := range results {
		var level string
		switch r.Status {
		case LinkBroken:
			level = "error"
		case LinkWarning:
			level = "warning"
		default:
			continue
		}
		fmt.Fprintf(&b, "::%s file=%s,title=%s::%s\n", level,
			annotationProperty(r.Source), annotationProperty(r.Field),
			annotationData(fmt.Sprintf("%s: %s (%s)", r.Field, r.URL, r.Detail)))
	}
	return b.String()
}

// FormatLinkReportMarkdown renders the issue body opened by the scheduled
// scan. It lists broken links first, since those are what needs fixing.
func FormatLinkReportMarkdown(results []LinkResult) string {
	broken, warnings := LinkSummary(results)
	var b strings.Builder
	b.WriteString("The scheduled link check found links in this repository that no longer resolve.\n\n")
	fmt.Fprintf(&b, "**%d broken**, %d warnings, %d links checked.\n\n", broken, warnings, len(results))

	write := func(title string, status LinkStatus) {
		var rows []LinkResult
		for _, r := range results {
			if r.Status == status {
				rows = append(rows, r)
			}
		}
		if len(rows) == 0 {
			return
		}
		fmt.Fprintf(&b, "### %s\n\n| File | Field | Link | Result |\n|---|---|---|---|\n", title)
		for _, r := range rows {
			fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n", r.Source, r.Field, mdEscape(r.URL), mdEscape(r.Detail))
		}
		b.WriteString("\n")
	}
	write("Broken", LinkBroken)
	write("Warnings", LinkWarning)
	b.WriteString("Fix the links (or replace them with the correct location) and this issue closes automatically on the next run. ")
	b.WriteString("Warnings are informational: the target could not be confirmed, often because the site blocks automated requests.\n")
	return b.String()
}

func newLink(source, field, raw, resolved string) Link {
	l := Link{Source: source, Field: field, URL: raw, Resolved: resolved}
	if isPlaceholder(raw) {
		l.Note = "placeholder has not been replaced"
	}
	return l
}

func hasScheme(v string) bool {
	i := strings.Index(v, "://")
	return i > 0 && !strings.ContainsAny(v[:i], "/ ")
}

func isHTTPURL(v string) bool {
	return strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://")
}

func githubHandleLike(v string) bool {
	return !strings.ContainsAny(v, "/.:")
}

// resolveRepoPath turns a repository-relative path into a browsable GitHub
// URL on the default branch.
func resolveRepoPath(repoURL, path string) (string, bool) {
	org, repo, err := ParseGitHubURL(repoURL)
	if err != nil || org == "" || repo == "" || !strings.Contains(repoURL, "github.com") {
		return "", false
	}
	path = strings.TrimPrefix(strings.TrimPrefix(path, "./"), "/")
	return fmt.Sprintf("https://github.com/%s/%s/blob/HEAD/%s", org, repo, path), true
}

func escapePath(seg []string) string {
	out := make([]string, len(seg))
	for i, s := range seg {
		out[i] = url.PathEscape(s)
	}
	return strings.Join(out, "/")
}

func relTo(root, path string) string {
	if root == "" {
		return filepath.ToSlash(path)
	}
	absRoot, err1 := filepath.Abs(root)
	absPath, err2 := filepath.Abs(path)
	if err1 == nil && err2 == nil {
		if r, err := filepath.Rel(absRoot, absPath); err == nil && !strings.HasPrefix(r, "..") {
			return filepath.ToSlash(r)
		}
	}
	return filepath.ToSlash(path)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// annotationData and annotationProperty apply the escaping GitHub Actions
// requires for workflow command values.
func annotationData(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

func annotationProperty(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C").Replace(s)
}

func mdEscape(s string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ").Replace(s)
}
