package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// maxQueryLimit is the largest page size accepted by query_projects/query_members.
	maxQueryLimit = 500
	// defaultQueryLimit is used when limit is missing or non-positive.
	defaultQueryLimit = 100
)

// toolHandler runs one tool against a loaded dataset. argsRaw is the raw
// "arguments" payload of the tools/call request (possibly empty).
type toolHandler func(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse

// toolSpec is the single source of truth for one tool: everything tools/list
// advertises and tools/call dispatches is derived from it.
type toolSpec struct {
	name string
	// descriptionFormat has one %s verb for the landscape name.
	descriptionFormat string
	// metrics, when non-empty, makes this a metric tool: the input schema is
	// the metric enum and handler is built by metricToolHandler.
	metrics    []string
	properties map[string]interface{}
	required   []string
	handler    toolHandler
}

func stringProp(description string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": description}
}

func integerProp(description string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": description}
}

const queryLimitDescription = "Maximum number of results to return (default: 100; values <= 0 use 100, values above 500 are clamped to 500)"

// toolSpecs lists every tool in the order tools/list advertises them.
var toolSpecs = []toolSpec{
	{
		name:              "query_projects",
		descriptionFormat: "Query %s projects with flexible filtering by maturity, dates, and name. Returns detailed project information. Archived projects are excluded. The total field is the number of matches ignoring limit/offset.",
		properties: map[string]interface{}{
			"maturity":        stringProp("Filter by maturity level (e.g., graduated, incubating, sandbox)"),
			"name":            stringProp("Filter by project name (case-insensitive substring match)"),
			"category":        stringProp("Filter by category (e.g., Provisioning, Runtime, Observability and Analysis)"),
			"subcategory":     stringProp("Filter by subcategory (e.g., Container Runtime, Service Mesh)"),
			"graduated_from":  stringProp("Filter projects graduated on or after this date (YYYY-MM-DD)"),
			"graduated_to":    stringProp("Filter projects graduated on or before this date (YYYY-MM-DD)"),
			"incubating_from": stringProp("Filter projects that reached incubating on or after this date (YYYY-MM-DD)"),
			"incubating_to":   stringProp("Filter projects that reached incubating on or before this date (YYYY-MM-DD)"),
			"accepted_from":   stringProp("Filter projects accepted on or after this date (YYYY-MM-DD)"),
			"accepted_to":     stringProp("Filter projects accepted on or before this date (YYYY-MM-DD)"),
			"limit":           integerProp(queryLimitDescription),
			"offset":          integerProp("Number of results to skip before returning (default: 0)"),
		},
		handler: handleQueryProjects,
	},
	{
		name:              "query_members",
		descriptionFormat: "Query %s members with flexible filtering by membership tier and join dates. Archived items are excluded. The total field is the number of matches ignoring limit/offset.",
		properties: map[string]interface{}{
			"tier":        stringProp("Filter by membership tier (e.g., Gold, Silver, Platinum, End User Supporter)"),
			"category":    stringProp("Filter by category (e.g., CNCF Members)"),
			"joined_from": stringProp("Filter members joined on or after this date (YYYY-MM-DD)"),
			"joined_to":   stringProp("Filter members joined on or before this date (YYYY-MM-DD)"),
			"limit":       integerProp(queryLimitDescription),
			"offset":      integerProp("Number of results to skip before returning (default: 0)"),
		},
		handler: handleQueryMembers,
	},
	{
		name:              "get_project_details",
		descriptionFormat: "Get detailed information about a specific %s project by name. An exact (case-insensitive) name match is preferred; otherwise substring matching is used and an error listing candidates is returned if it is ambiguous. Archived projects can be looked up.",
		properties: map[string]interface{}{
			"name": stringProp("Project name (case-insensitive; exact match preferred, otherwise unique substring)"),
		},
		required: []string{"name"},
		handler:  handleGetProjectDetails,
	},
	{
		name:              "get_member_details",
		descriptionFormat: "Get detailed information about a specific %s member by name.",
		properties: map[string]interface{}{
			"name": stringProp("Member name to search for (case-insensitive)"),
		},
		required: []string{"name"},
		handler:  handleGetMemberDetails,
	},
	{
		name:              "list_categories",
		descriptionFormat: "List all categories and subcategories in the %s landscape with item counts.",
		handler:           handleListCategories,
	},
	{
		name:              "landscape_summary",
		descriptionFormat: "Get a high-level statistical overview of the entire %s landscape. Archived items are excluded from totals.",
		handler:           handleLandscapeSummary,
	},
	{
		name:              "search_landscape",
		descriptionFormat: "Full-text search across all %s landscape items by name, description, category, subcategory, or homepage URL.",
		properties: map[string]interface{}{
			"query": stringProp("Search term (case-insensitive substring match)"),
			"limit": integerProp("Maximum results (default 20, max 100)"),
		},
		required: []string{"query"},
		handler:  handleSearchLandscape,
	},
	{
		name:              "compare_projects",
		descriptionFormat: "Side-by-side comparison of two or more %s projects. Each name is matched exactly (case-insensitive) first, then by substring; ambiguous substring matches return an error listing candidates.",
		properties: map[string]interface{}{
			"names": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Project names to compare (minimum 2)",
				"minItems":    2,
			},
		},
		required: []string{"names"},
		handler:  handleCompareProjects,
	},
	{
		name:              "funding_analysis",
		descriptionFormat: "Analyze funding data from Crunchbase for %s landscape members.",
		properties: map[string]interface{}{
			"tier": stringProp("Filter by membership tier (e.g. Gold, Silver)"),
			"year": integerProp("Filter funding rounds to a specific year"),
		},
		handler: handleFundingAnalysis,
	},
	{
		name:              "geographic_distribution",
		descriptionFormat: "Geographic breakdown of %s landscape members by Crunchbase location data.",
		properties: map[string]interface{}{
			"group_by": map[string]interface{}{
				"type":        "string",
				"description": "Group by: country (default), region, or city",
				"enum":        []string{"country", "region", "city"},
			},
		},
		handler: handleGeographicDistribution,
	},
	{
		name:              "project_metrics",
		descriptionFormat: "Answer questions about %s project counts and progress. Archived projects are excluded.",
		metrics: []string{
			"incubating_project_count",
			"sandbox_projects_joined_this_year",
			"projects_graduated_last_year",
		},
	},
	{
		name:              "membership_metrics",
		descriptionFormat: "Answer questions about %s membership activity and funding.",
		metrics: []string{
			"gold_members_joined_this_year",
			"silver_members_joined_this_year",
			"silver_members_raised_last_month",
			"gold_members_raised_this_year",
		},
	},
}

