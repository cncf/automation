package projects

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CheckBootstrapLinks verifies every link the scaffold is about to write, and
// every maintainer handle, so provisioning never ships metadata that is
// already broken. Rather than failing the run, it records the broken links in
// result.BrokenLinks (which GenerateProjectYAML then comments out under a
// TODO) and moves unknown handles to result.UnknownMaintainers. The full
// results are returned for reporting.
func CheckBootstrapLinks(ctx context.Context, result *BootstrapResult, checker *LinkChecker) ([]LinkResult, error) {
	result.BrokenLinks = nil
	data, err := GenerateProjectYAML(result)
	if err != nil {
		return nil, err
	}
	var p Project
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing generated %s: %w", ProjectFileName, err)
	}

	links := CollectProjectLinks(p, ProjectFileName)

	handles := append([]string(nil), result.Maintainers...)
	if result.ProjectLead != "" && !containsFold(handles, result.ProjectLead) {
		handles = append(handles, result.ProjectLead)
	}
	links = append(links, CollectMaintainerLinks(MaintainersConfig{Maintainers: []MaintainerEntry{{
		ProjectID: result.Slug,
		Teams:     []Team{{Name: result.Slug + "-maintainers", Members: handles}},
	}}}, MaintainersFileName)...)

	results := checker.Check(ctx, links)

	broken := map[string]string{}
	unknown := map[string]bool{}
	for _, r := range results {
		if r.Status != LinkBroken {
			continue
		}
		if r.Source == MaintainersFileName {
			unknown[strings.ToLower(normalizeHandle(r.URL))] = true
			continue
		}
		broken[r.URL] = r.Detail
	}

	if len(broken) > 0 {
		result.BrokenLinks = broken
		result.TODOs = append(result.TODOs, fmt.Sprintf(
			"%d link(s) returned 404 during provisioning and were commented out or flagged; add the correct links", len(broken)))
	}

	if len(unknown) > 0 {
		var kept []string
		for _, h := range result.Maintainers {
			if unknown[strings.ToLower(normalizeHandle(h))] {
				result.UnknownMaintainers = append(result.UnknownMaintainers, h)
			} else {
				kept = append(kept, h)
			}
		}
		result.Maintainers = kept

		if unknown[strings.ToLower(normalizeHandle(result.ProjectLead))] {
			result.ProjectLead = ""
			delete(result.Sources, "project_lead")
			if len(kept) > 0 {
				result.ProjectLead = kept[0]
				if result.Sources == nil {
					result.Sources = map[string]string{}
				}
				result.Sources["project_lead"] = "foundation-csv"
			}
		}
		result.TODOs = append(result.TODOs, fmt.Sprintf(
			"%d maintainer handle(s) have no GitHub account and were left out of the roster; see maintainers.yaml", len(result.UnknownMaintainers)))
	}

	return results, nil
}

// PruneBrokenLinks comments out every line of a generated project.yaml whose
// value is a broken link, with a TODO above it naming the dead URL. The
// result is still valid YAML that passes validation:
//
//   - a path: line takes its parent key with it, so no empty mapping is left;
//   - a list item takes its nested lines with it;
//   - maturity_log issue: is required, so it becomes the XXX placeholder,
//     which validation accepts and the link check reports as a warning;
//   - a repositories entry is kept with a TODO above it, because at least one
//     (primary) repository is required and there is no placeholder for it.
func PruneBrokenLinks(doc string, broken map[string]string) string {
	urls := make([]string, 0, len(broken))
	for u := range broken {
		urls = append(urls, u)
	}
	// Longest first, so a URL that is a prefix of another cannot match the
	// wrong line.
	sort.Slice(urls, func(i, j int) bool { return len(urls[i]) > len(urls[j]) })

	match := func(line string) (string, bool) {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return "", false
		}
		for _, u := range urls {
			if strings.Contains(line, `"`+u+`"`) {
				return u, true
			}
		}
		return "", false
	}

	lines := strings.Split(doc, "\n")
	out := make([]string, 0, len(lines)+len(broken)*2)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		u, ok := match(line)
		if !ok {
			out = append(out, line)
			continue
		}

		indent := leadingSpaces(line)
		trimmed := strings.TrimSpace(line)
		todo := fmt.Sprintf("# TODO: %s returned %s — add the correct link", u, broken[u])

		switch {
		case strings.HasPrefix(trimmed, "issue:"):
			out = append(out, fmt.Sprintf(`%sissue: "https://github.com/cncf/toc/issues/XXX" # TODO: %s returned %s — set the TOC issue URL`,
				pad(indent), u, broken[u]))

		case strings.HasPrefix(trimmed, "- url:"):
			out = append(out, fmt.Sprintf("%s# TODO: %s returned %s — fix or replace this repository URL",
				pad(indent), u, broken[u]), line)

		case strings.HasPrefix(trimmed, "path:") && len(out) > 0 && isParentKey(out[len(out)-1], indent):
			parent := out[len(out)-1]
			pIndent := leadingSpaces(parent)
			out[len(out)-1] = pad(pIndent) + todo
			out = append(out, commentOut(parent, pIndent), commentOut(line, pIndent))

		case strings.HasPrefix(trimmed, "- "):
			out = append(out, pad(indent)+todo, commentOut(line, indent))
			for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" && leadingSpaces(lines[i+1]) > indent {
				i++
				out = append(out, commentOut(lines[i], indent))
			}

		default:
			out = append(out, pad(indent)+todo, commentOut(line, indent))
		}
	}
	return strings.Join(out, "\n")
}

func isParentKey(line string, childIndent int) bool {
	t := strings.TrimSpace(line)
	return strings.HasSuffix(t, ":") && !strings.HasPrefix(t, "#") && leadingSpaces(line) < childIndent
}

func commentOut(line string, indent int) string {
	if len(line) < indent {
		return pad(indent) + "#"
	}
	return pad(indent) + "# " + line[indent:]
}

func leadingSpaces(s string) int {
	return len(s) - len(strings.TrimLeft(s, " "))
}

func pad(n int) string {
	return strings.Repeat(" ", n)
}

func normalizeHandle(h string) string {
	return strings.TrimPrefix(strings.TrimSpace(h), "@")
}

func containsFold(list []string, v string) bool {
	v = normalizeHandle(v)
	for _, s := range list {
		if strings.EqualFold(normalizeHandle(s), v) {
			return true
		}
	}
	return false
}
