package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// supportedProtocolVersions lists the MCP protocol revisions this server
// speaks, newest first.
var supportedProtocolVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

const (
	projectResourcePrefix = "landscape://project/"
	memberResourcePrefix  = "landscape://member/"
)

// negotiateProtocolVersion echoes the client's requested version when it is
// supported and otherwise answers with the latest version the server supports.
func negotiateProtocolVersion(params json.RawMessage) string {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if len(params) > 0 {
		// A malformed payload simply falls through to the latest version.
		_ = json.Unmarshal(params, &p)
	}
	for _, v := range supportedProtocolVersions {
		if v == p.ProtocolVersion {
			return v
		}
	}
	return supportedProtocolVersions[0]
}

// toolAnnotations describes every tool as a read-only, idempotent query over a
// closed dataset.
func toolAnnotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

func handleResourceTemplatesList(req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	templates := []map[string]string{
		{
			"uriTemplate": projectResourcePrefix + "{name}",
			"name":        fmt.Sprintf("%s Project Details", state.cfg.Name),
			"description": "Detailed information about a single project, looked up by name",
			"mimeType":    "application/json",
		},
		{
			"uriTemplate": memberResourcePrefix + "{name}",
			"name":        fmt.Sprintf("%s Member Details", state.cfg.Name),
			"description": "Detailed information about a single member, looked up by name",
			"mimeType":    "application/json",
		},
	}
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"resourceTemplates": templates}),
		ID:      req.ID,
	}
}

// resourceReader produces the text of one resource from a loaded dataset.
// notFoundIsInvalidParams marks readers whose errors mean the URI named
// something that does not exist (JSON-RPC -32602) rather than an internal fault.
type resourceReader struct {
	read                    func(ds *Dataset) (string, error)
	notFoundIsInvalidParams bool
}

// resolveResource maps a resource URI to its reader. It never touches the
// dataset, so malformed URIs are rejected without waiting for the load.
func resolveResource(uri string) (resourceReader, error) {
	switch {
	case uri == "landscape://categories":
		return resourceReader{read: listCategories}, nil
	case uri == "landscape://summary":
		return resourceReader{read: landscapeSummary}, nil
	case strings.HasPrefix(uri, projectResourcePrefix):
		name, err := entityName(uri, projectResourcePrefix)
		if err != nil {
			return resourceReader{}, err
		}
		return resourceReader{
			read:                    func(ds *Dataset) (string, error) { return getProjectDetails(ds, name) },
			notFoundIsInvalidParams: true,
		}, nil
	case strings.HasPrefix(uri, memberResourcePrefix):
		name, err := entityName(uri, memberResourcePrefix)
		if err != nil {
			return resourceReader{}, err
		}
		return resourceReader{
			read:                    func(ds *Dataset) (string, error) { return getMemberDetails(ds, name) },
			notFoundIsInvalidParams: true,
		}, nil
	default:
		return resourceReader{}, fmt.Errorf("unknown resource URI: %s", uri)
	}
}

func entityName(uri, prefix string) (string, error) {
	name, err := url.PathUnescape(strings.TrimPrefix(uri, prefix))
	if err != nil {
		return "", fmt.Errorf("invalid resource URI %s: %v", uri, err)
	}
	if name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("invalid resource URI: %s", uri)
	}
	return name, nil
}

func changesReportPrompt(cfgName, since string, now time.Time) (string, error) {
	if since == "" {
		since = now.AddDate(-1, 0, 0).Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", since); err != nil {
		return "", fmt.Errorf("invalid argument since: %q is not a YYYY-MM-DD date", since)
	}
	return fmt.Sprintf(`Please produce a report of changes in the %[1]s landscape since %[2]s. Use the available tools:
1. Use query_projects with graduated_from=%[2]s to find projects that graduated
2. Use query_projects with incubating_from=%[2]s to find projects that moved to incubating
3. Use query_projects with accepted_from=%[2]s to find newly accepted (new sandbox) projects
4. Use search_landscape and get_project_details to find archived projects (archived_at on or after %[2]s); query_projects excludes archived projects
5. Summarize promotions, new sandbox projects, and archived projects, with dates, and note overall trends`, cfgName, since), nil
}