// toolDefinition is a toolSpec resolved for one landscape configuration.
type toolDefinition struct {
	Name        string
	Description string
	Metrics     []string
	InputSchema map[string]interface{}
	Handler     toolHandler
}

type toolCatalogData struct {
	catalog         map[string]toolDefinition
	advertisedTools []string
}

func buildToolCatalog(cfg LandscapeConfig) toolCatalogData {
	data := toolCatalogData{
		catalog:         make(map[string]toolDefinition, len(toolSpecs)),
		advertisedTools: make([]string, 0, len(toolSpecs)),
	}
	for _, spec := range toolSpecs {
		def := toolDefinition{
			Name:        spec.name,
			Description: fmt.Sprintf(spec.descriptionFormat, cfg.Name),
			Metrics:     spec.metrics,
			InputSchema: spec.inputSchema(),
			Handler:     spec.handler,
		}
		if len(spec.metrics) > 0 {
			def.Handler = metricToolHandler(spec.metrics)
		}
		data.catalog[spec.name] = def
		data.advertisedTools = append(data.advertisedTools, spec.name)
	}
	return data
}

func (spec toolSpec) inputSchema() map[string]interface{} {
	schema := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
	}
	if len(spec.metrics) > 0 {
		schema["properties"] = map[string]interface{}{
			"metric": map[string]interface{}{
				"type": "string",
				"enum": spec.metrics,
			},
		}
		schema["required"] = []string{"metric"}
		return schema
	}
	if spec.properties != nil {
		schema["properties"] = spec.properties
	}
	if spec.required != nil {
		schema["required"] = spec.required
	}
	return schema
}

func handleToolsList(req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	tools := make([]map[string]interface{}, 0, len(state.tools.advertisedTools))
	for _, name := range state.tools.advertisedTools {
		def, ok := state.tools.catalog[name]
		if !ok {
			continue
		}
		tools = append(tools, map[string]interface{}{
			"name":        def.Name,
			"description": def.Description,
			"inputSchema": def.InputSchema,
			"annotations": toolAnnotations(),
		})
	}
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result: mustJSON(map[string]interface{}{
			"tools": tools,
		}),
		ID: req.ID,
	}
}

func handleToolsCall(ctx context.Context, req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	var payload struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params", nil)
	}

	def, ok := state.tools.catalog[payload.Name]
	if !ok {
		return errorResponse(req.ID, -32602, fmt.Sprintf("Unknown tool: %s", payload.Name), nil)
	}

	waitCtx, cancel := context.WithTimeout(ctx, datasetWaitTimeout)
	defer cancel()
	dataset, err := state.waitForDataset(waitCtx)
	if err != nil {
		return toolErrorResult(req.ID, datasetErrorText(err))
	}

	return def.Handler(req.ID, payload.Arguments, dataset)
}

// metricToolHandler builds the handler shared by the metric-based tools, which
// accept a required "metric" argument restricted to the allowed list.
func metricToolHandler(allowed []string) toolHandler {
	return func(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
		var args struct {
			Metric string `json:"metric"`
		}
		if len(argsRaw) > 0 {
			if err := json.Unmarshal(argsRaw, &args); err != nil {
				return toolErrorResult(id, "Invalid arguments: "+err.Error())
			}
		}
		if args.Metric == "" {
			return toolErrorResult(id, fmt.Sprintf("Metric is required; allowed metrics: %s", strings.Join(allowed, ", ")))
		}
		if !metricAllowed(allowed, args.Metric) {
			return toolErrorResult(id, fmt.Sprintf("Unsupported metric %q; allowed metrics: %s", args.Metric, strings.Join(allowed, ", ")))
		}

		result, err := executeMetric(args.Metric, ds, time.Now().UTC())
		if err != nil {
			return toolErrorResult(id, err.Error())
		}
		return textResult(id, result)
	}
}

// clampLimit applies the query_projects/query_members page-size rules.
func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultQueryLimit
	}
	if limit > maxQueryLimit {
		return maxQueryLimit
	}
	return limit
}

func metricAllowed(allowed []string, metric string) bool {
	for _, m := range allowed {
		if m == metric {
			return true
		}
	}
	return false
}

func handleQueryProjects(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Maturity       string `json:"maturity"`
		Name           string `json:"name"`
		Category       string `json:"category"`
		Subcategory    string `json:"subcategory"`
		GraduatedFrom  string `json:"graduated_from"`
		GraduatedTo    string `json:"graduated_to"`
		IncubatingFrom string `json:"incubating_from"`
		IncubatingTo   string `json:"incubating_to"`
		AcceptedFrom   string `json:"accepted_from"`
		AcceptedTo     string `json:"accepted_to"`
		Limit          int    `json:"limit"`
		Offset         int    `json:"offset"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}
	args.Limit = clampLimit(args.Limit)

	result, err := queryProjects(ds, ProjectQuery{
		Maturity:       args.Maturity,
		Name:           args.Name,
		Category:       args.Category,
		Subcategory:    args.Subcategory,
		GraduatedFrom:  args.GraduatedFrom,
		GraduatedTo:    args.GraduatedTo,
		IncubatingFrom: args.IncubatingFrom,
		IncubatingTo:   args.IncubatingTo,
		AcceptedFrom:   args.AcceptedFrom,
		AcceptedTo:     args.AcceptedTo,
		Limit:          args.Limit,
		Offset:         args.Offset,
	})
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleQueryMembers(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Tier       string `json:"tier"`
		Category   string `json:"category"`
		JoinedFrom string `json:"joined_from"`
		JoinedTo   string `json:"joined_to"`
		Limit      int    `json:"limit"`
		Offset     int    `json:"offset"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}
	args.Limit = clampLimit(args.Limit)

	result, err := queryMembers(ds, MemberQuery{
		Tier:       args.Tier,
		Category:   args.Category,
		JoinedFrom: args.JoinedFrom,
		JoinedTo:   args.JoinedTo,
		Limit:      args.Limit,
		Offset:     args.Offset,
	})
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleGetProjectDetails(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Name string `json:"name"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}
	if args.Name == "" {
		return toolErrorResult(id, "name parameter is required")
	}

	result, err := getProjectDetails(ds, args.Name)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleGetMemberDetails(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Name string `json:"name"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}
	if args.Name == "" {
		return toolErrorResult(id, "name parameter is required")
	}

	result, err := getMemberDetails(ds, args.Name)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleListCategories(id json.RawMessage, _ json.RawMessage, ds *Dataset) *jsonRPCResponse {
	result, err := listCategories(ds)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleLandscapeSummary(id json.RawMessage, _ json.RawMessage, ds *Dataset) *jsonRPCResponse {
	result, err := landscapeSummary(ds)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleSearchLandscape(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}
	if args.Query == "" {
		return toolErrorResult(id, "query parameter is required")
	}

	result, err := searchLandscape(ds, args.Query, args.Limit)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleCompareProjects(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Names []string `json:"names"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}
	if len(args.Names) < 2 {
		return toolErrorResult(id, "at least 2 project names are required")
	}

	result, err := compareProjects(ds, args.Names)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleFundingAnalysis(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Tier string `json:"tier"`
		Year int    `json:"year"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}

	result, err := fundingAnalysis(ds, args.Tier, args.Year)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}

func handleGeographicDistribution(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		GroupBy string `json:"group_by"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return toolErrorResult(id, "Invalid arguments: "+err.Error())
		}
	}

	result, err := geographicDistribution(ds, args.GroupBy)
	if err != nil {
		return toolErrorResult(id, err.Error())
	}

	return textResult(id, result)
}
